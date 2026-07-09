package stdlib_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbtype"
)

func assertSameTimeOfDayAndOffset(t *testing.T, got, want time.Time) {
	t.Helper()

	assert.Equal(t, want.Hour(), got.Hour())
	assert.Equal(t, want.Minute(), got.Minute())
	assert.Equal(t, want.Second(), got.Second())
	assert.Equal(t, want.Nanosecond(), got.Nanosecond())

	_, gotOffset := got.Zone()
	_, wantOffset := want.Zone()
	assert.Equal(t, wantOffset, gotOffset)
}

func TestStdlibTimetzRoundTrip(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		if getServerMajorVersion(t, db) <= 3 {
			t.Skip("time with time zone is not supported by server engine version <= 3")
		}

		_, err := db.Exec("recreate table stdlib_timetz_roundtrip (id integer primary key, tm time with time zone)")
		require.NoError(t, err)
		defer func() { _, _ = db.Exec("drop table stdlib_timetz_roundtrip") }()

		locMoscow, err := time.LoadLocation("Europe/Moscow")
		require.NoError(t, err)
		offsetPlus330 := time.FixedZone("+03:30", (3*60+30)*60)

		tests := []struct {
			id    int
			param any
			want  time.Time
		}{
			{
				id:    1,
				param: fbtype.Timetz{Time: time.Date(2025, 12, 17, 10, 16, 43, 123400000, locMoscow), Valid: true},
				want:  time.Date(2020, 1, 1, 10, 16, 43, 123400000, locMoscow),
			},
			{
				id:    2,
				param: "18:00:00.1234 +03:30",
				want:  time.Date(2020, 1, 1, 18, 0, 0, 123400000, offsetPlus330),
			},
		}

		for _, tt := range tests {
			_, err := db.Exec("insert into stdlib_timetz_roundtrip (id, tm) values (?, ?)", tt.id, tt.param)
			require.NoError(t, err)
		}

		for _, tt := range tests {
			var got time.Time
			err := db.QueryRow("select tm from stdlib_timetz_roundtrip where id = ?", tt.id).Scan(&got)
			require.NoError(t, err)
			assertSameTimeOfDayAndOffset(t, got, tt.want)

			var gotNull sql.NullTime
			err = db.QueryRow("select tm from stdlib_timetz_roundtrip where id = ?", tt.id).Scan(&gotNull)
			require.NoError(t, err)
			require.True(t, gotNull.Valid)
			assertSameTimeOfDayAndOffset(t, gotNull.Time, tt.want)

			var gotFbtype fbtype.Timetz
			err = db.QueryRow("select tm from stdlib_timetz_roundtrip where id = ?", tt.id).Scan(&gotFbtype)
			require.NoError(t, err)
			require.True(t, gotFbtype.Valid)
			assertSameTimeOfDayAndOffset(t, gotFbtype.Time, tt.want)

			var gotString string
			err = db.QueryRow("select tm from stdlib_timetz_roundtrip where id = ?", tt.id).Scan(&gotString)
			require.NoError(t, err)
			gotStringTime, err := time.Parse(time.RFC3339Nano, gotString)
			require.NoError(t, err)
			assertSameTimeOfDayAndOffset(t, gotStringTime, tt.want)
		}
	})
}

func TestStdlibTimetzNullAndErrorCases(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		if getServerMajorVersion(t, db) <= 3 {
			t.Skip("time with time zone is not supported by server engine version <= 3")
		}

		var nullTime sql.NullTime
		err := db.QueryRow("select cast(null as time with time zone) from rdb$database").Scan(&nullTime)
		require.NoError(t, err)
		assert.False(t, nullTime.Valid)

		var plainTime time.Time
		err = db.QueryRow("select cast(null as time with time zone) from rdb$database").Scan(&plainTime)
		require.Error(t, err)

		invalidInputs := []string{
			"not-a-time",
			"25:30:45 +03:00",
			"12:00:00 +24:00",
			"",
		}

		for _, input := range invalidInputs {
			var got time.Time
			err := db.QueryRow("select cast(? as time with time zone) from rdb$database", input).Scan(&got)
			require.Error(t, err, "input %q", input)
		}
	})
}
