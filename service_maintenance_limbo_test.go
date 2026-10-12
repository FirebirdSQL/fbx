package fbx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
)

type maintenanceClientScenario struct {
	Database            string
	Prepare             bool
	ParticipantDatabase string
	CommitParticipant   bool
}

type maintenanceClientReady struct {
	TransactionID int64
	AttachmentID  uint64
}

// Run the already compiled test binary as a client, then crash it after the
// server confirms the requested step. No driver cleanup can run in that process.
func crashMaintenanceClient(t *testing.T, ctx context.Context, config fbconn.Config, scenario maintenanceClientScenario) int64 {
	t.Helper()
	executable, err := os.Executable()
	require.NoError(t, err)
	input, err := json.Marshal(scenario)
	require.NoError(t, err)
	clientCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(clientCtx, executable, "-test.run=^TestServiceMaintenanceCrashClient$", "-test.parallel=1", "-test.timeout=1m")
	cmd.Env = append(os.Environ(), "FBX_TEST_MAINTENANCE_CLIENT=1")
	cmd.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			killErr := cmd.Process.Kill()
			// Wait reaps the client; its nonzero exit is expected after Kill.
			_ = cmd.Wait()
			if !errors.Is(killErr, os.ErrProcessDone) {
				require.NoError(t, killErr)
			}
		}
	})
	var ready maintenanceClientReady
	var output bytes.Buffer
	if err := json.NewDecoder(io.TeeReader(stdout, &output)).Decode(&ready); err != nil {
		waitErr := cmd.Wait()
		t.Fatalf("client failed before confirming its transaction: %v (exit: %v)\n%s%s", err, waitErr, output.String(), stderr.String())
	}
	require.Positive(t, ready.TransactionID)
	require.Positive(t, ready.AttachmentID)

	// MON$ snapshots are transaction-scoped. Every observer query is autocommit
	// so it sees the server's current attachment state rather than a cached view.
	config.Database = scenario.Database
	config.Autocommit = true
	observerCtx, observerCancel := context.WithTimeout(ctx, 10*time.Second)
	defer observerCancel()
	observer, err := fbconn.ConnectConfig(observerCtx, &config)
	require.NoError(t, err)
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, observer.Close(cleanup))
	}()
	countAttachments := func() int {
		result := observer.Exec(observerCtx, fmt.Sprintf("select cast(count(*) as varchar(20)) from MON$ATTACHMENTS where MON$ATTACHMENT_ID=%d", ready.AttachmentID)).Read()
		require.NoError(t, result.Err)
		require.Len(t, result.Rows, 1)
		require.Len(t, result.Rows[0], 1)
		count, err := strconv.Atoi(strings.TrimSpace(string(result.Rows[0][0])))
		require.NoError(t, err)
		return count
	}
	require.Equal(t, 1, countAttachments(), "the client must still be attached before the crash")
	require.NoError(t, cmd.Process.Kill())
	err = cmd.Wait()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, stderr.String())
	require.NoError(t, clientCtx.Err(), "client must be killed by the test, not by its timeout")

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for countAttachments() != 0 {
		select {
		case <-observerCtx.Done():
			t.Fatal("server did not release the crashed client's attachment")
		case <-ticker.C:
		}
	}
	return ready.TransactionID
}

// This entry point only runs in a subprocess created by crashMaintenanceClient.
func TestServiceMaintenanceCrashClient(t *testing.T) {
	if os.Getenv("FBX_TEST_MAINTENANCE_CLIENT") != "1" {
		t.Skip("only runs as the maintenance crash client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var scenario maintenanceClientScenario
	require.NoError(t, json.NewDecoder(os.Stdin).Decode(&scenario))
	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	config.Database = scenario.Database
	c, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, c.Close(cleanup))
	})
	var id int64
	if scenario.ParticipantDatabase != "" {
		id = prepareMaintenanceClientDistributedTransaction(t, ctx, c, *config, scenario)
	} else {
		tx, txID := maintenanceClientTransaction(t, ctx, c)
		id = txID
		if scenario.Prepare {
			prepareMaintenanceClientTransaction(t, ctx, c, tx, nil)
		}
	}
	require.NoError(t, json.NewEncoder(os.Stdout).Encode(maintenanceClientReady{TransactionID: id, AttachmentID: c.AttachmentID()}))
	<-ctx.Done()
	t.Fatal("parent did not crash the maintenance client")
}

func maintenanceClientTransaction(t *testing.T, ctx context.Context, c *fbconn.FbConn) (*fbconn.FBTransaction, int64) {
	t.Helper()
	tx, err := c.StartTransaction(ctx)
	require.NoError(t, err)
	result := c.ExecTx(ctx, tx, "select cast(current_transaction as varchar(30)) from RDB$DATABASE").Read()
	require.NoError(t, result.Err)
	require.Len(t, result.Rows, 1)
	require.Len(t, result.Rows[0], 1)
	id, err := strconv.ParseInt(strings.TrimSpace(string(result.Rows[0][0])), 10, 64)
	require.NoError(t, err)
	require.NoError(t, c.ExecTx(ctx, tx, fmt.Sprintf("insert into MAINT_DATA values (%d)", id)).Read().Err)
	return tx, id
}

func prepareMaintenanceClientTransaction(t *testing.T, ctx context.Context, c *fbconn.FbConn, tx *fbconn.FBTransaction, description []byte) {
	t.Helper()
	require.NoError(t, c.Client().WriteMessage(maintenancePrepareTransaction{handle: tx.Handle, description: description}))
	require.NoError(t, c.Client().Flush())
	message, err := c.ReceiveMessage(ctx)
	require.NoError(t, err)
	require.IsType(t, &fbproto.OpResponse{}, message)
}
