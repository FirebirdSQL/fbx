package fbconn

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

func TestBuildTraceStartActionSPB(t *testing.T) {
	spb, err := buildTraceStartActionSPB("т", "config\n")
	require.NoError(t, err)
	require.Equal(t, []byte{22, 3, 7, 0, 'c', 'o', 'n', 'f', 'i', 'g', '\n', 2, 2, 0, 0xd1, 0x82}, spb)
	spb, err = buildTraceStartActionSPB("", "c")
	require.NoError(t, err)
	require.Equal(t, []byte{22, 3, 1, 0, 'c'}, spb)
	for _, test := range []struct{ name, configuration string }{
		{"", ""}, {"bad\x00name", "config"}, {"name", "bad\x00config"},
		{strings.Repeat("n", 65536), "c"}, {"name", strings.Repeat("c", 65536)},
	} {
		_, err := buildTraceStartActionSPB(test.name, test.configuration)
		require.Error(t, err)
	}
	spb, err = buildTraceStartActionSPB("", strings.Repeat("c", 65531))
	require.NoError(t, err)
	require.Equal(t, uint16(65531), binary.LittleEndian.Uint16(spb[2:4]))
	// String lengths count bytes rather than Unicode characters.
	_, err = buildTraceStartActionSPB(strings.Repeat("т", 32768), "c")
	require.Error(t, err)
}

func TestBuildTraceStartActionSPBTotalLength(t *testing.T) {
	for _, test := range []struct {
		label, name, configuration string
		valid                      bool
	}{
		{"unnamed limit", "", strings.Repeat("c", 65531), true},
		{"unnamed overflow", "", strings.Repeat("c", 65532), false},
		{"maximum string", "", strings.Repeat("c", 65535), false},
		{"named limit", "n", strings.Repeat("c", 65527), true},
		{"named overflow", "n", strings.Repeat("c", 65528), false},
		{"long name limit", strings.Repeat("n", 65527), "c", true},
		{"long name overflow", strings.Repeat("n", 65528), "c", false},
		{"combined limit", strings.Repeat("n", 32764), strings.Repeat("c", 32764), true},
		{"combined overflow", strings.Repeat("n", 32765), strings.Repeat("c", 32764), false},
		{"UTF8 name limit", "т", strings.Repeat("c", 65526), true},
		{"UTF8 name overflow", "т", strings.Repeat("c", 65527), false},
		{"UTF8 configuration limit", "", strings.Repeat("т", 32765) + "c", true},
		{"UTF8 configuration overflow", "", strings.Repeat("т", 32766), false},
	} {
		t.Run(test.label, func(t *testing.T) {
			spb, err := buildTraceStartActionSPB(test.name, test.configuration)
			if test.valid {
				require.NoError(t, err)
				require.Len(t, spb, 65535)
			} else {
				require.ErrorContains(t, err, "trace start SPB")
				require.Nil(t, spb)
			}
		})
	}
}

