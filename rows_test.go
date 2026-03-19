package fbx_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbxtest"
)

type testRowScanner struct {
	name string
	age  int32
}

func (rs *testRowScanner) ScanRow(rows fbx.Rows) error {
	return rows.Scan(&rs.name, &rs.age)
}

func TestRowScanner(t *testing.T) {
	t.Parallel()

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		var s testRowScanner
		err := conn.QueryRow(ctx, "select 'Adam' as name, 72 as height from rdb$database").Scan(&s)
		require.NoError(t, err)
		require.Equal(t, "Adam", s.name)
		require.Equal(t, int32(72), s.age)
	})
}

type testErrRowScanner string

func (ers *testErrRowScanner) ScanRow(rows fbx.Rows) error {
	return errors.New(string(*ers))
}

func TestRowScannerErrorIsFatalToRows(t *testing.T) {
	t.Parallel()

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		s := testErrRowScanner("foo")
		err := conn.QueryRow(ctx, "select 'Adam' as name, 72 as height from rdb$database").Scan(&s)
		require.EqualError(t, err, "foo")
	})
}

func TestForEachRow(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		var actualResults []any

		rows, _ := conn.Query(
			context.Background(),
			"select n, n * 2 from ("+generateSeriesQuery("1", "?", "n")+")",
			3,
		)
		var a, b int
		ct, err := fbx.ForEachRow(rows, []any{&a, &b}, func() error {
			actualResults = append(actualResults, []any{a, b})
			return nil
		})
		require.NoError(t, err)

		expectedResults := []any{
			[]any{1, 2},
			[]any{2, 4},
			[]any{3, 6},
		}
		require.Equal(t, expectedResults, actualResults)
		require.EqualValues(t, 3, ct.RowsAffected())
	})
}

func TestRowsAffectedAggregatesSqlCounts(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, err := conn.Exec(ctx, fmt.Sprintf(`recreate table %s (id int)`, "rows_counts_test"))
		require.NoError(t, err)
		defer conn.Exec(ctx, fmt.Sprintf(`drop table %s`, "rows_counts_test"))

		_, err = conn.Exec(ctx, fmt.Sprintf(`insert into %s (id) values (1)`, "rows_counts_test"))
		require.NoError(t, err)

		ct, err := conn.Exec(ctx, fmt.Sprintf(`
execute block as
begin
	insert into %s (id) values (2);
	update %s set id = id + 10 where id = 1;
	delete from %s where id = 2;
end`, "rows_counts_test", "rows_counts_test", "rows_counts_test"))
		require.NoError(t, err)

		// Firebird reports counts for all operations inside the block (including internal reads),
		// so the aggregated RowsAffected should reflect the sum returned by isc_info_sql_records.
		require.EqualValues(t, 5, ct.RowsAffected())
	})
}

func TestForEachRowScanError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		var actualResults []any

		rows, _ := conn.Query(
			context.Background(),
			"select 'foo', 'bar' from ("+generateSeriesQuery("1", "?", "n")+")",
			3,
		)
		var a, b int
		ct, err := fbx.ForEachRow(rows, []any{&a, &b}, func() error {
			actualResults = append(actualResults, []any{a, b})
			return nil
		})
		require.Contains(t, err.Error(), "cannot scan text (type id 452) with subtype 4 in text format into *int")
		require.Equal(t, fbconn.QueryStatus{}, ct)
	})
}

func TestForEachRowAbort(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunWithQueryExecModes(ctx, t, defaultConnTestRunner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(
			context.Background(),
			"select n, n * 2 from ("+generateSeriesQuery("1", "?", "n")+")",
			3,
		)
		var a, b int
		ct, err := fbx.ForEachRow(rows, []any{&a, &b}, func() error {
			return errors.New("abort")
		})
		require.EqualError(t, err, "abort")
		require.Equal(t, fbconn.QueryStatus{}, ct)
	})
}

func ExampleForEachRow() {
	conn, err := fbx.Connect(context.Background(), os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		fmt.Printf("Unable to establish connection: %v", err)
		return
	}
	defer conn.Close(context.Background())

	rows, _ := conn.Query(
		context.Background(),
		"select n, n * 2 from ("+generateSeriesQuery("1", "?", "n")+")",
		3,
	)
	var a, b int
	_, err = fbx.ForEachRow(rows, []any{&a, &b}, func() error {
		fmt.Printf("%v, %v\n", a, b)
		return nil
	})
	if err != nil {
		fmt.Printf("ForEachRow error: %v", err)
		return
	}

	// Output:
	// 1, 2
	// 2, 4
	// 3, 6
}

