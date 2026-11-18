package fbconn

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

func TestServiceConnNBackup(t *testing.T) {
	client, server := net.Pipe()
	resultCh := make(chan serviceActionProbeResult, 1)
	go func() {
		defer func() { _ = server.Close() }()
		resultCh <- runServiceActionProbeServerWithOutput(server, "backup complete\n")
	}()
	conn, ctx := newStatisticsProbeConnection(t, client)
	var output bytes.Buffer
	require.NoError(t, conn.NBackup(ctx, "db", "full.nbk", NBackupOptions{Output: &output}))
	require.Equal(t, "backup complete\n", output.String())
	require.NoError(t, conn.Close(ctx))
	result := <-resultCh
	require.NoError(t, result.err)
	// isc_action_svc_nbak, dbname, nbk_file, nbk_level=0.
	require.Equal(t, []byte{20, 106, 2, 0, 'd', 'b', 6, 8, 0, 'f', 'u', 'l', 'l', '.', 'n', 'b', 'k', 5, 0, 0, 0, 0}, result.actionSPB)
}

func TestServiceConnNBackupOptions(t *testing.T) {
	const guid = "{01234567-89AB-CDEF-0123-456789ABCDEF}"
	for name, tt := range map[string]struct {
		options NBackupOptions
		want    []byte
	}{
		"incremental":   {NBackupOptions{Level: 2}, []byte{5, 2, 0, 0, 0}},
		"maximum level": {NBackupOptions{Level: 32767}, []byte{5, 255, 127, 0, 0}},
		"GUID":          {NBackupOptions{GUID: guid}, append([]byte{8, 38, 0}, guid...)},
		"no triggers":   {NBackupOptions{NoDatabaseTriggers: true}, []byte{5, 0, 0, 0, 0, 108, 1, 0, 0, 0}},
		"direct ON":     {NBackupOptions{DirectIO: new(true)}, []byte{5, 0, 0, 0, 0, 7, 2, 0, 'O', 'N'}},
		"direct OFF":    {NBackupOptions{DirectIO: new(false)}, []byte{5, 0, 0, 0, 0, 7, 3, 0, 'O', 'F', 'F'}},
		"keep days":     {NBackupOptions{CleanHistory: true, KeepDays: 7}, []byte{5, 0, 0, 0, 0, 9, 10, 7, 0, 0, 0}},
		"keep rows":     {NBackupOptions{CleanHistory: true, KeepRows: 258}, []byte{5, 0, 0, 0, 0, 9, 11, 2, 1, 0, 0}},
		"maximum days":  {NBackupOptions{CleanHistory: true, KeepDays: 1<<31 - 1}, []byte{5, 0, 0, 0, 0, 9, 10, 255, 255, 255, 127}},
		"maximum rows":  {NBackupOptions{CleanHistory: true, KeepRows: 1<<31 - 1}, []byte{5, 0, 0, 0, 0, 9, 11, 255, 255, 255, 127}},
	} {
		t.Run(name, func(t *testing.T) {
			client, server := net.Pipe()
			t.Cleanup(func() { _ = server.Close() })
			resultCh := make(chan serviceActionProbeResult, 1)
			go func() {
				defer func() { _ = server.Close() }()
				resultCh <- runServiceActionProbeServerWithOutput(server, "done\n")
			}()
			conn, ctx := newStatisticsProbeConnection(t, client)
			require.NoError(t, conn.NBackup(ctx, "db", "я", tt.options))
			require.NoError(t, conn.Close(ctx))
			result := <-resultCh
			require.NoError(t, result.err)
			// Literal wire values, including the UTF-8 byte length of the file name.
			require.Equal(t, append([]byte{20, 106, 2, 0, 'd', 'b', 6, 2, 0, 0xd1, 0x8f}, tt.want...), result.actionSPB)
		})
	}
}

