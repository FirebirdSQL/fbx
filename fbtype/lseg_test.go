package fbtype_test

import (
	"context"
	"testing"

	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestLsegTranscode(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "lseg", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Lseg{
			P:     [2]fbtype.Vec2{{3.14, 1.678}, {7.1, 5.2345678901}},
			Valid: true,
		}, Result: new(fbtype.Lseg), Test: isExpectedEq(fbtype.Lseg{
			P:     [2]fbtype.Vec2{{3.14, 1.678}, {7.1, 5.2345678901}},
			Valid: true,
		}),
		},
		{Param: fbtype.Lseg{
			P:     [2]fbtype.Vec2{{7.1, 1.678}, {-13.14, -5.234}},
			Valid: true,
		}, Result: new(fbtype.Lseg), Test: isExpectedEq(fbtype.Lseg{
			P:     [2]fbtype.Vec2{{7.1, 1.678}, {-13.14, -5.234}},
			Valid: true,
		}),
		},
		{Param: fbtype.Lseg{}, Result: new(fbtype.Lseg), Test: isExpectedEq(fbtype.Lseg{})},
		{Param: nil, Result: new(fbtype.Lseg), Test: isExpectedEq(fbtype.Lseg{})},
	})
}