func TestCollectRows(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, generateSeriesQuery("0", "99", "n"))
		numbers, err := fbx.CollectRows(rows, func(row fbx.CollectableRow) (int32, error) {
			var n int32
			err := row.Scan(&n)
			return n, err
		})
		require.NoError(t, err)

		assert.Len(t, numbers, 100)
		for i := range numbers {
			assert.Equal(t, int32(i), numbers[i])
		}
	})
}

func TestCollectRowsEmpty(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, "select * from rdb$database where false")
		numbers, err := fbx.CollectRows(rows, func(row fbx.CollectableRow) (int32, error) {
			var n int32
			err := row.Scan(&n)
			return n, err
		})
		require.NoError(t, err)
		require.NotNil(t, numbers)

		assert.Empty(t, numbers)
	})
}

// This example uses CollectRows with a manually written collector function. In most cases RowTo, RowToAddrOf,
// RowToStructByPos, RowToAddrOfStructByPos, or another generic function would be used.
func ExampleCollectRows() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := fbx.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		fmt.Printf("Unable to establish connection: %v", err)
		return
	}
	defer conn.Close(ctx)

	rows, _ := conn.Query(ctx, generateSeriesQuery("1", "5", ""))
	numbers, err := fbx.CollectRows(rows, func(row fbx.CollectableRow) (int32, error) {
		var n int32
		err := row.Scan(&n)
		return n, err
	})
	if err != nil {
		fmt.Printf("CollectRows error: %v", err)
		return
	}

	fmt.Println(numbers)

	// Output:
	// [1 2 3 4 5]
}

func TestCollectOneRow(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 42 from rdb$database`)
		n, err := fbx.CollectOneRow(rows, func(row fbx.CollectableRow) (int32, error) {
			var n int32
			err := row.Scan(&n)
			return n, err
		})
		assert.NoError(t, err)
		assert.Equal(t, int32(42), n)
	})
}

func TestCollectOneRowNotFound(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 42 from rdb$database where false`)
		n, err := fbx.CollectOneRow(rows, func(row fbx.CollectableRow) (int32, error) {
			var n int32
			err := row.Scan(&n)
			return n, err
		})
		assert.ErrorIs(t, err, fbx.ErrNoRows)
		assert.Equal(t, int32(0), n)
	})
}

func TestCollectOneRowIgnoresExtraRows(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, generateSeriesQuery("42", "99", ""))
		n, err := fbx.CollectOneRow(rows, func(row fbx.CollectableRow) (int32, error) {
			var n int32
			err := row.Scan(&n)
			return n, err
		})
		require.NoError(t, err)

		assert.NoError(t, err)
		assert.Equal(t, int32(42), n)
	})
}

func TestCollectOneRowPrefersFirebirdSQLErrorOverErrNoRows(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, err := conn.Exec(ctx, `recreate table "+t.Name()+" (name varchar(255) not null)`)
		require.NoError(t, err)

		var name string
		rows, _ := conn.Query(ctx, `insert into "+t.Name()+" (name) values ('foo1') returning name`)
		name, err = fbx.CollectOneRow(rows, func(row fbx.CollectableRow) (string, error) {
			var n string
			err := row.Scan(&n)
			return n, err
		})
		require.NoError(t, err)
		require.Equal(t, "foo1", name)

		rows, _ = conn.Query(ctx, `insert into "+t.Name()+" (name) values ('foo1') returning name`)
		name, err = fbx.CollectOneRow(rows, func(row fbx.CollectableRow) (string, error) {
			var n string
			err := row.Scan(&n)
			return n, err
		})
		require.NoError(t, err)
		require.Equal(t, "foo1", name)

		rows, _ = conn.Query(ctx, `insert into "+t.Name()+" (name) values ('foo1') returning name`)
		name, err = fbx.CollectOneRow(rows, func(row fbx.CollectableRow) (string, error) {
			var n string
			err := row.Scan(&n)
			return n, err
		})
		require.NoError(t, err)
		require.Equal(t, "foo1", name)

		//rows, _ = conn.Query(ctx, `insert into "+t.Name()+" (name) values ('foo2') returning name`)
		//name, err = fbx.CollectOneRow(rows, func(row fbx.CollectableRow) (string, error) {
		//	var n string
		//	err := row.Scan(&n)
		//	return n, err
		//})
		//require.Error(t, err)
		//var fbErr *fbconn.FbError
		//require.ErrorAs(t, err, &fbErr)
		//require.Equal(t, "23505", fbErr.SqlCode)
		//require.Equal(t, "", name)
	})
}

