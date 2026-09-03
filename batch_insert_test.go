package fbx_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
)

func TestConnBatchInsertWithAllQueryExecModes(t *testing.T) {
	for _, mode := range []fbx.QueryExecMode{
		fbx.QueryExecModeCacheStatement,
		fbx.QueryExecModeExec,
	} {
		t.Run(mode.String(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			cfg := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
			cfg.DefaultQueryExecMode = mode
			conn := mustConnect(t, cfg)
			defer closeConn(t, conn)
			skipIfBatchUnsupported(ctx, t, conn)

			mustExec(t, conn, `recreate table foo(
			a smallint,
			b int,
			c bigint,
			d varchar(64))`)

			inputRows := [][]any{
				{int16(0), int32(1), int64(2), "abc"},
				{nil, nil, nil, nil},
			}

			count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A", "B", "C", "D"}, fbx.BatchInsertRows(inputRows))
			if err != nil {
				t.Errorf("Unexpected error for BatchInsert: %v", err)
			}
			if int(count) != len(inputRows) {
				t.Errorf("Expected BatchInsert to return %d inserted rows, but got %d", len(inputRows), count)
			}

			rows, err := conn.Query(ctx, "select * from foo")
			if err != nil {
				t.Errorf("Unexpected error for Query: %v", err)
			}

			var outputRows [][]any
			for rows.Next() {
				row, err := rows.Values()
				if err != nil {
					t.Errorf("Unexpected error for rows.Values(): %v", err)
				}
				outputRows = append(outputRows, row)
			}

			if rows.Err() != nil {
				t.Errorf("Unexpected error for rows.Err(): %v", rows.Err())
			}

			if !reflect.DeepEqual(inputRows, outputRows) {
				t.Errorf("Input rows and output rows do not equal: %v -> %v", inputRows, outputRows)
			}

			ensureConnValid(t, conn)
		})
	}
}

func TestConnBatchInsertFromSmall(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	mustExec(t, conn, `recreate table foo(
		a smallint,
		b int,
		c bigint,
		d char(3),
		e varchar(3),
		f date
	)`)

	inputRows := [][]any{
		{int16(0), int32(1), int64(2), "abc", "efg", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
		{nil, nil, nil, nil, nil, nil},
	}

	count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A", "B", "C", "D", "E", "F"}, fbx.BatchInsertRows(inputRows))
	if err != nil {
		t.Errorf("Unexpected error for BatchInsert: %v", err)
	}
	if int(count) != len(inputRows) {
		t.Errorf("Expected BatchInsert to return %d inserted rows, but got %d", len(inputRows), count)
	}

	rows, err := conn.Query(ctx, "select * from foo")
	if err != nil {
		t.Errorf("Unexpected error for Query: %v", err)
	}

	var outputRows [][]any
	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			t.Errorf("Unexpected error for rows.Values(): %v", err)
		}
		outputRows = append(outputRows, row)
	}

	if rows.Err() != nil {
		t.Errorf("Unexpected error for rows.Err(): %v", rows.Err())
	}

	if !reflect.DeepEqual(inputRows, outputRows) {
		t.Errorf("Input rows and output rows do not equal: %v -> %v", inputRows, outputRows)
	}

	ensureConnValid(t, conn)
}

func TestConnBatchInsertSliceSmall(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	mustExec(t, conn, `recreate table foo(
		a smallint,
		b int,
		c bigint,
		d varchar(3),
		e char(3),
		f timestamp with time zone
	)`)

	tzedTime := time.Date(2010, 2, 3, 4, 5, 6, 0, time.FixedZone("+03:00", +3*60*60))

	inputRows := [][]any{
		{int16(0), int32(1), int64(2), "abc", "efg", tzedTime},
		{nil, nil, nil, nil, nil, nil},
	}

	count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A", "B", "C", "D", "E", "F"},
		fbx.BatchInsertSlice(len(inputRows), func(i int) ([]any, error) {
			return inputRows[i], nil
		}))
	if err != nil {
		t.Errorf("Unexpected error for BatchInsert: %v", err)
	}
	if int(count) != len(inputRows) {
		t.Errorf("Expected BatchInsert to return %d inserted rows, but got %d", len(inputRows), count)
	}

	rows, err := conn.Query(ctx, "select * from foo")
	if err != nil {
		t.Errorf("Unexpected error for Query: %v", err)
	}

	var outputRows [][]any
	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			t.Errorf("Unexpected error for rows.Values(): %v", err)
		}
		outputRows = append(outputRows, row)
	}

	if rows.Err() != nil {
		t.Errorf("Unexpected error for rows.Err(): %v", rows.Err())
	}

	if !reflect.DeepEqual(inputRows, outputRows) {
		t.Errorf("Input rows and output rows do not equal: %v -> %v", inputRows, outputRows)
	}

	ensureConnValid(t, conn)
}

