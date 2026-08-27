package stdlib_test

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxpool"
	"rdb.red-soft.ru/fbx/stdlib"
	"rdb.red-soft.ru/fbx/tracelog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openDB(t testing.TB, opts ...stdlib.OptionOpenDB) *sql.DB {
	t.Helper()
	config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	return stdlib.OpenDB(*config, opts...)
}

func getServerMajorVersion(t testing.TB, db *sql.DB) int {
	t.Helper()
	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()

	var major int
	err = conn.Raw(func(driverConn any) error {
		major = driverConn.(*stdlib.Conn).Conn().ServerVersion().Major
		return nil
	})
	require.NoError(t, err)
	return major
}

func closeDB(t testing.TB, db *sql.DB) {
	err := db.Close()
	require.NoError(t, err)
}

func testWithAllQueryExecModes(t *testing.T, f func(t *testing.T, db *sql.DB)) {
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

// Do a simple query to ensure the DB is still usable. This is of less use in stdlib as the connection pool should
// cover broken connections.
func ensureDBValid(t testing.TB, db *sql.DB) {
	var sum, rowCount int32

	rows, err := db.Query(generateSeriesQuery("1", "?", ""), 10)
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var n int32
		rows.Scan(&n)
		sum += n
		rowCount++
	}

	require.NoError(t, rows.Err())

	if rowCount != 10 {
		t.Error("Select called onDataRow wrong number of times")
	}
	if sum != 55 {
		t.Error("Wrong values returned")
	}
}

type preparer interface {
	Prepare(query string) (*sql.Stmt, error)
}

func prepareStmt(t *testing.T, p preparer, sql string) *sql.Stmt {
	stmt, err := p.Prepare(sql)
	require.NoError(t, err)
	return stmt
}

func closeStmt(t *testing.T, stmt *sql.Stmt) {
	err := stmt.Close()
	require.NoError(t, err)
}

func TestSQLOpen(t *testing.T) {
	tests := []struct {
		driverName string
	}{
		{driverName: "fbx"},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.driverName, func(t *testing.T) {
			db, err := sql.Open(tt.driverName, os.Getenv("FBX_TEST_DATABASE"))
			require.NoError(t, err)
			closeDB(t, db)
		})
	}
}

func TestStdlibBlobBinary(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table stdlib_blob_bin(id integer primary key, data blob sub_type 0)")
		require.NoError(t, err)

		data := make([]byte, 1024*1024*10)
		for i := range data {
			data[i] = byte((i % 251) + 1)
		}

		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()

		_, err = tx.Exec("insert into stdlib_blob_bin(id, data) values (?, ?)", 1, bytes.NewReader(data))
		require.NoError(t, err)

		rows, err := tx.Query("select data from stdlib_blob_bin where id = ?", 1)
		require.NoError(t, err)

		require.True(t, rows.Next())
		var blob stdlib.Blob
		err = rows.Scan(&blob)
		require.NoError(t, err)
		rows.Close()
		require.NoError(t, rows.Err())

		got, err := blob.Bytes()
		require.NoError(t, err)
		require.Equal(t, data, got)

		require.NoError(t, tx.Commit())
	})
}

func TestLazyBlobRemainsReadableAfterQueryRowReleasesConnection(t *testing.T) {
	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("FBX_TEST_DATABASE is not set")
	}

	ctx := context.Background()
	config, err := fbx.ParseConfig(connString)
	require.NoError(t, err)

	config.MaxInlineBlobSize = 0
	db := stdlib.OpenDB(*config)
	db.SetMaxIdleConns(0)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	_, err = db.ExecContext(ctx, `
recreate table stdlib_blob_query_row_lease (
	id integer primary key,
	data blob sub_type 0
)`)
	require.NoError(t, err)

	want := bytes.Repeat([]byte("lazy-blob-lease-regression\n"), 8192)
	_, err = db.ExecContext(
		ctx,
		"insert into stdlib_blob_query_row_lease(id, data) values (?, ?)",
		1,
		bytes.NewReader(want),
	)
	require.NoError(t, err)

	var blob stdlib.Blob
	err = db.QueryRowContext(
		ctx,
		"select data from stdlib_blob_query_row_lease where id = ?",
		1,
	).Scan(&blob)
	require.NoError(t, err)
	require.Zero(t, db.Stats().OpenConnections, "QueryRow.Scan must release the connection lease")

	got, err := blob.Bytes()
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestStdlibBlobWithArray(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table stdlib_blob_array(id integer primary key, data blob sub_type 0, numbers bigint[3])")
		require.NoError(t, err)

		inputBlob := []byte("blob-array-result")
		inputNumbers := []int64{7, 8, 9}

		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()

		_, err = tx.Exec(
			"insert into stdlib_blob_array(id, data, numbers) values (?, ?, ?)",
			1,
			bytes.NewReader(inputBlob),
			inputNumbers,
		)
		require.NoError(t, err)

		rows, err := tx.Query("select data, numbers from stdlib_blob_array where id = ?", 1)
		require.NoError(t, err)

		require.True(t, rows.Next())
		var (
			blob    stdlib.Blob
			numbers []int64
		)
		m := fbtype.NewMap()
		err = rows.Scan(&blob, m.SQLScanner(&numbers))
		require.NoError(t, err)
		rows.Close()
		require.NoError(t, rows.Err())

		gotBlob, err := blob.Bytes()
		require.NoError(t, err)
		require.Equal(t, inputBlob, gotBlob)
		require.Equal(t, inputNumbers, numbers)

		require.NoError(t, tx.Commit())
	})
}