func TestCollectExactlyOneRow(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 42 from rdb$database`)
		n, err := fbx.CollectExactlyOneRow(rows, func(row fbx.CollectableRow) (int32, error) {
			var n int32
			err := row.Scan(&n)
			return n, err
		})
		assert.NoError(t, err)
		assert.Equal(t, int32(42), n)
	})
}

func TestCollectExactlyOneRowNotFound(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 42 from rdb$database where false`)
		n, err := fbx.CollectExactlyOneRow(rows, func(row fbx.CollectableRow) (int32, error) {
			var n int32
			err := row.Scan(&n)
			return n, err
		})
		assert.ErrorIs(t, err, fbx.ErrNoRows)
		assert.Equal(t, int32(0), n)
	})
}

func TestCollectExactlyOneRowExtraRows(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, generateSeriesQuery("42", "99", ""))
		n, err := fbx.CollectExactlyOneRow(rows, func(row fbx.CollectableRow) (int32, error) {
			var n int32
			err := row.Scan(&n)
			return n, err
		})
		assert.ErrorIs(t, err, fbx.ErrTooManyRows)
		assert.Equal(t, int32(0), n)
	})
}

func TestRowTo(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, generateSeriesQuery("0", "99", ""))
		numbers, err := fbx.CollectRows(rows, fbx.RowTo[int32])
		require.NoError(t, err)

		assert.Len(t, numbers, 100)
		for i := range numbers {
			assert.Equal(t, int32(i), numbers[i])
		}
	})
}

func ExampleRowTo() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := fbx.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		fmt.Printf("Unable to establish connection: %v", err)
		return
	}
	defer conn.Close(ctx)

	rows, _ := conn.Query(ctx, generateSeriesQuery("1", "5", ""))
	numbers, err := fbx.CollectRows(rows, fbx.RowTo[int32])
	if err != nil {
		fmt.Printf("CollectRows error: %v", err)
		return
	}

	fmt.Println(numbers)

	// Output:
	// [1 2 3 4 5]
}

func TestRowToAddrOf(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, generateSeriesQuery("0", "99", ""))
		numbers, err := fbx.CollectRows(rows, fbx.RowToAddrOf[int32])
		require.NoError(t, err)

		assert.Len(t, numbers, 100)
		for i := range numbers {
			assert.Equal(t, int32(i), *numbers[i])
		}
	})
}

func ExampleRowToAddrOf() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := fbx.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		fmt.Printf("Unable to establish connection: %v", err)
		return
	}
	defer conn.Close(ctx)

	rows, _ := conn.Query(ctx, generateSeriesQuery("1", "5", ""))
	pNumbers, err := fbx.CollectRows(rows, fbx.RowToAddrOf[int32])
	if err != nil {
		fmt.Printf("CollectRows error: %v", err)
		return
	}

	for _, p := range pNumbers {
		fmt.Println(*p)
	}

	// Output:
	// 1
	// 2
	// 3
	// 4
	// 5
}

func TestRowToMap(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 'Joe' as name, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		slice, err := fbx.CollectRows(rows, fbx.RowToMap)
		require.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "Joe", slice[i]["NAME"])
			assert.EqualValues(t, i, slice[i]["AGE"])
		}
	})
}

func TestRowToStructByPos(t *testing.T) {
	type person struct {
		Name string
		Age  int32
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 'Joe' as name, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		slice, err := fbx.CollectRows(rows, fbx.RowToStructByPos[person])
		require.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "Joe", slice[i].Name)
			assert.EqualValues(t, i, slice[i].Age)
		}
	})
}

