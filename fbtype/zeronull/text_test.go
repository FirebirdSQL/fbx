package zeronull_test

import (
	"context"
	"testing"

	"github.com/firebirdsql/fbx/fbtype/zeronull"
	"github.com/firebirdsql/fbx/fbxtest"
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
