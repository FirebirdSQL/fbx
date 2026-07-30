// Package stdlib is the compatibility layer from fbx to database/sql.
//
// A database/sql connection can be established through sql.Open.
//
//	db, err := sql.Open("fbx", "firebird://user:secret@localhost:3050/fbx_test")
//	if err != nil {
//	  return err
//	}
//
// Or from a keyword/value string.
//
//	db, err := sql.Open("fbx", "user=user password=secret host=localhost port=3050 database=fbx_test")
//	if err != nil {
//	  return err
//	}
//
// Or from a *fbxpool.Pool.
//
//	pool, err := fbxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
//	if err != nil {
//	  return err
//	}
//
//	db := stdlib.OpenDBFromPool(pool)
//
// Or a fbx.ConnConfig can be used to set configuration not accessible via connection string. In this case the
// fbx.ConnConfig must first be registered with the driver. This registration returns a connection string which is used
// with sql.Open.
//
//	connConfig, _ := fbx.ParseConfig(os.Getenv("DATABASE_URL"))
//	connConfig.Tracer = &tracelog.TraceLog{Logger: myLogger, LogLevel: tracelog.LogLevelInfo}
//	connStr := stdlib.RegisterConnConfig(connConfig)
//	db, _ := sql.Open("fbx", connStr)
//
// fbx uses standard Firebird positional parameters in queries. e.g. ?, ?. It does not support named parameters.
//
//	db.QueryRow("select * from users where id=?", userID)
//
// (*sql.Conn) Raw() can be used to get a *fbx.Conn from the standard database/sql.DB connection pool. This allows
// operations that use fbx specific functionality.
//
//	// Given db is a *sql.DB
//	conn, err := db.Conn(context.Background())
//	if err != nil {
//	  // handle error from acquiring connection from DB pool
//	}
//
//	err = conn.Raw(func(driverConn any) error {
//	  conn := driverConn.(*stdlib.Conn).Conn() // conn is a *fbx.Conn
//	  // Do fbx specific stuff with conn
//	  conn.CopyFrom(...)
//	  return nil
//	})
//	if err != nil {
//	  // handle error that occurred while using *fbx.Conn
//	}
//
// # Firebird Specific Data Types
//
// The fbtype package provides support for Firebird specific types. *fbtype.Map.SQLScanner is an adapter that makes
// these types usable as a sql.Scanner.
//
//	_, err := db.Exec("recreate table array_example(bigint_array bigint[3])")
//	input := []int64{1,2,3}
//	_, err = db.Exec("insert into array_example(bigint_array) values(?)", input)
//
//	m := fbtype.NewMap()
//	var output []int64
//	err = db.QueryRow("select bigint_array from array_example").Scan(m.SQLScanner(&output))
package stdlib

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxpool"
)

var fbxDriver *Driver

func init() {
	fbxDriver = &Driver{
		configs: make(map[string]*fbx.ConnConfig),
	}

	// if fbx driver was already registered by different fbx major version then we
	// skip registration under the default name.
	if !slices.Contains(sql.Drivers(), "fbx") {
		sql.Register("fbx", fbxDriver)
	}
}

// OptionOpenDB options for configuring the driver when opening a new db pool.
type OptionOpenDB func(*connector)

// ShouldPingParams are passed to OptionShouldPing to decide whether to ping before reusing a connection.
type ShouldPingParams struct {
	// Conn is the underlying fbx connection.
	Conn *fbx.Conn
	// IdleDuration is how long it has been since ResetSession last ran.
	IdleDuration time.Duration
}

// OptionShouldPing controls whether stdlib should issue a liveness ping before reusing a connection.
// If the function returns true, stdlib will ping.
// If it returns false, stdlib will skip the ping.
// If not provided, default is ping only when IdleDuration > 1s.
func OptionShouldPing(f func(context.Context, ShouldPingParams) bool) OptionOpenDB {
	return func(dc *connector) { dc.ShouldPing = f }
}

