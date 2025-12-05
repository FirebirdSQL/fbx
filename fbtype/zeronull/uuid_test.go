package zeronull_test

import (
	"context"
	"testing"

	"rdb.red-soft.ru/fbx/fbtype/zeronull"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestUUIDTranscode(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "uuid", []fbxtest.ValueRoundTripTest{
		{
			Param:  (zeronull.UUID)([16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}),
			Result: new(zeronull.UUID),
			Test:   isExpectedEq((zeronull.UUID)([16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})),
		},
		{
			Param:  nil,
			Result: new(zeronull.UUID),
			Test:   isExpectedEq((zeronull.UUID)([16]byte{})),
		},
		{
			Param:  (zeronull.UUID)([16]byte{}),
			Result: new(any),
			Test:   isExpectedEq(nil),
		},
	})
}
