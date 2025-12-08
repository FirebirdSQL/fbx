package fbtype_test

import (
	"context"
	"testing"

	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func isExpectedEqPath(a any) func(any) bool {
	return func(v any) bool {
		ap := a.(fbtype.Path)
		vp := v.(fbtype.Path)

		if !(ap.Valid == vp.Valid && ap.Closed == vp.Closed && len(ap.P) == len(vp.P)) {
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

func TestPathTranscode(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "path", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Path{
			P:      []fbtype.Vec2{{3.14, 1.678901234}, {7.1, 5.234}},
			Closed: false,
			Valid:  true,
		}, Result: new(fbtype.Path), Test: isExpectedEqPath(fbtype.Path{
			P:      []fbtype.Vec2{{3.14, 1.678901234}, {7.1, 5.234}},
			Closed: false,
			Valid:  true,
		}),
		},
		{Param: fbtype.Path{
			P:      []fbtype.Vec2{{3.14, 1.678}, {7.1, 5.234}, {23.1, 9.34}},
			Closed: true,
			Valid:  true,
		}, Result: new(fbtype.Path), Test: isExpectedEqPath(fbtype.Path{
			P:      []fbtype.Vec2{{3.14, 1.678}, {7.1, 5.234}, {23.1, 9.34}},
			Closed: true,
			Valid:  true,
		}),
		},
		{Param: fbtype.Path{
			P:      []fbtype.Vec2{{7.1, 1.678}, {-13.14, -5.234}},
			Closed: true,
			Valid:  true,
		}, Result: new(fbtype.Path), Test: isExpectedEqPath(fbtype.Path{
			P:      []fbtype.Vec2{{7.1, 1.678}, {-13.14, -5.234}},
			Closed: true,
			Valid:  true,
		}),
		},
		{Param: fbtype.Path{}, Result: new(fbtype.Path), Test: isExpectedEqPath(fbtype.Path{})},
		{Param: nil, Result: new(fbtype.Path), Test: isExpectedEqPath(fbtype.Path{})},
	})
}
