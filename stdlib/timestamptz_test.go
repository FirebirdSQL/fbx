package stdlib_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbtype"
)

func TestStdlibTimestamptzRoundTripPreservesNamedLocation(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		if getServerMajorVersion(t, db) <= 3 {
			t.Skip("timestamp with time zone is not supported by server engine version <= 3")
		}

		_, err := db.Exec("recreate table stdlib_timestamptz_roundtrip (id integer primary key, ts timestamp with time zone)")
		require.NoError(t, err)
		defer func() { _, _ = db.Exec("drop table stdlib_timestamptz_roundtrip") }()

		locMoscow, err := time.LoadLocation("Europe/Moscow")
		require.NoError(t, err)
		locNewYork, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)
		locKolkata, err := time.LoadLocation("Asia/Kolkata")
		require.NoError(t, err)

		tests := []struct {
			id int
			ts time.Time
		}{
			{id: 1, ts: time.Date(2013, 1, 2, 3, 4, 5, 600000, locMoscow)},
			{id: 2, ts: time.Date(2025, 6, 1, 12, 30, 45, 123400000, locNewYork)},
			{id: 3, ts: time.Date(2025, 12, 10, 18, 0, 0, 0, locKolkata)},
		}

		for _, tt := range tests {
			_, err := db.Exec("insert into stdlib_timestamptz_roundtrip (id, ts) values (?, ?)", tt.id, tt.ts)
			require.NoError(t, err)
		}

		for _, tt := range tests {
			var got time.Time
			err := db.QueryRow("select ts from stdlib_timestamptz_roundtrip where id = ?", tt.id).Scan(&got)
			require.NoError(t, err)
			assert.Equal(t, tt.ts, got)

			var gotNull sql.NullTime
			err = db.QueryRow("select ts from stdlib_timestamptz_roundtrip where id = ?", tt.id).Scan(&gotNull)
			require.NoError(t, err)
			require.True(t, gotNull.Valid)
			assert.Equal(t, tt.ts, gotNull.Time)

			var gotFbtype fbtype.Timestamptz
			err = db.QueryRow("select ts from stdlib_timestamptz_roundtrip where id = ?", tt.id).Scan(&gotFbtype)
			require.NoError(t, err)
			require.True(t, gotFbtype.Valid)
			assert.Equal(t, tt.ts, gotFbtype.Time)

			var gotString string
			err = db.QueryRow("select ts from stdlib_timestamptz_roundtrip where id = ?", tt.id).Scan(&gotString)
			require.NoError(t, err)
			gotStringTime, err := time.Parse(time.RFC3339Nano, gotString)
			require.NoError(t, err)
			assert.True(t, gotStringTime.Equal(tt.ts), "string scan %q does not match %s", gotString, tt.ts)
		}
	})
}

func TestStdlibTimestamptzNullAndErrorCases(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		if getServerMajorVersion(t, db) <= 3 {
			t.Skip("timestamp with time zone is not supported by server engine version <= 3")
		}

		var nullTime sql.NullTime
		err := db.QueryRow("select cast(null as timestamp with time zone) from rdb$database").Scan(&nullTime)
		require.NoError(t, err)
		assert.False(t, nullTime.Valid)

		var plainTime time.Time
		err = db.QueryRow("select cast(null as timestamp with time zone) from rdb$database").Scan(&plainTime)
		require.Error(t, err)

		invalidInputs := []string{
			"not-a-timestamp",
			"2024-13-01T14:30:45Z",
			"2024-01-15T25:30:45Z",
			"",
		}

		for _, input := range invalidInputs {
			var got time.Time
			err := db.QueryRow("select cast(? as timestamp with time zone) from rdb$database", input).Scan(&got)
			require.Error(t, err, "input %q", input)
			assert.Contains(t, err.Error(), "cannot parse", "input %q", input)
		}
	})
}