func TestConnBatchInsertLarge(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	mustExec(t, conn, `recreate table foo(
		a smallint,
		b int,
		c bigint,
		d char(3),
		e varchar(3),
		f date,
		g timestamp with time zone
	)`)

	tzedTime := time.Date(2010, 2, 3, 4, 5, 6, 0, time.FixedZone("+03:00", +3*60*60))

	inputRows := [][]any{}

	for range 10_000 {
		inputRows = append(inputRows, []any{int16(0), int32(1), int64(2), "abc", "efg", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), tzedTime})
	}

	count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A", "B", "C", "D", "E", "F", "G"}, fbx.BatchInsertRows(inputRows))
	if err != nil {
		t.Errorf("Unexpected error for BatchInsert: %v", err)
	}
	if int(count) != len(inputRows) {
		t.Errorf("Expected BatchInsert to return %d inserted rows, but got %d", len(inputRows), count)
	}

	rows, err := conn.Query(ctx, "select * from foo")
	if err != nil {
		t.Errorf("Unexpected error for Query: %v", err)
	}

	var outputRows [][]any
	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			t.Errorf("Unexpected error for rows.Values(): %v", err)
		}
		outputRows = append(outputRows, row)
	}

	if rows.Err() != nil {
		t.Errorf("Unexpected error for rows.Err(): %v", rows.Err())
	}

	if !reflect.DeepEqual(inputRows, outputRows) {
		t.Errorf("Input rows and output rows do not equal")
	}

	ensureConnValid(t, conn)
}

type clientFailSource struct {
	count int
	err   error
}

func (cfs *clientFailSource) Next() bool {
	cfs.count++
	return cfs.count < 100
}

func (cfs *clientFailSource) Values() ([]any, error) {
	if cfs.count == 3 {
		cfs.err = fmt.Errorf("client error")
		return nil, cfs.err
	}
	return []any{"1234567890"}, nil
}

func (cfs *clientFailSource) Err() error {
	return cfs.err
}

func TestConnBatchInsertFailServerSideMidway(t *testing.T) {
	t.Skip("need way to abort during execution")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config := mustParseConfig(t, os.Getenv("FBX_TEST_DATABASE"))
	config.BatchInsertPendingSize = 17
	conn := mustConnect(t, config)
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	mustExec(t, conn, `recreate table foo(
		a int,
		b varchar(10) not null
	)`)

	inputRows := [][]any{
		{int32(1), "abc"},
		{int32(2), nil},
		{int32(3), "def"},
	}
	count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A", "B"}, fbx.BatchInsertRows(inputRows))
	if err == nil {
		t.Errorf("Expected BatchInsert return error, but it did not")
	}
	var batchExecutionError *fbconn.FbError
	if !errors.As(err, &batchExecutionError) {
		t.Errorf("Expected BatchInsert return fbconn.FbError, but instead it returned: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected BatchInsert to return 0 inserted rows, but got %d", count)
	}

	rows, err := conn.Query(ctx, "select * from foo")
	if err != nil {
		t.Errorf("Unexpected error for Query: %v", err)
	}

	var outputRows [][]any
	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			t.Errorf("Unexpected error for rows.Values(): %v", err)
		}
		outputRows = append(outputRows, row)
	}

	if rows.Err() != nil {
		t.Errorf("Unexpected error for rows.Err(): %v", rows.Err())
	}

	if len(outputRows) != 0 {
		t.Errorf("Expected 0 rows, but got %v", outputRows)
	}

	ensureConnValid(t, conn)
}

type failSource struct {
	count int
}

func (fs *failSource) Next() bool {
	time.Sleep(time.Millisecond * 100)
	fs.count++
	return fs.count < 100
}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (fs *failSource) Values() ([]any, error) {
	if fs.count == 1280 {
		return []any{nil}, nil
	}
	return []any{randomString(1024)}, nil
}

func (fs *failSource) Err() error {
	return nil
}

