package fbconn

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestBuildRepairActionSPB(t *testing.T) {
	for name, tt := range map[string]struct {
		options RepairOptions
		want    []byte
	}{
		"validate":               {RepairOptions{Operation: RepairValidate}, []byte{108, 1, 0, 0, 0}},
		"full read only ignore":  {RepairOptions{Operation: RepairValidate, Full: true, ReadOnly: true, IgnoreChecksums: true}, []byte{108, 177, 0, 0, 0}},
		"sweep":                  {RepairOptions{Operation: RepairSweep, ParallelWorkers: 1}, []byte{108, 2, 0, 0, 0, 52, 1, 0, 0, 0}},
		"sweep ignore checksums": {RepairOptions{Operation: RepairSweep, IgnoreChecksums: true}, []byte{108, 34, 0, 0, 0}},
		"mend":                   {RepairOptions{Operation: RepairMend}, []byte{108, 4, 0, 0, 0}},
		"kill shadows":           {RepairOptions{Operation: RepairKillShadows}, []byte{108, 64, 0, 0, 0}},
		"ICU":                    {RepairOptions{Operation: RepairFixICU}, []byte{108, 0, 8, 0, 0}},
		"ODS":                    {RepairOptions{Operation: RepairUpgradeODS}, []byte{108, 0, 16, 0, 0}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := buildRepairActionSPB("db", tt.options, ServerVersion{Major: 6})
			require.NoError(t, err)
			require.Equal(t, append([]byte{3, 106, 2, 0, 'd', 'b'}, tt.want...), got)
		})
	}
	// Semantic restrictions belong to fbservice.
	_, err := buildRepairActionSPB("db", RepairOptions{Operation: RepairSweep, ReadOnly: true}, ServerVersion{Major: 6})
	require.NoError(t, err)
}

func TestBuildPropertiesActionSPB(t *testing.T) {
	for name, tt := range map[string]struct {
		options DatabaseProperties
		want    []byte
	}{
		"read only":     {DatabaseProperties{AccessMode: DatabaseAccessModeReadOnly}, []byte{13, 39}},
		"read write":    {DatabaseProperties{AccessMode: DatabaseAccessModeReadWrite}, []byte{13, 40}},
		"sync":          {DatabaseProperties{WriteMode: DatabaseWriteModeSync}, []byte{12, 38}},
		"async":         {DatabaseProperties{WriteMode: DatabaseWriteModeAsync}, []byte{12, 37}},
		"cache":         {DatabaseProperties{PageBuffers: new(258)}, []byte{5, 2, 1, 0, 0}},
		"default cache": {DatabaseProperties{PageBuffers: new(0)}, []byte{5, 0, 0, 0, 0}},
		"sweep":         {DatabaseProperties{SweepInterval: new(258)}, []byte{6, 2, 1, 0, 0}},
		"disable sweep": {DatabaseProperties{SweepInterval: new(0)}, []byte{6, 0, 0, 0, 0}},
		"dialect 1":     {DatabaseProperties{SQLDialect: 1}, []byte{14, 1, 0, 0, 0}},
		"dialect 3":     {DatabaseProperties{SQLDialect: 3}, []byte{14, 3, 0, 0, 0}},
		"full pages":    {DatabaseProperties{ReserveSpace: new(false)}, []byte{11, 35}},
		"reserve":       {DatabaseProperties{ReserveSpace: new(true)}, []byte{11, 36}},
		"activate":      {DatabaseProperties{ActivateShadow: true}, []byte{108, 0, 1, 0, 0}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := buildPropertiesActionSPB("db", tt.options)
			require.NoError(t, err)
			require.Equal(t, append([]byte{8, 106, 2, 0, 'd', 'b'}, tt.want...), got)
		})
	}
	for i, mode := range []DatabaseMode{DatabaseModeMulti, DatabaseModeSingle, DatabaseModeFull} {
		for j, method := range []ShutdownMethod{ShutdownForce, ShutdownDenyAttachments, ShutdownDenyTransactions} {
			got, err := buildPropertiesActionSPB("db", DatabaseProperties{Shutdown: &ShutdownOptions{Mode: mode, Method: method, Timeout: 258}})
			require.NoError(t, err)
			require.Equal(t, []byte{8, 106, 2, 0, 'd', 'b', 44, byte(i + 1), byte(41 + j), 2, 1, 0, 0}, got)
		}
	}
	for i, mode := range []DatabaseMode{DatabaseModeNormal, DatabaseModeMulti, DatabaseModeSingle} {
		got, err := buildPropertiesActionSPB("db", DatabaseProperties{OnlineMode: mode})
		require.NoError(t, err)
		require.Equal(t, []byte{8, 106, 2, 0, 'd', 'b', 45, byte(i)}, got)
	}
	_, err := buildPropertiesActionSPB("db", DatabaseProperties{SQLDialect: 3, PageBuffers: new(12)})
	require.NoError(t, err) // gfix semantics are checked by fbservice.
}