func TestStdlibBlobText(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		tests := []struct {
			name    string
			charset string
			text    string
		}{
			{name: "utf8", charset: "UTF8", text: "blob-utf-" + string([]rune{0x20AC, 0x00A9})},
			{name: "unicode_fss", charset: "UNICODE_FSS", text: "blob-fss-" + string([]rune{0x0422, 0x0435, 0x0441, 0x0442})},
			{name: "ascii", charset: "ASCII", text: "blob-ascii"},
		}

		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				table := fmt.Sprintf("stdlib_blob_text_%s", tc.name)
				_, err := db.Exec(fmt.Sprintf("recreate table %s(id integer primary key, data blob sub_type 1 character set %s)", table, tc.charset))
				if err != nil {
					t.Skipf("charset %s not supported: %v", tc.charset, err)
				}

				_, err = db.Exec(fmt.Sprintf("insert into %s(id, data) values (?, ?)", table), 1, tc.text)
				require.NoError(t, err)

				tx, err := db.Begin()
				require.NoError(t, err)
				defer func() { _ = tx.Rollback() }()

				rows, err := tx.Query(fmt.Sprintf("select data from %s where id = ?", table), 1)
				require.NoError(t, err)

				require.True(t, rows.Next())
				var blob stdlib.Blob
				err = rows.Scan(&blob)
				require.NoError(t, err)
				rows.Close()
				require.NoError(t, rows.Err())

				got, err := blob.Text()
				require.NoError(t, err)
				require.Equal(t, tc.text, got)

				require.NoError(t, tx.Commit())
			})
		}
	})
}

func TestSQLOpenFromPool(t *testing.T) {
	pool, err := fbxpool.New(context.Background(), os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	db := stdlib.OpenDBFromPool(pool)
	ensureDBValid(t, db)

	db.Close()
}

func TestNormalLifeCycle(t *testing.T) {
	db := openDB(t)
	defer closeDB(t, db)

	stmt := prepareStmt(t, db, "select 'foo', n from ("+generateSeriesQuery("?", "?", "n")+")")
	defer closeStmt(t, stmt)

	rows, err := stmt.Query(int32(1), int32(10))
	require.NoError(t, err)

	rowCount := int64(0)

	for rows.Next() {
		rowCount++

		var s string
		var n int64
		err := rows.Scan(&s, &n)
		require.NoError(t, err)

		if s != "foo" {
			t.Errorf(`Expected "foo", received "%v"`, s)
		}
		if n != rowCount {
			t.Errorf("Expected %d, received %d", rowCount, n)
		}
	}
	require.NoError(t, rows.Err())

	require.EqualValues(t, 10, rowCount)

	err = rows.Close()
	require.NoError(t, err)

	ensureDBValid(t, db)
}

func TestStmtExec(t *testing.T) {
	db := openDB(t)
	defer closeDB(t, db)

	tx, err := db.Begin()
	require.NoError(t, err)

	createStmt := prepareStmt(t, tx, "recreate table t(a varchar(255) not null)")
	_, err = createStmt.Exec()
	require.NoError(t, err)
	closeStmt(t, createStmt)
	require.NoError(t, tx.Commit())

	//original test executed both statements in same transaction, but FB do not support
	//using table in same transaction it was created
	tx, err = db.Begin()
	require.NoError(t, err)

	insertStmt := prepareStmt(t, tx, "insert into t values(cast(? as varchar(255)))")
	result, err := insertStmt.Exec("foo")
	require.NoError(t, err)

	n, err := result.RowsAffected()
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	closeStmt(t, insertStmt)
	require.NoError(t, tx.Rollback())

	ensureDBValid(t, db)
}

func TestQueryCloseRowsEarly(t *testing.T) {
	db := openDB(t)
	defer closeDB(t, db)

	stmt := prepareStmt(t, db, "select 'foo', n from ("+generateSeriesQuery("?", "?", "n")+")")
	defer closeStmt(t, stmt)

	rows, err := stmt.Query(int32(1), int32(10))
	require.NoError(t, err)

	// Close rows immediately without having read them
	err = rows.Close()
	require.NoError(t, err)

	// Run the query again to ensure the connection and statement are still ok
	rows, err = stmt.Query(int32(1), int32(10))
	require.NoError(t, err)

	rowCount := int64(0)

	for rows.Next() {
		rowCount++

		var s string
		var n int64
		err := rows.Scan(&s, &n)
		require.NoError(t, err)
		if s != "foo" {
			t.Errorf(`Expected "foo", received "%v"`, s)
		}
		if n != rowCount {
			t.Errorf("Expected %d, received %d", rowCount, n)
		}
	}
	require.NoError(t, rows.Err())
	require.EqualValues(t, 10, rowCount)

	err = rows.Close()
	require.NoError(t, err)

	ensureDBValid(t, db)
}

func TestConnExec(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table t(a varchar(255) not null)")
		require.NoError(t, err)

		result, err := db.Exec("insert into t values('hey')")
		require.NoError(t, err)

		n, err := result.RowsAffected()
		require.NoError(t, err)
		require.EqualValues(t, 1, n)
	})
}

func TestConnQuery(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		rows, err := db.Query("select 'foo', n from ("+generateSeriesQuery("?", "?", "n")+")", int32(1), int32(10))
		require.NoError(t, err)

		rowCount := int64(0)

		for rows.Next() {
			rowCount++

			var s string
			var n int64
			err := rows.Scan(&s, &n)
			require.NoError(t, err)
			if s != "foo" {
				t.Errorf(`Expected "foo", received "%v"`, s)
			}
			if n != rowCount {
				t.Errorf("Expected %d, received %d", rowCount, n)
			}
		}
		require.NoError(t, rows.Err())
		require.EqualValues(t, 10, rowCount)

		err = rows.Close()
		require.NoError(t, err)
	})
}