// OptionBeforeConnect provides a callback for before connect. It is passed a shallow copy of the ConnConfig that will
// be used to connect, so only its immediate members should be modified. Used only if db is opened with *fbx.ConnConfig.
func OptionBeforeConnect(bc func(context.Context, *fbx.ConnConfig) error) OptionOpenDB {
	return func(dc *connector) {
		dc.BeforeConnect = bc
	}
}

// OptionAfterConnect provides a callback for after connect. Used only if db is opened with *fbx.ConnConfig.
func OptionAfterConnect(ac func(context.Context, *fbx.Conn) error) OptionOpenDB {
	return func(dc *connector) {
		dc.AfterConnect = ac
	}
}

// OptionResetSession provides a callback that can be used to add custom logic prior to executing a query on the
// connection if the connection has been used before.
// If ResetSessionFunc returns ErrBadConn error the connection will be discarded.
func OptionResetSession(rs func(context.Context, *fbx.Conn) error) OptionOpenDB {
	return func(dc *connector) {
		dc.ResetSession = rs
	}
}

// RandomizeHostOrderFunc is a BeforeConnect hook that randomizes the host order in the provided connConfig, so that a
// new host becomes primary each time. This is useful to distribute connections for multi-master databases like
// CockroachDB. If you use this you likely should set https://golang.org/pkg/database/sql/#DB.SetConnMaxLifetime as well
// to ensure that connections are periodically rebalanced across your nodes.
func RandomizeHostOrderFunc(ctx context.Context, connConfig *fbx.ConnConfig) error {
	if len(connConfig.Fallbacks) == 0 {
		return nil
	}

	newFallbacks := append([]*fbconn.FallbackConfig{{
		Host: connConfig.Host,
		Port: connConfig.Port,
	}}, connConfig.Fallbacks...)

	rand.Shuffle(len(newFallbacks), func(i, j int) {
		newFallbacks[i], newFallbacks[j] = newFallbacks[j], newFallbacks[i]
	})

	// Use the one that sorted last as the primary and keep the rest as the fallbacks
	newPrimary := newFallbacks[len(newFallbacks)-1]
	connConfig.Host = newPrimary.Host
	connConfig.Port = newPrimary.Port
	connConfig.Fallbacks = newFallbacks[:len(newFallbacks)-1]
	return nil
}

