package zeronull_test

import (
	"context"
	"testing"

	"rdb.red-soft.ru/fbx/fbtype/zeronull"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func isExpectedEq(a any) func(any) bool {
	return func(v any) bool {
		return a == v
	}
}

func TestFloat8Transcode(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "float", []fbxtest.ValueRoundTripTest{
		{
			Param:  (zeronull.Float8)(1),
			Result: new(zeronull.Float8),
			Test:   isExpectedEq((zeronull.Float8)(1)),
		},
		{
			Param:  nil,
			Result: new(zeronull.Float8),
			Test:   isExpectedEq((zeronull.Float8)(0)),
		},
		{
			Param:  (zeronull.Float8)(0),
			Result: new(any),
			Test:   isExpectedEq(nil),
		},
	})
}