func TestConnConcurrency(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table t (id integer primary key, str varchar(255), dur_str bigint)")
		require.NoError(t, err)

		var wg sync.WaitGroup

		concurrency := 50
		errChan := make(chan error, concurrency)

		for i := 1; i <= concurrency; i++ {
			wg.Add(1)

			go func(idx int) {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
				defer cancel()

				str := strconv.Itoa(idx)
				duration := time.Duration(idx) * time.Second
				_, err := db.ExecContext(ctx, "insert into t(id) values(?)", idx)
				if err != nil {
					errChan <- fmt.Errorf("insert failed: %d %w", idx, err)
					return
				}
				_, err = db.ExecContext(ctx, "update t set str = ? where id = ?", str, idx)
				if err != nil {
					errChan <- fmt.Errorf("update 1 failed: %d %w", idx, err)
					return
				}
				_, err = db.ExecContext(ctx, "update t set dur_str = ? where id = ?", duration, idx)
				if err != nil {
					errChan <- fmt.Errorf("update 2 failed: %d %w", idx, err)
					return
				}

				errChan <- nil
			}(i)
		}
		wg.Wait()
		for i := 1; i <= concurrency; i++ {
			err := <-errChan
			require.NoError(t, err)
		}

		for i := 1; i <= concurrency; i++ {
			wg.Add(1)

			go func(idx int) {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
				defer cancel()

				var id int
				var str string
				var duration int64 //fbtype.Interval
				err := db.QueryRowContext(ctx, "select id,str,dur_str from t where id = ?", idx).Scan(&id, &str, &duration)
				if err != nil {
					errChan <- fmt.Errorf("select failed: %d %w", idx, err)
					return
				}
				if id != idx {
					errChan <- fmt.Errorf("id mismatch: %d %d", idx, id)
					return
				}
				if str != strconv.Itoa(idx) {
					errChan <- fmt.Errorf("str mismatch: %d %s", idx, str)
					return
				}
				//expectedDuration := fbtype.Interval{
				//	Microseconds: int64(idx) * time.Second.Microseconds(),
				//	Valid:        true,
				//}
				if duration != int64(idx)*1000000000 {
					errChan <- fmt.Errorf("duration mismatch: %d %v", idx, duration)
					return
				}

				errChan <- nil
			}(i)
		}
		wg.Wait()
		for i := 1; i <= concurrency; i++ {
			err := <-errChan
			require.NoError(t, err)
		}
	})
}

func TestConnQueryDifferentScanPlansIssue781(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		var s string
		var b bool

		rows, err := db.Query("select true, 'foo' from rdb$database")
		require.NoError(t, err)

		require.True(t, rows.Next())
		require.NoError(t, rows.Scan(&b, &s))
		assert.Equal(t, true, b)
		assert.Equal(t, "foo", s)
	})
}

func TestConnQueryNull(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		rows, err := db.Query("select cast(? as integer) from rdb$database", nil)
		require.NoError(t, err)

		rowCount := int64(0)

		for rows.Next() {
			rowCount++

			var n sql.NullInt64
			err := rows.Scan(&n)
			require.NoError(t, err)
			if n.Valid != false {
				t.Errorf("Expected n to be null, but it was %v", n)
			}
		}
		require.NoError(t, rows.Err())
		require.EqualValues(t, 1, rowCount)

		err = rows.Close()
		require.NoError(t, err)
	})
}

func TestConnQueryRowByteSlice(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		expected := []byte{222, 173, 190, 239}
		var actual []byte

		err := db.QueryRow(`select cast(x'deadbeef' as varchar(16) character set octets) from rdb$database`).Scan(&actual)
		require.NoError(t, err)
		require.EqualValues(t, expected, actual)
	})
}

func TestConnQueryFailure(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Query("select 'foo from rdb$database")
		require.Error(t, err)
		require.IsType(t, new(fbconn.FbError), err)
	})
}

func TestConnSimpleSlicePassThrough(t *testing.T) {
	t.Skip("need implementation")
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		var n int64
		err := db.QueryRow("select cardinality($1::text[])", []string{"a", "b", "c"}).Scan(&n)
		require.NoError(t, err)
		assert.EqualValues(t, 3, n)
	})
}

func TestConnQueryScanGoArray(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table scan_go_array_test (id integer primary key, array bigint[3])")
		assert.NoError(t, err)
		m := fbtype.NewMap()
		for i, tt := range []any{
			[]int64(nil),
			[]int64{},
			[]int64{1, 2, 3},
		} {
			var actual []int64
			_, err := db.Exec(
				"insert into scan_go_array_test values(?, ?)",
				i,
				tt,
			)
			assert.NoErrorf(t, err, "%d", i)

			err = db.QueryRow(
				"select array from scan_go_array_test where id = ?",
				i,
			).Scan(m.SQLScanner(&actual))
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt, actual, "%d", i)

		}
	})
}

func TestConnQueryScanGoArrayNullClearsDestination(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table scan_go_array_null_test (id integer primary key, array bigint[3])")
		require.NoError(t, err)

		_, err = db.Exec("insert into scan_go_array_null_test values (?, ?)", 1, []int64{1, 2, 3})
		require.NoError(t, err)
		_, err = db.Exec("insert into scan_go_array_null_test values (?, ?)", 2, []int64(nil))
		require.NoError(t, err)

		rows, err := db.Query("select array from scan_go_array_null_test order by id")
		require.NoError(t, err)
		defer rows.Close()

		m := fbtype.NewMap()
		actual := []int64{99}

		require.True(t, rows.Next())
		require.NoError(t, rows.Scan(m.SQLScanner(&actual)))
		require.Equal(t, []int64{1, 2, 3}, actual)

		require.True(t, rows.Next())
		require.NoError(t, rows.Scan(m.SQLScanner(&actual)))
		require.Nil(t, actual)

		require.False(t, rows.Next())
		require.NoError(t, rows.Err())
	})
}

func TestConnQueryArrayDriverValueOwnsFetchBuffer(t *testing.T) {
	config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	config.FetchBatchSize = 1

	db := stdlib.OpenDB(*config)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	defer closeDB(t, db)

	_, err = db.Exec(`recreate table scan_array_buffer_lifetime (
		id integer not null primary key,
		array bigint[3] not null
	)`)
	require.NoError(t, err)
	_, err = db.Exec("insert into scan_array_buffer_lifetime values (?, ?)", 1, []int64{1, 2, 3})
	require.NoError(t, err)
	_, err = db.Exec("insert into scan_array_buffer_lifetime values (?, ?)", 2, []int64{10, 20, 30})
	require.NoError(t, err)

	rows, err := db.Query("select array from scan_array_buffer_lifetime order by id")
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next())
	var firstRaw any
	require.NoError(t, rows.Scan(&firstRaw))
	first, ok := firstRaw.(fbtype.ArraySQLScanner)
	require.True(t, ok)
	firstBuffer := bytes.Clone(first.ArrayBuffer)

	require.True(t, rows.Next())
	var secondRaw any
	require.NoError(t, rows.Scan(&secondRaw))
	require.IsType(t, fbtype.ArraySQLScanner{}, secondRaw)

	require.Equal(t, firstBuffer, first.ArrayBuffer)
	var firstValues []int64
	require.NoError(t, fbtype.NewMap().SQLScanner(&firstValues).Scan(first))
	require.Equal(t, []int64{1, 2, 3}, firstValues)
}

