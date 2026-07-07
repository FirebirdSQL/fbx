package fbtype_test

import (
	"context"
	"testing"
	"time"

	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func sameTimetzClockAndOffset(want time.Time) func(any) bool {
	return func(v any) bool {
		got := v.(time.Time)
		if got.Hour() != want.Hour() || got.Minute() != want.Minute() || got.Second() != want.Second() || got.Nanosecond() != want.Nanosecond() {
			return false
		}
		_, gotOffset := got.Zone()
		_, wantOffset := want.Zone()
		return gotOffset == wantOffset
	}
}

func sameFbtypeTimetzClockAndOffset(want time.Time) func(any) bool {
	return func(v any) bool {
		got := v.(fbtype.Timetz)
		if !got.Valid {
			return false
		}
		return sameTimetzClockAndOffset(want)(got.Time)
	}
}

func TestTimetzTranscode(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		if getServerMajorVersionForConn(ctx, tb, conn) <= 3 {
			t.Skip("time with time zone is not supported by server engine version <= 3")
		}
	})

	locMoscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatalf("load Europe/Moscow: %v", err)
	}

	offsetPlus330 := time.FixedZone("+03:30", (3*60+30)*60)

	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, fbxtest.KnownTypeIDQueryExecModes, "time with time zone", []fbxtest.ValueRoundTripTest{
		{
			Param:  fbtype.Timetz{Time: time.Date(2025, 12, 17, 10, 16, 43, 123400000, locMoscow), Valid: true},
			Result: new(fbtype.Timetz),
			Test:   sameFbtypeTimetzClockAndOffset(time.Date(2000, 1, 1, 10, 16, 43, 123400000, locMoscow)),
		},
		{
			Param:  time.Date(2025, 12, 10, 18, 0, 0, 123400000, offsetPlus330),
			Result: new(time.Time),
			Test:   sameTimetzClockAndOffset(time.Date(2000, 1, 1, 18, 0, 0, 123400000, offsetPlus330)),
		},
		{
			Param:  "11:31:12.1234 +03:30",
			Result: new(fbtype.Timetz),
			Test:   sameFbtypeTimetzClockAndOffset(time.Date(2000, 1, 1, 11, 31, 12, 123400000, offsetPlus330)),
		},
		{
			Param:  nil,
			Result: new(fbtype.Timetz),
			Test: func(v any) bool {
				return !v.(fbtype.Timetz).Valid
			},
		},
	})
}
