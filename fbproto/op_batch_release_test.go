package fbproto

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpBatchReleaseWrite(t *testing.T) {
	var buf bytes.Buffer

	err := (&OpBatchRelease{
		Kind:            Op_batch_rls,
		StatementHandle: 42,
	}).Write(NewXDRWriter(&buf))

	require.NoError(t, err)
	require.Equal(t, []byte{
		0x00, 0x00, 0x00, byte(Op_batch_rls),
		0x00, 0x00, 0x00, 0x2a,
	}, buf.Bytes())
}

func TestOpBatchReleaseWriteInvalidKind(t *testing.T) {
	var buf bytes.Buffer

	err := (&OpBatchRelease{
		Kind:            12345,
		StatementHandle: 42,
	}).Write(NewXDRWriter(&buf))

	require.Error(t, err)
	require.ErrorContains(t, err, "wrong batch release operation")
}

func TestOpBatchSyncWrite(t *testing.T) {
	var buf bytes.Buffer

	err := (&OpBatchSync{}).Write(NewXDRWriter(&buf))

	require.NoError(t, err)
	require.Equal(t, []byte{
		0x00, 0x00, 0x00, byte(Op_batch_sync),
	}, buf.Bytes())
}

func TestOpInfoBatchWrite(t *testing.T) {
	var buf bytes.Buffer
	err := (&OpInfoBatch{
		StatementHandle: 42,
		Items:           []byte{BatchInfoBufferBytes, BatchInfoDataBytes, BatchInfoBlobsBytes, BatchInfoBlobAlign, BatchInfoBlobHeader},
		BufferLength:    64,
	}).Write(NewXDRWriter(&buf))
	require.NoError(t, err)
	require.Equal(t, []byte{
		0, 0, 0, byte(Op_info_batch),
		0, 0, 0, 42,
		0, 0, 0, 0,
		0, 0, 0, 5, 10, 11, 12, 13, 14, 0, 0, 0, // items and XDR padding
		0, 0, 0, 64,
	}, buf.Bytes())
}

func TestOpBatchCompletionRead(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []int
		valid  bool
	}{
		{"empty", []int{42, 0, 0, 0, 0}, true},
		{"counts and errors", []int{42, 3, 3, 1, 1, 1, -1, -1, 1, Isc_arg_gds, Isc_cancelled, Isc_arg_end, 2}, true},
		{"negative records", []int{42, -1, 0, 0, 0}, false},
		{"negative updates", []int{42, 1, -1, 0, 0}, false},
		{"negative vectors", []int{42, 1, 0, -1, 0}, false},
		{"negative errors", []int{42, 1, 0, 0, -1}, false},
		{"too many updates", []int{42, 1, 2, 0, 0}, false},
		{"too many errors", []int{42, 1, 0, 1, 1}, false},
		{"invalid detailed row", []int{42, 1, 0, 1, 0, -1}, false},
		{"invalid simplified row", []int{42, 1, 0, 0, 1, 1}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var data bytes.Buffer
			w := NewXDRWriter(&data)
			for _, n := range tc.values {
				require.NoError(t, w.WriteInt32(n))
			}
			op := &OpBatchCompletion{}
			if !tc.valid {
				require.Error(t, op.Read(NewXDRReader(&data)))
				return
			}
			payload := append([]byte(nil), data.Bytes()...)
			require.NoError(t, w.WriteInt32(77))
			r := NewXDRReader(&data)
			require.NoError(t, op.Read(r))
			require.Equal(t, 42, op.StatementHandle)
			next, err := r.ReadInt32()
			require.NoError(t, err)
			require.Equal(t, 77, next)
			if op.TotalRecords > 0 {
				require.Equal(t, []int{1, -1, -1}, op.UpdateCounts)
				require.Equal(t, 1, op.DetailedErrors[0].Row)
				require.Equal(t, []int{2}, op.SimplifiedErrors)
			}
			previous := *op
			previous.DetailedErrors = append([]BatchDetailedError(nil), op.DetailedErrors...)
			require.NoError(t, op.Read(NewXDRReader(bytes.NewReader(payload))))
			require.Equal(t, previous, *op)
			require.Equal(t, cap(previous.UpdateCounts), cap(op.UpdateCounts))
			require.Equal(t, cap(previous.DetailedErrors), cap(op.DetailedErrors))
			require.Equal(t, cap(previous.SimplifiedErrors), cap(op.SimplifiedErrors))
			require.ErrorIs(t, op.Read(NewXDRReader(bytes.NewReader(payload[:len(payload)-1]))), io.ErrUnexpectedEOF)
			var empty bytes.Buffer
			for _, n := range []int{42, 0, 0, 0, 0} {
				require.NoError(t, NewXDRWriter(&empty).WriteInt32(n))
			}
			require.NoError(t, op.Read(NewXDRReader(&empty)))
			require.Empty(t, op.UpdateCounts)
			require.Empty(t, op.DetailedErrors)
			require.Empty(t, op.SimplifiedErrors)
			require.Equal(t, cap(previous.DetailedErrors), cap(op.DetailedErrors))
			for _, detail := range op.DetailedErrors[:cap(op.DetailedErrors)] {
				require.Equal(t, BatchDetailedError{}, detail)
			}
		})
	}
}

func TestOpBatchCompletionReadTruncated(t *testing.T) {
	const count = 1 << 20
	for _, tc := range []struct {
		name    string
		counts  []int
		element []int
	}{
		{"updates", []int{count, 0, 0}, []int{1}},
		{"detailed errors", []int{0, count, 0}, []int{0, Isc_arg_gds, Isc_cancelled, Isc_arg_end}},
		{"simplified errors", []int{0, 0, count}, []int{0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var data bytes.Buffer
			w := NewXDRWriter(&data)
			for _, n := range append([]int{42, count}, tc.counts...) {
				require.NoError(t, w.WriteInt32(n))
			}
			for _, n := range tc.element {
				require.NoError(t, w.WriteInt32(n))
			}
			for _, size := range []int{20, data.Len() - 1, data.Len()} {
				var op OpBatchCompletion
				err := op.Read(NewXDRReader(bytes.NewReader(data.Bytes()[:size])))
				if size == data.Len()-1 {
					require.ErrorIs(t, err, io.ErrUnexpectedEOF)
				} else {
					require.ErrorIs(t, err, io.EOF)
				}
				length := len(op.UpdateCounts) + len(op.DetailedErrors) + len(op.SimplifiedErrors)
				capacity := cap(op.UpdateCounts) + cap(op.DetailedErrors) + cap(op.SimplifiedErrors)
				if size == data.Len() {
					require.Equal(t, 1, length)
					require.Less(t, capacity, count)
				} else {
					require.Zero(t, length)
					require.Zero(t, capacity)
				}
			}
		})
	}
}
