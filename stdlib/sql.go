// Package stdlib provides the [database/sql] adapter for fbx.
//
// Importing stdlib makes the "fbx" driver available to [sql.Open]. Both URL
// and keyword/value connection strings are supported:
//
//	ctx := context.Background()
//	db, err := sql.Open("fbx", "firebird://user:secret@localhost:3050/fbx_test")
//	if err != nil {
//		return err
//	}
//	defer db.Close()
//
//	if err := db.PingContext(ctx); err != nil {
//		return err
//	}
//
// A keyword/value connection string can be used in the same way:
//
//	db, err := sql.Open("fbx", "user=user password=secret host=localhost port=3050 database=fbx_test")
//
// [sql.Open] may not establish a connection immediately. Use
// [sql.DB.PingContext] or execute a query to verify the connection settings.
//
// An existing [fbxpool.Pool] can be exposed through [database/sql]:
//
//	pool, err := fbxpool.New(ctx, os.Getenv("DATABASE_URL"))
//	if err != nil {
//		return err
//	}
//	defer pool.Close()
//
//	db := stdlib.OpenDBFromPool(pool)
//	defer db.Close()
//
// Use [OpenDB] to open a database from a customized [fbx.ConnConfig]. Use
// [RegisterConnConfig] only when the configuration must be passed to
// [sql.Open].
//
//	connConfig, err := fbx.ParseConfig(os.Getenv("DATABASE_URL"))
//	if err != nil {
//		return err
//	}
//	db := stdlib.OpenDB(*connConfig)
//
// Firebird parameters are positional and use question mark placeholders:
//
//	db.QueryRowContext(ctx, "select * from users where id=?", userID)
//
// Direct query methods also support named placeholder rewriting through
// [fbx.NamedArgs]. [sql.Named] names are not interpreted as Firebird
// placeholder names.
//
//	db.QueryRowContext(ctx, "select * from users where id=@id", fbx.NamedArgs{"id": userID})
//
// [sql.Conn.Raw] exposes the underlying [fbx.Conn] for the duration of its
// callback. The underlying connection must not be retained after the callback
// returns.
//
//	sqlConn, err := db.Conn(ctx)
//	if err != nil {
//		return err
//	}
//	defer sqlConn.Close()
//
//	err = sqlConn.Raw(func(driverConn any) error {
//		fbxConn := driverConn.(*stdlib.Conn).Conn()
//		_ = fbxConn.ServerVersion()
//		return nil
//	})
//
// # Firebird-specific data types
//
// The [fbtype] package provides Firebird-specific types and codecs.
// [fbtype.Map.SQLScanner] is primarily used to scan Firebird arrays through
// [database/sql]:
//
//	input := []int64{1, 2, 3}
//	_, err := db.Exec("insert into array_example(bigint_array) values(?)", input)
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

	// Keep the default registration owned by the first imported fbx major version.
	if !slices.Contains(sql.Drivers(), "fbx") {
		sql.Register("fbx", fbxDriver)
	}
}

// OptionOpenDB configures a connector created by [GetConnector] or
// [GetPoolConnector].
type OptionOpenDB func(*connector)

// ShouldPingParams contains the values passed to the function configured by
// [OptionShouldPing].
type ShouldPingParams struct {
	// Conn is the underlying fbx connection.
	Conn *fbx.Conn

	// IdleDuration is the elapsed time since this driver connection was created
	// or ResetSession previously ran. It approximates, but does not exactly
	// measure, the time spent idle in the database/sql pool.
	IdleDuration time.Duration
}

// OptionShouldPing sets the function that decides whether stdlib performs a
// liveness ping before database/sql reuses a connection. Returning true enables
// the ping. Without this option, connections are pinged when IdleDuration is
// greater than one second.
func OptionShouldPing(f func(context.Context, ShouldPingParams) bool) OptionOpenDB {
	return func(dc *connector) { dc.shouldPing = f }
}

// OptionBeforeConnect sets a function called before each connection created by
// [GetConnector] or [OpenDB]. The function receives a per-connection copy
// produced by [fbx.ConnConfig.Copy]. Pool connectors ignore this option;
// configure connection hooks on [fbxpool.Config] instead.
func OptionBeforeConnect(bc func(context.Context, *fbx.ConnConfig) error) OptionOpenDB {
	return func(dc *connector) {
		dc.beforeConnect = bc
	}
}

