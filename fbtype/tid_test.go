package fbtype_test

import (
	"context"
	"testing"

	"github.com/red-soft/fbx/fbtype"
	"github.com/red-soft/fbx/pgxtest"
)

func TestTIDCodec(t *testing.T) {
	skipCockroachDB(t, "Server does not support type tid")

	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "tid", []pgxtest.ValueRoundTripTest{
		{Param: fbtype.TID{BlockNumber: 42, OffsetNumber: 43, Valid: true}, Result: new(fbtype.TID), Test: isExpectedEq(fbtype.TID{BlockNumber: 42, OffsetNumber: 43, Valid: true})},
		{Param: fbtype.TID{BlockNumber: 4294967295, OffsetNumber: 65535, Valid: true}, Result: new(fbtype.TID), Test: isExpectedEq(fbtype.TID{BlockNumber: 4294967295, OffsetNumber: 65535, Valid: true})},
		{Param: fbtype.TID{BlockNumber: 42, OffsetNumber: 43, Valid: true}, Result: new(string), Test: isExpectedEq("(42,43)")},
		{Param: fbtype.TID{BlockNumber: 4294967295, OffsetNumber: 65535, Valid: true}, Result: new(string), Test: isExpectedEq("(4294967295,65535)")},
		{Param: fbtype.TID{}, Result: new(fbtype.TID), Test: isExpectedEq(fbtype.TID{})},
		{Param: nil, Result: new(fbtype.TID), Test: isExpectedEq(fbtype.TID{})},
	})
}
