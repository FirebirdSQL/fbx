package stdlib

import (
	"context"
	"database/sql/driver"
	"encoding/binary"
	"errors"
	"net"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/fbproto"
	"github.com/firebirdsql/fbx/fbtype"
)

func TestConnectorBeforeConnectGetsIndependentConfig(t *testing.T) {
	t.Parallel()

	config, err := fbx.ParseConfig("firebird://user:password@primary/database")
	if err != nil {
		t.Fatal(err)
	}
	config.RuntimeParams["application_name"] = "original"
	config.Fallbacks = []*fbconn.FallbackConfig{{Host: "fallback", Port: 3050}}

	callbackErr := errors.New("stop before connect")
	connector := GetConnector(*config, OptionBeforeConnect(func(_ context.Context, candidate *fbx.ConnConfig) error {
		if got := candidate.RuntimeParams["application_name"]; got != "original" {
			return errors.New("connector did not preserve its configuration snapshot")
		}
		if got := candidate.Fallbacks[0].Host; got != "fallback" {
			return errors.New("connector did not preserve its fallback snapshot")
		}
		candidate.RuntimeParams["application_name"] = "changed"
		candidate.Fallbacks[0].Host = "changed"
		return callbackErr
	}))
	config.RuntimeParams["application_name"] = "outside change"
	config.Fallbacks[0].Host = "outside change"

	_, err = connector.Connect(context.Background())
	if !errors.Is(err, callbackErr) {
		t.Fatalf("Connect error = %v, want %v", err, callbackErr)
	}
	if got := config.RuntimeParams["application_name"]; got != "outside change" {
		t.Fatalf("RuntimeParams mutation leaked from BeforeConnect: got %q", got)
	}
	if got := config.Fallbacks[0].Host; got != "outside change" {
		t.Fatalf("Fallbacks mutation leaked from BeforeConnect: got %q", got)
	}
}

func TestConnectorRejectsAutocommitDisabled(t *testing.T) {
	t.Parallel()

	config, err := fbx.ParseConfig("firebird://user:password@primary/database")
	if err != nil {
		t.Fatal(err)
	}
	config.Autocommit = false

	_, err = GetConnector(*config).Connect(context.Background())
	if !errors.Is(err, ErrAutocommitDisabled) {
		t.Fatalf("Connect error = %v, want %v", err, ErrAutocommitDisabled)
	}
}

func TestConnectorValidatesConfigAfterBeforeConnect(t *testing.T) {
	t.Parallel()

	config, err := fbx.ParseConfig("firebird://user:password@primary/database")
	if err != nil {
		t.Fatal(err)
	}

	connector := GetConnector(*config, OptionBeforeConnect(func(_ context.Context, candidate *fbx.ConnConfig) error {
		candidate.Autocommit = false
		return nil
	}))
	_, err = connector.Connect(context.Background())
	if !errors.Is(err, ErrAutocommitDisabled) {
		t.Fatalf("Connect error = %v, want %v", err, ErrAutocommitDisabled)
	}
}

func TestDriverOpenConnectorParsesName(t *testing.T) {
	t.Parallel()

	d := &Driver{}
	if _, err := d.OpenConnector("firebird://%"); err == nil {
		t.Fatal("OpenConnector accepted an invalid connection string")
	}
}

func TestDriverOpenConnectorRejectsAutocommitDisabled(t *testing.T) {
	t.Parallel()

	d := &Driver{}
	config, err := fbx.ParseConfig("firebird://user:password@primary/database")
	if err != nil {
		t.Fatal(err)
	}
	config.Autocommit = false
	name := d.registerConnConfig(config)
	defer d.unregisterConnConfig(name)

	_, err = d.OpenConnector(name)
	if !errors.Is(err, ErrAutocommitDisabled) {
		t.Fatalf("OpenConnector error = %v, want %v", err, ErrAutocommitDisabled)
	}
}

func TestDriverRegisteredConfigIsSnapshot(t *testing.T) {
	t.Parallel()

	d := &Driver{}
	config, err := fbx.ParseConfig("firebird://user:password@original/database")
	if err != nil {
		t.Fatal(err)
	}
	config.RuntimeParams["application_name"] = "original"

	name := d.registerConnConfig(config)
	config.Host = "changed"
	config.RuntimeParams["application_name"] = "changed"

	connector, err := d.OpenConnector(name)
	if err != nil {
		t.Fatal(err)
	}
	dc := connector.(*driverConnector)
	if dc.config.Host != "original" {
		t.Fatalf("registered Host = %q, want %q", dc.config.Host, "original")
	}
	if got := dc.config.RuntimeParams["application_name"]; got != "original" {
		t.Fatalf("registered application_name = %q, want %q", got, "original")
	}

	dc.config.RuntimeParams["application_name"] = "connector"
	d.configMu.RLock()
	registeredApplicationName := d.configs[name].RuntimeParams["application_name"]
	d.configMu.RUnlock()
	if registeredApplicationName != "original" {
		t.Fatalf("connector mutation leaked to registered config: got %q", registeredApplicationName)
	}
	d.unregisterConnConfig(name)
}