func TestRowToStructByPosIgnoredField(t *testing.T) {
	type person struct {
		Name string
		Age  int32 `db:"-"`
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 'Joe' as name from (`+generateSeriesQuery("0", "9", "n")+`)`)
		slice, err := fbx.CollectRows(rows, fbx.RowToStructByPos[person])
		require.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "Joe", slice[i].Name)
		}
	})
}

func TestRowToStructByPosEmbeddedStruct(t *testing.T) {
	type Name struct {
		First string
		Last  string
	}

	type person struct {
		Name
		Age int32
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 'John' as first_name, 'Smith' as last_name, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		slice, err := fbx.CollectRows(rows, fbx.RowToStructByPos[person])
		require.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "John", slice[i].Name.First)
			assert.Equal(t, "Smith", slice[i].Name.Last)
			assert.EqualValues(t, i, slice[i].Age)
		}
	})
}

func TestRowToStructByPosMultipleEmbeddedStruct(t *testing.T) {
	type Sandwich struct {
		Bread string
		Salad string
	}
	type Drink struct {
		Ml int
	}

	type meal struct {
		Sandwich
		Drink
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 'Baguette' as bread, 'Lettuce' as salad, drink_ml from (`+generateSeriesQuery("0", "9", "drink_ml")+`)`)
		slice, err := fbx.CollectRows(rows, fbx.RowToStructByPos[meal])
		require.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "Baguette", slice[i].Sandwich.Bread)
			assert.Equal(t, "Lettuce", slice[i].Sandwich.Salad)
			assert.EqualValues(t, i, slice[i].Drink.Ml)
		}
	})
}

func TestRowToStructByPosEmbeddedUnexportedStruct(t *testing.T) {
	type name struct {
		First string
		Last  string
	}

	type person struct {
		name
		Age int32
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 'John' as first_name, 'Smith' as last_name, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		slice, err := fbx.CollectRows(rows, fbx.RowToStructByPos[person])
		require.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "John", slice[i].name.First)
			assert.Equal(t, "Smith", slice[i].name.Last)
			assert.EqualValues(t, i, slice[i].Age)
		}
	})
}

// Pointer to struct is not supported. But check that we don't panic.
func TestRowToStructByPosEmbeddedPointerToStruct(t *testing.T) {
	type Name struct {
		First string
		Last  string
	}

	type person struct {
		*Name
		Age int32
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 'John' as first_name, 'Smith' as last_name, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		_, err := fbx.CollectRows(rows, fbx.RowToStructByPos[person])
		require.EqualError(t, err, "got 3 values, but dst struct has only 2 fields")
	})
}

func ExampleRowToStructByPos() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := fbx.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		fmt.Printf("Unable to establish connection: %v", err)
		return
	}
	defer conn.Close(ctx)

	// Setup example schema and data.
	_, err = conn.Exec(ctx, `
recreate table products (
	id int generated by default as identity primary key,
	name varchar(100) not null,
	price int not null
)`)

	if err != nil {
		fmt.Printf("Unable to setup example schema and data: %v", err)
		return
	}

	//FIXME: fbx rewrite someday with batch insert?
	conn.Exec(ctx, `insert into products (name, price) values ('Cheeseburger', 10)`)
	conn.Exec(ctx, `insert into products (name, price) values ('Double Cheeseburger', 14)`)
	conn.Exec(ctx, `insert into products (name, price) values ('Fries', 5)`)
	conn.Exec(ctx, `insert into products (name, price) values ('Soft Drink', 3)`)

	type product struct {
		ID    int32
		Name  string
		Price int32
	}

	rows, _ := conn.Query(ctx, "select * from products where price < ? order by price desc", 12)
	products, err := fbx.CollectRows(rows, fbx.RowToStructByPos[product])
	if err != nil {
		fmt.Printf("CollectRows error: %v", err)
		return
	}

	for _, p := range products {
		fmt.Printf("%s: $%d\n", p.Name, p.Price)
	}

	// Output:
	// Cheeseburger: $10
	// Fries: $5
	// Soft Drink: $3
}

func TestRowToAddrOfStructPos(t *testing.T) {
	type person struct {
		Name string
		Age  int32
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 'Joe' as name, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		slice, err := fbx.CollectRows(rows, fbx.RowToAddrOfStructByPos[person])
		require.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "Joe", slice[i].Name)
			assert.EqualValues(t, i, slice[i].Age)
		}
	})
}

