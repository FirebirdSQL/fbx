package zeronull_test

import (
	"context"
	"testing"

	"github.com/red-soft/fbx/pgtype/zeronull"
	"github.com/red-soft/fbx/pgxtest"
)

func TestTextTranscode(t *testing.T) {
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "text", []pgxtest.ValueRoundTripTest{
		{
			(zeronull.Text)("foo"),
			new(zeronull.Text),
			isExpectedEq((zeronull.Text)("foo")),
		},
		{
			nil,
			new(zeronull.Text),
			isExpectedEq((zeronull.Text)("")),
		},
		{
			(zeronull.Text)(""),
			new(any),
			isExpectedEq(nil),
		},
	})
}