func TestBuildMaintenanceActionValidation(t *testing.T) {
	for _, database := range []string{"", "-?", "d\x00b", strings.Repeat("d", 65536)} {
		_, err := buildRepairActionSPB(database, RepairOptions{Operation: RepairValidate}, ServerVersion{Major: 6})
		require.Error(t, err)
		_, err = buildPropertiesActionSPB(database, DatabaseProperties{SQLDialect: 3})
		require.Error(t, err)
		_, err = buildResolveLimboActionSPB(database, 42, LimboCommit)
		require.Error(t, err)
	}
	for _, database := range []string{"./-db.fdb", "база с пробелами", strings.Repeat("d", 65535)} {
		_, err := buildRepairActionSPB(database, RepairOptions{Operation: RepairValidate}, ServerVersion{Major: 6})
		require.NoError(t, err)
	}
	for _, options := range []RepairOptions{
		{}, {Operation: 255}, {Operation: RepairSweep, ParallelWorkers: -1},
		{Operation: RepairSweep, SweepTimeout: -1}, {Operation: RepairSweep, SweepTimeout: 2147484},
	} {
		_, err := buildRepairActionSPB("db", options, ServerVersion{Major: 6})
		require.Error(t, err)
	}
	for _, options := range []DatabaseProperties{
		{AccessMode: 255}, {WriteMode: 255}, {OnlineMode: DatabaseModeFull}, {SQLDialect: 2},
		{PageBuffers: new(-1)}, {SweepInterval: new(-1)},
		{Shutdown: &ShutdownOptions{}},
		{Shutdown: &ShutdownOptions{Mode: DatabaseModeFull, Method: 255}},
		{Shutdown: &ShutdownOptions{Mode: DatabaseModeFull, Method: ShutdownForce, Timeout: -1}},
	} {
		_, err := buildPropertiesActionSPB("db", options)
		require.Error(t, err)
	}
}

func TestBuildResolveLimboActionSPB(t *testing.T) {
	for _, tt := range []struct {
		action       LimboTransactionAction
		tag32, tag64 byte
	}{
		{LimboCommit, 15, 49}, {LimboRollback, 34, 50}, {LimboRecover, 17, 51},
	} {
		for _, id := range []int64{1, math.MaxInt32, math.MaxInt32 + 1, math.MaxUint32, 1 << 40, math.MaxInt64} {
			got, err := buildResolveLimboActionSPB("db", id, tt.action)
			require.NoError(t, err)
			want := []byte{3, 106, 2, 0, 'd', 'b'}
			if id <= math.MaxInt32 {
				want = append(want, tt.tag32)
				want = binary.LittleEndian.AppendUint32(want, uint32(id))
			} else {
				want = append(want, tt.tag64)
				want = binary.LittleEndian.AppendUint64(want, uint64(id))
			}
			require.Equal(t, want, got)
		}
	}
	for _, id := range []int64{-1, 0} {
		_, err := buildResolveLimboActionSPB("db", id, LimboCommit)
		require.Error(t, err)
	}
	_, err := buildResolveLimboActionSPB("db", 1, 255)
	require.Error(t, err)
}