func TestConnBatchInsertFailServerSideMidwayAbortsWithoutWaiting(t *testing.T) {
	t.Skip("need way to abort during execution")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	mustExec(t, conn, `recreate table foo(
		a varchar(1024) not null
	)`)

	startTime := time.Now()

	count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A"}, &failSource{})
	if err == nil {
		t.Errorf("Expected BatchInsert return error, but it did not")
	}
	var fbError *fbconn.FbError
	if !errors.As(err, &fbError) {
		t.Errorf("Expected BatchInsert return fbconn.FbError, but instead it returned: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected BatchInsert to return 0 inserted rows, but got %d", count)
	}

	endTime := time.Now()
	batchTime := endTime.Sub(startTime)
	if batchTime > time.Second {
		t.Errorf("Failing BatchInsert shouldn't have taken so long: %v", batchTime)
	}

	rows, err := conn.Query(ctx, "select * from foo")
	if err != nil {
		t.Errorf("Unexpected error for Query: %v", err)
	}

	var outputRows [][]any
	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			t.Errorf("Unexpected error for rows.Values(): %v", err)
		}
		outputRows = append(outputRows, row)
	}

	if rows.Err() != nil {
		t.Errorf("Unexpected error for rows.Err(): %v", rows.Err())
	}

	if len(outputRows) != 0 {
		t.Errorf("Expected 0 rows, but got %v", outputRows)
	}

	ensureConnValid(t, conn)
}

type slowFailRaceSource struct {
	count int
}

func (fs *slowFailRaceSource) Next() bool {
	time.Sleep(time.Millisecond)
	fs.count++
	return fs.count < 1000
}

func (fs *slowFailRaceSource) Values() ([]any, error) {
	if fs.count == 500 {
		return []any{nil, nil}, nil
	}
	return []any{1, "1234567890"}, nil
}

func (fs *slowFailRaceSource) Err() error {
	return nil
}

func TestConnBatchInsertSlowFailRace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	mustExec(t, conn, `recreate table foo(
		a int not null,
		b varchar(10) not null
	)`)

	count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A", "B"}, &slowFailRaceSource{})
	if err == nil {
		t.Errorf("Expected BatchInsert return error, but it did not")
	}
	var fbError *fbconn.FbError
	if !errors.As(err, &fbError) {
		t.Errorf("Expected BatchInsert return fbconn.FbError, but instead it returned: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected BatchInsert to return 0 inserted rows, but got %d", count)
	}

	ensureConnValid(t, conn)
}

func TestConnBatchInsertBatchInsertSourceErrorMidway(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	mustExec(t, conn, `recreate table foo(
		a varchar(10) not null
	)`)

	count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A"}, &clientFailSource{})
	if err == nil {
		t.Errorf("Expected BatchInsert return error, but it did not")
	}
	if count != 0 {
		t.Errorf("Expected BatchInsert to return 0 inserted rows, but got %d", count)
	}

	rows, err := conn.Query(ctx, "select * from foo")
	if err != nil {
		t.Errorf("Unexpected error for Query: %v", err)
	}

	var outputRows [][]any
	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			t.Errorf("Unexpected error for rows.Values(): %v", err)
		}
		outputRows = append(outputRows, row)
	}

	if rows.Err() != nil {
		t.Errorf("Unexpected error for rows.Err(): %v", rows.Err())
	}

	if len(outputRows) != 0 {
		t.Errorf("Expected 0 rows, but got %v", len(outputRows))
	}

	ensureConnValid(t, conn)
}

type clientFinalErrSource struct {
	count int
}

func (cfs *clientFinalErrSource) Next() bool {
	cfs.count++
	return cfs.count < 5
}

func (cfs *clientFinalErrSource) Values() ([]any, error) {
	return []any{"1234567890"}, nil
}

func (cfs *clientFinalErrSource) Err() error {
	return fmt.Errorf("final error")
}

func TestConnBatchInsertBatchInsertSourceErrorEnd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	mustExec(t, conn, `recreate table foo(
		a varchar(10) not null
	)`)

	count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A"}, &clientFinalErrSource{})
	if err == nil {
		t.Errorf("Expected BatchInsert return error, but it did not")
	}
	if count != 0 {
		t.Errorf("Expected BatchInsert to return 0 inserted rows, but got %d", count)
	}

	rows, err := conn.Query(ctx, "select * from foo")
	if err != nil {
		t.Errorf("Unexpected error for Query: %v", err)
	}

	var outputRows [][]any
	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			t.Errorf("Unexpected error for rows.Values(): %v", err)
		}
		outputRows = append(outputRows, row)
	}

	if rows.Err() != nil {
		t.Errorf("Unexpected error for rows.Err(): %v", rows.Err())
	}

	if len(outputRows) != 0 {
		t.Errorf("Expected 0 rows, but got %v", outputRows)
	}

	ensureConnValid(t, conn)
}