func TestServiceConnNBackupValidationBeforeStart(t *testing.T) {
	var conn *ServiceConn
	tests := map[string]struct {
		database, file string
		options        NBackupOptions
		want           string
	}{
		"empty database":            {"", "b", NBackupOptions{}, "database"},
		"NUL database":              {"d\x00b", "b", NBackupOptions{}, "NUL"},
		"empty file":                {"db", "", NBackupOptions{}, "backup file"},
		"NUL file":                  {"db", "b\x00x", NBackupOptions{}, "NUL"},
		"stdout":                    {"db", "stdout", NBackupOptions{}, "server-side"},
		"long file":                 {"db", strings.Repeat("я", 32768), NBackupOptions{}, "65535"},
		"negative level":            {"db", "b", NBackupOptions{Level: -1}, "level"},
		"level overflow":            {"db", "b", NBackupOptions{Level: 32768}, "level"},
		"GUID with level":           {"db", "b", NBackupOptions{Level: 1, GUID: "{01234567-89AB-CDEF-0123-456789ABCDEF}"}, "mutually exclusive"},
		"malformed GUID":            {"db", "b", NBackupOptions{GUID: "not a GUID"}, "GUID must"},
		"GUID without braces":       {"db", "b", NBackupOptions{GUID: "01234567-89AB-CDEF-0123-456789ABCDEF"}, "GUID must"},
		"GUID with suffix":          {"db", "b", NBackupOptions{GUID: "{01234567-89AB-CDEF-0123-456789ABCDEF}x"}, "GUID must"},
		"GUID not hex":              {"db", "b", NBackupOptions{GUID: "{Z1234567-89AB-CDEF-0123-456789ABCDEF}"}, "GUID must"},
		"history without retention": {"db", "b", NBackupOptions{CleanHistory: true}, "requires exactly one"},
		"days without cleanup":      {"db", "b", NBackupOptions{KeepDays: 7}, "requires exactly one"},
		"rows without cleanup":      {"db", "b", NBackupOptions{KeepRows: 7}, "requires exactly one"},
		"both retention units":      {"db", "b", NBackupOptions{CleanHistory: true, KeepDays: 7, KeepRows: 1}, "mutually exclusive"},
		"negative days":             {"db", "b", NBackupOptions{CleanHistory: true, KeepDays: -1}, "between 0"},
		"negative rows":             {"db", "b", NBackupOptions{CleanHistory: true, KeepRows: -1}, "between 0"},
		"total overflow":            {"db", strings.Repeat("b", 65522), NBackupOptions{}, "action SPB"},
	}
	if strconv.IntSize > 32 {
		overflow := int64(1 << 31)
		tests["days overflow"] = struct {
			database, file string
			options        NBackupOptions
			want           string
		}{
			"db", "b", NBackupOptions{CleanHistory: true, KeepDays: int(overflow)}, "between 0"}
		tests["rows overflow"] = struct {
			database, file string
			options        NBackupOptions
			want           string
		}{
			"db", "b", NBackupOptions{CleanHistory: true, KeepRows: int(overflow)}, "between 0"}
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, conn.NBackup(context.Background(), tt.database, tt.file, tt.options), tt.want)
		})
	}
}

func TestServiceConnNRestoreAndNFixupValidationBeforeStart(t *testing.T) {
	var conn *ServiceConn
	ctx := context.Background()
	for _, database := range []string{"", "db\x00suffix", strings.Repeat("d", 65536)} {
		err := conn.NRestore(ctx, []string{"a"}, database, NRestoreOptions{})
		require.ErrorContains(t, err, "database")
		require.ErrorContains(t, conn.NFixup(ctx, database, NFixupOptions{}), "database")
	}
	for _, files := range [][]string{nil, {}, {""}, {"a", ""}, {"a\x00b"}, {"stdin"}, {strings.Repeat("f", 65536)}, {strings.Repeat("f", 32763), strings.Repeat("f", 32763)}} {
		err := conn.NRestore(ctx, files, "db", NRestoreOptions{})
		require.Error(t, err)
		require.NotContains(t, err.Error(), "connection is nil")
	}
	require.ErrorContains(t, conn.NFixup(ctx, strings.Repeat("d", 65532), NFixupOptions{}), "action SPB")
	require.ErrorContains(t, conn.NFixup(ctx, strings.Repeat("d", 65527), NFixupOptions{PreserveSequence: true}), "action SPB")
}