func TestRowToStructByName(t *testing.T) {
	type person struct {
		Last      string
		First     string
		Age       int32
		AccountID string
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 'John' as first, 'Smith' as last, n as age, 'd5e49d3f' as account_id from (`+generateSeriesQuery("0", "9", "n")+`)`)
		slice, err := fbx.CollectRows(rows, fbx.RowToStructByName[person])
		assert.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "Smith", slice[i].Last)
			assert.Equal(t, "John", slice[i].First)
			assert.EqualValues(t, i, slice[i].Age)
			assert.Equal(t, "d5e49d3f", slice[i].AccountID)
		}
		assert.NoError(t, rows.Err())

		// check missing fields in a returned row
		rows, _ = conn.Query(ctx, `select 'Smith' as last, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		_, err = fbx.CollectRows(rows, fbx.RowToStructByName[person])
		assert.ErrorContains(t, err, "cannot find field First in returned row")

		// check missing field in a destination struct
		rows, _ = conn.Query(ctx, `select 'John' as first, 'Smith' as last, n as age, 'd5e49d3f' as account_id, null as ignore from(`+generateSeriesQuery("0", "9", "n")+`)`)
		_, err = fbx.CollectRows(rows, fbx.RowToAddrOfStructByName[person])
		assert.ErrorContains(t, err, "struct doesn't have corresponding row field IGNORE")
	})
}

func TestRowToStructByNameDbTags(t *testing.T) {
	type person struct {
		Last             string `db:"LAST_NAME"`
		First            string `db:"FIRST_NAME"`
		Age              int32  `db:"AGE"`
		AccountID        string `db:"ACCOUNT_ID"`
		AnotherAccountID string `db:"ACCOUNT__ID"`
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 'John' as first_name, 'Smith' as last_name, n as age, 'd5e49d3f' as account_id, '5e49d321' as account__id from (`+generateSeriesQuery("0", "9", "n")+`)`)
		slice, err := fbx.CollectRows(rows, fbx.RowToStructByName[person])
		assert.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "Smith", slice[i].Last)
			assert.Equal(t, "John", slice[i].First)
			assert.EqualValues(t, i, slice[i].Age)
			assert.Equal(t, "d5e49d3f", slice[i].AccountID)
			assert.Equal(t, "5e49d321", slice[i].AnotherAccountID)
		}

		// check missing fields in a returned row
		rows, _ = conn.Query(ctx, `select 'Smith' as last_name, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		_, err = fbx.CollectRows(rows, fbx.RowToStructByName[person])
		assert.ErrorContains(t, err, "cannot find field FIRST_NAME in returned row")

		// check missing field in a destination struct
		rows, _ = conn.Query(ctx, `select 'John' as first_name, 'Smith' as last_name, n as age, 'd5e49d3f' as account_id, '5e49d321' as account__id, null as ignore from (`+generateSeriesQuery("0", "9", "n")+`)`)
		_, err = fbx.CollectRows(rows, fbx.RowToAddrOfStructByName[person])
		assert.ErrorContains(t, err, "struct doesn't have corresponding row field IGNORE")
	})
}