func TestNilConnectorInputsPanic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func()
	}{
		{name: "pool", call: func() { GetPoolConnector(nil) }},
		{name: "registered config", call: func() { RegisterConnConfig(nil) }},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("call did not panic")
				}
			}()
			tt.call()
		})
	}
}

func TestNumericRowValueFunc(t *testing.T) {
	t.Parallel()

	encodeInt64 := func(v int64) []byte {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(v))
		return buf
	}

	valueFunc := numericRowValueFunc(
		fbtype.NewMap(),
		fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT64, 1, 8, -2),
	)
	tests := []struct {
		src  []byte
		want string
	}{
		{src: encodeInt64(12345), want: "123.45"},
		{src: encodeInt64(-67890), want: "-678.90"},
	}

	for _, tt := range tests {
		got, err := valueFunc(tt.src)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Fatalf("got %#v (%T), want %q", got, got, tt.want)
		}
	}
}

func TestRawBytesRowValueFuncReturnsCopy(t *testing.T) {
	t.Parallel()

	src := []byte{1, 2, 3, 4}
	value, err := rawBytesRowValueFunc(src)
	if err != nil {
		t.Fatal(err)
	}

	got, ok := value.([]byte)
	if !ok {
		t.Fatalf("got %T, want []byte", value)
	}
	src[0] = 9
	if got[0] != 1 {
		t.Fatalf("returned bytes alias source: got %v", got)
	}

	nullValue, err := rawBytesRowValueFunc(nil)
	if err != nil {
		t.Fatal(err)
	}
	if nullValue != nil {
		t.Fatalf("NULL: got %#v (%T), want nil", nullValue, nullValue)
	}
}

func TestNamedValuesFromValues(t *testing.T) {
	t.Parallel()

	values := []driver.Value{int64(42), "answer", nil}
	got := namedValuesFromValues(values)
	want := []driver.NamedValue{
		{Ordinal: 1, Value: int64(42)},
		{Ordinal: 2, Value: "answer"},
		{Ordinal: 3, Value: nil},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestStmtCloseIsIdempotentWithSharedStatement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nativeConn, err := fbx.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := nativeConn.Close(context.Background()); err != nil {
			t.Errorf("close native connection: %v", err)
		}
	}()

	conn := &Conn{
		conn:        nativeConn,
		psRefCounts: make(map[*fbconn.StatementDescription]int),
	}
	const query = "select 42 from rdb$database"

	firstDriverStmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	secondDriverStmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	first := firstDriverStmt.(*Stmt)
	second := secondDriverStmt.(*Stmt)

	if got := conn.psRefCounts[first.sd]; got != 2 {
		t.Fatalf("initial reference count = %d, want 2", got)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := first.Query(nil); !errors.Is(err, errStmtClosed) {
		t.Fatalf("Query after Close error = %v, want %v", err, errStmtClosed)
	}
	if got := conn.psRefCounts[first.sd]; got != 1 {
		t.Fatalf("reference count after repeated Close = %d, want 1", got)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := conn.psRefCounts[first.sd]; ok {
		t.Fatal("reference count remains after the final Close")
	}
}

func TestStmtCloseAfterCachedStatementEvictionDoesNotLeak(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		t.Fatal(err)
	}
	config.DefaultQueryExecMode = fbx.QueryExecModeCacheStatement
	config.StatementCacheCapacity = 1

	nativeConn, err := fbx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := nativeConn.Close(context.Background()); err != nil {
			t.Errorf("close native connection: %v", err)
		}
	}()

	conn := &Conn{
		conn:        nativeConn,
		psRefCounts: make(map[*fbconn.StatementDescription]int),
	}
	const statementSQL = "select cast('rs_328477_stdlib' as varchar(64)) from rdb$database"
	if _, err := conn.ExecContext(ctx, statementSQL, nil); err != nil {
		t.Fatal(err)
	}

	driverStmt, err := conn.PrepareContext(ctx, statementSQL)
	if err != nil {
		t.Fatal(err)
	}
	stmt := driverStmt.(*Stmt)
	if got := conn.psRefCounts[stmt.sd]; got != 1 {
		t.Fatalf("initial reference count = %d, want 1", got)
	}
	if got := countNativeStatementsForSQL(t, nativeConn, statementSQL); got != 2 {
		t.Fatalf("initial native statement count = %d, want 2", got)
	}

	const evictingSQL = "select cast('rs_328477_stdlib_evict' as varchar(64)) from rdb$database"
	if _, err := conn.ExecContext(ctx, evictingSQL, nil); err != nil {
		t.Fatal(err)
	}
	const cleanupSQL = "select cast('rs_328477_stdlib_cleanup' as varchar(64)) from rdb$database"
	if _, err := conn.ExecContext(ctx, cleanupSQL, nil); err != nil {
		t.Fatal(err)
	}
	if got := conn.psRefCounts[stmt.sd]; got != 1 {
		t.Fatalf("reference count after cache eviction = %d, want 1", got)
	}

	if err := stmt.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := conn.psRefCounts[stmt.sd]; ok {
		t.Fatal("reference count remains after Close")
	}
	if got := countNativeStatementsForSQL(t, nativeConn, statementSQL); got != 0 {
		t.Fatalf("native statement count after Close = %d, want 0", got)
	}
}