func TestConnBatchInsertAutomaticStringConversion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	mustExec(t, conn, `recreate table foo(
		a bigint
	)`)

	inputRows := [][]any{
		{"42"},
		{"7"},
		{8},
	}

	count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A"}, fbx.BatchInsertRows(inputRows))
	require.NoError(t, err)
	require.EqualValues(t, len(inputRows), count)

	rows, _ := conn.Query(ctx, "select * from foo")
	nums, err := fbx.CollectRows(rows, fbx.RowTo[int64])
	require.NoError(t, err)

	require.Equal(t, []int64{42, 7, 8}, nums)

	ensureConnValid(t, conn)
}

func TestConnBatchInsertAutomaticStringConversionArray(t *testing.T) {
	t.Skip("not implemented")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	mustExec(t, conn, `recreate table foo(
		a numeric[2]
	)`)

	inputRows := [][]any{
		{[]string{"42"}},
		{[]string{"7"}},
		{[]string{"8", "9"}},
		{[][]string{{"10", "11"}, {"12", "13"}}},
	}

	count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A"}, fbx.BatchInsertRows(inputRows))
	require.NoError(t, err)
	require.EqualValues(t, len(inputRows), count)

	// Test reads as int64 and flattened array for simplicity.
	rows, _ := conn.Query(ctx, "select * from foo")
	nums, err := fbx.CollectRows(rows, fbx.RowTo[[]int64])
	require.NoError(t, err)
	require.Equal(t, [][]int64{{42}, {7}, {8, 9}, {10, 11, 12, 13}}, nums)

	ensureConnValid(t, conn)
}

func TestBatchInsertFunc(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	mustExec(t, conn, `recreate table foo(
		a int
	)`)

	dataCh := make(chan int, 1)

	const channelItems = 10
	go func() {
		for i := range channelItems {
			dataCh <- i
		}
		close(dataCh)
	}()

	count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A"},
		fbx.BatchInsertFunc(func() ([]any, error) {
			v, ok := <-dataCh
			if !ok {
				return nil, nil
			}
			return []any{v}, nil
		}))

	require.ErrorIs(t, err, nil)
	require.EqualValues(t, channelItems, count)

	rows, err := conn.Query(ctx, "select * from foo order by a")
	require.NoError(t, err)
	nums, err := fbx.CollectRows(rows, fbx.RowTo[int64])
	require.NoError(t, err)
	require.Equal(t, []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, nums)

	// simulate a failure
	count, err = conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A"},
		fbx.BatchInsertFunc(func() func() ([]any, error) {
			x := 9
			return func() ([]any, error) {
				x++
				if x > 100 {
					return nil, fmt.Errorf("simulated error")
				}
				return []any{x}, nil
			}
		}()))
	require.NotErrorIs(t, err, nil)
	require.EqualValues(t, 0, count) // no change, due to error

	ensureConnValid(t, conn)
}

func exampleBatchSupported(ctx context.Context, conn *fbx.Conn) bool {
	return conn.ServerVersion().AtLeast(4, 0, 0)
}

func ExampleBatchInsertRows() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cfg, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		fmt.Println(err)
		return
	}

	conn, err := fbx.ConnectConfig(ctx, cfg)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		_ = conn.Close(ctx)
	}()

	if !exampleBatchSupported(ctx, conn) {
		// to silently skip example on unsupported engine
		fmt.Printf("Batch count: %d\n", 3)
		return
	}

	_, err = conn.Exec(ctx, `recreate table foo(
		a smallint,
		b int,
		c bigint,
		d varchar(3))`)
	if err != nil {
		fmt.Println(err)
		return
	}

	inputRows := [][]any{
		{1, 2, 3, "qwe"},
		{4, 5, 6, "asd"},
		{7, 8, 9, "zxc"},
	}

	count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"}, []string{"A", "B", "C", "D"}, fbx.BatchInsertRows(inputRows))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Batch count: %d\n", count)

	// Output:
	// Batch count: 3
}

func ExampleBatchInsertFunc() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cfg, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		fmt.Println(err)
		return
	}

	conn, err := fbx.ConnectConfig(ctx, cfg)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		_ = conn.Close(ctx)
	}()

	if !exampleBatchSupported(ctx, conn) {
		// to silently skip example on unsupported engine
		fmt.Printf("Batch count: %d\n", 3)
		return
	}

	_, err = conn.Exec(ctx, `recreate table foo(
		a smallint,
		b int,
		c bigint)`)
	if err != nil {
		fmt.Println(err)
		return
	}

	dataCh := make(chan []any, 3)

	const channelItems = 3
	go func() {
		for i := range channelItems {
			dataCh <- []any{i, i + 1, i + 2}
		}
		close(dataCh)
	}()

	count, err := conn.BatchInsert(context.Background(), fbx.Identifier{"FOO"}, []string{"A", "B", "C"},
		fbx.BatchInsertFunc(func() ([]any, error) {
			v, ok := <-dataCh
			if !ok {
				return nil, nil
			}
			return v, nil
		}))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Batch count: %d\n", count)

	// Output:
	// Batch count: 3
}

