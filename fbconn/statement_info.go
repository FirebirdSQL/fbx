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
			p.FieldDescriptors = []fbproto.FieldDescriptor{}
			p.ParametersDescriptors = []fbproto.FieldDescriptor{}
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
		previousItem := item
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
		case fbproto.Isc_info_sql_relation_schema:
			if field.Schema, err = r.ReadStringValue(); err != nil {
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
			if previousItem == fbproto.Isc_info_sql_describe_end && len(*p.currentDescriptor) > 0 {
				*p.currentDescriptor = (*p.currentDescriptor)[:len(*p.currentDescriptor)-1]
			}
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
	parametrsNumber := len(descriptors) * 2

	blr := fbproto.NewBLRWriterV5()
	blr.PutTag(fbproto.Isc_blr_begin)
	blr.PutTag(fbproto.Isc_blr_message)
	blr.PutByte(0) // message number
	blr.PutInt16(int16(parametrsNumber))

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

			blr.PutTag(fbproto.Isc_blr_varying2)
			blr.PutByte(charsetID)
			blr.PutByte(collationID)
			blr.PutInt16(int16(dataLength))
		case fbproto.SQL_TYPE_TEXT:
			charsetID := byte(x.Subtype & 0xFF)
			collationID := byte((x.Subtype >> 8) & 0xFF)

			blr.PutTag(fbproto.Isc_blr_text2)
			blr.PutByte(charsetID)
			blr.PutByte(collationID)
			blr.PutInt16(int16(dataLength))
		case fbproto.SQL_TYPE_LONG:
			blr.PutTypeWithScale(fbproto.Isc_blr_long, byte(sqlscale))
		case fbproto.SQL_TYPE_SHORT:
			blr.PutTypeWithScale(fbproto.Isc_blr_short, byte(sqlscale))
		case fbproto.SQL_TYPE_INT64:
			blr.PutTypeWithScale(fbproto.Isc_blr_int64, byte(sqlscale))
		case fbproto.SQL_TYPE_INT128:
			blr.PutTypeWithScale(fbproto.Isc_blr_int128, byte(sqlscale))
		case fbproto.SQL_TYPE_QUAD:
			blr.PutTypeWithScale(fbproto.Isc_blr_quad, byte(sqlscale))
		case fbproto.SQL_TYPE_DEC_FIXED: // OBSOLETED
			blr.PutTypeWithScale(fbproto.Isc_blr_int128, byte(sqlscale))
		case fbproto.SQL_TYPE_DOUBLE:
			blr.PutTag(fbproto.Isc_blr_double)
		case fbproto.SQL_TYPE_FLOAT:
			blr.PutTag(fbproto.Isc_blr_float)
		case fbproto.SQL_TYPE_D_FLOAT:
			blr.PutTag(fbproto.Isc_blr_d_float)
		case fbproto.SQL_TYPE_DATE:
			blr.PutTag(fbproto.Isc_blr_sql_date)
		case fbproto.SQL_TYPE_TIME:
			blr.PutTag(fbproto.Isc_blr_sql_time)
		case fbproto.SQL_TYPE_TIMESTAMP:
			blr.PutTag(fbproto.Isc_blr_timestamp)
		case fbproto.SQL_TYPE_BLOB:
			blr.PutTypeWithScale(fbproto.Isc_blr_quad, 0)
		case fbproto.SQL_TYPE_ARRAY:
			blr.PutTypeWithScale(fbproto.Isc_blr_quad, 0)
		case fbproto.SQL_TYPE_BOOLEAN:
			blr.PutTag(fbproto.Isc_blr_bool)
		case fbproto.SQL_TYPE_DEC64:
			blr.PutTag(fbproto.Isc_blr_dec64)
		case fbproto.SQL_TYPE_DEC128:
			blr.PutTag(fbproto.Isc_blr_dec128)
		case fbproto.SQL_TYPE_TIME_TZ:
			blr.PutTag(fbproto.Isc_blr_sql_time_tz)
		case fbproto.SQL_TYPE_TIMESTAMP_TZ:
			blr.PutTag(fbproto.Isc_blr_timestamp_tz)
		case fbproto.SQL_TYPE_NULL:
			blr.PutTag(fbproto.Isc_blr_text)
			blr.PutByte(0)
			blr.PutByte(0)
		default:
			panic(fmt.Sprintf("unknown SQL type %d", x.Type))
		}
		blr.PutTypeWithScale(fbproto.Isc_blr_short, 0)
	}
	blr.PutTag(fbproto.Isc_blr_end)
	blr.PutTag(fbproto.Isc_blr_eoc)

	return blr.Bytes()
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