// OptionAfterConnect sets a function called after each connection created by
// [GetConnector] or [OpenDB]. Returning an error aborts that connection attempt.
// Pool connectors ignore this option; configure connection hooks on
// [fbxpool.Config] instead.
func OptionAfterConnect(ac func(context.Context, *fbx.Conn) error) OptionOpenDB {
	return func(dc *connector) {
		dc.afterConnect = ac
	}
}

// OptionResetSession sets a function called before database/sql reuses a
// connection. Returning [driver.ErrBadConn] tells database/sql not to reuse the
// driver connection.
func OptionResetSession(rs func(context.Context, *fbx.Conn) error) OptionOpenDB {
	return func(dc *connector) {
		dc.resetSession = rs
	}
}

// RandomizeHostOrderFunc is an [OptionBeforeConnect] hook that randomly selects
// the primary host from the configured primary and fallback hosts. It can
// distribute new connections among equivalent servers. Use
// [sql.DB.SetConnMaxLifetime] when existing connections should also be replaced
// periodically.
func RandomizeHostOrderFunc(_ context.Context, connConfig *fbx.ConnConfig) error {
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

	// Use the last shuffled host as the primary and keep the rest as fallbacks.
	newPrimary := newFallbacks[len(newFallbacks)-1]
	connConfig.Host = newPrimary.Host
	connConfig.Port = newPrimary.Port
	connConfig.Fallbacks = newFallbacks[:len(newFallbacks)-1]
	return nil
}