func TestConnQueryScanArray(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table scan_array_test (id integer primary key, array bigint[3])")
		assert.NoError(t, err)
		m := fbtype.NewMap()
		for i, tt := range []any{
			fbtype.Array[int64]{Elements: nil, Dims: nil, Valid: false},
			fbtype.Array[int64]{Elements: []int64{}, Dims: []fbtype.ArrayDimension{{Length: 0, LowerBound: 1}}, Valid: true},
			fbtype.Array[int64]{Elements: []int64{1, 2, 3}, Dims: []fbtype.ArrayDimension{{Length: 3, LowerBound: 1}}, Valid: true},
		} {
			var actual fbtype.Array[int64]
			_, err := db.Exec(
				"insert into scan_array_test values(?, ?)",
				i,
				tt,
			)
			assert.NoErrorf(t, err, "%d", i)

			err = db.QueryRow(
				"select array from scan_array_test where id = ?",
				i,
			).Scan(m.SQLScanner(&actual))
			assert.NoErrorf(t, err, "%d", i)
			assert.Equalf(t, tt, actual, "%d", i)

		}
	})
}

// Test type that fbx would handle natively in binary, but since it is not a
// database/sql native type should be passed through as a string
func TestConnQueryRowFbxBinary(t *testing.T) {
	t.Skip()
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		sql := "select cast(? as integer) from rdb$database"
		expected := "{1,2,3}"
		var actual string

		err := db.QueryRow(sql, expected).Scan(&actual)
		require.NoError(t, err)
		require.EqualValues(t, expected, actual)
	})
}

func TestConnQueryJSONIntoByteSlice(t *testing.T) {
	t.Skip("need implementation")

	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec(`
		create temporary table docs(
			body json not null
		);

		insert into docs(body) values('{"foo": "bar"}');
`)
		require.NoError(t, err)

		sql := `select * from docs`
		expected := []byte(`{"foo": "bar"}`)
		var actual []byte

		err = db.QueryRow(sql).Scan(&actual)
		if err != nil {
			t.Errorf("Unexpected failure: %v (sql -> %v)", err, sql)
		}

		if !bytes.Equal(actual, expected) {
			t.Errorf(`Expected "%v", got "%v" (sql -> %v)`, string(expected), string(actual), sql)
		}

		_, err = db.Exec(`drop table docs`)
		require.NoError(t, err)
	})
}

func TestConnExecInsertByteSliceIntoJSON(t *testing.T) {
	t.Skip("need implementation")

	db := openDB(t)
	defer closeDB(t, db)

	_, err := db.Exec(`
		create temporary table docs(
			body json not null
		);
`)
	require.NoError(t, err)

	expected := []byte(`{"foo": "bar"}`)

	_, err = db.Exec(`insert into docs(body) values($1)`, expected)
	require.NoError(t, err)

	var actual []byte
	err = db.QueryRow(`select body from docs`).Scan(&actual)
	require.NoError(t, err)

	if !bytes.Equal(actual, expected) {
		t.Errorf(`Expected "%v", got "%v"`, string(expected), string(actual))
	}

	_, err = db.Exec(`drop table docs`)
	require.NoError(t, err)
}

func TestTransactionLifeCycle(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table t(a varchar(255) not null)")
		require.NoError(t, err)

		tx, err := db.Begin()
		require.NoError(t, err)

		_, err = tx.Exec("insert into t values('hi')")
		require.NoError(t, err)

		err = tx.Rollback()
		require.NoError(t, err)

		var n int64
		err = db.QueryRow("select count(*) from t").Scan(&n)
		require.NoError(t, err)
		require.EqualValues(t, 0, n)

		tx, err = db.Begin()
		require.NoError(t, err)

		_, err = tx.Exec("insert into t values('hi')")
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		err = db.QueryRow("select count(*) from t").Scan(&n)
		require.NoError(t, err)
		require.EqualValues(t, 1, n)
	})
}

func TestConnBeginTxIsolation(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		var defaultMode int
		err := db.QueryRow("select mon$isolation_mode from mon$transactions where mon$transaction_id = current_transaction").Scan(&defaultMode)
		require.NoError(t, err)

		serverMajorVersion := getServerMajorVersion(t, db)
		readCommittedFbIso := 4
		if serverMajorVersion <= 3 {
			readCommittedFbIso = 2
		}

		supportedTests := []struct {
			sqlIso sql.IsolationLevel
			fbIso  int
		}{
			{sqlIso: sql.LevelDefault, fbIso: defaultMode},
			{sqlIso: sql.LevelReadCommitted, fbIso: readCommittedFbIso},
			{sqlIso: sql.LevelRepeatableRead, fbIso: 1},
			{sqlIso: sql.LevelSnapshot, fbIso: 1},
			{sqlIso: sql.LevelSerializable, fbIso: 0},
		}
		for i, tt := range supportedTests {
			func() {
				tx, err := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: tt.sqlIso})
				if err != nil {
					t.Errorf("%d. BeginTx failed: %v", i, err)
					return
				}
				defer tx.Rollback()

				var fbIso int
				err = tx.QueryRow("select mon$isolation_mode from mon$transactions where mon$transaction_id = current_transaction").Scan(&fbIso)
				if err != nil {
					t.Errorf("%d. QueryRow failed: %v", i, err)
				}

				if fbIso != tt.fbIso {
					t.Errorf("%d. fbIso => %d, want %d", i, fbIso, tt.fbIso)
				}
			}()
		}

		unsupportedTests := []struct {
			sqlIso sql.IsolationLevel
		}{
			{sqlIso: sql.LevelWriteCommitted},
			{sqlIso: sql.LevelLinearizable},
		}
		for i, tt := range unsupportedTests {
			tx, err := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: tt.sqlIso})
			if err == nil {
				t.Errorf("%d. BeginTx should have failed", i)
				tx.Rollback()
			}
		}
	})
}

