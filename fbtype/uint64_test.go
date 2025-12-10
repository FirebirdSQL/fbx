package fbtype_test

import (
	"context"
	"testing"

	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestUint64Codec(t *testing.T) {
	t.Skip()

	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, fbxtest.KnownOIDQueryExecModes, "xid8", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Uint64{Uint64: 1 << 36, Valid: true}, Result: new(fbtype.Uint64), Test: isExpectedEq(fbtype.Uint64{Uint64: 1 << 36, Valid: true})},
		{Param: fbtype.Uint64{}, Result: new(fbtype.Uint64), Test: isExpectedEq(fbtype.Uint64{})},
		{Param: nil, Result: new(fbtype.Uint64), Test: isExpectedEq(fbtype.Uint64{})},
		{Param: uint64(1 << 36), Result: new(uint64), Test: isExpectedEq(uint64(1 << 36))},
		{Param: "1147", Result: new(string), Test: isExpectedEq("1147")},
	})
}