func TestRowToStructByNameEmbeddedStruct(t *testing.T) {
	type Name struct {
		Last  string `db:"LAST_NAME"`
		First string `db:"FIRST_NAME"`
	}

	type person struct {
		Ignore bool `db:"-"`
		Name
		Age int32
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 'John' as first_name, 'Smith' as last_name, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		slice, err := fbx.CollectRows(rows, fbx.RowToStructByName[person])
		assert.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "Smith", slice[i].Name.Last)
			assert.Equal(t, "John", slice[i].Name.First)
			assert.EqualValues(t, i, slice[i].Age)
		}

		// check missing fields in a returned row
		rows, _ = conn.Query(ctx, `select 'Smith' as last_name, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		_, err = fbx.CollectRows(rows, fbx.RowToStructByName[person])
		assert.ErrorContains(t, err, "cannot find field FIRST_NAME in returned row")

		// check missing field in a destination struct
		rows, _ = conn.Query(ctx, `select 'John' as first_name, 'Smith' as last_name, n as age, null as ignore from (`+generateSeriesQuery("0", "9", "n")+`)`)
		_, err = fbx.CollectRows(rows, fbx.RowToAddrOfStructByName[person])
		assert.ErrorContains(t, err, "struct doesn't have corresponding row field IGNORE")
	})
}

func ExampleRowToStructByName() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := fbx.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		fmt.Printf("Unable to establish connection: %v", err)
		return
	}
	defer conn.Close(ctx)

	// Setup example schema and data.
	_, err = conn.Exec(ctx, `
recreate table products (
	id int generated by default as identity primary key,
	name varchar(100) not null,
	price int not null
)`)

	if err != nil {
		fmt.Printf("Unable to setup example schema and data: %v", err)
		return
	}

	//FIXME: fbx rewrite someday with batch insert?
	conn.Exec(ctx, `insert into products (name, price) values ('Cheeseburger', 10)`)
	conn.Exec(ctx, `insert into products (name, price) values ('Double Cheeseburger', 14)`)
	conn.Exec(ctx, `insert into products (name, price) values ('Fries', 5)`)
	conn.Exec(ctx, `insert into products (name, price) values ('Soft Drink', 3)`)

	type product struct {
		ID    int32
		Name  string
		Price int32
	}

	rows, _ := conn.Query(ctx, "select * from products where price < ? order by price desc", 12)
	products, err := fbx.CollectRows(rows, fbx.RowToStructByName[product])
	if err != nil {
		fmt.Printf("CollectRows error: %v", err)
		return
	}

	for _, p := range products {
		fmt.Printf("%s: $%d\n", p.Name, p.Price)
	}

	// Output:
	// Cheeseburger: $10
	// Fries: $5
	// Soft Drink: $3
}

func TestRowToStructByNameLax(t *testing.T) {
	type person struct {
		Last   string
		First  string
		Age    int32
		Ignore bool `db:"-"`
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 'John' as first, 'Smith' as last, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		slice, err := fbx.CollectRows(rows, fbx.RowToStructByNameLax[person])
		assert.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "Smith", slice[i].Last)
			assert.Equal(t, "John", slice[i].First)
			assert.EqualValues(t, i, slice[i].Age)
		}

		// check missing fields in a returned row
		rows, _ = conn.Query(ctx, `select 'John' as first, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		slice, err = fbx.CollectRows(rows, fbx.RowToStructByNameLax[person])
		assert.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "John", slice[i].First)
			assert.EqualValues(t, i, slice[i].Age)
		}

		// check extra fields in a returned row
		rows, _ = conn.Query(ctx, `select 'John' as first, 'Smith' as last, n as age, null as ignore from (`+generateSeriesQuery("0", "9", "n")+`)`)
		_, err = fbx.CollectRows(rows, fbx.RowToAddrOfStructByNameLax[person])
		assert.ErrorContains(t, err, "struct doesn't have corresponding row field IGNORE")

		// check missing fields in a destination struct
		rows, _ = conn.Query(ctx, `select 'Smith' as last, 'D.' as middle, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		_, err = fbx.CollectRows(rows, fbx.RowToAddrOfStructByNameLax[person])
		assert.ErrorContains(t, err, "struct doesn't have corresponding row field MIDDLE")

		// check ignored fields in a destination struct
		rows, _ = conn.Query(ctx, `select 'Smith' as last, n as age, null as ignore from (`+generateSeriesQuery("0", "9", "n")+`)`)
		_, err = fbx.CollectRows(rows, fbx.RowToAddrOfStructByNameLax[person])
		assert.ErrorContains(t, err, "struct doesn't have corresponding row field IGNORE")
	})
}

func TestRowToStructByNameLaxEmbeddedStruct(t *testing.T) {
	type Name struct {
		Last  string `db:"LAST_NAME"`
		First string `db:"FIRST_NAME"`
	}

	type person struct {
		Ignore bool `db:"-"`
		Name
		Age int32
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `select 'John' as first_name, 'Smith' as last_name, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		slice, err := fbx.CollectRows(rows, fbx.RowToStructByNameLax[person])
		assert.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "Smith", slice[i].Name.Last)
			assert.Equal(t, "John", slice[i].Name.First)
			assert.EqualValues(t, i, slice[i].Age)
		}

		// check missing fields in a returned row
		rows, _ = conn.Query(ctx, `select 'John' as first_name, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		slice, err = fbx.CollectRows(rows, fbx.RowToStructByNameLax[person])
		assert.NoError(t, err)

		assert.Len(t, slice, 10)
		for i := range slice {
			assert.Equal(t, "John", slice[i].Name.First)
			assert.EqualValues(t, i, slice[i].Age)
		}

		// check extra fields in a returned row
		rows, _ = conn.Query(ctx, `select 'John' as first_name, 'Smith' as last_name, n as age, null as ignore from (`+generateSeriesQuery("0", "9", "n")+`)`)
		_, err = fbx.CollectRows(rows, fbx.RowToAddrOfStructByNameLax[person])
		assert.ErrorContains(t, err, "struct doesn't have corresponding row field IGNORE")

		// check missing fields in a destination struct
		rows, _ = conn.Query(ctx, `select 'Smith' as last_name, 'D.' as middle_name, n as age from (`+generateSeriesQuery("0", "9", "n")+`)`)
		_, err = fbx.CollectRows(rows, fbx.RowToAddrOfStructByNameLax[person])
		assert.ErrorContains(t, err, "struct doesn't have corresponding row field MIDDLE_NAME")

		// check ignored fields in a destination struct
		rows, _ = conn.Query(ctx, `select 'Smith' as last_name, n as age, null as ignore from (`+generateSeriesQuery("0", "9", "n")+`)`)
		_, err = fbx.CollectRows(rows, fbx.RowToAddrOfStructByNameLax[person])
		assert.ErrorContains(t, err, "struct doesn't have corresponding row field IGNORE")
	})
}

func TestRowToStructByNameLaxRowValue(t *testing.T) {
	//FIXME: need row type support
	t.Skip("need implementation in RDB")
	type AnotherTable struct{}
	type User struct {
		UserID int    `json:"userId" db:"USER_ID"`
		Name   string `json:"name" db:"NAME"`
	}
	type UserAPIKey struct {
		UserAPIKeyID int `json:"userApiKeyId" db:"USER_API_KEY_ID"`
		UserID       int `json:"userId" db:"USER_ID"`

		User         *User         `json:"user" db:"USR"`
		AnotherTable *AnotherTable `json:"anotherTable" db:"ANOTHER_TABLE"`
	}

	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		rows, _ := conn.Query(ctx, `
		WITH user_api_keys AS (
			SELECT 1 AS user_id, 101 AS user_api_key_id, 'abc123' AS api_key from rdb$database
		), users AS (
			SELECT 1 AS user_id, 'John Doe' AS name from rdb$database
		)
		SELECT user_api_keys.user_api_key_id, user_api_keys.user_id, row(users.*) as user
		FROM user_api_keys
		LEFT JOIN users ON users.user_id = user_api_keys.user_id
		WHERE user_api_keys.api_key = 'abc123';
		`)
		slice, err := fbx.CollectRows(rows, fbx.RowToStructByNameLax[UserAPIKey])

		assert.NoError(t, err)
		assert.ElementsMatch(t, slice, []UserAPIKey{{UserAPIKeyID: 101, UserID: 1, User: &User{UserID: 1, Name: "John Doe"}, AnotherTable: nil}})
	})
}

func ExampleRowToStructByNameLax() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := fbx.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		fmt.Printf("Unable to establish connection: %v", err)
		return
	}
	defer conn.Close(ctx)

	// Setup example schema and data.
	_, err = conn.Exec(ctx, `
recreate table products (
	id int generated by default as identity primary key,
	name varchar(100) not null,
	price int not null)
`)
	if err != nil {
		fmt.Printf("Unable to setup example schema and data: %v", err)
		return
	}
	_, err = conn.Exec(ctx, `insert into products (name, price) values('Cheeseburger', 10)`)
	_, err = conn.Exec(ctx, `insert into products (name, price) values('Double Cheeseburger', 14)`)
	_, err = conn.Exec(ctx, `insert into products (name, price) values('Fries', 5)`)
	_, err = conn.Exec(ctx, `insert into products (name, price) values('Soft Drink', 3)`)
	if err != nil {
		fmt.Printf("Unable to setup example schema and data: %v", err)
		return
	}

	type product struct {
		ID    int32
		Name  string
		Type  string
		Price int32
	}

	rows, _ := conn.Query(ctx, "select * from products where price < ? order by price desc", 12)
	products, err := fbx.CollectRows(rows, fbx.RowToStructByNameLax[product])
	if err != nil {
		fmt.Printf("CollectRows error: %v", err)
		return
	}

	for _, p := range products {
		fmt.Printf("%s: $%d\n", p.Name, p.Price)
	}

	// Output:
	// Cheeseburger: $10
	// Fries: $5
	// Soft Drink: $3
}
