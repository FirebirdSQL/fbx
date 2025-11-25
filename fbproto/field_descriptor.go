package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-row-data

import (
	"bytes"
	"fmt"
	"io"

	"github.com/red-soft/fbx/fbproto/vax"
)

type FieldDescriptor struct {
	Index             int
	Type              int
	Format            int16 // 0 - text, 1 - binary. PG flavor flag for compatibility. Probable should be removed after reworking type conversion module
	Scale             int
	Subtype           int
	Length            int
	NullOk            bool
	Name              string
	TableAliasName    string
	RelName           string
	OwnerName         string
	OriginalName      string
	OriginalTableName string
	Nullable          bool
}

type StatementInfo struct {
	FieldDescriptors      []FieldDescriptor
	ParametersDescriptors []FieldDescriptor
	StatementType         int
	Counts                SQLCounts
	currentDescriptor     *[]FieldDescriptor
}

type SQLCounts struct {
	SelectCount int64
	InsertCount int64
	UpdateCount int64
	DeleteCount int64
}

func (p *StatementInfo) Parse(buf []byte) error {
	reader := bytes.NewBuffer(buf)
	var item byte
	var err error
	for {
		if item, err = reader.ReadByte(); err != nil {
			return err
		}
		switch item {
		case Isc_info_end:
			return nil
		case Isc_info_sql_stmt_type:
			p.StatementType = p.readInt32Value(reader)
		case Isc_info_sql_bind:
			p.currentDescriptor = &p.ParametersDescriptors
			if err = p.feedDescriptors(reader); err != nil {
				return err
			}
		case Isc_info_sql_select:
			p.currentDescriptor = &p.FieldDescriptors
			if err = p.feedDescriptors(reader); err != nil {
				return err
			}
		case Isc_info_sql_records:
			if err = p.parseSQLCounts(reader); err != nil {
				return err
			}
		case Isc_info_truncated:
			return StatementInfoTruncationError{0}
		default:
			return fmt.Errorf("unknown statement info item: %d", item)
		}
	}
}

func (p *StatementInfo) readInt32Value(reader *bytes.Buffer) int {
	l := vax.Integer2(reader.Bytes(), 0)
	reader.Next(2)
	val := vax.Integer(reader.Bytes(), 0, l)
	reader.Next(l)
	return val
}

func (p *StatementInfo) readStringValue(reader *bytes.Buffer) string {
	l := vax.Integer2(reader.Bytes(), 0)
	reader.Next(2)
	return string(reader.Next(l))
}

type StatementInfoTruncationError struct {
	Index int
}

func (s StatementInfoTruncationError) Error() string {
	return fmt.Sprintf("info truncated at field with index %d", s.Index)
}

func (p *StatementInfo) feedDescriptors(reader *bytes.Buffer) error {
	var item byte
	var err error
	// Skipping isc_info_sql_num_variables/isc_info_sql_describe_vars item
	if _, err := reader.ReadByte(); err != nil {
		return err
	}
	descriptorCount := p.readInt32Value(reader)
	if descriptorCount == 0 {
		return nil
	}
	field := FieldDescriptor{}
	for {
		if len(*p.currentDescriptor) == descriptorCount {
			return nil
		}
		item, err = reader.ReadByte()
		if err != nil {
			return err
		}
		switch item {
		case Isc_info_sql_sqlda_seq:
			field.Index = p.readInt32Value(reader)
		case Isc_info_sql_type:
			field.Type = p.readInt32Value(reader)
			field.Nullable = (field.Type & 1) != 0
			field.Type &= ^1
		case Isc_info_sql_sub_type:
			field.Subtype = p.readInt32Value(reader)
		case Isc_info_sql_scale:
			field.Scale = p.readInt32Value(reader)
		case Isc_info_sql_length:
			field.Length = p.readInt32Value(reader)
		case Isc_info_sql_field:
			field.OriginalName = p.readStringValue(reader)
		case Isc_info_sql_alias:
			field.Name = p.readStringValue(reader)
		case Isc_info_sql_relation:
			field.OriginalTableName = p.readStringValue(reader)
		case Isc_info_sql_relation_alias:
			field.TableAliasName = p.readStringValue(reader)
		case Isc_info_sql_owner:
			field.OwnerName = p.readStringValue(reader)
		case Isc_info_sql_describe_end:
			field.Format = FbTypeIntoFormat(field.Type, field.Subtype)
			*p.currentDescriptor = append(*p.currentDescriptor, field)
		case Isc_info_sql_null_ind:
			field.NullOk = p.readInt32Value(reader) == 1
		case Isc_info_truncated:
			return StatementInfoTruncationError{Index: field.Index}
		default:
			return fmt.Errorf("unknown descriptor info item: %d", item)
		}
	}
}

