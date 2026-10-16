package fbproto

import (
	"bytes"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestXDRReaderBufferLengths(t *testing.T) {
	methods := []struct {
		name     string
		prefixed bool
		padded   bool
		read     func(*XDRReader, int) ([]byte, error)
	}{
		{"ReadBytes", false, false, func(r *XDRReader, n int) ([]byte, error) { return r.ReadBytes(int64(n)) }},
		{"ReadBufferLen", false, true, func(r *XDRReader, n int) ([]byte, error) { return r.ReadBufferLen(int64(n)) }},
		{"ReadBuffer", true, true, func(r *XDRReader, _ int) ([]byte, error) { return r.ReadBuffer() }},
		{"ReadBufferIntoNil", true, true, func(r *XDRReader, _ int) ([]byte, error) { return r.ReadBufferInto(nil) }},
		{"ReadBufferInto", true, true, func(r *XDRReader, _ int) ([]byte, error) {
			dst := make([]byte, 8)
			return r.ReadBufferInto(&dst)
		}},
	}
	for _, method := range methods {
		t.Run(method.name, func(t *testing.T) {
			for _, tc := range []struct {
				name   string
				length int
				data   []byte
			}{
				{"negative", -1, []byte{1, 2, 3, 4}},
				{"min_int32", math.MinInt32, []byte{1, 2, 3, 4}},
				{"empty", 0, []byte{}},
				{"value", 3, []byte("abc")},
				{"aligned", 4, []byte("abcd")},
				{"truncated", 3, []byte("ab")},
			} {
				t.Run(tc.name, func(t *testing.T) {
					var input bytes.Buffer
					w := NewXDRWriter(&input)
					if method.prefixed {
						require.NoError(t, w.WriteInt32(tc.length))
					}
					input.Write(tc.data)
					valid := tc.length >= 0 && tc.length == len(tc.data)
					if valid {
						if method.padded {
							input.Write(make([]byte, (4-tc.length)&3))
						}
						require.NoError(t, w.WriteInt32(42))
					}
					r := NewXDRReader(&input)
					value, err := method.read(r, tc.length)
					switch {
					case tc.length < 0:
						require.ErrorContains(t, err, "invalid buffer length")
						require.Nil(t, value)
						require.Equal(t, tc.data, input.Bytes(), "must not read payload after an invalid length")
					case !valid:
						require.ErrorIs(t, err, io.ErrUnexpectedEOF)
					default:
						require.NoError(t, err)
						require.Equal(t, tc.data, value)
						next, err := r.ReadInt32()
						require.NoError(t, err)
						require.Equal(t, 42, next, "padding must not consume the next value")
					}
				})
			}
			if method.prefixed {
				t.Run("truncated_prefix", func(t *testing.T) {
					_, err := method.read(NewXDRReader(bytes.NewBuffer([]byte{0, 0, 0})), 0)
					require.ErrorIs(t, err, io.ErrUnexpectedEOF)
				})
			}
			if method.padded {
				t.Run("truncated_padding", func(t *testing.T) {
					var input bytes.Buffer
					if method.prefixed {
						require.NoError(t, NewXDRWriter(&input).WriteInt32(1))
					}
					input.Write([]byte{'a', 0})
					_, err := method.read(NewXDRReader(&input), 1)
					require.ErrorIs(t, err, io.ErrUnexpectedEOF)
				})
			}
		})
	}
}

func TestReadRowDataVaryingLength(t *testing.T) {
	fields := []FieldDescriptor{{Type: SQL_TYPE_VARYING, Length: 8}, {Type: SQL_TYPE_LONG}}
	var scratch rowScratch
	for _, tc := range []struct {
		name   string
		length int
		data   []byte
	}{
		{"negative", -1, nil},
		{"min_int32", math.MinInt32, nil},
		{"value", 3, []byte("abc")},
		{"empty", 0, []byte{}},
		{"truncated", 3, []byte("ab")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var input bytes.Buffer
			w := NewXDRWriter(&input)
			require.NoError(t, w.WriteInt32(0))
			require.NoError(t, w.WriteInt32(tc.length))
			input.Write(tc.data)
			valid := tc.length >= 0 && tc.length == len(tc.data)
			if valid {
				input.Write(make([]byte, (4-tc.length)&3))
				require.NoError(t, w.WriteInt32(42))
			}
			row, err := ReadRowData(NewXDRReader(&input), fields, &scratch)
			switch {
			case tc.length < 0:
				require.ErrorContains(t, err, "invalid row field length")
				require.Nil(t, row)
			case !valid:
				require.ErrorIs(t, err, io.ErrUnexpectedEOF)
			default:
				require.NoError(t, err)
				require.Equal(t, [][]byte{tc.data, {0, 0, 0, 42}}, row.RowData)
				require.Zero(t, input.Len())
			}
		})
	}
}

