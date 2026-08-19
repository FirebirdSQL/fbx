package fbconn

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn/ctxwatch"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestBuildRestoreActionSPB(t *testing.T) {
	options := RestoreOptions{
		Verbose:                 true,
		Replace:                 true,
		AccessMode:              RestoreAccessModeReadOnly,
		DeactivateIndexes:       true,
		SkipShadows:             true,
		SkipValidityConstraints: true,
		CommitAfterEachTable:    true,
		UseAllPageSpace:         true,
		PageBuffers:             512,
		PageSize:                16384,
		ParallelWorkers:         4,
	}

	spb, err := buildRestoreActionSPB(
		"/backup/employee.fbk",
		"/data/employee.fdb",
		options,
	)
	require.NoError(t, err)

	offset := 0
	require.Equal(t, byte(fbproto.Isc_action_svc_restore), spb[offset])
	offset++
	offset = requireServiceActionString(t, spb, offset, fbproto.Isc_spb_dbname, "/data/employee.fdb")
	offset = requireServiceActionString(
		t,
		spb,
		offset,
		fbproto.Isc_spb_bkp_file,
		"/backup/employee.fbk",
	)
	require.Equal(t, byte(fbproto.Isc_spb_res_buffers), spb[offset])
	offset++
	require.Equal(t, uint32(512), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, byte(fbproto.Isc_spb_res_page_size), spb[offset])
	offset++
	require.Equal(t, uint32(16384), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, byte(fbproto.Isc_spb_res_access_mode), spb[offset])
	offset++
	require.Equal(t, byte(fbproto.Isc_spb_res_am_readonly), spb[offset])
	offset++
	require.Equal(t, byte(fbproto.Isc_spb_verbose), spb[offset])
	offset++
	require.Equal(t, byte(fbproto.Isc_spb_res_parallel_workers), spb[offset])
	offset++
	require.Equal(t, uint32(4), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, byte(fbproto.Isc_spb_options), spb[offset])
	offset++
	require.Equal(t, uint32(0x5f00), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, len(spb), offset)
}

func TestBuildRestoreActionSPBUsesCreateAndAccessModeFromBackupByDefault(t *testing.T) {
	spb, err := buildRestoreActionSPB(
		"/backup/employee.fbk",
		"/data/employee.fdb",
		RestoreOptions{},
	)
	require.NoError(t, err)

	offset := 1
	offset = requireServiceActionString(
		t,
		spb,
		offset,
		fbproto.Isc_spb_dbname,
		"/data/employee.fdb",
	)
	offset = requireServiceActionString(
		t,
		spb,
		offset,
		fbproto.Isc_spb_bkp_file,
		"/backup/employee.fbk",
	)
	require.Equal(t, byte(fbproto.Isc_spb_options), spb[offset])
	offset++
	require.Equal(
		t,
		uint32(fbproto.Isc_spb_res_create),
		binary.LittleEndian.Uint32(spb[offset:offset+4]),
	)
	offset += 4
	require.Equal(t, len(spb), offset)
}

func TestBuildRestoreActionSPBAccessMode(t *testing.T) {
	tests := map[string]struct {
		accessMode RestoreAccessMode
		wireMode   byte
	}{
		"read-only": {
			accessMode: RestoreAccessModeReadOnly,
			wireMode:   fbproto.Isc_spb_res_am_readonly,
		},
		"read-write": {
			accessMode: RestoreAccessModeReadWrite,
			wireMode:   fbproto.Isc_spb_res_am_readwrite,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			spb, err := buildRestoreActionSPB(
				"/backup/employee.fbk",
				"/data/employee.fdb",
				RestoreOptions{AccessMode: tt.accessMode},
			)
			require.NoError(t, err)

			offset := 1
			offset = requireServiceActionString(
				t,
				spb,
				offset,
				fbproto.Isc_spb_dbname,
				"/data/employee.fdb",
			)
			offset = requireServiceActionString(
				t,
				spb,
				offset,
				fbproto.Isc_spb_bkp_file,
				"/backup/employee.fbk",
			)
			require.Equal(t, byte(fbproto.Isc_spb_res_access_mode), spb[offset])
			offset++
			require.Equal(t, tt.wireMode, spb[offset])
			offset++
			require.Equal(t, byte(fbproto.Isc_spb_options), spb[offset])
		})
	}
}

