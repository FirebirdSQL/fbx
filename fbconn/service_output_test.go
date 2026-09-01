package fbconn

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestParseServiceOutput(t *testing.T) {
	tests := []struct {
		name      string
		response  []byte
		wantData  []byte
		wantState serviceOutputState
		wantError string
	}{
		{
			name:      "data",
			response:  serviceOutputResponse([]byte("gbak: done\n")),
			wantData:  []byte("gbak: done\n"),
			wantState: serviceOutputData,
		},
		{
			name:      "timeout",
			response:  []byte{fbproto.Isc_info_svc_to_eof, 0, 0, fbproto.Isc_info_svc_timeout},
			wantState: serviceOutputTimeout,
		},
		{
			name:      "truncated",
			response:  []byte{fbproto.Isc_info_truncated},
			wantState: serviceOutputTruncated,
		},
		{
			name:      "complete in stream",
			response:  []byte{fbproto.Isc_info_svc_to_eof, 0, 0, fbproto.Isc_info_end},
			wantState: serviceOutputComplete,
		},
		{
			name:      "complete item",
			response:  []byte{fbproto.Isc_info_end},
			wantState: serviceOutputComplete,
		},
		{
			name:      "empty",
			wantError: "empty service output response",
		},
		{
			name:      "short data",
			response:  []byte{fbproto.Isc_info_svc_to_eof, 4, 0, 'a'},
			wantError: "declares 4 bytes",
		},
		{
			name:      "unknown item",
			response:  []byte{255},
			wantError: "unexpected service output item",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, state, err := parseServiceOutput(test.response)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantData, data)
			require.Equal(t, test.wantState, state)
		})
	}
}

func TestBuildServiceOutputSendItems(t *testing.T) {
	require.Equal(t, []byte{
		fbproto.Isc_spb_utf8_filename, 0, 0, 0, 0,
		fbproto.Isc_info_svc_timeout, 1, 0, 0, 0, 1,
	}, buildServiceOutputSendItems())
}

func TestWriteServiceOutputChecksContextBeforeWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	output := new(countingWriter)

	err := writeServiceOutput(ctx, output, []byte("output"))

	require.Error(t, err)
	require.Zero(t, output.calls)
}

func TestMalformedServiceOutputClosesConnection(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	serverDone := make(chan error, 1)
	go func() {
		defer func() { _ = serverConn.Close() }()
		serverDone <- runMalformedServiceOutputProbeServer(serverConn)
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

	err = conn.Backup(
		context.Background(),
		"/data/employee.fdb",
		"/backup/employee.fbk",
		BackupOptions{},
	)

	require.ErrorContains(t, err, "parse service output")
	require.ErrorContains(t, err, "unexpected service output item")
	require.True(t, conn.conn.IsClosed())
	select {
	case <-conn.conn.CleanupDone():
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	require.NoError(t, <-serverDone)
	require.NoError(t, conn.Close(context.Background()))
}

func runMalformedServiceOutputProbeServer(conn net.Conn) error {
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
	if _, err := readProbeServiceStart(r, 42); err != nil {
		return err
	}
	if err := writeProbeResponse(w, 0); err != nil {
		return err
	}
	if _, _, _, err := readProbeServiceInfo(r, 42); err != nil {
		return err
	}
	if err := writeProbeResponseData(w, 0, []byte{255}); err != nil {
		return err
	}

	_, err = r.ReadInt32()
	if !errors.Is(err, io.EOF) {
		return errors.New("client did not close after malformed service output")
	}
	return nil
}

func serviceOutputResponse(data []byte) []byte {
	response := []byte{fbproto.Isc_info_svc_to_eof, 0, 0}
	binary.LittleEndian.PutUint16(response[1:3], uint16(len(data)))
	return append(response, data...)
}

type countingWriter struct {
	calls int
}

func (w *countingWriter) Write(data []byte) (int, error) {
	w.calls++
	return len(data), nil
}
