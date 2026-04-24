package fbconn

import (
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbproto/utils"
)

func tagLengthValue(tag byte, value int) []byte {
	data := utils.Int32_to_bytes(int32(value))
	buf := []byte{tag}                                           // tag
	buf = append(buf, utils.Int16_to_bytes(int16(len(data)))...) // data length
	return append(buf, data...)                                  // data
}

func createDescriptorBlock(index int) []byte {
	buf := []byte{}

	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_sqlda_seq, index)...)
	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_type, fbproto.SQL_TYPE_LONG)...)
	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_sub_type, 0)...)
	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_scale, 0)...)
	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_length, 4)...)
	buf = append(buf, fbproto.Isc_info_sql_describe_end)

	return buf
}

func createTruncatedDescriptorBlock(index int) []byte {
	buf := []byte{}

	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_sqlda_seq, index)...)
	buf = append(buf, fbproto.Isc_info_truncated)

	return buf
}
func TestInfoTruncatedError(t *testing.T) {
	tests := []struct {
		name                               string
		createTestBuffer                   func() []byte
		expectedFiledDescriptorsLength     int
		expectedParameterDescriptorsLength int
		expectedIndex                      int
		expectedError                      bool
	}{
		{
			// If an InfoTruncatedError error occurs in the parseInfo function,
			// the descriptors slices should be cleared, and currentIndex should be set to 0.
			name: "Clearing descriptors after InfoTruncatedError",
			createTestBuffer: func() []byte {
				buf := []byte{}
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf = append(buf, fbproto.Isc_info_sql_bind)
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 1)...) // Expected number of ParameterDescriptors
				buf = append(buf, createDescriptorBlock(1)...)
				buf = append(buf, fbproto.Isc_info_sql_select)
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 2)...) // Expected number of FieldDescriptors
				buf = append(buf, createDescriptorBlock(1)...)
				buf = append(buf, createDescriptorBlock(2)...)
				buf = append(buf, fbproto.Isc_info_truncated)
				return buf
			},
			expectedFiledDescriptorsLength:     0,
			expectedParameterDescriptorsLength: 0,
			expectedIndex:                      0,
			expectedError:                      true,
		},
		{
			name: "Successful reading of descriptors",
			createTestBuffer: func() []byte {
				buf := []byte{}
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf = append(buf, fbproto.Isc_info_sql_bind)
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 1)...) // Expected number of ParameterDescriptors
				buf = append(buf, createDescriptorBlock(1)...)
				buf = append(buf, fbproto.Isc_info_sql_select)
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 2)...) // Expected number of FieldDescriptors
				buf = append(buf, createDescriptorBlock(1)...)
				buf = append(buf, createDescriptorBlock(2)...)
				buf = append(buf, fbproto.Isc_info_end)
				return buf
			},
			expectedFiledDescriptorsLength:     2,
			expectedParameterDescriptorsLength: 1,
			expectedError:                      false,
		},
		{
			// If the InfoTruncatedError error occurred at the lower level, in the feedDescriptors function
			// after writing the FieldDescriptors, the last recorded FieldDescriptors should be deleted from FieldDescriptors slice
			// and its index should be written to currentIndex.
			name: "InfoTruncatedError immediately after reading the previous FieldDescriptors",
			createTestBuffer: func() []byte {
				buf := []byte{}
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf = append(buf, fbproto.Isc_info_sql_select)
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 4)...) // Expected number of FieldDescriptors
				buf = append(buf, createDescriptorBlock(1)...)
				buf = append(buf, createDescriptorBlock(2)...)
				buf = append(buf, createDescriptorBlock(3)...)
				buf = append(buf, fbproto.Isc_info_truncated)
				return buf
			},
			expectedFiledDescriptorsLength:     2,
			expectedParameterDescriptorsLength: 0,
			expectedIndex:                      3,
			expectedError:                      true,
		},
		{
			// If the InfoTruncatedError error occurred at the lower level, in the feedDescriptors function
			// after writing the ParameterDescriptors, the last recorded ParameterDescriptors should be deleted
			// from ParameterDescriptors slice and its index should be written to currentIndex.
			name: "InfoTruncatedError immediately after reading the previous ParameterDescriptors",
			createTestBuffer: func() []byte {
				buf := []byte{}
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf = append(buf, fbproto.Isc_info_sql_bind)
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 3)...) // Expected number of ParameterDescriptors
				buf = append(buf, createDescriptorBlock(1)...)
				buf = append(buf, createDescriptorBlock(2)...)
				buf = append(buf, fbproto.Isc_info_truncated)
				return buf
			},
			expectedFiledDescriptorsLength:     0,
			expectedParameterDescriptorsLength: 1,
			expectedIndex:                      2,
			expectedError:                      true,
		},
		{
			// If the InfoTruncatedError error occurred at the lower level in the feedDescriptors function
			// while reading the FieldDescriptors, its index should be written to currentIndex.
			name: "InfoTruncatedError while reading the FieldDescriptors",
			createTestBuffer: func() []byte {
				buf := []byte{}
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf = append(buf, fbproto.Isc_info_sql_select)
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 4)...) // Expected number of FieldDescriptors
				buf = append(buf, createDescriptorBlock(1)...)
				buf = append(buf, createDescriptorBlock(2)...)
				buf = append(buf, createTruncatedDescriptorBlock(3)...)
				return buf
			},
			expectedFiledDescriptorsLength:     2,
			expectedParameterDescriptorsLength: 0,
			expectedIndex:                      3,
			expectedError:                      true,
		},
		{
			// If the InfoTruncatedError error occurred at the lower level in the feedDescriptors function
			// while reading the ParameterDescriptors, its index should be written to currentIndex.
			name: "InfoTruncatedError while reading the ParameterDescriptors",
			createTestBuffer: func() []byte {
				buf := []byte{}
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf = append(buf, fbproto.Isc_info_sql_bind)
				buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 3)...) // Expected number of ParameterDescriptors
				buf = append(buf, createDescriptorBlock(1)...)
				buf = append(buf, createTruncatedDescriptorBlock(2)...)
				return buf
			},
			expectedFiledDescriptorsLength:     0,
			expectedParameterDescriptorsLength: 1,
			expectedIndex:                      2,
			expectedError:                      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info := StatementInfo{}
			err := info.parseInfo(test.createTestBuffer())
			if test.expectedError {
				var truncErr InfoTruncationError
				require.ErrorAs(t, err, &truncErr)
				require.Equal(t, test.expectedIndex, truncErr.Index)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, info.FieldDescriptors, test.expectedFiledDescriptorsLength)
			require.Len(t, info.ParametersDescriptors, test.expectedParameterDescriptorsLength)
		})
	}
}