func TestBuildRestoreActionSPBAcceptsMaximumValues(t *testing.T) {
	maxLengthValue := strings.Repeat("x", maxServiceActionStringLength)

	spb, err := buildRestoreActionSPB(
		maxLengthValue,
		maxLengthValue,
		RestoreOptions{
			PageBuffers:     maxServiceActionInt32,
			PageSize:        32768,
			ParallelWorkers: maxServiceActionInt32,
		},
	)
	require.NoError(t, err)

	offset := 1
	offset = requireServiceActionString(
		t,
		spb,
		offset,
		fbproto.Isc_spb_dbname,
		maxLengthValue,
	)
	offset = requireServiceActionString(
		t,
		spb,
		offset,
		fbproto.Isc_spb_bkp_file,
		maxLengthValue,
	)
	require.Equal(t, byte(fbproto.Isc_spb_res_buffers), spb[offset])
	offset++
	require.Equal(t, uint32(maxServiceActionInt32), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, byte(fbproto.Isc_spb_res_page_size), spb[offset])
	offset++
	require.Equal(t, uint32(32768), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, byte(fbproto.Isc_spb_res_parallel_workers), spb[offset])
	offset++
	require.Equal(t, uint32(maxServiceActionInt32), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, byte(fbproto.Isc_spb_options), spb[offset])
	offset++
	require.Equal(
		t,
		uint32(fbproto.Isc_spb_res_create),
		binary.LittleEndian.Uint32(spb[offset:offset+4]),
	)
	offset += 4
	require.Equal(t, len(spb), offset)
}

func TestBuildRestoreActionSPBValidation(t *testing.T) {
	type testCase struct {
		backupFile string
		database   string
		options    RestoreOptions
		wantError  string
	}

	tests := map[string]testCase{
		"empty backup file": {
			database:  "/data/employee.fdb",
			wantError: "restore backup file must not be empty",
		},
		"empty database": {
			backupFile: "/backup/employee.fbk",
			wantError:  "restore database must not be empty",
		},
		"backup file is too long": {
			backupFile: strings.Repeat("b", maxServiceActionStringLength+1),
			database:   "/data/employee.fdb",
			wantError:  "maximum is 65535",
		},
		"database is too long": {
			backupFile: "/backup/employee.fbk",
			database:   strings.Repeat("d", maxServiceActionStringLength+1),
			wantError:  "maximum is 65535",
		},
		"negative page buffers": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{PageBuffers: -1},
			wantError:  "restore page buffers must be between",
		},
		"invalid page size": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{PageSize: 12345},
			wantError:  "restore page size must be one of",
		},
		"negative parallel workers": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{ParallelWorkers: -1},
			wantError:  "restore parallel workers must be between",
		},
		"invalid access mode": {
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{AccessMode: RestoreAccessMode(255)},
			wantError:  "invalid restore access mode 255",
		},
	}
	if strconv.IntSize > 32 {
		aboveMaxInt32 := int64(maxServiceActionInt32) + 1
		tests["page buffers overflow"] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{PageBuffers: int(aboveMaxInt32)},
			wantError:  "restore page buffers must be between",
		}
		tests["parallel workers overflow"] = testCase{
			backupFile: "/backup/employee.fbk",
			database:   "/data/employee.fdb",
			options:    RestoreOptions{ParallelWorkers: int(aboveMaxInt32)},
			wantError:  "restore parallel workers must be between",
		}
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := buildRestoreActionSPB(tt.backupFile, tt.database, tt.options)
			require.ErrorContains(t, err, tt.wantError)
		})
	}
}