func (p *StatementInfo) parseSQLCounts(reader *bytes.Buffer) error {
	if reader.Len() < 2 {
		return io.ErrUnexpectedEOF
	}

	length := vax.Integer2(reader.Bytes(), 0)
	reader.Next(2)

	if length == 0 {
		return nil
	}
	if reader.Len() < length {
		return io.ErrUnexpectedEOF
	}

	for length > 0 {
		item, err := reader.ReadByte()
		if err != nil {
			return err
		}
		length--

		switch item {
		case Isc_info_end:
			return nil
		case Isc_info_truncated:
			return StatementInfoTruncationError{}
		case Isc_info_error:
			return fmt.Errorf("sql counts info unavailable")
		case Isc_info_req_select_count, Isc_info_req_insert_count, Isc_info_req_update_count, Isc_info_req_delete_count:
			before := reader.Len()
			val, err := p.readInt64Value(reader)
			if err != nil {
				return err
			}
			consumed := before - reader.Len()
			if consumed > length {
				return io.ErrUnexpectedEOF
			}
			length -= consumed

			switch item {
			case Isc_info_req_select_count:
				p.Counts.SelectCount = val
			case Isc_info_req_insert_count:
				p.Counts.InsertCount = val
			case Isc_info_req_update_count:
				p.Counts.UpdateCount = val
			case Isc_info_req_delete_count:
				p.Counts.DeleteCount = val
			}
		default:
			return fmt.Errorf("unsupported sql counts info item: %d", item)
		}
	}

	return nil
}

func (p *StatementInfo) readInt64Value(reader *bytes.Buffer) (int64, error) {
	if reader.Len() < 2 {
		return 0, io.ErrUnexpectedEOF
	}

	length := vax.Integer2(reader.Bytes(), 0)
	reader.Next(2)

	if length == 0 {
		return 0, nil
	}
	if length > 8 {
		return 0, fmt.Errorf("invalid int64 length: %d", length)
	}
	if reader.Len() < length {
		return 0, io.ErrUnexpectedEOF
	}

	value := vax.Long(reader.Bytes(), 0, length)
	reader.Next(length)

	return value, nil
}

func FbTypeIntoFormat(fbtype int, subtype int) int16 {
	switch fbtype {
	case SQL_TYPE_VARYING, SQL_TYPE_TEXT:
		return 0
	}
	return 1
}

func (p *StatementInfo) GetRowBlr() []byte {
	return p.CalculateBlr(p.FieldDescriptors)
}

func (p *StatementInfo) GetParametersBlr() []byte {
	return p.CalculateBlr(p.ParametersDescriptors)
}

