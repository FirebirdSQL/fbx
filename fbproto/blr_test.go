package fbproto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBLRWriter(t *testing.T) {
	blr := NewBLRWriter(32)
	require.NotNil(t, blr)
	assert.Equal(t, []byte{}, blr.Bytes())
	assert.Equal(t, 32, cap(blr.buf))
}

func TestBLRWriterPutByte(t *testing.T) {
	blr := NewBLRWriter(0)
	blr.PutByte(0x42)
	blr.PutByte(0x00)
	blr.PutByte(0xFF)
	assert.Equal(t, []byte{0x42, 0x00, 0xFF}, blr.Bytes())
}

func TestBLRWriterPutTag(t *testing.T) {
	blr := NewBLRWriter(0)
	blr.PutTag(Isc_blr_version5)
	blr.PutTag(Isc_blr_begin)
	assert.Equal(t, []byte{Isc_blr_version5, Isc_blr_begin}, blr.Bytes())
}

func TestBLRWriterPutInt16(t *testing.T) {
	blr := NewBLRWriter(2)
	blr.PutInt16(-32768)
	assert.Equal(t, []byte{0x00, 0x80}, blr.Bytes())
}

func TestBLRWriterPutInt32(t *testing.T) {
	blr := NewBLRWriter(4)
	blr.PutInt32(0x04030201)
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, blr.Bytes())
}

func TestBLRWriterPutInt64(t *testing.T) {
	blr := NewBLRWriter(8)
	blr.PutInt64(0x0807060504030201)
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, blr.Bytes())
}

func TestBLRWriterPutBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
		want  []byte
	}{
		{name: "nil", bytes: nil, want: []byte{}},
		{name: "empty", bytes: []byte{}, want: []byte{}},
		{name: "data", bytes: []byte{1, 2, 3}, want: []byte{1, 2, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blr := NewBLRWriter(3)
			blr.PutBytes(tt.bytes)
			assert.Equal(t, tt.want, blr.Bytes())
		})
	}
}

func TestBLRWriterPutString(t *testing.T) {

	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			"empty",
			func(t *testing.T) {
				blr := NewBLRWriter(0)
				err := blr.PutString("")
				require.NoError(t, err)
				assert.Equal(t, []byte{0}, blr.Bytes())
			},
		},
		{
			"data",
			func(t *testing.T) {
				blr := NewBLRWriter(5)
				err := blr.PutString("test")
				require.NoError(t, err)
				assert.Equal(t, []byte{4, 't', 'e', 's', 't'}, blr.Bytes())
			},
		},
		{
			"max_255",
			func(t *testing.T) {
				blr := NewBLRWriter(256)
				s := strings.Repeat("a", 255)
				err := blr.PutString(s)
				require.NoError(t, err)
				got := blr.Bytes()
				require.Len(t, got, 256)
				assert.Equal(t, byte(255), got[0])
				assert.Equal(t, s, string(got[1:]))
			},
		},
		{
			"error_on_256",
			func(t *testing.T) {
				blr := NewBLRWriter(0)
				err := blr.PutString(strings.Repeat("a", 256))
				require.ErrorContains(t, err, "buffer more then expected")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, test.testFunc)
	}
}

func TestBLRWriterPutTagWithByte(t *testing.T) {
	blr := NewBLRWriter(2)
	blr.PutTagWithByte(Isc_blr_long, 2)
	assert.Equal(t, []byte{Isc_blr_long, 2}, blr.Bytes())
}

func TestBLRWriterBytesAppendsAccumulate(t *testing.T) {
	blr := NewBLRWriter(8)
	blr.PutTag(Isc_blr_version5)
	blr.PutInt16(258)
	blr.PutBytes([]byte{0, 1, 2, 3, 4})
	assert.Equal(t, []byte{Isc_blr_version5, 0x02, 0x01, 0, 1, 2, 3, 4}, blr.Bytes())
}