func TestServiceConnRestore(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	resultCh := make(chan serviceActionProbeResult, 1)
	go func() {
		defer func() { _ = serverConn.Close() }()
		resultCh <- runServiceActionProbeServer(serverConn)
	}()

	config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee")
	require.NoError(t, err)
	config.AuthPluginList = []string{"Legacy_Auth"}
	config.LookupFunc = func(context.Context, string) ([]string, error) {
		return []string{"127.0.0.1"}, nil
	}
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		return clientConn, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := ConnectServiceConfig(ctx, testServiceConfig(config))
	require.NoError(t, err)

	var output bytes.Buffer
	err = conn.Restore(
		context.Background(),
		"/backup/employee.fbk",
		"/data/employee.fdb",
		RestoreOptions{
			Output:  &output,
			Verbose: true,
		},
	)
	require.NoError(t, err)
	require.Equal(t, "gbak: writing data\n", output.String())
	require.False(t, conn.actionActive)
	require.NoError(t, conn.Close(context.Background()))

	select {
	case result := <-resultCh:
		require.NoError(t, result.err)
		require.Equal(t, byte(fbproto.Isc_action_svc_restore), result.actionSPB[0])
		require.Equal(t, 42, result.detachedHandle)
		require.Equal(t, fbproto.Op_disconnect, result.finalOpcode)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestServiceConnRestoreCanBeCanceledWhileWaitingForOutput(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	queryStarted := make(chan struct{})
	serverDone := make(chan error, 1)
	go func() {
		defer func() { _ = serverConn.Close() }()
		serverDone <- runCanceledRestoreProbeServer(serverConn, queryStarted)
	}()

	config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee")
	require.NoError(t, err)
	config.AuthPluginList = []string{"Legacy_Auth"}
	config.LookupFunc = func(context.Context, string) ([]string, error) {
		return []string{"127.0.0.1"}, nil
	}
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		return clientConn, nil
	}
	config.BuildContextWatcherHandler = func(conn *FbConn) ctxwatch.Handler {
		return &DeadlineContextWatcherHandler{Conn: conn.conn}
	}

	conn, err := ConnectServiceConfig(context.Background(), testServiceConfig(config))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	restoreDone := make(chan error, 1)
	go func() {
		restoreDone <- conn.Restore(
			ctx,
			"/backup/employee.fbk",
			"/data/employee.fdb",
			RestoreOptions{},
		)
	}()

	select {
	case <-queryStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("restore did not start output polling")
	}
	cancel()

	select {
	case err := <-restoreDone:
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorContains(t, err, "restore database")
	case <-time.After(2 * time.Second):
		t.Fatal("restore did not stop after context cancellation")
	}
	require.True(t, conn.conn.IsClosed())
	select {
	case <-conn.conn.CleanupDone():
	case <-time.After(2 * time.Second):
		t.Fatal("service connection cleanup did not complete")
	}
	require.NoError(t, <-serverDone)
	require.NoError(t, conn.Close(context.Background()))
}

func runCanceledRestoreProbeServer(conn net.Conn, queryStarted chan<- struct{}) error {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return err
	}
	r := fbproto.NewXDRReader(conn)
	w := fbproto.NewXDRWriter(conn)
	if err := readProbeConnect(r); err != nil {
		return err
	}
	if err := writeAuthenticatedProbeAccept(w, encoding); err != nil {
		return err
	}
	if err := readProbeServiceAttach(r, encoding); err != nil {
		return err
	}
	if err := writeProbeResponse(w, 42); err != nil {
		return err
	}
	actionSPB, err := readProbeServiceStart(r, 42)
	if err != nil {
		return err
	}
	if len(actionSPB) == 0 || actionSPB[0] != fbproto.Isc_action_svc_restore {
		return errors.New("expected restore service action")
	}
	if err := writeProbeResponse(w, 0); err != nil {
		return err
	}
	if _, _, _, err := readProbeServiceInfo(r, 42); err != nil {
		return err
	}
	close(queryStarted)

	_, err = r.ReadInt32()
	if !errors.Is(err, io.EOF) {
		return errors.New("client did not close after context cancellation")
	}
	return nil
}