func ExampleBatchInsertSlice() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cfg, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		fmt.Println(err)
		return
	}

	conn, err := fbx.ConnectConfig(ctx, cfg)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		_ = conn.Close(ctx)
	}()

	if !exampleBatchSupported(ctx, conn) {
		// to silently skip example on unsupported engine
		fmt.Printf("Batch count: %d\n", 3)
		return
	}

	_, err = conn.Exec(ctx, `recreate table foo(
		a smallint,
		b int,
		c bigint)`)
	if err != nil {
		fmt.Println(err)
		return
	}

	dataCh := make(chan []any, 3)

	const channelItems = 3
	go func() {
		for i := range channelItems {
			n := i + 1
			dataCh <- []any{n, n + 1, n + 2}
		}
		close(dataCh)
	}()

	count, err := conn.BatchInsert(context.Background(), fbx.Identifier{"FOO"}, []string{"A", "B", "C"},
		fbx.BatchInsertFunc(func() ([]any, error) {
			v, ok := <-dataCh
			if !ok {
				return nil, nil
			}
			return v, nil
		}))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Batch count: %d\n", count)

	// Output:
	// Batch count: 3
}

func TestConnBatchInsertBoundaryAndVariableLength(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn := mustConnectString(t, os.Getenv("FBX_TEST_DATABASE"))
	defer closeConn(t, conn)
	skipIfBatchUnsupported(ctx, t, conn)

	mustExec(t, conn, `recreate table foo(
		a smallint,
		s varchar(20),
		b int,
		c varchar(5),
		d bigint,
		e smallint
	)`)

	inputRows := [][]any{
		{int16(0x1234), "x", int32(0x7eadbeef), "ab", int64(0x0102030405060708), int16(math.MaxInt16)},
		{int16(math.MinInt16), "abcdefghijklmnopqrst", int32(math.MinInt32), "abcde", int64(math.MinInt64), int16(-12345)},
		{int16(-1), "", int32(-1), "", int64(-1), int16(0x5a5a)},
		{int16(0x00ff), "hello", int32(0x00ff00ff), "z", int64(0x00ff00ff00ff00ff), int16(0x7fff)},
		{nil, nil, nil, nil, nil, nil},
	}

	count, err := conn.BatchInsert(ctx, fbx.Identifier{"FOO"},
		[]string{"A", "S", "B", "C", "D", "E"}, fbx.BatchInsertRows(inputRows))
	if err != nil {
		t.Fatalf("BatchInsert error: %v", err)
	}
	if int(count) != len(inputRows) {
		t.Fatalf("expected %d inserted, got %d", len(inputRows), count)
	}

	rows, err := conn.Query(ctx, "select a, s, b, c, d, e from foo")
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	var outputRows [][]any
	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			t.Fatalf("rows.Values error: %v", err)
		}
		outputRows = append(outputRows, row)
	}
	if rows.Err() != nil {
		t.Fatalf("rows.Err: %v", rows.Err())
	}
	if len(outputRows) != len(inputRows) {
		t.Fatalf("row count mismatch: in=%d out=%d", len(inputRows), len(outputRows))
	}

	expectByB := map[int32][]any{}
	var expectNull []any
	for _, r := range inputRows {
		if r[2] == nil {
			expectNull = r
			continue
		}
		expectByB[r[2].(int32)] = r
	}

	for _, r := range outputRows {
		if r[2] == nil {
			if !reflect.DeepEqual(r, expectNull) {
				t.Errorf("NULL row mismatch:\n got  %#v\n want %#v", r, expectNull)
			}
			continue
		}
		b, ok := r[2].(int32)
		if !ok {
			t.Fatalf("column b is %T not int32", r[2])
		}
		want := expectByB[b]
		if !reflect.DeepEqual(r, want) {
			t.Errorf("row for b=%d mismatch (corruption!):\n got  %#v\n want %#v", b, r, want)
		}

		for _, idx := range []int{0, 5} {
			if _, ok := r[idx].(int16); !ok {
				t.Errorf("column idx %d expected int16, got %T (%v)", idx, r[idx], r[idx])
			}
		}
	}

	ensureConnValid(t, conn)
}