func TestBLRWriterPutType(t *testing.T) {
	tests := []struct {
		name    string
		sqltype int
		subtype int
		scale   int
		length  int
		want    []byte
	}{
		{
			name:    "varying",
			sqltype: SQL_TYPE_VARYING,
			subtype: 0x0102,
			length:  10,
			want:    []byte{Isc_blr_varying2, 0x02, 0x01, 10, 0},
		},
		{
			name:    "text",
			sqltype: SQL_TYPE_TEXT,
			subtype: 0x0304,
			length:  20,
			want:    []byte{Isc_blr_text2, 0x04, 0x03, 20, 0},
		},
		{
			name:    "long",
			sqltype: SQL_TYPE_LONG,
			scale:   2,
			want:    []byte{Isc_blr_long, 2},
		},
		{
			name:    "short",
			sqltype: SQL_TYPE_SHORT,
			scale:   1,
			want:    []byte{Isc_blr_short, 1},
		},
		{
			name:    "int64",
			sqltype: SQL_TYPE_INT64,
			scale:   3,
			want:    []byte{Isc_blr_int64, 3},
		},
		{
			name:    "int128",
			sqltype: SQL_TYPE_INT128,
			scale:   4,
			want:    []byte{Isc_blr_int128, 4},
		},
		{
			name:    "quad",
			sqltype: SQL_TYPE_QUAD,
			scale:   5,
			want:    []byte{Isc_blr_quad, 5},
		},
		{
			name:    "dec_fixed",
			sqltype: SQL_TYPE_DEC_FIXED,
			scale:   6,
			want:    []byte{Isc_blr_int128, 6},
		},
		{
			name:    "double",
			sqltype: SQL_TYPE_DOUBLE,
			want:    []byte{Isc_blr_double},
		},
		{
			name:    "float",
			sqltype: SQL_TYPE_FLOAT,
			want:    []byte{Isc_blr_float},
		},
		{
			name:    "d_float",
			sqltype: SQL_TYPE_D_FLOAT,
			want:    []byte{Isc_blr_d_float},
		},
		{
			name:    "date",
			sqltype: SQL_TYPE_DATE,
			want:    []byte{Isc_blr_sql_date},
		},
		{
			name:    "time",
			sqltype: SQL_TYPE_TIME,
			want:    []byte{Isc_blr_sql_time},
		},
		{
			name:    "timestamp",
			sqltype: SQL_TYPE_TIMESTAMP,
			want:    []byte{Isc_blr_timestamp},
		},
		{
			name:    "blob",
			sqltype: SQL_TYPE_BLOB,
			want:    []byte{Isc_blr_quad, 0},
		},
		{
			name:    "array",
			sqltype: SQL_TYPE_ARRAY,
			want:    []byte{Isc_blr_quad, 0},
		},
		{
			name:    "boolean",
			sqltype: SQL_TYPE_BOOLEAN,
			want:    []byte{Isc_blr_bool},
		},
		{
			name:    "dec64",
			sqltype: SQL_TYPE_DEC64,
			want:    []byte{Isc_blr_dec64},
		},
		{
			name:    "dec128",
			sqltype: SQL_TYPE_DEC128,
			want:    []byte{Isc_blr_dec128},
		},
		{
			name:    "time_tz",
			sqltype: SQL_TYPE_TIME_TZ,
			want:    []byte{Isc_blr_sql_time_tz},
		},
		{
			name:    "timestamp_tz",
			sqltype: SQL_TYPE_TIMESTAMP_TZ,
			want:    []byte{Isc_blr_timestamp_tz},
		},
		{
			name:    "time_tz_ex",
			sqltype: SQL_TIME_TZ_EX,
			want:    []byte{Isc_blr_ex_time_tz},
		},
		{
			name:    "timestamp_tz_ex",
			sqltype: SQL_TIMESTAMP_TZ_EX,
			want:    []byte{Isc_blr_ex_timestamp_tz},
		},
		{
			name:    "null",
			sqltype: SQL_TYPE_NULL,
			want:    []byte{Isc_blr_text, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blr := NewBLRWriter(0)
			err := blr.PutType(tt.sqltype, tt.subtype, tt.scale, tt.length)
			require.NoError(t, err)
			assert.Equal(t, tt.want, blr.Bytes())
		})
	}
}

func TestBLRWriterPutTypeUnknownError(t *testing.T) {
	blr := NewBLRWriter(0)
	err := blr.PutType(99999, 0, 0, 0)
	require.ErrorContains(t, err, "unknown SQL type")
}