func GetConnector(config fbx.ConnConfig, opts ...OptionOpenDB) driver.Connector {
	c := connector{
		ConnConfig:    config,
		BeforeConnect: func(context.Context, *fbx.ConnConfig) error { return nil }, // noop before connect by default
		AfterConnect:  func(context.Context, *fbx.Conn) error { return nil },       // noop after connect by default
		ResetSession:  func(context.Context, *fbx.Conn) error { return nil },       // noop reset session by default
		driver:        fbxDriver,
	}

	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// GetPoolConnector creates a new driver.Connector from the given *fbxpool.Pool. By using this be sure to set the
// maximum idle connections of the *sql.DB created with this connector to zero since they must be managed from the
// *fbxpool.Pool. This is required to avoid acquiring all the connections from the fbxpool and starving any direct
// users of the fbxpool.
func GetPoolConnector(pool *fbxpool.Pool, opts ...OptionOpenDB) driver.Connector {
	c := connector{
		pool:         pool,
		ResetSession: func(context.Context, *fbx.Conn) error { return nil }, // noop reset session by default
		driver:       fbxDriver,
	}

	for _, opt := range opts {
		opt(&c)
	}

	return c
}

func OpenDB(config fbx.ConnConfig, opts ...OptionOpenDB) *sql.DB {
	c := GetConnector(config, opts...)
	return sql.OpenDB(c)
}

// OpenDBFromPool creates a new *sql.DB from the given *fbxpool.Pool. Note that this method automatically sets the
// maximum number of idle connections in *sql.DB to zero, since they must be managed from the *fbxpool.Pool. This is
// required to avoid acquiring all the connections from the fbxpool and starving any direct users of the fbxpool. Note
// that closing the returned *sql.DB will not close the *fbxpool.Pool.
func OpenDBFromPool(pool *fbxpool.Pool, opts ...OptionOpenDB) *sql.DB {
	c := GetPoolConnector(pool, opts...)
	db := sql.OpenDB(c)
	db.SetMaxIdleConns(0)
	return db
}

type connector struct {
	fbx.ConnConfig
	pool          *fbxpool.Pool
	BeforeConnect func(context.Context, *fbx.ConnConfig) error // function to call before creation of every new connection
	AfterConnect  func(context.Context, *fbx.Conn) error       // function to call after creation of every new connection
	ResetSession  func(context.Context, *fbx.Conn) error       // function is called before a connection is reused
	ShouldPing    func(context.Context, ShouldPingParams) bool // function to decide if stdlib should ping before reusing a connection
	driver        *Driver
}

// Connect implement driver.Connector interface
func (c connector) Connect(ctx context.Context) (driver.Conn, error) {
	var (
		connConfig fbx.ConnConfig
		conn       *fbx.Conn
		close      func(context.Context) error
		err        error
	)

	if c.pool == nil {
		// Create a shallow copy of the config, so that BeforeConnect can safely modify it
		connConfig = c.ConnConfig

		if err = c.BeforeConnect(ctx, &connConfig); err != nil {
			return nil, err
		}

		if conn, err = fbx.ConnectConfig(ctx, &connConfig); err != nil {
			return nil, err
		}

		if err = c.AfterConnect(ctx, conn); err != nil {
			return nil, err
		}

		close = conn.Close
	} else {
		var pconn *fbxpool.Conn

		pconn, err = c.pool.Acquire(ctx)
		if err != nil {
			return nil, err
		}

		conn = pconn.Conn()

		close = func(_ context.Context) error {
			pconn.Release()
			return nil
		}
	}

	return &Conn{
		conn:             conn,
		close:            close,
		driver:           c.driver,
		connConfig:       connConfig,
		resetSessionFunc: c.ResetSession,
		shouldPing:       c.ShouldPing,
		psRefCounts:      make(map[*fbconn.StatementDescription]int),
	}, nil
}

// Driver implement driver.Connector interface
func (c connector) Driver() driver.Driver {
	return c.driver
}

// GetDefaultDriver returns the driver initialized in the init function
// and used when the fbx driver is registered.
func GetDefaultDriver() driver.Driver {
	return fbxDriver
}

type Driver struct {
	configMutex sync.Mutex
	configs     map[string]*fbx.ConnConfig
	sequence    int
}

func (d *Driver) Open(name string) (driver.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Ensure eventual timeout
	defer cancel()

	connector, err := d.OpenConnector(name)
	if err != nil {
		return nil, err
	}
	return connector.Connect(ctx)
}

func (d *Driver) OpenConnector(name string) (driver.Connector, error) {
	return &driverConnector{driver: d, name: name}, nil
}

func (d *Driver) registerConnConfig(c *fbx.ConnConfig) string {
	d.configMutex.Lock()
	connStr := fmt.Sprintf("registeredConnConfig%d", d.sequence)
	d.sequence++
	d.configs[connStr] = c
	d.configMutex.Unlock()
	return connStr
}

func (d *Driver) unregisterConnConfig(connStr string) {
	d.configMutex.Lock()
	delete(d.configs, connStr)
	d.configMutex.Unlock()
}

type driverConnector struct {
	driver *Driver
	name   string
}

func (dc *driverConnector) Connect(ctx context.Context) (driver.Conn, error) {
	var connConfig *fbx.ConnConfig

	dc.driver.configMutex.Lock()
	connConfig = dc.driver.configs[dc.name]
	dc.driver.configMutex.Unlock()

	if connConfig == nil {
		var err error
		connConfig, err = fbx.ParseConfig(dc.name)
		if err != nil {
			return nil, err
		}
	}

	conn, err := fbx.ConnectConfig(ctx, connConfig)
	if err != nil {
		return nil, err
	}

	c := &Conn{
		conn:             conn,
		close:            conn.Close,
		driver:           dc.driver,
		connConfig:       *connConfig,
		resetSessionFunc: func(context.Context, *fbx.Conn) error { return nil },
		psRefCounts:      make(map[*fbconn.StatementDescription]int),
	}

	return c, nil
}

func (dc *driverConnector) Driver() driver.Driver {
	return dc.driver
}

// RegisterConnConfig registers a ConnConfig and returns the connection string to use with Open.
func RegisterConnConfig(c *fbx.ConnConfig) string {
	return fbxDriver.registerConnConfig(c)
}

// UnregisterConnConfig removes the ConnConfig registration for connStr.
func UnregisterConnConfig(connStr string) {
	fbxDriver.unregisterConnConfig(connStr)
}

type Conn struct {
	conn                 *fbx.Conn
	close                func(context.Context) error
	driver               *Driver
	connConfig           fbx.ConnConfig
	resetSessionFunc     func(context.Context, *fbx.Conn) error       // Function is called before a connection is reused
	shouldPing           func(context.Context, ShouldPingParams) bool // Function to decide if stdlib should ping before reusing a connection
	lastResetSessionTime time.Time

	// psRefCounts contains reference counts for prepared statements. Prepare uses the underlying fbx logic to generate
	// deterministic statement names from the statement text. If this query has already been prepared then the existing
	// *fbconn.StatementDescription will be returned. However, this means that if Close is called on the returned Stmt
	// then the underlying prepared statement will be closed even when the underlying prepared statement is still in use
	// by another database/sql Stmt. To prevent this psRefCounts keeps track of how many database/sql statements are using
	// the same underlying statement and only closes the underlying statement when the reference count reaches 0.
	psRefCounts map[*fbconn.StatementDescription]int
}

// Conn returns the underlying *fbx.Conn
func (c *Conn) Conn() *fbx.Conn {
	return c.conn
}

func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

func (c *Conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if c.conn.IsClosed() {
		return nil, driver.ErrBadConn
	}

	sd, err := c.conn.Prepare(ctx, query, query)
	if err != nil {
		return nil, err
	}
	c.psRefCounts[sd]++

	return &Stmt{sd: sd, conn: c}, nil
}

func (c *Conn) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	return c.close(ctx)
}