func TestReadSliceLengths(t *testing.T) {
	for _, tc := range []struct {
		name     string
		dataType uint32
		length   int
		words    []int
		want     []byte
		wantErr  string
	}{
		{"negative_fixed", Isc_blr_long, 4, []int{-1, -1}, nil, "invalid p_slr_length"},
		{"negative_varying", Isc_blr_varying, 8, []int{-1, -1}, nil, "invalid p_slr_length"},
		{"min_int32", Isc_blr_long, 4, []int{math.MinInt32, math.MinInt32}, nil, "invalid p_slr_length"},
		{"negative_second_length", Isc_blr_long, 4, []int{4, -1}, nil, "invalid slice length"},
		{"mismatched", Isc_blr_long, 4, []int{4, 8}, nil, "not equal"},
		{"negative_element", Isc_blr_varying, 8, []int{10, 10, -1}, nil, "invalid buffer length"},
		{"empty_fixed", Isc_blr_long, 4, []int{0, 0}, []byte{}, ""},
		{"empty_varying", Isc_blr_varying, 8, []int{0, 0}, []byte{}, ""},
		{"fixed", Isc_blr_long, 4, []int{4, 4, 42}, []byte{0, 0, 0, 42}, ""},
		{"varying", Isc_blr_varying, 8, []int{10, 10, 1, 0x61000000}, []byte{0, 0, 0, 1, 'a'}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var input bytes.Buffer
			w := NewXDRWriter(&input)
			require.NoError(t, w.WriteInt32(Op_slice_response))
			for _, n := range tc.words {
				require.NoError(t, w.WriteInt32(n))
			}
			msg, err := NewClient(&input, io.Discard, nil).ReadSlice(tc.dataType, tc.length)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Nil(t, msg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, msg.(*OpSliceResponse).SliceData)
			require.Zero(t, input.Len())
		})
	}
}

func TestReadSliceConvertsFailedResponseToErrorResponse(t *testing.T) {
	t.Parallel()

	var input bytes.Buffer
	w := NewXDRWriter(&input)
	require.NoError(t, w.WriteInt32(Op_response))
	require.NoError(t, w.WriteInt32(7))
	require.NoError(t, w.WriteInt64(11))
	require.NoError(t, w.WriteBuffer(nil))
	require.NoError(t, w.WriteInt32(Isc_arg_gds))
	require.NoError(t, w.WriteInt32(Isc_att_shutdown))
	require.NoError(t, w.WriteInt32(Isc_arg_end))

	encoding, err := NewEncoding("UTF8")
	require.NoError(t, err)
	client := NewClient(&input, io.Discard, encoding)

	msg, err := client.ReadSlice(0, 0)
	require.NoError(t, err)

	response, ok := msg.(*ErrorResponse)
	require.True(t, ok)
	require.Equal(t, 7, response.Object)
	require.EqualValues(t, 11, response.BlobId)
	require.Equal(t, []int{Isc_att_shutdown}, response.GdsCodes)
}