func nbackupOutputCalls() map[string]func(context.Context, *ServiceConn, io.Writer) error {
	return map[string]func(context.Context, *ServiceConn, io.Writer) error{
		"nbackup": func(ctx context.Context, c *ServiceConn, out io.Writer) error {
			return c.NBackup(ctx, "db", "backup", NBackupOptions{Output: out})
		},
		"nrestore": func(ctx context.Context, c *ServiceConn, out io.Writer) error {
			return c.NRestore(ctx, []string{"backup"}, "db", NRestoreOptions{Output: out})
		},
		"nfixup": func(ctx context.Context, c *ServiceConn, out io.Writer) error {
			return c.NFixup(ctx, "db", NFixupOptions{Output: out})
		},
	}
}

func TestServiceConnNBackupSPBLengthBoundaries(t *testing.T) {
	var conn *ServiceConn
	const guid = "{01234567-89AB-CDEF-0123-456789ABCDEF}"
	for _, tt := range []struct {
		options NBackupOptions
		maxFile int
	}{
		{NBackupOptions{}, 65521},
		{NBackupOptions{NoDatabaseTriggers: true}, 65516},
		{NBackupOptions{DirectIO: new(true)}, 65516},
		{NBackupOptions{DirectIO: new(false)}, 65515},
		{NBackupOptions{CleanHistory: true, KeepRows: 1}, 65515},
		{NBackupOptions{GUID: guid}, 65485},
		{NBackupOptions{GUID: guid, NoDatabaseTriggers: true, DirectIO: new(false), CleanHistory: true, KeepDays: 1}, 65468},
	} {
		require.ErrorContains(t, conn.NBackup(context.Background(), "db", strings.Repeat("f", tt.maxFile), tt.options), "connection is nil")
		require.ErrorContains(t, conn.NBackup(context.Background(), "db", strings.Repeat("f", tt.maxFile+1), tt.options), "action SPB")
	}
	for _, tt := range []struct {
		options NRestoreOptions
		maxFile int
	}{
		{NRestoreOptions{}, 65526},
		{NRestoreOptions{InPlace: true}, 65521},
		{NRestoreOptions{NoDatabaseTriggers: true, InPlace: true, PreserveSequence: true}, 65521},
	} {
		require.ErrorContains(t, conn.NRestore(context.Background(), []string{strings.Repeat("f", tt.maxFile)}, "db", tt.options), "connection is nil")
		require.ErrorContains(t, conn.NRestore(context.Background(), []string{strings.Repeat("f", tt.maxFile+1)}, "db", tt.options), "action SPB")
	}
	for _, preserve := range []bool{false, true} {
		maxDatabase := 65531
		if preserve {
			maxDatabase = 65526
		}
		require.ErrorContains(t, conn.NFixup(context.Background(), strings.Repeat("d", maxDatabase), NFixupOptions{PreserveSequence: preserve}), "connection is nil")
		require.ErrorContains(t, conn.NFixup(context.Background(), strings.Repeat("d", maxDatabase+1), NFixupOptions{PreserveSequence: preserve}), "action SPB")
	}
}

