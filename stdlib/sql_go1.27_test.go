//go:build go1.27

package stdlib_test

import (
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fbx "github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/fbtype"
	"github.com/firebirdsql/fbx/stdlib"
)

func testWithKnownTypeQueryExecModes(t *testing.T, f func(t *testing.T, db *sql.DB)) {
	for _, mode := range []fbx.QueryExecMode{
		fbx.QueryExecModeCacheStatement,
		fbx.QueryExecModeExec,
	} {
		t.Run(mode.String(),
			func(t *testing.T) {
				config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
				require.NoError(t, err)

				config.DefaultQueryExecMode = mode
				db := stdlib.OpenDB(*config)
				defer func() {
					err := db.Close()
					require.NoError(t, err)
				}()

				f(t, db)

				ensureDBValid(t, db)
			},
		)
	}
}

func TestScanNonEmptyBlob(t *testing.T) {
	db := openDB(t)

	rows, err := db.Query("select cast('blob' as blob sub_type text) from rdb$database")
	require.NoError(t, err)
	require.True(t, rows.Next())
	var blob stdlib.Blob
	err = rows.Scan(&blob)
	// Register cleanup after Scan so a panic inside database/sql does not hang in Close.
	defer closeDB(t, db)
	defer closeResource(t, rows)
	require.NoError(t, err)
	defer closeResource(t, &blob)

	text, err := blob.Text()
	require.NoError(t, err)
	require.Equal(t, "blob", text)
}

func TestScanBlobIntoInterface(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		tests := []struct {
			name    string
			query   string
			subType int32
		}{
			{name: "text", query: "select cast('blob' as blob sub_type text) from rdb$database", subType: 1},
			{name: "binary", query: "select cast('blob' as blob sub_type binary) from rdb$database", subType: 0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var actual any = "previous value"
				err := db.QueryRow(tt.query).Scan(&actual)
				require.NoError(t, err)
				blob, ok := actual.(stdlib.Blob)
				require.True(t, ok)
				defer closeResource(t, &blob)
				require.Equal(t, tt.subType, blob.SubType)

				data, err := blob.Bytes()
				require.NoError(t, err)
				require.Equal(t, []byte("blob"), data)
			})
		}
	})
}

func TestScanBlobNullClearsDestination(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		var actual any = "previous value"
		err := db.QueryRow("select cast(null as blob sub_type text) from rdb$database").Scan(&actual)
		require.NoError(t, err)
		require.Nil(t, actual)

		var blob stdlib.Blob
		err = db.QueryRow("select cast('blob' as blob sub_type text) from rdb$database").Scan(&blob)
		require.NoError(t, err)
		closeResource(t, &blob)

		err = db.QueryRow("select cast(null as blob sub_type text) from rdb$database").Scan(&blob)
		require.NoError(t, err)
		require.Equal(t, stdlib.Blob{}, blob)

		err = db.QueryRow("select cast('' as blob sub_type text) from rdb$database").Scan(&actual)
		require.NoError(t, err)
		emptyBlob, ok := actual.(stdlib.Blob)
		require.True(t, ok)
		defer closeResource(t, &emptyBlob)
		data, err := emptyBlob.Bytes()
		require.NoError(t, err)
		require.Empty(t, data)
	})
}

func TestScanBlobPreservesScannerError(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		for _, query := range []string{
			"select cast('blob' as blob sub_type text) from rdb$database",
			"select cast('blob' as blob sub_type binary) from rdb$database",
			"select cast(null as blob sub_type text) from rdb$database",
		} {
			t.Run(query, func(t *testing.T) {
				errRejected := errors.New("blob rejected by scanner")
				s := &scanOnce{err: errRejected}
				err := db.QueryRow(query).Scan(s)
				require.ErrorIs(t, err, errRejected)
				require.Equal(t, 1, s.calls)
			})
		}
	})
}

func TestScanBlobRejectsIncompatibleDestination(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		for _, query := range []string{
			"select cast('blob' as blob sub_type text) from rdb$database",
			"select cast(null as blob sub_type text) from rdb$database",
		} {
			t.Run(query, func(t *testing.T) {
				actual := 42
				err := db.QueryRow(query).Scan(&actual)
				require.Error(t, err)
				require.Equal(t, 42, actual)
			})
		}
	})
}

func TestScanBlobMixedColumns(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		var number int
		var first, second stdlib.Blob
		var nullBlob any = "previous value"
		err := db.QueryRow(`select cast(42 as integer),
			cast('first' as blob sub_type text),
			cast(null as blob sub_type text),
			cast('second' as blob sub_type text)
			from rdb$database`).Scan(&number, &first, &nullBlob, &second)
		require.NoError(t, err)
		defer closeResource(t, &first)
		defer closeResource(t, &second)
		require.Equal(t, 42, number)
		require.Nil(t, nullBlob)

		text, err := first.Text()
		require.NoError(t, err)
		require.Equal(t, "first", text)
		text, err = second.Text()
		require.NoError(t, err)
		require.Equal(t, "second", text)
	})
}

