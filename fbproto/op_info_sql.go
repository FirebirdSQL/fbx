package fbproto

import "github.com/red-soft/fbx/fbproto/utils"

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-statements-information

type OpInfoSQL struct {
	StatementHandle           int
	StatementInformationItems []byte
	MaxBufferLength           int
}

func (op *OpInfoSQL) Write(x *XDRWriter) error {
	var err error
	// p_operation
	if err = x.WriteInt32(op_info_sql); err != nil {
		return err
	}

	//p_info_object
	if err = x.WriteInt32(op.StatementHandle); err != nil {
		return err
	}

	//p_info_incarnation
	if err = x.WriteInt32(0); err != nil {
		return err
	}

	//p_info_items
	if err = x.WriteBuffer(op.StatementInformationItems); err != nil {
		return err
	}

	//p_info_buffer_length
	bufLength := 1024
	if op.MaxBufferLength != 0 {
		bufLength = op.MaxBufferLength
	}
	if err = x.WriteInt32(bufLength); err != nil {
		return err
	}

	return nil
}

func GetSelectDescribeInfoItemsStart() []byte {
	return append([]byte{Isc_info_sql_stmt_type}, GetSelectDescribeInfoItems()...)
}

func GetSelectDescribeInfoItemsContinue(index int) []byte {
	return append(append([]byte{isc_info_sql_sqlda_start, 2}, utils.Int16_to_bytes(int16(index))...), GetSelectDescribeInfoItems()...)
}

func GetSelectDescribeInfoItems() []byte {
	return []byte{
		Isc_info_sql_stmt_type,
		Isc_info_sql_select,
		isc_info_sql_describe_vars,
		isc_info_sql_sqlda_seq,
		isc_info_sql_type,
		isc_info_sql_sub_type,
		isc_info_sql_scale,
		isc_info_sql_length,
		isc_info_sql_null_ind,
		isc_info_sql_field,
		isc_info_sql_relation,
		isc_info_sql_owner,
		isc_info_sql_alias,
		isc_info_sql_describe_end,
	}
}