func (c *Conn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *Conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if c.conn.IsClosed() {
		return nil, driver.ErrBadConn
	}

	var fbxOpts fbx.TxOptions
	switch sql.IsolationLevel(opts.Isolation) {
	case sql.LevelDefault:
		// Use server / database default isolation. This is important for Firebird where the default is
		// SNAPSHOT (concurrency) instead of READ COMMITTED.
		// Leave fbxOpts.IsoLevel empty so fbx.Conn.BeginTx can use server defaults.
	case sql.LevelReadCommitted:
		fbxOpts.IsoLevel = fbx.ReadCommitted
	case sql.LevelRepeatableRead, sql.LevelSnapshot:
		fbxOpts.IsoLevel = fbx.RepeatableRead
	case sql.LevelSerializable:
		fbxOpts.IsoLevel = fbx.Serializable
	default:
		return nil, fmt.Errorf("unsupported isolation: %v", opts.Isolation)
	}

	if opts.ReadOnly {
		fbxOpts.AccessMode = fbx.ReadOnly
	}

	tx, err := c.conn.BeginTx(ctx, fbxOpts)
	if err != nil {
		return nil, err
	}

	return wrapTx{ctx: ctx, tx: tx}, nil
}

func (c *Conn) ExecContext(ctx context.Context, query string, argsV []driver.NamedValue) (driver.Result, error) {
	if c.conn.IsClosed() {
		return nil, driver.ErrBadConn
	}

	args := make([]any, len(argsV))
	convertNamedArguments(args, argsV)

	commandTag, err := c.conn.Exec(ctx, query, args...)
	// if we got a network error before we had a chance to send the query, retry
	if err != nil {
		if fbconn.SafeToRetry(err) {
			return nil, driver.ErrBadConn
		}
	}
	return driver.RowsAffected(commandTag.RowsAffected()), err
}

