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

// GetRowBlr generates a BLR buffer from FieldDescriptors
func (p *StatementInfo) GetRowBlr() []byte {
	return fbproto.CalculateBLR(p.FieldDescriptors, nil)
}

// GetParametersBlr generates a BLR buffer from ParametersDescriptors and a set of values
func (p *StatementInfo) GetParametersBlr(values [][]byte) []byte {
	return fbproto.CalculateBLR(p.ParametersDescriptors, values)
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