func (p *StatementInfo) CalculateBlr(descriptors []FieldDescriptor) []byte {
	ln := len(descriptors) * 2
	blr := make([]byte, (ln*4)+8)
	// FIXME: fbx remove magic numbers and use blr codes
	blr[0] = 5
	blr[1] = 2
	blr[2] = 4
	blr[3] = 0
	blr[4] = byte(ln & 255)
	blr[5] = byte(ln >> 8)
	n := 6

	for _, x := range descriptors {
		sqlscale := x.Scale
		if sqlscale < 0 {
			sqlscale += 256
		}
		switch x.Type {
		case SQL_TYPE_VARYING:
			blr[n] = 37
			blr[n+1] = byte(x.Length & 255)
			blr[n+2] = byte(x.Length >> 8)
			n += 3
		case SQL_TYPE_TEXT:
			blr[n] = 14
			blr[n+1] = byte(x.Length & 255)
			blr[n+2] = byte(x.Length >> 8)
			n += 3
		case SQL_TYPE_LONG:
			blr[n] = 8
			blr[n+1] = byte(sqlscale)
			n += 2
		case SQL_TYPE_SHORT:
			blr[n] = 7
			blr[n+1] = byte(sqlscale)
			n += 2
		case SQL_TYPE_INT64:
			blr[n] = 16
			blr[n+1] = byte(sqlscale)
			n += 2
		case SQL_TYPE_INT128:
			blr[n] = 26
			blr[n+1] = byte(sqlscale)
			n += 2
		case SQL_TYPE_QUAD:
			blr[n] = 9
			blr[n+1] = byte(sqlscale)
			n += 2
		case SQL_TYPE_DEC_FIXED: // OBSOLETED
			blr[n] = 26
			blr[n+1] = byte(sqlscale)
			n += 2
		case SQL_TYPE_DOUBLE:
			blr[n] = 27
			n++
		case SQL_TYPE_FLOAT:
			blr[n] = 10
			n++
		case SQL_TYPE_D_FLOAT:
			blr[n] = 11
			n++
		case SQL_TYPE_DATE:
			blr[n] = 12
			n++
		case SQL_TYPE_TIME:
			blr[n] = 13
			n++
		case SQL_TYPE_TIMESTAMP:
			blr[n] = 35
			n++
		case SQL_TYPE_BLOB:
			blr[n] = 9
			blr[n+1] = 0
			n += 2
		case SQL_TYPE_ARRAY:
			blr[n] = 9
			blr[n+1] = 0
			n += 2
		case SQL_TYPE_BOOLEAN:
			blr[n] = 23
			n++
		case SQL_TYPE_DEC64:
			blr[n] = 24
			n++
		case SQL_TYPE_DEC128:
			blr[n] = 25
			n++
		case SQL_TYPE_TIME_TZ:
			blr[n] = 28
			n++
		case SQL_TYPE_TIMESTAMP_TZ:
			blr[n] = 29
			n++
		default:
			panic(fmt.Sprintf("unknown SQL type %d", x.Type))
		}
		// [blr_short, 0]
		blr[n] = 7
		blr[n+1] = 0
		n += 2
	}
	// [blr_end, blr_eoc]
	blr[n] = 255
	blr[n+1] = 76
	n += 2

	return blr[:n]
}

// GetIOLength returns physical length of field by its type
// If length is 0 - field length should be obtained from stream
// If length < 0 - field data used as is without padding
// If length > 0 - padding bytes must be obtained after field data (note +1 to length below to distinguish from 0, it should be subtracted later)
func (f *FieldDescriptor) GetIOLength() int {
	switch f.Type {
	case SQL_TYPE_TEXT:
		return f.Length + 1 // +1 to distinguish from 0
	case SQL_TYPE_VARYING:
		return 0
	case SQL_TYPE_SHORT, SQL_TYPE_LONG, SQL_TYPE_FLOAT, SQL_TYPE_TIME, SQL_TYPE_DATE:
		return -4
	case SQL_TYPE_DOUBLE, SQL_TYPE_TIMESTAMP, SQL_TYPE_BLOB, SQL_TYPE_ARRAY, SQL_TYPE_QUAD, SQL_TYPE_INT64, SQL_TYPE_DEC64, SQL_TYPE_TIME_TZ:
		return -8
	case SQL_TYPE_TIMESTAMP_TZ, SQL_TIME_TZ_EX:
		return -12
	case SQL_TIMESTAMP_TZ_EX, SQL_TYPE_DEC128, SQL_TYPE_INT128:
		return -16
	case SQL_TYPE_NULL:
		return 0
	case SQL_TYPE_BOOLEAN:
		return 1 + 1 // +1 to distinguish from 0
	default:
		panic(fmt.Sprintf("can not get IO length of unknown field descriptor type: %d", f.Type))
	}
}

// CalculateIoLength returns physical length of field by its type and data
// See GetIOLength
func (f *FieldDescriptor) CalculateIoLength(fieldData []byte) int {
	t := f.Type
	if t == SQL_TYPE_TEXT {
		return len(fieldData) + 1
	}
	return f.GetIOLength()
}

func (f *FieldDescriptor) GetPaddingByte() byte {
	switch f.Type {
	case SQL_TYPE_TEXT, SQL_TYPE_VARYING:
		{
			if f.GetCharacterSetId() == CS_BINARY {
				return 0x00
			}
			return 0x20
		}
	default:
		return 0x00
	}
}

func (f *FieldDescriptor) GetCharacterSetId() int {
	switch f.Type {
	case SQL_TYPE_TEXT, SQL_TYPE_VARYING:
		return f.Subtype & 0xFF
	case SQL_TYPE_BLOB:
		if f.Subtype == BLOB_SUB_TYPE_TEXT {
			return f.Scale & 0xFF
		}
		return CS_BINARY
	default:
		return CS_dynamic
	}
}
