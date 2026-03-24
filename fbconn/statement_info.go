package fbconn

import (
	"fmt"

	"rdb.red-soft.ru/fbx/fbproto"
)

type StatementInfo struct {
	FieldDescriptors      []fbproto.FieldDescriptor
	ParametersDescriptors []fbproto.FieldDescriptor
	StatementType         int
	Counts                SQLCounts
	currentDescriptor     *[]fbproto.FieldDescriptor
}

type SQLCounts struct {
	SelectCount int64
	InsertCount int64
	UpdateCount int64
	DeleteCount int64
}

func (p *StatementInfo) parseInfo(buf []byte) error {
	r := NewServerInfoBufferReader(buf)
	var item byte
	var err error
	for {
		if item, err = r.ReadByte(); err != nil {
			return err
		}
		switch item {
		case fbproto.Isc_info_end:
			return nil
		case fbproto.Isc_info_sql_stmt_type:
			p.StatementType, err = r.ReadInt32Value()
			if err != nil {
				return err
			}
		case fbproto.Isc_info_sql_bind:
			p.currentDescriptor = &p.ParametersDescriptors
			if err = p.feedDescriptors(r); err != nil {
				return err
			}
		case fbproto.Isc_info_sql_select:
			p.currentDescriptor = &p.FieldDescriptors
			if err = p.feedDescriptors(r); err != nil {
				return err
			}
		case fbproto.Isc_info_truncated:
			return InfoTruncationError{0}
		default:
			return fmt.Errorf("unknown statement info item: %d", item)
		}
	}
}

func (p *StatementInfo) feedDescriptors(r *ServerInfoBufferReader) error {
	var item byte
	var err error
	var descriptorCount int
	// Skipping isc_info_sql_num_variables/isc_info_sql_describe_vars item
	if _, err := r.ReadByte(); err != nil {
		return err
	}
	if descriptorCount, err = r.ReadInt32Value(); err != nil {
		return err
	}
	if descriptorCount == 0 {
		return nil
	}
	field := fbproto.FieldDescriptor{}
	for {
		if len(*p.currentDescriptor) == descriptorCount {
			return nil
		}
		if item, err = r.ReadByte(); err != nil {
			return err
		}
		switch item {
		case fbproto.Isc_info_sql_sqlda_seq:
			if field.Index, err = r.ReadInt32Value(); err != nil {
				return err
			}
		case fbproto.Isc_info_sql_type:
			if field.Type, err = r.ReadInt32Value(); err != nil {
				return err
			}
			field.Nullable = (field.Type & 1) != 0
			field.Type &= ^1
		case fbproto.Isc_info_sql_sub_type:
			if field.Subtype, err = r.ReadInt32Value(); err != nil {
				return err
			}
		case fbproto.Isc_info_sql_scale:
			if field.Scale, err = r.ReadInt32Value(); err != nil {
				return err
			}
		case fbproto.Isc_info_sql_length:
			if field.Length, err = r.ReadInt32Value(); err != nil {
				return err
			}
		case fbproto.Isc_info_sql_field:
			if field.OriginalName, err = r.ReadStringValue(); err != nil {
				return err
			}
		case fbproto.Isc_info_sql_alias:
			if field.Name, err = r.ReadStringValue(); err != nil {
				return err
			}
		case fbproto.Isc_info_sql_relation:
			if field.OriginalTableName, err = r.ReadStringValue(); err != nil {
				return err
			}
		case fbproto.Isc_info_sql_relation_alias:
			if field.TableAliasName, err = r.ReadStringValue(); err != nil {
				return err
			}
		case fbproto.Isc_info_sql_owner:
			if field.OwnerName, err = r.ReadStringValue(); err != nil {
				return err
			}
		case fbproto.Isc_info_sql_describe_end:
			*p.currentDescriptor = append(*p.currentDescriptor, field)
		case fbproto.Isc_info_sql_null_ind:
			var nullOk int
			if nullOk, err = r.ReadInt32Value(); err != nil {
				return err
			}
			field.NullOk = nullOk == 1
		case fbproto.Isc_info_truncated:
			return InfoTruncationError{Index: field.Index}
		default:
			return fmt.Errorf("unknown descriptor info item: %d", item)
		}
	}
}

func (p *StatementInfo) GetRowBlr() []byte {
	return p.CalculateBlr(p.FieldDescriptors, nil)
}

func (p *StatementInfo) GetParametersBlr(values [][]byte) []byte {
	return p.CalculateBlr(p.ParametersDescriptors, values)
}

