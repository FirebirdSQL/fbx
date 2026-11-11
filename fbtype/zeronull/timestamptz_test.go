package zeronull_test

import (
	"context"
	"testing"
	"time"

	"github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/fbtype/zeronull"
	"github.com/firebirdsql/fbx/fbxtest"
)

func isExpectedEqTimestamptz(a any) func(any) bool {
	return func(v any) bool {
		at := time.Time(a.(zeronull.Timestamptz))
		vt := time.Time(v.(zeronull.Timestamptz))

		return at.Equal(vt)
	}
}

func TestTimestamptzTranscode(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		if getServerMajorVersionForConn(ctx, tb, conn) <= 3 {
			t.Skip("timestamp with time zone is not supported by server engine version <= 3")
		}
	})

	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "timestamp with time zone", []fbxtest.ValueRoundTripTest{
		{
			Param:  (zeronull.Timestamptz)(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
			Result: new(zeronull.Timestamptz),
			Test:   isExpectedEqTimestamptz((zeronull.Timestamptz)(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))),
		},
		{
			Param:  nil,
			Result: new(zeronull.Timestamptz),
			Test:   isExpectedEqTimestamptz((zeronull.Timestamptz)(time.Time{})),
		},
		{
			Param:  (zeronull.Timestamptz)(time.Time{}),
			Result: new(any),
			Test:   isExpectedEq(nil),
		},
	})
}