func TestStaleStmtCloseDoesNotDeallocateReplacement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nativeConn, err := fbx.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := nativeConn.Close(context.Background()); err != nil {
			t.Errorf("close native connection: %v", err)
		}
	}()

	conn := &Conn{
		conn:        nativeConn,
		psRefCounts: make(map[*fbconn.StatementDescription]int),
	}
	const query = "select cast(328477 as integer) from rdb$database"
	staleDriverStmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	staleStmt := staleDriverStmt.(*Stmt)

	if err := nativeConn.DeallocateAll(ctx); err != nil {
		t.Fatal(err)
	}
	replacementDriverStmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	replacementStmt := replacementDriverStmt.(*Stmt)
	if staleStmt.sd == replacementStmt.sd {
		t.Fatal("replacement reused the stale statement description")
	}

	if err := staleStmt.Close(); err != nil {
		t.Fatal(err)
	}
	if got := countNativeStatementsForSQL(t, nativeConn, query); got != 1 {
		t.Fatalf("native statement count after stale Close = %d, want 1", got)
	}
	rows, err := replacementStmt.Query(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}

	if err := replacementStmt.Close(); err != nil {
		t.Fatal(err)
	}
	if got := countNativeStatementsForSQL(t, nativeConn, query); got != 0 {
		t.Fatalf("native statement count after replacement Close = %d, want 0", got)
	}
}

func countNativeStatementsForSQL(t testing.TB, conn *fbx.Conn, sql string) int64 {
	t.Helper()

	var count int64
	err := conn.QueryRow(
		context.Background(),
		"select count(*) from mon$statements where mon$attachment_id = current_connection and cast(mon$sql_text as varchar(255)) = ?",
		fbx.QueryExecModeExec,
		sql,
	).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	return count
}

type blockNextWriteConn struct {
	net.Conn

	blockNext     atomic.Bool
	writeStarted  chan struct{}
	continueWrite chan struct{}
}

func (c *blockNextWriteConn) Write(p []byte) (int, error) {
	if c.blockNext.CompareAndSwap(true, false) {
		close(c.writeStarted)
		<-c.continueWrite
	}
	return c.Conn.Write(p)
}