func (c *Conn) QueryContext(ctx context.Context, query string, argsV []driver.NamedValue) (driver.Rows, error) {
	if c.conn.IsClosed() {
		return nil, driver.ErrBadConn
	}

	args := make([]any, len(argsV))
	convertNamedArguments(args, argsV)

	rows, err := c.conn.Query(ctx, query, args...)
	if err != nil {
		if fbconn.SafeToRetry(err) {
			return nil, driver.ErrBadConn
		}
		return nil, err
	}

	// Preload first row because otherwise we won't know what columns are available when database/sql asks.
	more := rows.Next()
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	return &Rows{conn: c, rows: rows, skipNext: true, skipNextMore: more}, nil
}

func (c *Conn) Ping(ctx context.Context) error {
	if c.conn.IsClosed() {
		return driver.ErrBadConn
	}

	err := c.conn.Ping(ctx)
	if err != nil {
		if c.conn.IsClosed() {
			return driver.ErrBadConn
		}
		return err
	}

	return nil
}

func (c *Conn) CheckNamedValue(*driver.NamedValue) error {
	// Underlying fbx supports sql.Scanner and driver.Valuer interfaces natively. So everything can be passed through directly.
	return nil
}

func (c *Conn) ResetSession(ctx context.Context) error {
	if c.conn.IsClosed() {
		return driver.ErrBadConn
	}

	// Discard connection if it has an open transaction. This can happen if the
	// application did not properly commit or rollback a transaction.
	if c.conn.FbConn().ActiveExplicitTx() {
		return driver.ErrBadConn
	}

	now := time.Now()
	idle := now.Sub(c.lastResetSessionTime)

	doPing := idle > time.Second // default behavior: ping only if idle > 1s

	if c.shouldPing != nil {
		doPing = c.shouldPing(ctx, ShouldPingParams{
			Conn:         c.conn,
			IdleDuration: idle,
		})
	}

	if doPing {
		if err := c.conn.FbConn().Ping(ctx); err != nil {
			return driver.ErrBadConn
		}
	}

	c.lastResetSessionTime = now

	return c.resetSessionFunc(ctx, c.conn)
}

type Stmt struct {
	sd   *fbconn.StatementDescription
	conn *Conn
}

func (s *Stmt) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	refCount := s.conn.psRefCounts[s.sd]
	if refCount == 1 {
		delete(s.conn.psRefCounts, s.sd)
	} else {
		s.conn.psRefCounts[s.sd]--
		return nil
	}

	return s.conn.conn.Deallocate(ctx, s.sd.SQL)
}

func (s *Stmt) NumInput() int {
	return len(s.sd.Parameters)
}

func (s *Stmt) Exec(argsV []driver.Value) (driver.Result, error) {
	return nil, errors.New("Stmt.Exec deprecated and not implemented")
}

func (s *Stmt) ExecContext(ctx context.Context, argsV []driver.NamedValue) (driver.Result, error) {
	return s.conn.ExecContext(ctx, s.sd.SQL, argsV)
}

func (s *Stmt) Query(argsV []driver.Value) (driver.Rows, error) {
	return nil, errors.New("Stmt.Query deprecated and not implemented")
}

func (s *Stmt) QueryContext(ctx context.Context, argsV []driver.NamedValue) (driver.Rows, error) {
	return s.conn.QueryContext(ctx, s.sd.SQL, argsV)
}

type rowValueFunc func(src []byte) (driver.Value, error)

type Rows struct {
	conn         *Conn
	rows         fbx.Rows
	valueFuncs   []rowValueFunc
	blobColumns  []bool
	hasBlobCols  bool
	skipNext     bool
	skipNextMore bool

	columnNames []string
}

