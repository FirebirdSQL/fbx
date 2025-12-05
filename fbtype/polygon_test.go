package fbtype_test

import (
	"context"
	"testing"

	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func isExpectedEqPolygon(a any) func(any) bool {
	return func(v any) bool {
		ap := a.(fbtype.Polygon)
		vp := v.(fbtype.Polygon)

		if !(ap.Valid == vp.Valid && len(ap.P) == len(vp.P)) {
			return false
		}

		for i := range ap.P {
			if ap.P[i] != vp.P[i] {
				return false
			}
		}

		return true
	}
}

func TestPolygonTranscode(t *testing.T) {
	skipCockroachDB(t, "Server does not support type polygon")

	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "polygon", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Polygon{
			P:     []fbtype.Vec2{{3.14, 1.678901234}, {7.1, 5.234}, {5.0, 3.234}},
			Valid: true,
		}, Result: new(fbtype.Polygon), Test: isExpectedEqPolygon(fbtype.Polygon{
			P:     []fbtype.Vec2{{3.14, 1.678901234}, {7.1, 5.234}, {5.0, 3.234}},
			Valid: true,
		}),
		},
		{Param: fbtype.Polygon{
			P:     []fbtype.Vec2{{3.14, -1.678}, {7.1, -5.234}, {23.1, 9.34}},
			Valid: true,
		}, Result: new(fbtype.Polygon), Test: isExpectedEqPolygon(fbtype.Polygon{
			P:     []fbtype.Vec2{{3.14, -1.678}, {7.1, -5.234}, {23.1, 9.34}},
			Valid: true,
		}),
		},
		{Param: fbtype.Polygon{}, Result: new(fbtype.Polygon), Test: isExpectedEqPolygon(fbtype.Polygon{})},
		{Param: nil, Result: new(fbtype.Polygon), Test: isExpectedEqPolygon(fbtype.Polygon{})},
	})
}