func TestContinueReadingDescriptorsAfterInfoTruncatedError(t *testing.T) {

	tests := []struct {
		name                                string
		createTruncatedBuffer               func() []byte
		createContinueBuffer                func() []byte
		expectedFiledDescriptorsLength1     int
		expectedFiledDescriptorsLength2     int
		expectedParameterDescriptorsLength1 int
		expectedParameterDescriptorsLength2 int
	}{
		{
			name: "While reading FieldDescriptors",
			createTruncatedBuffer: func() []byte {
				buf1 := []byte{}
				buf1 = append(buf1, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf1 = append(buf1, fbproto.Isc_info_sql_select)
				buf1 = append(buf1, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 4)...) // Expected number of descriptors
				buf1 = append(buf1, createDescriptorBlock(1)...)
				buf1 = append(buf1, createDescriptorBlock(2)...)
				buf1 = append(buf1, createDescriptorBlock(3)...)
				buf1 = append(buf1, fbproto.Isc_info_truncated)
				return buf1
			},
			createContinueBuffer: func() []byte {
				buf2 := []byte{}
				buf2 = append(buf2, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf2 = append(buf2, fbproto.Isc_info_sql_select)
				buf2 = append(buf2, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 4)...) // Expected number of descriptors
				buf2 = append(buf2, createDescriptorBlock(3)...)
				buf2 = append(buf2, createDescriptorBlock(4)...)
				buf2 = append(buf2, fbproto.Isc_info_end)
				return buf2
			},

			expectedFiledDescriptorsLength1:     2,
			expectedFiledDescriptorsLength2:     4,
			expectedParameterDescriptorsLength1: 0,
			expectedParameterDescriptorsLength2: 0,
		},
		{
			name: "Immediately after reading the previous FieldDescriptors",
			createTruncatedBuffer: func() []byte {
				buf1 := []byte{}
				buf1 = append(buf1, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf1 = append(buf1, fbproto.Isc_info_sql_select)
				buf1 = append(buf1, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 4)...) // Expected number of descriptors
				buf1 = append(buf1, createDescriptorBlock(1)...)
				buf1 = append(buf1, createDescriptorBlock(2)...)
				buf1 = append(buf1, createTruncatedDescriptorBlock(3)...)
				return buf1
			},
			createContinueBuffer: func() []byte {
				buf2 := []byte{}
				buf2 = append(buf2, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf2 = append(buf2, fbproto.Isc_info_sql_select)
				buf2 = append(buf2, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 4)...) // Expected number of descriptors
				buf2 = append(buf2, createDescriptorBlock(3)...)
				buf2 = append(buf2, createDescriptorBlock(4)...)
				buf2 = append(buf2, fbproto.Isc_info_end)
				return buf2
			},
			expectedFiledDescriptorsLength1:     2,
			expectedFiledDescriptorsLength2:     4,
			expectedParameterDescriptorsLength1: 0,
			expectedParameterDescriptorsLength2: 0,
		},
		{
			name: "While reading ParameterDescriptors",
			createTruncatedBuffer: func() []byte {
				buf1 := []byte{}
				buf1 = append(buf1, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf1 = append(buf1, fbproto.Isc_info_sql_bind)
				buf1 = append(buf1, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 4)...) // Expected number of descriptors
				buf1 = append(buf1, createDescriptorBlock(1)...)
				buf1 = append(buf1, createDescriptorBlock(2)...)
				buf1 = append(buf1, createDescriptorBlock(3)...)
				buf1 = append(buf1, fbproto.Isc_info_truncated)
				return buf1
			},
			createContinueBuffer: func() []byte {
				buf2 := []byte{}
				buf2 = append(buf2, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf2 = append(buf2, fbproto.Isc_info_sql_bind)
				buf2 = append(buf2, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 4)...) // Expected number of descriptors
				buf2 = append(buf2, createDescriptorBlock(3)...)
				buf2 = append(buf2, createDescriptorBlock(4)...)
				buf2 = append(buf2, fbproto.Isc_info_end)
				return buf2
			},
			expectedFiledDescriptorsLength1:     0,
			expectedFiledDescriptorsLength2:     0,
			expectedParameterDescriptorsLength1: 2,
			expectedParameterDescriptorsLength2: 4,
		},
		{
			name: "Immediately after reading the previous ParameterDescriptors",
			createTruncatedBuffer: func() []byte {
				buf1 := []byte{}
				buf1 = append(buf1, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf1 = append(buf1, fbproto.Isc_info_sql_bind)
				buf1 = append(buf1, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 4)...) // Expected number of descriptors
				buf1 = append(buf1, createDescriptorBlock(1)...)
				buf1 = append(buf1, createDescriptorBlock(2)...)
				buf1 = append(buf1, createTruncatedDescriptorBlock(3)...)
				return buf1
			},
			createContinueBuffer: func() []byte {
				buf2 := []byte{}
				buf2 = append(buf2, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
				buf2 = append(buf2, fbproto.Isc_info_sql_bind)
				buf2 = append(buf2, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 4)...) // Expected number of descriptors
				buf2 = append(buf2, createDescriptorBlock(3)...)
				buf2 = append(buf2, createDescriptorBlock(4)...)
				buf2 = append(buf2, fbproto.Isc_info_end)
				return buf2
			},
			expectedFiledDescriptorsLength1:     0,
			expectedFiledDescriptorsLength2:     0,
			expectedParameterDescriptorsLength1: 2,
			expectedParameterDescriptorsLength2: 4,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info := StatementInfo{}
			err := info.parseInfo(test.createTruncatedBuffer())

			var truncErr InfoTruncationError
			require.ErrorAs(t, err, &truncErr)
			require.Equal(t, 3, truncErr.Index)
			require.Len(t, info.FieldDescriptors, test.expectedFiledDescriptorsLength1)
			require.Len(t, info.ParametersDescriptors, test.expectedParameterDescriptorsLength1)

			err = info.parseInfo(test.createContinueBuffer())

			require.NoError(t, err)
			require.Len(t, info.FieldDescriptors, test.expectedFiledDescriptorsLength2)
			require.Len(t, info.ParametersDescriptors, test.expectedParameterDescriptorsLength2)
		})
	}
}