func (r *Rows) Columns() []string {
	if r.columnNames == nil {
		fields := r.rows.FieldDescriptions()
		r.columnNames = make([]string, len(fields))
		for i, fd := range fields {
			r.columnNames[i] = string(fd.Name)
		}
	}

	return r.columnNames
}

// ColumnTypeDatabaseTypeName returns the database system type name. If the name is unknown the TypeID is returned.
func (r *Rows) ColumnTypeDatabaseTypeName(index int) string {
	fd := r.rows.FieldDescriptions()[index]
	if dt, ok := r.conn.conn.TypeMap().TypeForTypeID(fd.Type, fd.Subtype); ok {
		return strings.ToUpper(dt.Name)
	}

	return strconv.FormatInt(int64(fd.Type), 10)
}

// ColumnTypeLength returns the length of the column type if the column is a
// variable length type. If the column is not a variable length type ok
// should return false.
func (r *Rows) ColumnTypeLength(index int) (int64, bool) {
	fd := r.rows.FieldDescriptions()[index]

	switch fd.Type {
	case fbproto.SQL_TYPE_TEXT:
		return characterLength(fd), true
	case fbproto.SQL_TYPE_VARYING:
		return characterLength(fd), true
	default:
		return 0, false
	}
}

func characterLength(fd fbconn.FieldDescription) int64 {
	charsetId := int32(fd.Subtype) & 0xFF
	name, err := fbproto.CharacterSetIdIntoName(charsetId)
	if err != nil {
		return int64(fd.Length)
	}
	encoding, err := fbproto.NewEncoding(name)
	if err != nil || encoding.MaxBytesPerChar == 0 {
		return int64(fd.Length)
	}

	maxBytes := uint32(encoding.MaxBytesPerChar)
	if maxBytes == 0 {
		return int64(fd.Length)
	}
	if fd.Length%maxBytes == 0 {
		return int64(fd.Length / maxBytes)
	}
	return int64(fd.Length)
}

// ColumnTypePrecisionScale should return the precision and scale for decimal
// types. If not applicable, ok should be false.
func (r *Rows) ColumnTypePrecisionScale(index int) (precision, scale int64, ok bool) {
	fd := r.rows.FieldDescriptions()[index]

	if !isNumericSubtype(fd.Subtype) {
		return 0, 0, false
	}

	precision, ok = numericPrecisionForField(fd)
	if !ok {
		return 0, 0, false
	}

	scale = int64(fd.Scale)
	if scale < 0 {
		scale = -scale
	}

	return precision, scale, true
}

func isNumericSubtype(subtype int32) bool {
	return subtype == 1 || subtype == 2
}

func numericPrecisionForField(fd fbconn.FieldDescription) (int64, bool) {
	switch fd.Type {
	case fbproto.SQL_TYPE_SHORT:
		return 4, true
	case fbproto.SQL_TYPE_LONG:
		return 9, true
	case fbproto.SQL_TYPE_INT64:
		return 18, true
	case fbproto.SQL_TYPE_INT128:
		return 38, true
	case fbproto.SQL_TYPE_DEC64:
		return 16, true
	case fbproto.SQL_TYPE_DEC128:
		return 34, true
	default:
		return 0, false
	}
}

// ColumnTypeScanType returns the value type that can be used to scan types into.
func (r *Rows) ColumnTypeScanType(index int) reflect.Type {
	fd := r.rows.FieldDescriptions()[index]

	if isNumericSubtype(fd.Subtype) {
		return reflect.TypeOf(float64(0))
	}

	switch fd.Type {
	case fbproto.SQL_TYPE_BLOB:
		return reflect.TypeOf(Blob{})
	case fbproto.SQL_TYPE_D_FLOAT:
		return reflect.TypeOf(float64(0))
	case fbproto.SQL_TYPE_FLOAT:
		return reflect.TypeOf(float32(0))
	case fbproto.SQL_TYPE_INT64:
		return reflect.TypeOf(int64(0))
	case fbproto.SQL_TYPE_LONG:
		return reflect.TypeOf(int32(0))
	case fbproto.SQL_TYPE_SHORT:
		return reflect.TypeOf(int16(0))
	case fbproto.SQL_TYPE_BOOLEAN:
		return reflect.TypeOf(false)
	case fbproto.SQL_TYPE_DATE, fbproto.SQL_TYPE_TIMESTAMP, fbproto.SQL_TYPE_TIMESTAMP_TZ, fbproto.SQL_TYPE_TIME:
		return reflect.TypeOf(time.Time{})
	default:
		return reflect.TypeOf("")
	}
}