// GetConnector returns a [driver.Connector] that creates fbx connections using
// a snapshot of config and the supplied opts. Pass the result to [sql.OpenDB].
func GetConnector(config fbx.ConnConfig, opts ...OptionOpenDB) driver.Connector {
	c := connector{
		config:        *config.Copy(),
		beforeConnect: func(context.Context, *fbx.ConnConfig) error { return nil },
		afterConnect:  func(context.Context, *fbx.Conn) error { return nil },
		resetSession:  func(context.Context, *fbx.Conn) error { return nil },
		driver:        fbxDriver,
	}

	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// GetPoolConnector returns a [driver.Connector] that acquires connections from
// pool. A [sql.DB] created from this connector must have its maximum idle
// connections set to zero so database/sql does not retain connections acquired
// from the underlying pool. [OpenDBFromPool] applies this setting automatically.
// GetPoolConnector panics if pool is nil.
func GetPoolConnector(pool *fbxpool.Pool, opts ...OptionOpenDB) driver.Connector {
	if pool == nil {
		panic("stdlib: GetPoolConnector called with nil pool")
	}

	c := connector{
		pool:         pool,
		resetSession: func(context.Context, *fbx.Conn) error { return nil },
		driver:       fbxDriver,
	}

	for _, opt := range opts {
		opt(&c)
	}

	return c
}

// OpenDB returns a [sql.DB] that creates connections using config and opts.
// The first connection is established lazily when the database is used.
func OpenDB(config fbx.ConnConfig, opts ...OptionOpenDB) *sql.DB {
	c := GetConnector(config, opts...)
	return sql.OpenDB(c)
}

// OpenDBFromPool returns a [sql.DB] that acquires connections from pool. It sets
// the maximum number of idle database/sql connections to zero so the underlying
// pool retains ownership of idle connections. Closing the returned database
// does not close pool. OpenDBFromPool panics if pool is nil.
func OpenDBFromPool(pool *fbxpool.Pool, opts ...OptionOpenDB) *sql.DB {
	c := GetPoolConnector(pool, opts...)
	db := sql.OpenDB(c)
	db.SetMaxIdleConns(0)
	return db
}

type connector struct {
	config        fbx.ConnConfig
	pool          *fbxpool.Pool
	beforeConnect func(context.Context, *fbx.ConnConfig) error
	afterConnect  func(context.Context, *fbx.Conn) error
	resetSession  func(context.Context, *fbx.Conn) error
	shouldPing    func(context.Context, ShouldPingParams) bool
	driver        *Driver
}

// Connect implements [driver.Connector] by opening a native connection or
// acquiring one from the configured pool.
func (c connector) Connect(ctx context.Context) (driver.Conn, error) {
	var (
		connConfig fbx.ConnConfig
		conn       *fbx.Conn
		closeConn  func(context.Context) error
		err        error
	)

	if c.pool == nil {
		connConfig = *c.config.Copy()

		if err = c.beforeConnect(ctx, &connConfig); err != nil {
			return nil, err
		}

		if conn, err = fbx.ConnectConfig(ctx, &connConfig); err != nil {
			return nil, err
		}

		if err = c.afterConnect(ctx, conn); err != nil {
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			closeErr := conn.Close(closeCtx)
			cancel()
			if closeErr != nil {
				return nil, errors.Join(err, fmt.Errorf("close connection after AfterConnect error: %w", closeErr))
			}
			return nil, err
		}

		closeConn = conn.Close
	} else {
		var pconn *fbxpool.Conn

		pconn, err = c.pool.Acquire(ctx)
		if err != nil {
			return nil, err
		}

		conn = pconn.Conn()

		closeConn = func(_ context.Context) error {
			pconn.Release()
			return nil
		}
	}

	return &Conn{
		conn:                 conn,
		close:                closeConn,
		driver:               c.driver,
		connConfig:           connConfig,
		resetSessionFunc:     c.resetSession,
		shouldPing:           c.shouldPing,
		lastResetSessionTime: time.Now(),
		psRefCounts:          make(map[*fbconn.StatementDescription]int),
	}, nil
}

// Driver returns the driver associated with the connector.
func (c connector) Driver() driver.Driver {
	return c.driver
}

// GetDefaultDriver returns this package's default driver. Package initialization
// registers it under the "fbx" name unless that name is already owned by
// another imported driver. The returned driver can be registered manually under
// another name.
func GetDefaultDriver() driver.Driver {
	return fbxDriver
}

// Driver implements [driver.Driver] and [driver.DriverContext] for Firebird
// connections.
type Driver struct {
	configMu sync.RWMutex
	configs  map[string]*fbx.ConnConfig
	sequence int
}

// Open implements [driver.Driver]. It bounds the connection attempt because
// this legacy interface does not accept a context.
func (d *Driver) Open(name string) (driver.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	connector, err := d.OpenConnector(name)
	if err != nil {
		return nil, err
	}
	return connector.Connect(ctx)
}

// OpenConnector implements [driver.DriverContext]. It resolves name once and
// returns a connector with a fixed connection configuration.
func (d *Driver) OpenConnector(name string) (driver.Connector, error) {
	d.configMu.RLock()
	connConfig := d.configs[name]
	d.configMu.RUnlock()

	if connConfig == nil {
		var err error
		connConfig, err = fbx.ParseConfig(name)
		if err != nil {
			return nil, err
		}
	}

	return &driverConnector{driver: d, config: connConfig.Copy()}, nil
}

func (d *Driver) registerConnConfig(c *fbx.ConnConfig) string {
	connConfig := c.Copy()

	d.configMu.Lock()
	defer d.configMu.Unlock()

	if d.configs == nil {
		d.configs = make(map[string]*fbx.ConnConfig)
	}
	connStr := fmt.Sprintf("registeredConnConfig%d", d.sequence)
	d.sequence++
	d.configs[connStr] = connConfig
	return connStr
}

func (d *Driver) unregisterConnConfig(connStr string) {
	d.configMu.Lock()
	defer d.configMu.Unlock()

	delete(d.configs, connStr)
}

type driverConnector struct {
	driver *Driver
	config *fbx.ConnConfig
}

// Connect implements [driver.Connector] using its parsed configuration.
func (dc *driverConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := fbx.ConnectConfig(ctx, dc.config)
	if err != nil {
		return nil, err
	}

	return &Conn{
		conn:                 conn,
		close:                conn.Close,
		driver:               dc.driver,
		connConfig:           *dc.config,
		resetSessionFunc:     func(context.Context, *fbx.Conn) error { return nil },
		lastResetSessionTime: time.Now(),
		psRefCounts:          make(map[*fbconn.StatementDescription]int),
	}, nil
}

// Driver returns the driver associated with the connector.
func (dc *driverConnector) Driver() driver.Driver {
	return dc.driver
}

// RegisterConnConfig stores a copy of c and returns an opaque connection string
// for use with this package's default [Driver]. Once [sql.Open] returns
// successfully, the registration may be removed because its connector retains
// another copy. RegisterConnConfig panics if c is nil.
func RegisterConnConfig(c *fbx.ConnConfig) string {
	if c == nil {
		panic("stdlib: RegisterConnConfig called with nil config")
	}
	return fbxDriver.registerConnConfig(c)
}

// UnregisterConnConfig removes the configuration registered for connStr. It has
// no effect if connStr is not registered.
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

func numericRowValueFunc(m *fbtype.Map, typeMetadata fbtype.TypeMetadata) rowValueFunc {
	var d fbtype.Numeric
	scanPlan := m.PlanScan(typeMetadata, &d)
	return func(src []byte) (driver.Value, error) {
		if err := scanPlan.Scan(src, &d); err != nil {
			return nil, err
		}
		return d.Value()
	}
}

func rawBytesRowValueFunc(src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}
	return slices.Clone(src), nil
}

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

