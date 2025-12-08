package fbtype_test

import (
	"context"
	"testing"

	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestBoxCodec(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "box", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Box{
			P:     [2]fbtype.Vec2{{7.1, 5.2345678}, {3.14, 1.678}},
			Valid: true,
		}, Result: new(fbtype.Box), Test: isExpectedEq(fbtype.Box{
			P:     [2]fbtype.Vec2{{7.1, 5.2345678}, {3.14, 1.678}},
			Valid: true,
		}),
		},
		{Param: fbtype.Box{
			P:     [2]fbtype.Vec2{{7.1, 5.2345678}, {-13.14, -5.234}},
			Valid: true,
		}, Result: new(fbtype.Box), Test: isExpectedEq(fbtype.Box{
			P:     [2]fbtype.Vec2{{7.1, 5.2345678}, {-13.14, -5.234}},
			Valid: true,
		}),
		},
		{Param: fbtype.Box{}, Result: new(fbtype.Box), Test: isExpectedEq(fbtype.Box{})},
		{Param: nil, Result: new(fbtype.Box), Test: isExpectedEq(fbtype.Box{})},
	})
}
