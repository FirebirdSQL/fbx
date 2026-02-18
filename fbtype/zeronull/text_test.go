package zeronull_test

import (
	"context"
	"testing"

	"rdb.red-soft.ru/fbx/fbtype/zeronull"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestTextTranscode(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "varchar(255)", []fbxtest.ValueRoundTripTest{
		{
			Param:  (zeronull.Text)("foo"),
			Result: new(zeronull.Text),
			Test:   isExpectedEq((zeronull.Text)("foo")),
		},
		{
			Param:  nil,
			Result: new(zeronull.Text),
			Test:   isExpectedEq((zeronull.Text)("")),
		},
		{
			Param:  (zeronull.Text)(""),
			Result: new(any),
			Test:   isExpectedEq(nil),
		},
	})
}