func isScaledNumericField(fd fbconn.FieldDescription) bool {
	if !isNumericSubtype(fd.Subtype) {
		return false
	}

	switch fd.Type {
	case fbproto.SQL_TYPE_SHORT, fbproto.SQL_TYPE_LONG, fbproto.SQL_TYPE_INT64, fbproto.SQL_TYPE_INT128:
		return true
	default:
		return false
	}
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
	return columnTypeScanType(r.rows.FieldDescriptions()[index])
}

func columnTypeScanType(fd fbconn.FieldDescription) reflect.Type {
	if isScaledNumericField(fd) {
		return reflect.TypeOf("")
	}

	switch fd.Type {
	case fbproto.SQL_TYPE_BLOB:
		return reflect.TypeOf(Blob{})
	case fbproto.SQL_TYPE_DOUBLE:
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
	case fbproto.SQL_TYPE_DATE, fbproto.SQL_TYPE_TIMESTAMP, fbproto.SQL_TYPE_TIMESTAMP_TZ, fbproto.SQL_TYPE_TIME, fbproto.SQL_TYPE_TIME_TZ:
		return reflect.TypeOf(time.Time{})
	case fbproto.SQL_TYPE_INT128, fbproto.SQL_TYPE_DEC64, fbproto.SQL_TYPE_DEC128:
		return reflect.TypeOf("")
	case fbproto.SQL_TYPE_TEXT, fbproto.SQL_TYPE_VARYING:
		if fd.Subtype&0xFF == fbproto.CS_BINARY {
			return reflect.TypeOf([]byte(nil))
		}
		return reflect.TypeOf("")
	case fbproto.SQL_TYPE_ARRAY, fbproto.SQL_TYPE_NULL:
		return reflect.TypeOf((*any)(nil)).Elem()
	default:
		return reflect.TypeOf((*any)(nil)).Elem()
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
			if isScaledNumericField(fd) {
				r.valueFuncs[i] = numericRowValueFunc(m, typeMetadata)
				continue
			}

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
				var d int16
				scanPlan := m.PlanScan(typeMetadata, &d)
				r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
					err := scanPlan.Scan(src, &d)
					return int64(d), err
				}
			case fbproto.SQL_TYPE_LONG:
				var d int32
				scanPlan := m.PlanScan(typeMetadata, &d)
				r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
					err := scanPlan.Scan(src, &d)
					return int64(d), err
				}
			case fbproto.SQL_TYPE_INT64:
				var d int64
				scanPlan := m.PlanScan(typeMetadata, &d)
				r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
					err := scanPlan.Scan(src, &d)
					return d, err
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
			case fbproto.SQL_TYPE_TIME_TZ:
				var d fbtype.Timetz
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
				var d string
				scanPlan := m.PlanScan(typeMetadata, &d)
				r.valueFuncs[i] = func(src []byte) (driver.Value, error) {
					err := scanPlan.Scan(src, &d)
					return d, err
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
					if fd.ArrayData == nil {
						return nil, errors.New("array field is missing element metadata")
					}

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
			case fbproto.SQL_TYPE_TEXT, fbproto.SQL_TYPE_VARYING:
				if fd.Subtype&0xFF == fbproto.CS_BINARY {
					var d []byte
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
			default:
				r.valueFuncs[i] = rawBytesRowValueFunc
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