func TestConnBeginTxReadOnly(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		tx, err := db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
		require.NoError(t, err)
		defer tx.Rollback()

		var ro int
		err = tx.QueryRow("select mon$read_only from mon$transactions where mon$transaction_id = current_transaction").Scan(&ro)
		if err != nil {
			t.Errorf("QueryRow failed: %v", err)
		}

		if ro != 1 {
			t.Errorf("ro => %d, want %d", ro, 1)
		}
	})
}

func TestBeginTxContextCancel(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		ctx, cancelFn := context.WithCancel(context.Background())

		_, err := db.Exec("drop table t")
		if err != nil {
			var fbErr *fbconn.FbError
			if err != nil && !errors.As(err, &fbErr) || fbErr.Code != "42S02" {
				t.Fatalf(`err => %v, want FbError{Code: "42S02"}`, err)
			}
		}

		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		_, err = tx.Exec("create table t(id int)")
		require.NoError(t, err)

		cancelFn()

		err = tx.Commit()
		if err != context.Canceled && err != sql.ErrTxDone {
			t.Fatalf("err => %v, want %v or %v", err, context.Canceled, sql.ErrTxDone)
		}

		var n int
		err = db.QueryRow("select count(*) from t").Scan(&n)
		if fbEr, ok := err.(*fbconn.FbError); !ok || fbEr.Code != "42S02" {
			t.Fatalf(`err => %v, want FbError{Code: "42S02"}`, err)
		}
	})
}

func TestConnRaw(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		conn, err := db.Conn(context.Background())
		require.NoError(t, err)

		var n int
		err = conn.Raw(func(driverConn any) error {
			conn := driverConn.(*stdlib.Conn).Conn()
			return conn.QueryRow(context.Background(), "select 42 from rdb$database").Scan(&n)
		})
		require.NoError(t, err)
		assert.EqualValues(t, 42, n)
	})
}

func TestConnPingContextSuccess(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		err := db.PingContext(context.Background())
		require.NoError(t, err)
	})
}

func TestConnPrepareContextSuccess(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		stmt, err := db.PrepareContext(context.Background(), "select current_timestamp from rdb$database")
		require.NoError(t, err)
		err = stmt.Close()
		require.NoError(t, err)
	})
}

func TestConnMultiplePrepareAndDeallocate(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		sql := "select 42 from rdb$database"
		stmt1, err := db.PrepareContext(context.Background(), sql)
		require.NoError(t, err)
		stmt2, err := db.PrepareContext(context.Background(), sql)
		require.NoError(t, err)
		err = stmt1.Close()
		require.NoError(t, err)

		var preparedStmtCount int64
		err = db.QueryRowContext(context.Background(), "select count(*) from mon$statements where mon$attachment_id = current_connection and cast(mon$sql_text as varchar(255)) = ?", sql).Scan(&preparedStmtCount)
		require.NoError(t, err)
		require.EqualValues(t, 1, preparedStmtCount)

		err = stmt2.Close() // err isn't as useful as it should be as database/sql will ignore errors from Deallocate.
		require.NoError(t, err)

		err = db.QueryRowContext(context.Background(), "select count(*) from mon$statements where mon$attachment_id = current_connection and cast(mon$sql_text as varchar(255)) = ?", sql).Scan(&preparedStmtCount)
		require.NoError(t, err)
		require.EqualValues(t, 0, preparedStmtCount)
	})
}

func TestStmtCloseAfterDeallocateFailureIsIdempotent(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		sqlConn, err := db.Conn(context.Background())
		require.NoError(t, err)
		defer sqlConn.Close()

		err = sqlConn.Raw(func(driverConn any) error {
			conn := driverConn.(*stdlib.Conn)

			stmt, err := conn.PrepareContext(context.Background(), "select 42 from rdb$database")
			if err != nil {
				return err
			}

			if err := conn.Conn().Close(context.Background()); err != nil {
				return err
			}

			require.Error(t, stmt.Close())
			require.NoError(t, stmt.Close())
			return nil
		})
		require.NoError(t, err)
	})
}

func TestConnExecContextSuccess(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.ExecContext(context.Background(), "recreate table exec_context_test(id int primary key)")
		require.NoError(t, err)
	})
}

func TestConnQueryContextSuccess(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		rows, err := db.QueryContext(context.Background(), generateSeriesQuery("1", "10", "n"))
		require.NoError(t, err)

		for rows.Next() {
			var n int64
			err := rows.Scan(&n)
			require.NoError(t, err)
		}
		require.NoError(t, rows.Err())
	})
}

func TestRowsColumnTypeDatabaseTypeName(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		rows, err := db.Query("select cast(42 as bigint) from rdb$database")
		require.NoError(t, err)

		columnTypes, err := rows.ColumnTypes()
		require.NoError(t, err)
		require.Len(t, columnTypes, 1)

		if columnTypes[0].DatabaseTypeName() != "BIGINT" {
			t.Errorf("columnTypes[0].DatabaseTypeName() => %v, want %v", columnTypes[0].DatabaseTypeName(), "INT8")
		}

		err = rows.Close()
		require.NoError(t, err)
	})
}

func TestStmtExecContextSuccess(t *testing.T) {
	db := openDB(t)
	defer closeDB(t, db)

	_, err := db.Exec("recreate table t(id int primary key)")
	require.NoError(t, err)

	stmt, err := db.Prepare("insert into t(id) values(cast(? as int))")
	require.NoError(t, err)
	defer stmt.Close()

	_, err = stmt.ExecContext(context.Background(), 42)
	require.NoError(t, err)

	ensureDBValid(t, db)
}

func TestStmtExecContextCancel(t *testing.T) {
	db := openDB(t)
	defer closeDB(t, db)

	_, err := db.Exec("recreate procedure fbsleep (delay float not null) returns (o float) external name 'fbxci!fbsleep' engine udr")
	require.NoError(t, err)

	_, err = db.Exec("recreate table t(id int primary key)")
	require.NoError(t, err)

	stmt, err := db.Prepare("insert into t(id) select cast(? as smallint) from fbsleep(5)")
	require.NoError(t, err)
	defer stmt.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = stmt.ExecContext(ctx, 42)
	if !fbconn.Timeout(err) {
		t.Errorf("expected timeout error, got %v", err)
	}

	ensureDBValid(t, db)
}