func TestServiceConnNBackupOutputFailures(t *testing.T) {
	for name, call := range nbackupOutputCalls() {
		for _, short := range []bool{false, true} {
			t.Run(name+"/short="+strconv.FormatBool(short), func(t *testing.T) {
				client, server := net.Pipe()
				t.Cleanup(func() { _ = server.Close() })
				done := make(chan error, 1)
				go func() {
					defer func() { _ = server.Close() }()
					done <- runBackupOutputProbeServer(server)
				}()
				conn, ctx := newStatisticsProbeConnection(t, client)
				want := errors.New("writer failed")
				writer := &maintenanceTestWriter{err: want}
				if short {
					writer.err, writer.short = nil, true
					want = io.ErrShortWrite
				}
				err := call(ctx, conn, writer)
				require.ErrorIs(t, err, want)
				require.False(t, SafeToRetry(err))
				require.False(t, writer.closed)
				require.NoError(t, conn.Close(ctx))
				require.NoError(t, <-done)
			})
		}
	}
}

func TestServiceConnNBackupNetworkCancellation(t *testing.T) {
	for name, call := range nbackupOutputCalls() {
		t.Run(name, func(t *testing.T) {
			client, server := net.Pipe()
			t.Cleanup(func() { _ = server.Close() })
			started, done := make(chan struct{}), make(chan error, 1)
			actions := map[string]byte{"nbackup": fbproto.Isc_action_svc_nbak, "nrestore": fbproto.Isc_action_svc_nrest, "nfixup": fbproto.Isc_action_svc_nfix}
			go func() {
				defer func() { _ = server.Close() }()
				done <- runCanceledServiceActionProbeServer(server, started, actions[name])
			}()
			conn, deadline := newStatisticsProbeConnection(t, client)
			ctx, cancel := context.WithCancel(deadline)
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- call(ctx, conn, nil) }()
			select {
			case <-started:
				cancel()
			case <-deadline.Done():
				t.Fatal(deadline.Err())
			}
			select {
			case err := <-result:
				require.ErrorIs(t, err, context.Canceled)
				require.False(t, SafeToRetry(err))
			case <-deadline.Done():
				t.Fatal(deadline.Err())
			}
			require.True(t, conn.conn.IsClosed())
			select {
			case <-conn.conn.CleanupDone():
			case <-deadline.Done():
				t.Fatal("connection cleanup did not finish")
			}
			require.NoError(t, <-done)
			require.NoError(t, conn.Close(deadline))
		})
	}
}

func TestServiceConnNRestoreAndNFixup(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func(context.Context, *ServiceConn) error
		spb  []byte
	}{
		{"restore chain", func(ctx context.Context, c *ServiceConn) error {
			return c.NRestore(ctx, []string{"a", "b"}, "db", NRestoreOptions{})
		}, []byte{21, 106, 2, 0, 'd', 'b', 6, 1, 0, 'a', 6, 1, 0, 'b'}},
		{"restore options", func(ctx context.Context, c *ServiceConn) error {
			return c.NRestore(ctx, []string{"b"}, "db", NRestoreOptions{NoDatabaseTriggers: true, InPlace: true, PreserveSequence: true})
		}, []byte{21, 106, 2, 0, 'd', 'b', 6, 1, 0, 'b', 108, 7, 0, 0, 0}},
		{"fixup", func(ctx context.Context, c *ServiceConn) error {
			return c.NFixup(ctx, "db", NFixupOptions{})
		}, []byte{31, 106, 2, 0, 'd', 'b'}},
		{"fixup preserve sequence", func(ctx context.Context, c *ServiceConn) error {
			return c.NFixup(ctx, "db", NFixupOptions{PreserveSequence: true})
		}, []byte{31, 106, 2, 0, 'd', 'b', 108, 4, 0, 0, 0}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client, server := net.Pipe()
			resultCh := make(chan serviceActionProbeResult, 1)
			go func() {
				defer func() { _ = server.Close() }()
				resultCh <- runServiceActionProbeServerWithOutput(server, "done\n")
			}()
			conn, ctx := newStatisticsProbeConnection(t, client)
			require.NoError(t, tt.run(ctx, conn))
			require.NoError(t, conn.Close(ctx))
			result := <-resultCh
			require.NoError(t, result.err)
			require.Equal(t, tt.spb, result.actionSPB)
		})
	}
}
