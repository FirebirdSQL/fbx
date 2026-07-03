package fbconn

import (
	"context"
	"encoding/binary"
	"testing"

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
