package fbtype_test

import (
	"context"
	"testing"

	"github.com/red-soft/fbx/fbtype"
	"github.com/red-soft/fbx/fbxtest"
)

func TestUint32Codec(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, fbxtest.KnownOIDQueryExecModes, "oid", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Uint32{Uint32: fbtype.TextOID, Valid: true}, Result: new(fbtype.Uint32), Test: isExpectedEq(fbtype.Uint32{Uint32: fbtype.TextOID, Valid: true})},
		{Param: fbtype.Uint32{}, Result: new(fbtype.Uint32), Test: isExpectedEq(fbtype.Uint32{})},
		{Param: nil, Result: new(fbtype.Uint32), Test: isExpectedEq(fbtype.Uint32{})},
		{Param: "1147", Result: new(string), Test: isExpectedEq("1147")},
	})
}