func TestLimboTransactionWriterFragments(t *testing.T) {
	// Single and distributed 32/64-bit IDs, participant IDs, all variable metadata.
	data, want := limboDetailsProbeData()
	for size := 1; size <= len(data); size++ {
		var w limboTransactionWriter
		for offset := 0; offset < len(data); offset += size {
			p := data[offset:min(offset+size, len(data))]
			n, err := w.Write(p)
			require.NoError(t, err)
			require.Equal(t, len(p), n)
		}
		require.Empty(t, w.pending)
		require.Equal(t, want, w.transactions)
	}
	for _, data := range [][]byte{{255}, {47, 255, 255, 255, 255, 255, 255, 255, 255}} {
		var w limboTransactionWriter
		_, err := w.Write(data)
		require.Error(t, err)
	}
}

func maintenanceOutputCalls() map[string]func(context.Context, *ServiceConn, io.Writer) error {
	return map[string]func(context.Context, *ServiceConn, io.Writer) error{
		"repair": func(ctx context.Context, c *ServiceConn, w io.Writer) error {
			return c.Repair(ctx, "db", RepairOptions{Operation: RepairValidate, Output: w})
		},
		"properties": func(ctx context.Context, c *ServiceConn, w io.Writer) error {
			return c.SetDatabaseProperties(ctx, "db", DatabaseProperties{SQLDialect: 3, Output: w})
		},
		"resolve": func(ctx context.Context, c *ServiceConn, w io.Writer) error {
			return c.ResolveLimboTransaction(ctx, "db", 42, LimboCommit, w)
		},
	}
}

func TestServiceConnMaintenanceOutput(t *testing.T) {
	for name, call := range maintenanceOutputCalls() {
		t.Run(name, func(t *testing.T) {
			client, server := net.Pipe()
			done := make(chan serviceActionProbeResult, 1)
			go func() {
				defer func() { _ = server.Close() }()
				done <- runServiceActionProbeServerWithOutput(server, "report\n")
			}()
			conn, ctx := newStatisticsProbeConnection(t, client)
			var out maintenanceTestWriter
			require.NoError(t, call(ctx, conn, &out))
			require.Equal(t, "report\n", out.String())
			require.False(t, out.closed, "the output belongs to the caller")
			require.False(t, conn.actionActive)
			require.NoError(t, conn.Close(ctx))
			require.NoError(t, (<-done).err)
		})
	}
}

func TestServiceConnLimboTransactions(t *testing.T) {
	data, _ := limboDetailsProbeData()
	for name, tt := range map[string]struct {
		chunks  [][]byte
		want    []int64
		wantErr string
	}{
		"empty":            {},
		"fragmented":       {chunks: [][]byte{{19, 42}, {0, 0, 0, 47, 0}, {0, 0, 0, 0, 1, 0, 0}}, want: []int64{42, 1 << 40}},
		"distributed":      {chunks: [][]byte{data}, want: []int64{math.MaxUint32, 42, 1 << 40, 2 << 40, 99, 100}},
		"truncated ID":     {chunks: [][]byte{{19, 1, 0}}, wantErr: "truncated transaction information"},
		"truncated string": {chunks: [][]byte{{26, 10, 0, 'a'}}, wantErr: "truncated transaction information"},
		"unknown tag":      {chunks: [][]byte{{255}}, wantErr: "unknown limbo information tag"},
	} {
		t.Run(name, func(t *testing.T) {
			client, server := net.Pipe()
			done := make(chan error, 1)
			go func() { defer func() { _ = server.Close() }(); done <- runLimboProbeServer(server, tt.chunks) }()
			conn, ctx := newStatisticsProbeConnection(t, client)
			ids, err := conn.LimboTransactions(ctx, "db")
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Nil(t, ids)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, ids)
				require.False(t, conn.actionActive)
			}
			require.NoError(t, conn.Close(ctx))
			serverErr := <-done
			if tt.wantErr == "" {
				require.NoError(t, serverErr)
			}
		})
	}
}