func TestStmtQueryContextSuccess(t *testing.T) {
	db := openDB(t)
	defer closeDB(t, db)

	stmt, err := db.Prepare(generateSeriesQuery("1", "?", "n"))
	require.NoError(t, err)
	defer stmt.Close()

	rows, err := stmt.QueryContext(context.Background(), 5)
	require.NoError(t, err)

	for rows.Next() {
		var n int64
		if err := rows.Scan(&n); err != nil {
			t.Error(err)
		}
	}

	if rows.Err() != nil {
		t.Error(rows.Err())
	}

	ensureDBValid(t, db)
}

func TestStmtLegacyExecAndQuery(t *testing.T) {
	db := openDB(t)
	defer closeDB(t, db)

	const table = "stdlib_legacy_stmt_test"
	_, err := db.Exec("recreate table " + table + " (id integer)")
	require.NoError(t, err)
	defer func() {
		_, _ = db.Exec("drop table " + table)
	}()

	sqlConn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer sqlConn.Close()

	err = sqlConn.Raw(func(driverConn any) error {
		conn := driverConn.(*stdlib.Conn)

		insert, err := conn.PrepareContext(context.Background(), "insert into "+table+" (id) values (?)")
		if err != nil {
			return err
		}
		defer insert.Close()

		result, err := insert.Exec([]driver.Value{int64(73)})
		if err != nil {
			return err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected != 1 {
			return fmt.Errorf("RowsAffected = %d, want 1", rowsAffected)
		}

		query, err := conn.PrepareContext(context.Background(), "select id from "+table+" where id = ?")
		if err != nil {
			return err
		}
		defer query.Close()

		rows, err := query.Query([]driver.Value{int64(73)})
		if err != nil {
			return err
		}
		defer rows.Close()

		values := make([]driver.Value, len(rows.Columns()))
		if err := rows.Next(values); err != nil {
			return err
		}
		if len(values) != 1 || values[0] != int64(73) {
			return fmt.Errorf("query values = %#v, want []driver.Value{int64(73)}", values)
		}
		return nil
	})
	require.NoError(t, err)
}

func TestConnValidatorDiscardsClosedConnection(t *testing.T) {
	db := openDB(t)
	defer closeDB(t, db)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	sqlConn, err := db.Conn(context.Background())
	require.NoError(t, err)
	err = sqlConn.Raw(func(driverConn any) error {
		conn := driverConn.(*stdlib.Conn)
		if !conn.IsValid() {
			return errors.New("new connection is invalid")
		}
		if err := conn.Conn().Close(context.Background()); err != nil {
			return err
		}
		if conn.IsValid() {
			return errors.New("closed connection is still valid")
		}
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, sqlConn.Close())
	require.Equal(t, 0, db.Stats().OpenConnections)

	require.NoError(t, db.Ping())
	require.Equal(t, 1, db.Stats().OpenConnections)
}

func TestRowsColumnTypes(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		columnTypesTests := []struct {
			Name     string
			TypeName string
			Length   struct {
				Len int64
				OK  bool
			}
			DecimalSize struct {
				Precision int64
				Scale     int64
				OK        bool
			}
			ScanType reflect.Type
		}{
			{
				Name:     "A",
				TypeName: "BIGINT",
				Length: struct {
					Len int64
					OK  bool
				}{
					Len: 0,
					OK:  false,
				},
				DecimalSize: struct {
					Precision int64
					Scale     int64
					OK        bool
				}{
					Precision: 0,
					Scale:     0,
					OK:        false,
				},
				ScanType: reflect.TypeOf(int64(0)),
			}, {
				Name:     "BAR",
				TypeName: "VARCHAR",
				Length: struct {
					Len int64
					OK  bool
				}{
					Len: 10,
					OK:  true,
				},
				DecimalSize: struct {
					Precision int64
					Scale     int64
					OK        bool
				}{
					Precision: 0,
					Scale:     0,
					OK:        false,
				},
				ScanType: reflect.TypeOf(""),
			}, {
				Name:     "DE",
				TypeName: "NUMERIC",
				Length: struct {
					Len int64
					OK  bool
				}{
					Len: 0,
					OK:  false,
				},
				DecimalSize: struct {
					Precision int64
					Scale     int64
					OK        bool
				}{
					Precision: 9,
					Scale:     2,
					OK:        true,
				},
				ScanType: reflect.TypeOf(""),
			}, {
				Name:     "D",
				TypeName: "TIMETZ",
				Length: struct {
					Len int64
					OK  bool
				}{
					Len: 0,
					OK:  false,
				},
				DecimalSize: struct {
					Precision int64
					Scale     int64
					OK        bool
				}{
					Precision: 0,
					Scale:     0,
					OK:        false,
				},
				ScanType: reflect.TypeOf(time.Time{}),
			},
		}

		query := "SELECT cast(1 as bigint) AS a, cast('bar' as varchar(10)) AS bar, cast(1.28 as numeric(9, 2)) AS de"
		if getServerMajorVersion(t, db) > 3 {
			query += ", cast('12:00:00' as time with time zone) as d"
		} else {
			columnTypesTests = columnTypesTests[:3]
		}
		query += " from rdb$database"

		rows, err := db.Query(query)
		require.NoError(t, err)

		columns, err := rows.ColumnTypes()
		require.NoError(t, err)
		assert.Len(t, columns, len(columnTypesTests))

		for i, tt := range columnTypesTests {
			t.Logf("test column %d", i)
			c := columns[i]
			if c.Name() != tt.Name {
				t.Errorf("(%d) got: %s, want: %s", i, c.Name(), tt.Name)
			}
			if c.DatabaseTypeName() != tt.TypeName {
				t.Errorf("(%d) got: %s, want: %s", i, c.DatabaseTypeName(), tt.TypeName)
			}
			l, ok := c.Length()
			if l != tt.Length.Len {
				t.Errorf("(%d) got: %d, want: %d", i, l, tt.Length.Len)
			}
			if ok != tt.Length.OK {
				t.Errorf("(%d) got: %t, want: %t", i, ok, tt.Length.OK)
			}
			p, s, ok := c.DecimalSize()
			if p != tt.DecimalSize.Precision {
				t.Errorf("(%d) got: %d, want: %d", i, p, tt.DecimalSize.Precision)
			}
			if s != tt.DecimalSize.Scale {
				t.Errorf("(%d) got: %d, want: %d", i, s, tt.DecimalSize.Scale)
			}
			if ok != tt.DecimalSize.OK {
				t.Errorf("(%d) got: %t, want: %t", i, ok, tt.DecimalSize.OK)
			}
			if c.ScanType() != tt.ScanType {
				t.Errorf("(%d) got: %v, want: %v", i, c.ScanType(), tt.ScanType)
			}
		}
	})
}