func TestGoArray(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table go_array_test (id integer primary key, names varchar(10)[2])")
		require.NoError(t, err)

		expected := []string{"John", "Jane"}
		_, err = db.Exec("insert into go_array_test values (?, ?)", 1, expected)
		require.NoError(t, err)

		var names []string
		err = db.QueryRow("select names from go_array_test where id = ?", 1).Scan(&names)
		require.NoError(t, err)
		require.Equal(t, expected, names)

		_, err = db.Exec("insert into go_array_test values (?, ?)", 2, names)
		require.NoError(t, err)
		names = nil
		err = db.QueryRow("select names from go_array_test where id = ?", 2).Scan(&names)
		require.NoError(t, err)
		require.Equal(t, expected, names)

		_, err = db.Exec("insert into go_array_test values (?, ?)", 3, nil)
		require.NoError(t, err)
		err = db.QueryRow("select names from go_array_test where id = ?", 3).Scan(&names)
		require.NoError(t, err)
		require.Nil(t, names)
	})
}

func TestFbTypeFlatArray(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table go_flat_array_test (id integer primary key, names varchar(10)[2])")
		require.NoError(t, err)

		expected := fbtype.FlatArray[string]{"John", "Jane"}
		_, err = db.Exec("insert into go_flat_array_test values (?, ?)", 1, expected)
		require.NoError(t, err)

		var names fbtype.FlatArray[string]
		err = db.QueryRow("select names from go_flat_array_test where id = ?", 1).Scan(&names)
		require.NoError(t, err)
		require.Equal(t, expected, names)

		_, err = db.Exec("insert into go_flat_array_test values (?, ?)", 2, names)
		require.NoError(t, err)
		names = nil
		err = db.QueryRow("select names from go_flat_array_test where id = ?", 2).Scan(&names)
		require.NoError(t, err)
		require.Equal(t, expected, names)

		_, err = db.Exec("insert into go_flat_array_test values (?, ?)", 3, nil)
		require.NoError(t, err)
		err = db.QueryRow("select names from go_flat_array_test where id = ?", 3).Scan(&names)
		require.NoError(t, err)
		require.Nil(t, names)
	})
}

func TestFbTypeArray(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table go_matrix_test (id integer primary key, matrix bigint[2,3])")
		require.NoError(t, err)

		expected := fbtype.Array[int64]{
			Elements: []int64{1, 2, 3, 4, 5, 6},
			Dims: []fbtype.ArrayDimension{
				{Length: 2, LowerBound: 1},
				{Length: 3, LowerBound: 1},
			},
			Valid: true,
		}
		_, err = db.Exec("insert into go_matrix_test values (?, ?)", 1, expected)
		require.NoError(t, err)

		var matrix fbtype.Array[int64]
		err = db.QueryRow("select matrix from go_matrix_test where id = ?", 1).Scan(&matrix)
		require.NoError(t, err)
		require.Equal(t, expected, matrix)

		_, err = db.Exec("insert into go_matrix_test values (?, ?)", 2, matrix)
		require.NoError(t, err)
		matrix = fbtype.Array[int64]{}
		err = db.QueryRow("select matrix from go_matrix_test where id = ?", 2).Scan(&matrix)
		require.NoError(t, err)
		require.Equal(t, expected, matrix)

		_, err = db.Exec("insert into go_matrix_test values (?, ?)", 3, nil)
		require.NoError(t, err)
		err = db.QueryRow("select matrix from go_matrix_test where id = ?", 3).Scan(&matrix)
		require.NoError(t, err)
		require.Equal(t, fbtype.Array[int64]{}, matrix)
	})
}

// Boolean stringification must agree across query execution modes.
func TestScanBoolToString(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		var s string
		err := db.QueryRow("select true from rdb$database").Scan(&s)
		require.NoError(t, err)
		require.Equal(t, "true", s)

		err = db.QueryRow("select false from rdb$database").Scan(&s)
		require.NoError(t, err)
		require.Equal(t, "false", s)
	})
}

func TestScanInterfacePreservesDatabaseSQLValues(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		for _, tt := range []struct {
			query string
			want  any
		}{
			{"select cast(42 as smallint) from rdb$database", int64(42)},
			{"select cast(42 as integer) from rdb$database", int64(42)},
			{"select cast(42 as bigint) from rdb$database", int64(42)},
			{"select cast(1.5 as float) from rdb$database", float64(1.5)},
			{"select cast(1.25 as numeric(9, 2)) from rdb$database", "1.25"},
			{"select cast(x'39303037313939323534373430393933' as varchar(16) character set octets) from rdb$database", []byte("9007199254740993")},
			{"select cast(x'6e756c6c' as varchar(4) character set octets) from rdb$database", []byte("null")},
			{"select cast(null as integer) from rdb$database", nil},
		} {
			t.Run(tt.query, func(t *testing.T) {
				var got any
				err := db.QueryRow(tt.query).Scan(&got)
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)

				var indirect *any
				err = db.QueryRow(tt.query).Scan(&indirect)
				require.NoError(t, err)
				if tt.want == nil {
					assert.Nil(t, indirect)
				} else {
					require.NotNil(t, indirect)
					assert.Equal(t, tt.want, *indirect)
				}
			})
		}
	})
}