func TestStmtCloseClearsReferenceAfterNonFatalDeallocateError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		t.Fatal(err)
	}
	originalDial := config.DialFunc
	var wireConn *blockNextWriteConn
	config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := originalDial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		wireConn = &blockNextWriteConn{
			Conn:          conn,
			writeStarted:  make(chan struct{}),
			continueWrite: make(chan struct{}),
		}
		return wireConn, nil
	}

	nativeConn, err := fbx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := nativeConn.Close(context.Background()); err != nil {
			t.Errorf("close native connection: %v", err)
		}
	}()

	conn := &Conn{
		conn:        nativeConn,
		psRefCounts: make(map[*fbconn.StatementDescription]int),
	}
	const query = "select 42 from rdb$database"
	driverStmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	stmt := driverStmt.(*Stmt)

	wireConn.blockNext.Store(true)
	pingErr := make(chan error, 1)
	go func() {
		pingErr <- nativeConn.Ping(ctx)
	}()

	select {
	case <-wireConn.writeStarted:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	writeReleased := false
	defer func() {
		if !writeReleased {
			close(wireConn.continueWrite)
		}
	}()

	if err := stmt.Close(); err == nil {
		t.Fatal("Close succeeded while the connection was busy")
	}
	if nativeConn.IsClosed() {
		t.Fatal("non-fatal Deallocate error closed the connection")
	}
	if !stmt.closed {
		t.Fatal("Stmt remains open after Deallocate error")
	}
	if _, ok := conn.psRefCounts[stmt.sd]; ok {
		t.Fatal("reference count remains after Deallocate error")
	}
	if err := stmt.Close(); err != nil {
		t.Fatalf("repeated Close returned an error: %v", err)
	}
	if _, err := stmt.Query(nil); !errors.Is(err, errStmtClosed) {
		t.Fatalf("Query after failed Close error = %v, want %v", err, errStmtClosed)
	}

	close(wireConn.continueWrite)
	writeReleased = true
	if err := <-pingErr; err != nil {
		t.Fatal(err)
	}

	retryDriverStmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	retryStmt := retryDriverStmt.(*Stmt)
	if got := conn.psRefCounts[retryStmt.sd]; got != 1 {
		t.Fatalf("retry reference count = %d, want 1", got)
	}
	if err := retryStmt.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestColumnTypeScanType(t *testing.T) {
	t.Parallel()

	anyType := reflect.TypeOf((*any)(nil)).Elem()
	tests := []struct {
		name string
		fd   fbconn.FieldDescription
		want reflect.Type
	}{
		{name: "numeric", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_LONG, Subtype: 1}, want: reflect.TypeOf("")},
		{name: "blob", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_BLOB}, want: reflect.TypeOf(Blob{})},
		{name: "double", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_DOUBLE}, want: reflect.TypeOf(float64(0))},
		{name: "float", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_FLOAT}, want: reflect.TypeOf(float32(0))},
		{name: "bigint", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_INT64}, want: reflect.TypeOf(int64(0))},
		{name: "integer", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_LONG}, want: reflect.TypeOf(int32(0))},
		{name: "smallint", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_SHORT}, want: reflect.TypeOf(int16(0))},
		{name: "boolean", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_BOOLEAN}, want: reflect.TypeOf(false)},
		{name: "date", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_DATE}, want: reflect.TypeOf(time.Time{})},
		{name: "time", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_TIME}, want: reflect.TypeOf(time.Time{})},
		{name: "timestamp", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_TIMESTAMP}, want: reflect.TypeOf(time.Time{})},
		{name: "int128", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_INT128}, want: reflect.TypeOf("")},
		{name: "decfloat16", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_DEC64}, want: reflect.TypeOf("")},
		{name: "text", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_VARYING}, want: reflect.TypeOf("")},
		{name: "octets", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_VARYING, Subtype: fbproto.CS_BINARY}, want: reflect.TypeOf([]byte(nil))},
		{name: "array", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_ARRAY}, want: anyType},
		{name: "null", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_NULL}, want: anyType},
		{name: "legacy d float", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_D_FLOAT}, want: anyType},
		{name: "unknown", fd: fbconn.FieldDescription{Type: 65001}, want: anyType},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := columnTypeScanType(tt.fd); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNumericPrecisionForField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fd        fbconn.FieldDescription
		wantPrec  int64
		wantFound bool
	}{
		{
			name:      "type short",
			fd:        fbconn.FieldDescription{Type: fbproto.SQL_TYPE_SHORT, Length: 16},
			wantPrec:  4,
			wantFound: true,
		},
		{
			name:      "type long",
			fd:        fbconn.FieldDescription{Type: fbproto.SQL_TYPE_LONG, Length: 16},
			wantPrec:  9,
			wantFound: true,
		},
		{
			name:      "type int64",
			fd:        fbconn.FieldDescription{Type: fbproto.SQL_TYPE_INT64, Length: 16},
			wantPrec:  18,
			wantFound: true,
		},
		{
			name:      "type int128",
			fd:        fbconn.FieldDescription{Type: fbproto.SQL_TYPE_INT128, Length: 2},
			wantPrec:  38,
			wantFound: true,
		},
		{
			name:      "unknown length",
			fd:        fbconn.FieldDescription{Type: fbproto.SQL_TYPE_NULL, Length: 3},
			wantPrec:  0,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotPrec, gotFound := numericPrecisionForField(tt.fd)
			if gotFound != tt.wantFound {
				t.Fatalf("found: got %v, want %v", gotFound, tt.wantFound)
			}
			if gotPrec != tt.wantPrec {
				t.Fatalf("precision: got %d, want %d", gotPrec, tt.wantPrec)
			}
		})
	}
}