func TestRowsColumnTypeNullable(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		_, err := db.Exec("recreate table stdlib_column_nullable(id integer not null primary key, required_value integer not null, optional_value integer)")
		require.NoError(t, err)

		rows, err := db.Query("select id, required_value, optional_value from stdlib_column_nullable")
		require.NoError(t, err)
		defer rows.Close()

		columns, err := rows.ColumnTypes()
		require.NoError(t, err)
		require.Len(t, columns, 3)

		tests := []struct {
			index int
			want  bool
		}{
			{index: 0, want: false},
			{index: 1, want: false},
			{index: 2, want: true},
		}
		for _, tt := range tests {
			got, ok := columns[tt.index].Nullable()
			require.True(t, ok, "column %d", tt.index)
			require.Equal(t, tt.want, got, "column %d", tt.index)
		}
	})
}

func TestRowsDatabaseSQLValueTypes(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		var numericValue, doubleValue, octetsValue, timeValue any
		err := db.QueryRow(`
			select
				cast(9007199254740993 as numeric(18, 0)),
				cast(123.5 as double precision),
				cast(x'deadbeef' as varchar(4) character set octets),
				cast('14:30:45.1234' as time)
			from rdb$database
		`).Scan(&numericValue, &doubleValue, &octetsValue, &timeValue)
		require.NoError(t, err)

		require.IsType(t, "", numericValue)
		require.Equal(t, "9007199254740993", numericValue)
		require.IsType(t, float64(0), doubleValue)
		require.Equal(t, float64(123.5), doubleValue)
		require.IsType(t, []byte(nil), octetsValue)
		require.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, octetsValue)
		require.IsType(t, time.Time{}, timeValue)
		require.Equal(t, time.Date(2000, 1, 1, 14, 30, 45, 123400000, time.UTC), timeValue)
	})
}

func TestQueryLifeCycle(t *testing.T) {
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		rows, err := db.Query("SELECT 'foo', n FROM ("+generateSeriesQuery("?", "?", "n")+") WHERE 3 = ?", 1, 10, 3)
		require.NoError(t, err)

		rowCount := int64(0)

		for rows.Next() {
			rowCount++
			var (
				s string
				n int64
			)

			err := rows.Scan(&s, &n)
			require.NoError(t, err)

			if s != "foo" {
				t.Errorf(`Expected "foo", received "%v"`, s)
			}

			if n != rowCount {
				t.Errorf("Expected %d, received %d", rowCount, n)
			}
		}
		require.NoError(t, rows.Err())

		err = rows.Close()
		require.NoError(t, err)

		rows, err = db.Query("select 1 from rdb$database where false")
		require.NoError(t, err)

		rowCount = int64(0)

		for rows.Next() {
			rowCount++
		}
		require.NoError(t, rows.Err())
		require.EqualValues(t, 0, rowCount)

		err = rows.Close()
		require.NoError(t, err)
	})
}

func TestScanJSONIntoJSONRawMessage(t *testing.T) {
	t.Skip("need implementation")
	testWithAllQueryExecModes(t, func(t *testing.T, db *sql.DB) {
		var msg json.RawMessage

		err := db.QueryRow("select '{}'::json").Scan(&msg)
		require.NoError(t, err)
		require.EqualValues(t, []byte("{}"), []byte(msg))
	})
}

type testLog struct {
	lvl  tracelog.LogLevel
	msg  string
	data map[string]any
}

type testLogger struct {
	logs []testLog
}

func (l *testLogger) Log(ctx context.Context, lvl tracelog.LogLevel, msg string, data map[string]any) {
	l.logs = append(l.logs, testLog{lvl: lvl, msg: msg, data: data})
}

func TestRegisterConnConfig(t *testing.T) {
	connConfig, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	logger := &testLogger{}
	connConfig.Tracer = &tracelog.TraceLog{Logger: logger, LogLevel: tracelog.LogLevelInfo}

	// Issue 947: Register and unregister a ConnConfig and ensure that the
	// returned connection string is not reused.
	connStr := stdlib.RegisterConnConfig(connConfig)
	require.Equal(t, "registeredConnConfig0", connStr)
	stdlib.UnregisterConnConfig(connStr)

	connStr = stdlib.RegisterConnConfig(connConfig)
	defer stdlib.UnregisterConnConfig(connStr)
	require.Equal(t, "registeredConnConfig1", connStr)

	db, err := sql.Open("fbx", connStr)
	require.NoError(t, err)
	defer closeDB(t, db)

	var n int64
	err = db.QueryRow("select 1 from rdb$database").Scan(&n)
	require.NoError(t, err)

	l := logger.logs[len(logger.logs)-1]
	assert.Equal(t, "Query", l.msg)
	assert.Equal(t, "select 1 from rdb$database", l.data["sql"])
}

func TestOptionBeforeAfterConnect(t *testing.T) {
	config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	var beforeConnConfigs []*fbx.ConnConfig
	var afterConns []*fbx.Conn
	db := stdlib.OpenDB(*config,
		stdlib.OptionBeforeConnect(func(ctx context.Context, connConfig *fbx.ConnConfig) error {
			beforeConnConfigs = append(beforeConnConfigs, connConfig)
			return nil
		}),
		stdlib.OptionAfterConnect(func(ctx context.Context, conn *fbx.Conn) error {
			afterConns = append(afterConns, conn)
			return nil
		}))
	defer closeDB(t, db)

	// Force it to close and reopen a new connection after each query
	db.SetMaxIdleConns(0)

	_, err = db.Exec("select 1 from rdb$database")
	require.NoError(t, err)

	_, err = db.Exec("select 1 from rdb$database")
	require.NoError(t, err)

	require.Len(t, beforeConnConfigs, 2)
	require.Len(t, afterConns, 2)

	// Note: BeforeConnect creates a shallow copy, so the config contents will be the same but we wean to ensure they
	// are different objects, so can't use require.NotEqual
	require.False(t, config == beforeConnConfigs[0])
	require.False(t, beforeConnConfigs[0] == beforeConnConfigs[1])
}

