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
	if err = x.WriteInt32(Op_info_sql); err != nil {
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
	return append(append([]byte{Isc_info_sql_sqlda_start, 2}, utils.Int16_to_bytes(int16(index))...), GetSelectDescribeInfoItems()...)
}

func GetSelectDescribeInfoItems() []byte {
	return []byte{
		Isc_info_sql_stmt_type,
		Isc_info_sql_select,
		Isc_info_sql_describe_vars,
		Isc_info_sql_sqlda_seq,
		Isc_info_sql_type,
		Isc_info_sql_sub_type,
		Isc_info_sql_scale,
		Isc_info_sql_length,
		Isc_info_sql_null_ind,
		Isc_info_sql_field,
		Isc_info_sql_relation,
		Isc_info_sql_owner,
		Isc_info_sql_alias,
		Isc_info_sql_describe_end,

		Isc_info_sql_bind,
		Isc_info_sql_describe_vars,
		Isc_info_sql_sqlda_seq,
		Isc_info_sql_type,
		Isc_info_sql_sub_type,
		Isc_info_sql_scale,
		Isc_info_sql_length,
		Isc_info_sql_describe_end,
	}
}