func (r *Rows) Close() error {
	r.rows.Close()
	return r.rows.Err()
}

func (r *Rows) Next(dest []driver.Value) error {
	m := r.conn.conn.TypeMap()
	fieldDescriptions := r.rows.FieldDescriptions()

	if r.valueFuncs == nil {
		r.valueFuncs = make([]rowValueFunc, len(fieldDescriptions))
		r.blobColumns = make([]bool, len(fieldDescriptions))

		for i, fd := range fieldDescriptions {
			typeMetadata := fbtype.NewTypeMetadata(fd.Type, fd.Subtype, fd.Length, fd.Scale)

			switch fd.Type {
			case fbproto.SQL_TYPE_BLOB:
				r.blobColumns[i] = true
				r.hasBlobCols = true
				continue
			case fbproto.SQL_TYPE_BOOLEAN:
				var d bool
				scanPlan := m.PlanScan(typeMetadata, &d)
				r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
					err := scanPlan.Scan(src, &d)
					return d, err
				}
			case fbproto.SQL_TYPE_DATE:
				var d fbtype.Date
				scanPlan := m.PlanScan(typeMetadata, &d)
				r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
					err := scanPlan.Scan(src, &d)
					if err != nil {
						return nil, err
					}
					return d.Value()
				}
			case fbproto.SQL_TYPE_FLOAT:
				var d float32
				scanPlan := m.PlanScan(typeMetadata, &d)
				r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
					err := scanPlan.Scan(src, &d)
					return float64(d), err
				}
			case fbproto.SQL_TYPE_DOUBLE:
				var d float64
				scanPlan := m.PlanScan(typeMetadata, &d)
				r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
					err := scanPlan.Scan(src, &d)
					return d, err
				}
			case fbproto.SQL_TYPE_SHORT:
				if isNumericSubtype(fd.Subtype) {
					var d float64
					scanPlan := m.PlanScan(typeMetadata, &d)
					r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
						err := scanPlan.Scan(src, &d)
						return d, err
					}
				} else {
					var d int16
					scanPlan := m.PlanScan(typeMetadata, &d)
					r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
						err := scanPlan.Scan(src, &d)
						return int64(d), err
					}
				}

			case fbproto.SQL_TYPE_LONG:
				if isNumericSubtype(fd.Subtype) {
					var d float64
					scanPlan := m.PlanScan(typeMetadata, &d)
					r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
						err := scanPlan.Scan(src, &d)
						return d, err
					}
				} else {
					var d int32
					scanPlan := m.PlanScan(typeMetadata, &d)
					r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
						err := scanPlan.Scan(src, &d)
						return int64(d), err
					}
				}

			case fbproto.SQL_TYPE_INT64:
				if isNumericSubtype(fd.Subtype) {
					var d float64
					scanPlan := m.PlanScan(typeMetadata, &d)
					r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
						err := scanPlan.Scan(src, &d)
						return d, err
					}
				} else {
					var d int64
					scanPlan := m.PlanScan(typeMetadata, &d)
					r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
						err := scanPlan.Scan(src, &d)
						return d, err
					}
				}
			case fbproto.SQL_TYPE_TIMESTAMP:
				var d fbtype.Timestamp
				scanPlan := m.PlanScan(typeMetadata, &d)
				r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
					err := scanPlan.Scan(src, &d)
					if err != nil {
						return nil, err
					}
					return d.Value()
				}
			case fbproto.SQL_TYPE_TIMESTAMP_TZ:
				var d fbtype.Timestamptz
				scanPlan := m.PlanScan(typeMetadata, &d)
				r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
					err := scanPlan.Scan(src, &d)
					if err != nil {
						return nil, err
					}
					return d.Value()
				}
			case fbproto.SQL_TYPE_TIME:
				var d time.Time
				scanPlan := m.PlanScan(typeMetadata, &d)
				r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
					err := scanPlan.Scan(src, &d)
					return d, err
				}
			case fbproto.SQL_TYPE_INT128:
				if isNumericSubtype(fd.Subtype) {
					var d float64
					scanPlan := m.PlanScan(typeMetadata, &d)
					r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
						err := scanPlan.Scan(src, &d)
						return d, err
					}
				} else {
					var d string
					scanPlan := m.PlanScan(typeMetadata, &d)
					r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
						err := scanPlan.Scan(src, &d)
						return d, err
					}
				}
			case fbproto.SQL_TYPE_DEC64, fbproto.SQL_TYPE_DEC128:
				var d string
				scanPlan := m.PlanScan(typeMetadata, &d)
				r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
					err := scanPlan.Scan(src, &d)
					return d, err
				}
			case fbproto.SQL_TYPE_ARRAY:
				r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
					var array fbtype.ArraySQLScanner
					var err error
					array.ArrayBuffer = src
					array.ArrayData, err = fbtype.NewArrayData(fd.ArrayData.TypeId,
						fd.ArrayData.SubType,
						fd.ArrayData.Length,
						fd.ArrayData.Scale,
						fd.ArrayData.Dimensions)
					return array, err
				}
			default:
				if (fd.Subtype & 0xFF) != fbproto.CS_BINARY {
					var d string
					scanPlan := m.PlanScan(typeMetadata, &d)
					r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
						err := scanPlan.Scan(src, &d)
						return d, err
					}
				} else {
					var d []byte
					scanPlan := m.PlanScan(typeMetadata, &d)
					r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
						err := scanPlan.Scan(src, &d)
						return d, err
					}
				}
			}
		}
	}

	var more bool
	if r.skipNext {
		more = r.skipNextMore
		r.skipNext = false
	} else {
		more = r.rows.Next()
	}

	if !more {
		if r.rows.Err() == nil {
			return io.EOF
		} else {
			return r.rows.Err()
		}
	}

	var blobValues []any
	if r.hasBlobCols {
		values, err := r.rows.Values()
		if err != nil {
			return err
		}
		blobValues = values
	}

	for i, rv := range r.rows.RawValues() {
		if rv != nil {
			if r.blobColumns[i] {
				value := blobValues[i]
				if value == nil {
					dest[i] = nil
					continue
				}
				reader, ok := value.(io.ReadCloser)
				if !ok {
					return fmt.Errorf("blob column %d returned %T", i, value)
				}
				blobValue, err := buildBlobValue(reader, &fieldDescriptions[i], r.conn)
				if err != nil {
					_ = reader.Close()
					return err
				}
				dest[i] = blobValue
				continue
			}
			var err error
			dest[i], err = r.valueFuncs[i](rv)
			if err != nil {
				return fmt.Errorf("convert field %d failed: %w", i, err)
			}
		} else {
			dest[i] = nil
		}
	}

	return nil
}

func convertNamedArguments(args []any, argsV []driver.NamedValue) {
	for i, v := range argsV {
		if v.Value != nil {
			args[i] = v.Value.(any)
		} else {
			args[i] = nil
		}
	}
}

type wrapTx struct {
	ctx context.Context
	tx  fbx.Tx
}

func (wtx wrapTx) Commit() error { return wtx.tx.Commit(wtx.ctx) }

func (wtx wrapTx) Rollback() error { return wtx.tx.Rollback(wtx.ctx) }
