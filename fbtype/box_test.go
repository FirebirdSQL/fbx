package fbtype_test

import (
	"context"
	"testing"

	"github.com/red-soft/fbx/fbtype"
	"github.com/red-soft/fbx/fbxtest"
)

func TestBoxCodec(t *testing.T) {
	skipCockroachDB(t, "Server does not support box type")

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