func (p *StatementInfo) CalculateBlr(descriptors []fbproto.FieldDescriptor, values [][]byte) []byte {
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

	for i, x := range descriptors {
		sqlscale := x.Scale
		if sqlscale < 0 {
			sqlscale += 256
		}
		dataLength := x.Length
		if values != nil {
			if values[i] != nil {
				dataLength = len(values[i])
			} else {
				dataLength = 0
			}
		}
		switch x.Type {
		case fbproto.SQL_TYPE_VARYING:
			charsetID := byte(x.Subtype & 0xFF)
			collationID := byte((x.Subtype >> 8) & 0xFF)
			blr[n] = 38 // blr_varying2
			blr[n+1] = charsetID
			blr[n+2] = collationID
			blr[n+3] = byte(dataLength & 255)
			blr[n+4] = byte(dataLength >> 8)
			n += 5
		case fbproto.SQL_TYPE_TEXT:
			charsetID := byte(x.Subtype & 0xFF)
			collationID := byte((x.Subtype >> 8) & 0xFF)
			blr[n] = 15 // blr_text2
			blr[n+1] = charsetID
			blr[n+2] = collationID
			blr[n+3] = byte(dataLength & 255)
			blr[n+4] = byte(dataLength >> 8)
			n += 5
		case fbproto.SQL_TYPE_LONG:
			blr[n] = 8
			blr[n+1] = byte(sqlscale)
			n += 2
		case fbproto.SQL_TYPE_SHORT:
			blr[n] = 7
			blr[n+1] = byte(sqlscale)
			n += 2
		case fbproto.SQL_TYPE_INT64:
			blr[n] = 16
			blr[n+1] = byte(sqlscale)
			n += 2
		case fbproto.SQL_TYPE_INT128:
			blr[n] = 26
			blr[n+1] = byte(sqlscale)
			n += 2
		case fbproto.SQL_TYPE_QUAD:
			blr[n] = 9
			blr[n+1] = byte(sqlscale)
			n += 2
		case fbproto.SQL_TYPE_DEC_FIXED: // OBSOLETED
			blr[n] = 26
			blr[n+1] = byte(sqlscale)
			n += 2
		case fbproto.SQL_TYPE_DOUBLE:
			blr[n] = 27
			n++
		case fbproto.SQL_TYPE_FLOAT:
			blr[n] = 10
			n++
		case fbproto.SQL_TYPE_D_FLOAT:
			blr[n] = 11
			n++
		case fbproto.SQL_TYPE_DATE:
			blr[n] = 12
			n++
		case fbproto.SQL_TYPE_TIME:
			blr[n] = 13
			n++
		case fbproto.SQL_TYPE_TIMESTAMP:
			blr[n] = 35
			n++
		case fbproto.SQL_TYPE_BLOB:
			blr[n] = 9
			blr[n+1] = 0
			n += 2
		case fbproto.SQL_TYPE_ARRAY:
			blr[n] = 9
			blr[n+1] = 0
			n += 2
		case fbproto.SQL_TYPE_BOOLEAN:
			blr[n] = 23
			n++
		case fbproto.SQL_TYPE_DEC64:
			blr[n] = 24
			n++
		case fbproto.SQL_TYPE_DEC128:
			blr[n] = 25
			n++
		case fbproto.SQL_TYPE_TIME_TZ:
			blr[n] = 28
			n++
		case fbproto.SQL_TYPE_TIMESTAMP_TZ:
			blr[n] = 29
			n++
		case fbproto.SQL_TYPE_NULL:
			blr[n] = 14
			blr[n+1] = 0
			blr[n+2] = 0
			n += 3
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

func (p *SQLCounts) parseInfo(buf []byte) error {
	r := NewServerInfoBufferReader(buf)
	var item byte
	var err error
	for {
		if item, err = r.ReadByte(); err != nil {
			return err
		}
		switch item {
		case fbproto.Isc_info_end:
			return nil
		case fbproto.Isc_info_sql_records:
			if err = p.parseSQLCounts(r); err != nil {
				return err
			}
		case fbproto.Isc_info_truncated:
			return InfoTruncationError{0}
		default:
			return fmt.Errorf("unknown statement info item: %d", item)
		}
	}
}

func (p *SQLCounts) parseSQLCounts(r *ServerInfoBufferReader) error {
	length, err := r.ReadInt32ValueNoPrefix()
	if err != nil {
		return err
	}

	if length == 0 {
		return nil
	}

	for {
		item, err := r.ReadByte()
		if err != nil {
			return err
		}
		switch item {
		case fbproto.Isc_info_end:
			return nil
		case fbproto.Isc_info_truncated:
			return InfoTruncationError{}
		case fbproto.Isc_info_error:
			return fmt.Errorf("sql counts info unavailable")
		case fbproto.Isc_info_req_select_count, fbproto.Isc_info_req_insert_count, fbproto.Isc_info_req_update_count, fbproto.Isc_info_req_delete_count:
			val, err := r.ReadInt64Value()
			if err != nil {
				return err
			}
			switch item {
			case fbproto.Isc_info_req_select_count:
				p.SelectCount = val
			case fbproto.Isc_info_req_insert_count:
				p.InsertCount = val
			case fbproto.Isc_info_req_update_count:
				p.UpdateCount = val
			case fbproto.Isc_info_req_delete_count:
				p.DeleteCount = val
			}
		default:
			return fmt.Errorf("unsupported sql counts info item: %d", item)
		}
	}
}
