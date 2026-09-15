package fbtype_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
	"rdb.red-soft.ru/fbx/stdlib"
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
			Test:   sameFbtypeTimetzClockAndOffset(time.Date(2020, 1, 1, 10, 16, 43, 123400000, locMoscow)),
		},
		{
			Param:  time.Date(2025, 12, 10, 18, 0, 0, 123400000, offsetPlus330),
			Result: new(time.Time),
			Test:   sameTimetzClockAndOffset(time.Date(2020, 1, 1, 18, 0, 0, 123400000, offsetPlus330)),
		},
		{
			Param:  "11:31:12.1234 +03:30",
			Result: new(fbtype.Timetz),
			Test:   sameFbtypeTimetzClockAndOffset(time.Date(2020, 1, 1, 11, 31, 12, 123400000, offsetPlus330)),
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

func TestTimestamptzDSTTranscode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defaultConnTestRunner.RunTest(ctx, t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		if getServerMajorVersionForConn(ctx, tb, conn) <= 3 {
			t.Skip("timestamp with time zone is not supported by server engine version <= 3")
		}
		loc, err := time.LoadLocation("Europe/Berlin")
		require.NoError(t, err)
		db := stdlib.OpenDB(*conn.Config())
		defer db.Close()
		const query = "select cast(? as timestamp with time zone) from rdb$database"
		for _, tc := range []struct {
			name, text string
			utcHour    int
		}{
			{"first", "2025-10-26T02:30:00.123400+02:00[Europe/Berlin]", 0},
			{"second", "2025-10-26T02:30:00.123400+01:00[Europe/Berlin]", 1},
		} {
			t.Run(tc.name, func(t *testing.T) {
				original := time.Date(2025, 10, 26, tc.utcHour, 30, 0, 123400000, time.UTC).In(loc)
				checkTime := func(got time.Time) {
					t.Helper()
					require.True(t, got.Equal(original), "got %s, want %s", got, original)
					require.Equal(t, "Europe/Berlin", got.Location().String())
					_, wantOffset := original.Zone()
					_, gotOffset := got.Zone()
					require.Equal(t, wantOffset, gotOffset)
				}

				var text string
				require.NoError(t, conn.QueryRow(ctx, query, original).Scan(&text))
				require.Equal(t, tc.text, text)
				var native fbtype.Timestamptz
				require.NoError(t, conn.QueryRow(ctx, query, text).Scan(&native))
				require.True(t, native.Valid)
				checkTime(native.Time)

				b, err := json.Marshal(native)
				require.NoError(t, err)
				wantJSON, err := json.Marshal(tc.text)
				require.NoError(t, err)
				require.Equal(t, string(wantJSON), string(b))
				var restored fbtype.Timestamptz
				require.NoError(t, json.Unmarshal(b, &restored))
				checkTime(restored.Time)
				var roundTrip time.Time
				require.NoError(t, conn.QueryRow(ctx, query, restored).Scan(&roundTrip))
				checkTime(roundTrip)

				for _, value := range []any{original, text, restored} {
					var sqlValue fbtype.Timestamptz
					require.NoError(t, db.QueryRowContext(ctx, query, value).Scan(&sqlValue))
					require.True(t, sqlValue.Valid)
					checkTime(sqlValue.Time)
				}
			})
		}
	})
}