func TestServiceConnTraceActions(t *testing.T) {
	for _, test := range []struct {
		name string
		spb  []byte
		run  func(*ServiceConn, context.Context, TraceOptions) error
	}{
		{"start", []byte{22, 3, 1, 0, 'c', 2, 1, 0, 'n'}, func(c *ServiceConn, ctx context.Context, options TraceOptions) error {
			return c.TraceStart(ctx, "n", "c", options)
		}},
		{"stop", []byte{23, 1, 4, 3, 2, 1}, func(c *ServiceConn, ctx context.Context, options TraceOptions) error {
			return c.TraceStop(ctx, 0x01020304, options)
		}},
		{"suspend", []byte{24, 1, 42, 0, 0, 0}, func(c *ServiceConn, ctx context.Context, options TraceOptions) error {
			return c.TraceSuspend(ctx, 42, options)
		}},
		{"resume", []byte{25, 1, 255, 255, 255, 127}, func(c *ServiceConn, ctx context.Context, options TraceOptions) error {
			return c.TraceResume(ctx, 1<<31-1, options)
		}},
		{"list", []byte{26}, (*ServiceConn).TraceList},
	} {
		t.Run(test.name, func(t *testing.T) {
			client, server := net.Pipe()
			resultCh := make(chan serviceActionProbeResult, 1)
			// Include empty polls, partial lines, split UTF-8, and data followed by
			// truncation. Every byte must reach the writer exactly once.
			responses := [][]byte{
				{fbproto.Isc_info_truncated},
				{fbproto.Isc_info_svc_to_eof, 0, 0, fbproto.Isc_info_svc_timeout},
				serviceOutputResponse([]byte("Trace session ID ")),
				{fbproto.Isc_info_svc_to_eof, 0, 0, fbproto.Isc_info_data_not_ready, fbproto.Isc_info_end},
				append(serviceOutputResponse([]byte("42 started\n\xd1")), fbproto.Isc_info_truncated),
				serviceOutputResponse([]byte("\x82\x00event\r\n")),
				{fbproto.Isc_info_svc_to_eof, 0, 0, fbproto.Isc_info_end},
			}
			go func() {
				defer func() { _ = server.Close() }()
				resultCh <- runServiceActionProbeServerWithResponses(server, responses)
			}()
			conn, ctx := newStatisticsProbeConnection(t, client)
			var output bytes.Buffer
			require.NoError(t, test.run(conn, ctx, TraceOptions{Output: &output}))
			require.Equal(t, "Trace session ID 42 started\nт\x00event\r\n", output.String())
			require.False(t, conn.actionActive)
			require.NoError(t, conn.Close(ctx))
			select {
			case result := <-resultCh:
				require.NoError(t, result.err)
				require.Equal(t, test.spb, result.actionSPB)
				require.Equal(t, []byte{fbproto.Isc_info_svc_to_eof}, result.receiveItems)
				require.Equal(t, []int{1024, 2048, 2048, 2048, 2048, 2048, 2048}, result.bufferLengths)
				require.Equal(t, 42, result.detachedHandle)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
	}
}

func TestServiceConnTraceRejectsInvalidIDs(t *testing.T) {
	ids := []int{-1, 0}
	if strconv.IntSize > 32 {
		overflow := int64(1) << 31
		ids = append(ids, int(overflow))
	}
	var conn *ServiceConn
	for _, run := range []func(context.Context, int, TraceOptions) error{conn.TraceStop, conn.TraceSuspend, conn.TraceResume} {
		for _, id := range ids {
			require.ErrorContains(t, run(context.Background(), id, TraceOptions{}), "trace session ID")
		}
	}
}

type traceShortWriter struct{}

func (traceShortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestServiceConnTraceOutputError(t *testing.T) {
	for _, output := range []io.Writer{errorWriter{}, traceShortWriter{}} {
		client, server := net.Pipe()
		done := make(chan error, 1)
		go func() {
			defer func() { _ = server.Close() }()
			done <- runBackupOutputProbeServer(server)
		}()
		conn, ctx := newStatisticsProbeConnection(t, client)
		err := conn.TraceStart(ctx, "name", "config", TraceOptions{Output: output})
		require.ErrorContains(t, err, "trace start: write service output")
		if _, short := output.(traceShortWriter); short {
			require.ErrorIs(t, err, io.ErrShortWrite)
		}
		require.NoError(t, conn.Close(ctx))
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
}

func TestServiceConnTraceCancellation(t *testing.T) {
	client, server := net.Pipe()
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		defer func() { _ = server.Close() }()
		done <- runCanceledServiceActionProbeServer(server, started, fbproto.Isc_action_svc_trace_start)
	}()
	conn, deadline := newStatisticsProbeConnection(t, client)
	ctx, cancel := context.WithCancel(deadline)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- conn.TraceStart(ctx, "trace", "config", TraceOptions{}) }()
	select {
	case <-started:
		cancel()
	case <-deadline.Done():
		t.Fatal(deadline.Err())
	}
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-deadline.Done():
		t.Fatal(deadline.Err())
	}
	require.True(t, conn.conn.IsClosed())
	require.NoError(t, conn.Close(deadline))
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-deadline.Done():
		t.Fatal(deadline.Err())
	}
}