func runLimboProbeServer(conn net.Conn, chunks [][]byte) error {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return err
	}
	r, w := fbproto.NewXDRReader(conn), fbproto.NewXDRWriter(conn)
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
	spb, err := readProbeServiceStart(r, 42)
	if err != nil {
		return err
	}
	if !bytes.Equal(spb, []byte{3, 106, 2, 0, 'd', 'b', 108, 8, 0, 0, 0}) {
		return fmt.Errorf("unexpected limbo SPB: %v", spb)
	}
	if err := writeProbeResponse(w, 0); err != nil {
		return err
	}
	responses := [][]byte{{66, 0, 0, fbproto.Isc_info_svc_timeout, 1}, {66, 0, 0, fbproto.Isc_info_data_not_ready, 1}}
	for _, chunk := range chunks {
		response := []byte{66, byte(len(chunk)), byte(len(chunk) >> 8)}
		response = append(response, chunk...)
		responses = append(responses, append(response, 1))
	}
	responses = append(responses, []byte{66, 0, 0, 1})
	for _, response := range responses {
		_, items, _, err := readProbeServiceInfo(r, 42)
		if err != nil {
			return err
		}
		if !bytes.Equal(items, []byte{66}) {
			return fmt.Errorf("unexpected limbo query items: %v", items)
		}
		if err := writeProbeResponseData(w, 0, response); err != nil {
			return err
		}
	}
	_, _, err = readProbeServiceDetach(r)
	return err
}

func TestMaintenanceActionIntegerLimits(t *testing.T) {
	maxValue := int64(math.MaxInt32)
	value := int(maxValue)
	for _, properties := range []DatabaseProperties{
		{PageBuffers: &value}, {SweepInterval: &value},
		{Shutdown: &ShutdownOptions{Mode: DatabaseModeFull, Method: ShutdownForce, Timeout: value}},
	} {
		_, err := buildPropertiesActionSPB("db", properties)
		require.NoError(t, err)
	}
	if int64(int(maxValue+1)) != maxValue+1 {
		return
	}
	value = int(maxValue + 1)
	for _, properties := range []DatabaseProperties{
		{PageBuffers: &value}, {SweepInterval: &value},
		{Shutdown: &ShutdownOptions{Mode: DatabaseModeFull, Method: ShutdownForce, Timeout: value}},
	} {
		_, err := buildPropertiesActionSPB("db", properties)
		require.Error(t, err)
	}
	_, err := buildRepairActionSPB("db", RepairOptions{Operation: RepairSweep, ParallelWorkers: value}, ServerVersion{Major: 6})
	require.Error(t, err)
}

func TestServiceConnLimboTransactionsCancellation(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { _ = serverConn.Close() })
	queryStarted := make(chan struct{})
	serverDone := make(chan error, 1)
	go func() {
		defer func() { _ = serverConn.Close() }()
		serverDone <- runCanceledServiceActionProbeServer(serverConn, queryStarted, fbproto.Isc_action_svc_repair)
	}()
	conn, timeoutCtx := newStatisticsProbeConnection(t, clientConn)
	ctx, cancel := context.WithCancel(timeoutCtx)
	defer cancel()
	type result struct {
		ids []int64
		err error
	}
	done := make(chan result, 1)
	go func() {
		ids, err := conn.LimboTransactions(ctx, "db")
		done <- result{ids, err}
	}()
	select {
	case <-queryStarted:
	case <-timeoutCtx.Done():
		t.Fatal("limbo query did not start")
	}
	cancel()
	select {
	case got := <-done:
		require.ErrorIs(t, got.err, context.Canceled)
		require.Nil(t, got.ids)
	case <-timeoutCtx.Done():
		t.Fatal("limbo query did not stop after cancellation")
	}
	require.True(t, conn.conn.IsClosed())
	select {
	case <-conn.conn.CleanupDone():
	case <-timeoutCtx.Done():
		t.Fatal("service connection cleanup did not complete")
	}
	select {
	case err := <-serverDone:
		require.NoError(t, err)
	case <-timeoutCtx.Done():
		t.Fatal("server did not observe connection closure")
	}
	require.NoError(t, conn.Close(timeoutCtx))
}