func TestCalculateBLREmptyDescriptors(t *testing.T) {
	got, err := CalculateBLR(nil, nil)
	require.NoError(t, err)
	want := []byte{
		Isc_blr_version5,
		Isc_blr_begin,
		Isc_blr_message,
		0,
		0, 0,
		Isc_blr_end,
		Isc_blr_eoc,
	}
	assert.Equal(t, want, got)
}

func TestCalculateBLRMultipleTypes(t *testing.T) {
	descriptors := []FieldDescriptor{
		{Type: SQL_TYPE_SHORT, Scale: 1},
		{Type: SQL_TYPE_BOOLEAN},
		{Type: SQL_TYPE_DOUBLE},
	}
	got, err := CalculateBLR(descriptors, nil)
	require.NoError(t, err)
	want := []byte{
		Isc_blr_version5,
		Isc_blr_begin,
		Isc_blr_message,
		0,
		6, 0,
		Isc_blr_short, 1,
		Isc_blr_short, 0,
		Isc_blr_bool,
		Isc_blr_short, 0,
		Isc_blr_double,
		Isc_blr_short, 0,
		Isc_blr_end,
		Isc_blr_eoc,
	}
	assert.Equal(t, want, got)
}

func TestCalculateBLRNegativeScale(t *testing.T) {
	descriptors := []FieldDescriptor{{
		Type:  SQL_TYPE_INT64,
		Scale: -2, // becomes 254 after += 256
	}}
	got, err := CalculateBLR(descriptors, nil)
	require.NoError(t, err)

	assert.Equal(t, byte(Isc_blr_int64), got[6])
	assert.Equal(t, byte(254), got[7])
}

func TestCalculateBLRVarying(t *testing.T) {
	descriptors := []FieldDescriptor{{
		Type:    SQL_TYPE_VARYING,
		Subtype: 0x0001,
		Length:  50,
	}}
	got, err := CalculateBLR(descriptors, nil)
	require.NoError(t, err)

	want := []byte{
		Isc_blr_version5,
		Isc_blr_begin,
		Isc_blr_message,
		0,
		2, 0,
		Isc_blr_varying2, 1, 0, 50, 0,
		Isc_blr_short, 0,
		Isc_blr_end,
		Isc_blr_eoc,
	}
	assert.Equal(t, want, got)
}

func TestCalculateBLRMismatchValuesCountValuesSlice(t *testing.T) {
	tests := []struct {
		name          string
		descriptors   []FieldDescriptor
		values        [][]byte
		expectedError string
	}{
		{
			name:          "less values then expected",
			descriptors:   []FieldDescriptor{{Type: SQL_TYPE_LONG}},
			values:        [][]byte{},
			expectedError: "BLR values count mismatch: got 0, want 1",
		},
		{
			name:          "more values then expected",
			descriptors:   []FieldDescriptor{{Type: SQL_TYPE_LONG}},
			values:        [][]byte{{1}, {2}},
			expectedError: "BLR values count mismatch: got 2, want 1",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := CalculateBLR(test.descriptors, test.values)
			require.ErrorContains(t, err, test.expectedError)
		})
	}
}

func TestCalculateBLRBlobAndArray(t *testing.T) {
	descriptors := []FieldDescriptor{
		{Type: SQL_TYPE_BLOB},
		{Type: SQL_TYPE_ARRAY},
	}
	got, err := CalculateBLR(descriptors, nil)
	require.NoError(t, err)

	want := []byte{
		Isc_blr_version5,
		Isc_blr_begin,
		Isc_blr_message,
		0,
		4, 0,
		Isc_blr_quad, 0,
		Isc_blr_short, 0,
		Isc_blr_quad, 0,
		Isc_blr_short, 0,
		Isc_blr_end,
		Isc_blr_eoc,
	}
	assert.Equal(t, want, got)
}

func TestCalculateBLRUnknownTypeError(t *testing.T) {
	descriptors := []FieldDescriptor{{Type: 12345}}
	_, err := CalculateBLR(descriptors, nil)
	require.ErrorContains(t, err, "unknown SQL type")
}
