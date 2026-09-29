package fbconn

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestGetSelectDescribeInfoItemsRequestsSchemaOnlyForFirebird6(t *testing.T) {

	const version5 = 5
	const version6 = 6
	items := GetSelectDescribeInfoItems(version5)
	assert.Equal(t, 0, bytes.Count(items, []byte{fbproto.Isc_info_sql_relation_schema}))

	items = GetSelectDescribeInfoItems(version6)
	assert.Equal(t, 2, bytes.Count(items, []byte{fbproto.Isc_info_sql_relation_schema}))

}

func createDescriptorBlockWithSchema(index int, schema, relation, field string) []byte {
	buf := []byte{}
	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_sqlda_seq, index)...)
	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_type, fbproto.SQL_TYPE_LONG)...)
	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_sub_type, 0)...)
	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_scale, 0)...)
	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_length, 4)...)
	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_null_ind, 1)...)
	buf = append(buf, tagBufferValue(fbproto.Isc_info_sql_field, []byte(field))...)
	buf = append(buf, tagBufferValue(fbproto.Isc_info_sql_relation, []byte(relation))...)
	buf = append(buf, tagBufferValue(fbproto.Isc_info_sql_relation_schema, []byte(schema))...)
	buf = append(buf, tagBufferValue(fbproto.Isc_info_sql_owner, []byte("SYSDBA"))...)
	buf = append(buf, tagBufferValue(fbproto.Isc_info_sql_alias, []byte(field))...)
	buf = append(buf, fbproto.Isc_info_sql_describe_end)
	return buf
}

func TestStatementInfoParsesRelationSchema(t *testing.T) {

	buf := []byte{}
	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_stmt_type, fbproto.Isc_info_sql_stmt_select)...)
	buf = append(buf, fbproto.Isc_info_sql_select)
	buf = append(buf, tagLengthValue(fbproto.Isc_info_sql_describe_vars, 2)...)
	buf = append(buf, createDescriptorBlockWithSchema(1, "SCHEMA1", "TEST", "VAL")...)
	buf = append(buf, createDescriptorBlockWithSchema(2, "", "", "EXPR")...)
	buf = append(buf, fbproto.Isc_info_end)

	info := StatementInfo{}
	require.NoError(t, info.parseInfo(buf))

	require.Len(t, info.FieldDescriptors, 2)
	assert.Equal(t, "SCHEMA1", info.FieldDescriptors[0].Schema)
	assert.Equal(t, "TEST", info.FieldDescriptors[0].OriginalTableName)
	assert.Equal(t, "VAL", info.FieldDescriptors[0].OriginalName)

	assert.Empty(t, info.FieldDescriptors[1].Schema)
	assert.Empty(t, info.FieldDescriptors[1].OriginalTableName)
	assert.Equal(t, "EXPR", info.FieldDescriptors[1].OriginalName)
}

func TestArraySDLIncludesSchema(t *testing.T) {
	dims := []map[string]int{{"Lower": 1, "Upper": 3}}
	base := Array{
		TableName:         "T_ARR",
		FieldName:         "VAL",
		Dimensions:        dims,
		CurrentDimensions: dims,
		Length:            2,
		BlrTypeId:         fbproto.Isc_blr_short,
		TypeId:            fbproto.SQL_TYPE_SHORT,
	}
	withSchema := base
	withSchema.Schema = "MY_SCHEMA"
	withoutSchema := base

	sdlWith := withSchema.generateSDL()
	sdlWithout := withoutSchema.generateSDL()

	schemaItem := append([]byte{fbproto.Isc_sdl_schema, byte(len("MY_SCHEMA"))}, []byte("MY_SCHEMA")...)
	assert.True(t, bytes.Contains(sdlWith, schemaItem))
	assert.False(t, bytes.Contains(sdlWithout, []byte{fbproto.Isc_sdl_schema}))
}