func TestScanInterfacePreservesArray(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table go_interface_array_test (id integer primary key, numbers integer[2])")
		require.NoError(t, err)
		_, err = db.Exec("insert into go_interface_array_test values (?, ?)", 1, []int32{1, 2})
		require.NoError(t, err)

		var got any
		err = db.QueryRow("select numbers from go_interface_array_test where id = ?", 1).Scan(&got)
		require.NoError(t, err)
		array, ok := got.(fbtype.ArraySQLScanner)
		require.True(t, ok)
		var actual []int32
		require.NoError(t, fbtype.NewMap().SQLScanner(&actual).Scan(array))
		require.Equal(t, []int32{1, 2}, actual)

		var indirect *any
		err = db.QueryRow("select numbers from go_interface_array_test where id = ?", 1).Scan(&indirect)
		require.NoError(t, err)
		require.NotNil(t, indirect)
		array, ok = (*indirect).(fbtype.ArraySQLScanner)
		require.True(t, ok)
		actual = nil
		require.NoError(t, fbtype.NewMap().SQLScanner(&actual).Scan(array))
		require.Equal(t, []int32{1, 2}, actual)
	})
}

func TestScanScalarPreservesDatabaseSQLConversions(t *testing.T) {
	type namedString string
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		for _, tt := range []struct{ query, want string }{
			{"select true from rdb$database", "true"},
			{"select cast('2026-09-07' as date) from rdb$database", "2026-09-07T00:00:00Z"},
			{"select cast(x'6162' as varchar(2) character set octets) from rdb$database", "ab"},
		} {
			t.Run(tt.query, func(t *testing.T) {
				var s string
				err := db.QueryRow(tt.query).Scan(&s)
				require.NoError(t, err)
				assert.Equal(t, tt.want, s)

				var b []byte
				err = db.QueryRow(tt.query).Scan(&b)
				require.NoError(t, err)
				assert.Equal(t, []byte(tt.want), b)

				var indirect *string
				err = db.QueryRow(tt.query).Scan(&indirect)
				require.NoError(t, err)
				require.NotNil(t, indirect)
				assert.Equal(t, tt.want, *indirect)

				rows, err := db.Query(tt.query)
				require.NoError(t, err)
				defer closeResource(t, rows)
				require.True(t, rows.Next())
				var raw sql.RawBytes
				err = rows.Scan(&raw)
				require.NoError(t, err)
				assert.Equal(t, sql.RawBytes(tt.want), raw)
				require.NoError(t, rows.Close())
			})
		}

		var named namedString
		err := db.QueryRow("select cast(x'6162' as varchar(2) character set octets) from rdb$database").Scan(&named)
		require.NoError(t, err)
		assert.Equal(t, namedString("ab"), named)

		var s string
		err = db.QueryRow("select cast(null as varchar(10)) from rdb$database").Scan(&s)
		require.Error(t, err)
	})
}

type scanOnce struct {
	calls int
	err   error
}

func (s *scanOnce) Scan(any) error {
	s.calls++
	if s.calls == 1 {
		return s.err
	}
	return nil
}

func TestScanPreservesScannerError(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		for _, query := range []string{"select cast(42 as integer) from rdb$database", "select cast(x'31' as varchar(1) character set octets) from rdb$database", "select cast(null as integer) from rdb$database"} {
			errRejected := errors.New("value rejected by scanner")
			s := &scanOnce{err: errRejected}
			err := db.QueryRow(query).Scan(s)
			assert.ErrorIs(t, err, errRejected)
			assert.Equal(t, 1, s.calls)
		}
	})
}

func TestScanRejectsNilDestination(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		for _, dest := range []any{nil, (*int64)(nil), (*string)(nil), (*[]int32)(nil), (*sql.NullString)(nil)} {
			for _, query := range []string{"select cast(42 as integer) from rdb$database", "select cast(null as integer) from rdb$database"} {
				err := db.QueryRow(query).Scan(dest)
				require.Error(t, err)
			}
		}
	})
}

func TestScanNativeByteSlices(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table go_byte_array_test (id integer primary key, numbers smallint[3])")
		require.NoError(t, err)
		_, err = db.Exec("insert into go_byte_array_test values (?, ?)", 1, fbtype.FlatArray[byte]{1, 2, 3})
		require.NoError(t, err)

		var a fbtype.FlatArray[byte]
		err = db.QueryRow("select numbers from go_byte_array_test where id = ?", 1).Scan(&a)
		require.NoError(t, err)
		assert.Equal(t, fbtype.FlatArray[byte]{1, 2, 3}, a)

		backing := make([]byte, 16)
		b := fbtype.PreallocBytes(backing)
		err = db.QueryRow("select cast(x'6162' as varchar(2) character set octets) from rdb$database").Scan(&b)
		require.NoError(t, err)
		assert.Equal(t, fbtype.PreallocBytes("ab"), b)
		assert.Same(t, &backing[0], &b[0])
	})
}