func TestRandomizeHostOrderFunc(t *testing.T) {
	config, err := fbx.ParseConfig("firebird://host1,host2,host3")
	require.NoError(t, err)

	// Test that at some point we connect to all 3 hosts
	hostsNotSeenYet := map[string]struct{}{
		"host1": {},
		"host2": {},
		"host3": {},
	}

	// If we don't succeed within this many iterations, something is certainly wrong
	for i := 0; i < 100000; i++ {
		connCopy := *config
		stdlib.RandomizeHostOrderFunc(context.Background(), &connCopy)

		delete(hostsNotSeenYet, connCopy.Host)
		if len(hostsNotSeenYet) == 0 {
			return
		}

	hostCheckLoop:
		for _, h := range []string{"host1", "host2", "host3"} {
			if connCopy.Host == h {
				continue
			}
			for _, f := range connCopy.Fallbacks {
				if f.Host == h {
					continue hostCheckLoop
				}
			}
			require.Failf(t, "got configuration from RandomizeHostOrderFunc that did not have all the hosts", "%+v", connCopy)
		}
	}

	require.Fail(t, "did not get all hosts as primaries after many randomizations")
}

func TestResetSessionHookCalled(t *testing.T) {
	var mockCalled bool

	connConfig, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	db := stdlib.OpenDB(*connConfig, stdlib.OptionResetSession(func(ctx context.Context, conn *fbx.Conn) error {
		mockCalled = true

		return nil
	}))

	defer closeDB(t, db)

	err = db.Ping()
	require.NoError(t, err)

	err = db.Ping()
	require.NoError(t, err)

	require.True(t, mockCalled)
}

func TestCheckIdleConn(t *testing.T) {
	controllerConn, err := sql.Open("fbx", os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeDB(t, controllerConn)

	db, err := sql.Open("fbx", os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeDB(t, db)

	var conns []*sql.Conn
	for i := 0; i < 3; i++ {
		c, err := db.Conn(context.Background())
		require.NoError(t, err)
		conns = append(conns, c)
	}

	require.EqualValues(t, 3, db.Stats().OpenConnections)

	var ids []uint64
	for _, c := range conns {
		err := c.Raw(func(driverConn any) error {
			res := driverConn.(*stdlib.Conn).Conn().FbConn().Exec(context.Background(), "select mon$attachment_id from mon$attachments where mon$attachment_id = current_connection").Read()
			require.NoError(t, res.Err)
			ids = append(ids, binary.BigEndian.Uint64(res.Rows[0][0]))
			return nil
		})
		require.NoError(t, err)
		err = c.Close()
		require.NoError(t, err)
	}

	// The database/sql connection pool seems to automatically close idle connections to only keep 2 alive.
	// require.EqualValues(t, 3, db.Stats().OpenConnections)

	for _, id := range ids {
		_, err = controllerConn.ExecContext(context.Background(), `delete from mon$attachments where mon$attachment_id = ?`, id)
		require.NoError(t, err)
	}

	// All conns are dead they don't know it and neither does the pool. But because of database/sql automatically closing
	// idle connections we can't be sure how many we should have. require.EqualValues(t, 3, db.Stats().OpenConnections)

	// Wait long enough so the pool will realize it needs to check the connections.
	time.Sleep(time.Second)

	// Pool should try all existing connections and find them dead, then create a new connection which should successfully ping.
	err = db.PingContext(context.Background())
	require.NoError(t, err)

	// The original 3 conns should have been terminated and the a new conn established for the ping.
	require.EqualValues(t, 1, db.Stats().OpenConnections)
	c, err := db.Conn(context.Background())
	require.NoError(t, err)

	var attachmentID uint64
	err = c.Raw(func(driverConn any) error {
		attachmentID = driverConn.(*stdlib.Conn).Conn().FbConn().AttachmentID()
		return nil
	})
	require.NoError(t, err)
	err = c.Close()
	require.NoError(t, err)

	require.NotContains(t, ids, attachmentID)
}

func TestOptionShouldPing_HookCalledOnReuse(t *testing.T) {
	hookCalled := false

	db := openDB(t,
		stdlib.OptionShouldPing(func(context.Context, stdlib.ShouldPingParams) bool {
			hookCalled = true
			// Return false to avoid relying on actual ping behavior.
			return false
		}),
	)
	defer closeDB(t, db)

	// Ensure reuse (so ResetSession runs)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Establish the connection
	require.NoError(t, db.Ping())

	// Reuse the connection -> should trigger ResetSession -> ShouldPing
	_, err := db.Exec("select 1 from rdb$database")
	require.NoError(t, err)

	require.True(t, hookCalled, "hook should be called on reuse")
}

func TestOptionShouldPing_FirstReuseIdleDuration(t *testing.T) {
	var idleDuration time.Duration
	var hookCalled bool

	db := openDB(t,
		stdlib.OptionShouldPing(func(_ context.Context, params stdlib.ShouldPingParams) bool {
			hookCalled = true
			idleDuration = params.IdleDuration
			return false
		}),
	)
	defer closeDB(t, db)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	require.NoError(t, db.Ping())
	_, err := db.Exec("select 1 from rdb$database")
	require.NoError(t, err)

	require.True(t, hookCalled)
	require.Less(t, idleDuration, time.Minute)
}

func TestOptionAfterConnect_ErrorClosesConnection(t *testing.T) {
	callbackErr := errors.New("after-connect error")
	var connected *fbx.Conn

	db := openDB(t,
		stdlib.OptionAfterConnect(func(_ context.Context, conn *fbx.Conn) error {
			connected = conn
			return callbackErr
		}),
	)
	defer closeDB(t, db)

	err := db.Ping()
	require.ErrorIs(t, err, callbackErr)
	require.NotNil(t, connected)
	defer connected.Close(context.Background())
	require.True(t, connected.IsClosed())
}
