package fbx

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/internal/stmtcache"
)

// ConnConfig contains all the options used to establish a connection. It must be created by ParseConfig and
// then it can be modified. A manually initialized ConnConfig will cause ConnectConfig to panic.
type ConnConfig struct {
	fbconn.Config

	Tracer QueryTracer

	// Original connection string that was parsed into config.
	connString string

	// StatementCacheCapacity is maximum size of the statement cache used when executing a query with "cache_statement"
	// query exec mode.
	StatementCacheCapacity int

	// DefaultQueryExecMode controls the default mode for executing queries. By default fbx prepares statements and
	// may cache statement/description metadata to reduce round trips and parsing overhead.
	DefaultQueryExecMode QueryExecMode

	// BatchInsertPendingSize is maximum size of pending data in batch before
	// [Conn.BatchInsert] will flush it to server.
	BatchInsertPendingSize int

	createdByParseConfig bool // Used to enforce created by ParseConfig rule.
}

// ParseConfigOptions contains options that control how a config is built.
type ParseConfigOptions struct {
	fbconn.ParseConfigOptions
}

// Copy returns a deep copy of the config that is safe to use and modify.
// The only exception is the tls.Config:
// according to the tls.Config docs it must not be modified after creation.
func (cc *ConnConfig) Copy() *ConnConfig {
	newConfig := new(ConnConfig)
	*newConfig = *cc
	newConfig.Config = *newConfig.Config.Copy()
	return newConfig
}

// ConnString returns the connection string as parsed by fbx.ParseConfig into fbx.ConnConfig.
func (cc *ConnConfig) ConnString() string {
	return cc.connString
}

// Conn is a Firebird connection handle. It is not safe for concurrent usage. Use a connection pool to manage access
// to multiple database connections from multiple goroutines.
type Conn struct {
	fbConn             *fbconn.FbConn
	config             *ConnConfig // config used when establishing this connection
	preparedStatements map[string]*fbconn.Statement
	preparedBatches    map[string]*PreparedBatch
	statementCache     stmtcache.Cache

	queryTracer   QueryTracer
	prepareTracer PrepareTracer

	doneChan   chan struct{}
	closedChan chan error

	typeMap *fbtype.Map

	wbuf []byte
	eqb  ExtendedQueryBuilder
}

// ServerVersion identifies the Firebird or RedDatabase server version.
type ServerVersion = fbconn.ServerVersion

// Identifier a Firebird identifier or name. Identifiers can be composed of
// multiple parts such as ["schema", "table"] or ["table", "column"].
type Identifier []string

// Sanitize returns a sanitized string safe for SQL interpolation.
func (ident Identifier) Sanitize() string {
	parts := make([]string, len(ident))
	for i := range ident {
		s := strings.ReplaceAll(ident[i], string([]byte{0}), "")
		parts[i] = `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return strings.Join(parts, ".")
}

var (
	// ErrNoRows occurs when rows are expected but none are returned.
	ErrNoRows = newProxyErr(sql.ErrNoRows, "no rows in result set")
	// ErrTooManyRows occurs when more rows than expected are returned.
	ErrTooManyRows = errors.New("too many rows in result set")
)

func newProxyErr(background error, msg string) error {
	return &proxyError{
		msg:        msg,
		background: background,
	}
}

type proxyError struct {
	msg        string
	background error
}

func (err *proxyError) Error() string { return err.msg }

func (err *proxyError) Unwrap() error { return err.background }

var (
	errDisabledStatementCache = fmt.Errorf("cannot use QueryExecModeCacheStatement with disabled statement cache")
)

// cleanupGracePeriod bounds cleanup network calls (op_free_statement, etc.) that must
// run even after the user's ctx has been cancelled.
const cleanupGracePeriod = 5 * time.Second

// cleanupContext returns a context for releasing server-side resources after the
// user's request has finished.
func cleanupContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(parent), cleanupGracePeriod)
}

// Connect establishes a connection with a Firebird server with a connection string. See
// [fbconn.Connect], [fbx.ParseConfig], [fbconn.ParseConfig] for details.
func Connect(ctx context.Context, connString string) (*Conn, error) {
	connConfig, err := ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	return connect(ctx, connConfig)
}

// CreateDatabase creates a Firebird/RedDatabase database and returns an attached connection to it.
// The connection string accepts the same options as Connect.
func CreateDatabase(ctx context.Context, connString string) (*Conn, error) {
	connConfig, err := ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	connConfig.CreateDatabase = true
	return connect(ctx, connConfig)
}

// DropDatabase connects to a Firebird/RedDatabase database, drops it, and closes the connection.
func DropDatabase(ctx context.Context, connString string) error {
	connConfig, err := ParseConfig(connString)
	if err != nil {
		return err
	}
	connConfig.CreateDatabase = false

	conn, err := connect(ctx, connConfig)
	if err != nil {
		return err
	}
	err = conn.DropDatabase(ctx)
	if !conn.IsClosed() {
		err = errors.Join(err, conn.Close(context.Background()))
	}
	return err
}

// ConnectWithOptions behaves exactly like Connect with the addition of ParseConfigOptions.
func ConnectWithOptions(ctx context.Context, connString string, options ParseConfigOptions) (*Conn, error) {
	connConfig, err := ParseConfigWithOptions(connString, options)
	if err != nil {
		return nil, err
	}
	return connect(ctx, connConfig)
}

// CreateDatabaseWithOptions behaves exactly like CreateDatabase with the addition of ParseConfigOptions.
func CreateDatabaseWithOptions(ctx context.Context, connString string, options ParseConfigOptions) (*Conn, error) {
	connConfig, err := ParseConfigWithOptions(connString, options)
	if err != nil {
		return nil, err
	}
	connConfig.CreateDatabase = true
	return connect(ctx, connConfig)
}

// DropDatabaseWithOptions behaves exactly like DropDatabase with the addition of ParseConfigOptions.
func DropDatabaseWithOptions(ctx context.Context, connString string, options ParseConfigOptions) error {
	connConfig, err := ParseConfigWithOptions(connString, options)
	if err != nil {
		return err
	}
	connConfig.CreateDatabase = false

	conn, err := connect(ctx, connConfig)
	if err != nil {
		return err
	}
	err = conn.DropDatabase(ctx)
	if !conn.IsClosed() {
		err = errors.Join(err, conn.Close(context.Background()))
	}
	return err
}

// ConnectConfig establishes a connection with a Firebird server with a configuration struct.
// connConfig must have been created by ParseConfig.
func ConnectConfig(ctx context.Context, connConfig *ConnConfig) (*Conn, error) {
	// In general this improves safety. In particular avoid the config.Config.OnNotification mutation from affecting other
	// connections with the same config.
	connConfig = connConfig.Copy()

	return connect(ctx, connConfig)
}

// CreateDatabaseConfig creates a Firebird/RedDatabase database using connConfig and returns an attached connection to it.
// connConfig must have been created by ParseConfig.
func CreateDatabaseConfig(ctx context.Context, connConfig *ConnConfig) (*Conn, error) {
	connConfig = connConfig.Copy()
	connConfig.CreateDatabase = true
	return connect(ctx, connConfig)
}

// DropDatabaseConfig connects using connConfig, drops the attached database, and closes the connection.
// connConfig must have been created by ParseConfig.
func DropDatabaseConfig(ctx context.Context, connConfig *ConnConfig) error {
	connConfig = connConfig.Copy()
	connConfig.CreateDatabase = false

	conn, err := connect(ctx, connConfig)
	if err != nil {
		return err
	}
	err = conn.DropDatabase(ctx)
	if !conn.IsClosed() {
		err = errors.Join(err, conn.Close(context.Background()))
	}
	return err
}

// ParseConfigWithOptions behaves exactly as ParseConfig does with the addition of ParseConfigOptions.
func ParseConfigWithOptions(connString string, options ParseConfigOptions) (*ConnConfig, error) {
	config, err := fbconn.ParseConfigWithOptions(connString, options.ParseConfigOptions)
	if err != nil {
		return nil, err
	}

	statementCacheCapacity := 512
	if s, ok := config.RuntimeParams["statement_cache_capacity"]; ok {
		delete(config.RuntimeParams, "statement_cache_capacity")
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fbconn.NewParseConfigError(connString, "cannot parse statement_cache_capacity", err)
		}
		statementCacheCapacity = int(n)
	}

	defaultQueryExecMode := QueryExecModeCacheStatement
	if s, ok := config.RuntimeParams["default_query_exec_mode"]; ok {
		delete(config.RuntimeParams, "default_query_exec_mode")
		switch s {
		case "cache_statement":
			defaultQueryExecMode = QueryExecModeCacheStatement
		case "exec":
			defaultQueryExecMode = QueryExecModeExec
		default:
			return nil, fbconn.NewParseConfigError(connString, "invalid default_query_exec_mode", fmt.Errorf("unknown value %q", s))
		}
	}

	batchInsertPendingSize := defaultPreparedBatchFlushSize
	if s, ok := config.RuntimeParams["batch_insert_pending_size"]; ok {
		delete(config.RuntimeParams, "batch_insert_pending_size")
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fbconn.NewParseConfigError(connString, "cannot parse batch_insert_pending_size", err)
		}
		batchInsertPendingSize = int(n)
	}

	connConfig := &ConnConfig{
		Config:                 *config,
		createdByParseConfig:   true,
		StatementCacheCapacity: statementCacheCapacity,
		DefaultQueryExecMode:   defaultQueryExecMode,
		BatchInsertPendingSize: batchInsertPendingSize,
		connString:             connString,
	}

	return connConfig, nil
}

// ParseConfig creates a ConnConfig from a connection string. ParseConfig handles all options that [fbconn.ParseConfig]
// does. In addition, it accepts the following options:
//
//   - default_query_exec_mode.
//     Possible values: "cache_statement" and "exec". See
//     QueryExecMode constant documentation for the meaning of these values. Default: "cache_statement".
//
//   - statement_cache_capacity.
//     The maximum size of the statement cache used when executing a query with "cache_statement" query exec mode.
//     Default: 512.
//
//   - batch_insert_pending_size
//     The maximum size of pending data in batch before [Conn.BatchInsert] will flush it to server.
//     Default: 128*1024.
func ParseConfig(connString string) (*ConnConfig, error) {
	return ParseConfigWithOptions(connString, ParseConfigOptions{})
}

// connect connects to a database. connect takes ownership of config. The caller must not use or access it again.
func connect(ctx context.Context, config *ConnConfig) (c *Conn, err error) {
	if connectTracer, ok := config.Tracer.(ConnectTracer); ok {
		ctx = connectTracer.TraceConnectStart(ctx, TraceConnectStartData{ConnConfig: config})
		defer func() {
			connectTracer.TraceConnectEnd(ctx, TraceConnectEndData{Conn: c, Err: err})
		}()
	}

	// Default values are set in ParseConfig. Enforce initial creation by ParseConfig rather than setting defaults from
	// zero values.
	if !config.createdByParseConfig {
		panic("config must be created by ParseConfig")
	}

	c = &Conn{
		config:      config,
		typeMap:     fbtype.NewMap(),
		queryTracer: config.Tracer,
	}

	if t, ok := c.queryTracer.(PrepareTracer); ok {
		c.prepareTracer = t
	}

	c.fbConn, err = fbconn.ConnectConfig(ctx, &config.Config)
	if err != nil {
		return nil, err
	}

	c.preparedStatements = make(map[string]*fbconn.Statement)
	c.preparedBatches = make(map[string]*PreparedBatch)
	c.doneChan = make(chan struct{})
	c.closedChan = make(chan error)
	c.wbuf = make([]byte, 0, 1024)

	if c.config.StatementCacheCapacity > 0 {
		c.statementCache = stmtcache.NewLRUCache(c.config.StatementCacheCapacity)
	}

	return c, nil
}

// Close closes a connection. If no request is in progress, it waits for the
// server to confirm detach and returns any error. An open result prevents
// confirmation and causes Close to return an error. Close always closes the
// network connection. Pass a context with a deadline to limit network waits.
// Calling Close again after the connection is closed returns nil.
func (c *Conn) Close(ctx context.Context) error {
	if c.IsClosed() {
		return nil
	}

	err := c.fbConn.Close(ctx)
	return err
}

// DropDatabase drops the currently attached database and closes the connection.
func (c *Conn) DropDatabase(ctx context.Context) error {
	if c.IsClosed() {
		return nil
	}

	return c.fbConn.DropDatabase(ctx)
}

// Prepare creates a prepared statement with name and sql. sql can contain placeholders for bound parameters. These
// placeholders are referenced positionally as ?. name can be used instead of sql with [Conn.Query], [Conn.QueryRow], and
// [Conn.Exec] to execute the statement.
// Named statements belong to the connection and can be reused in any query
// execution mode until explicitly deallocated or the connection is closed.
//
// If sql starts with "batch ", Prepare creates a named batch.
// Use [Conn.Exec] to add rows, [Conn.Flush] to send queued rows, and
// [Conn.ExecBatch] to execute the batch.
//
// The underlying Firebird identifier for the prepared statement will be name if name != sql or a digest of sql if
// name == sql.
//
// Prepare is idempotent; i.e. it is safe to call Prepare multiple times with the same name and sql arguments. This
// allows a code path to Prepare and [Conn.Query]/[Conn.Exec] without concern for if the statement has already been prepared.
func (c *Conn) Prepare(ctx context.Context, name, sql string) (sd *fbconn.StatementDescription, err error) {
	return c.prepareTx(ctx, nil, name, sql)
}

func (c *Conn) prepareTx(ctx context.Context, tx *dbTx, name string, sql string) (*fbconn.StatementDescription, error) {
	if normalizedSQL, ok := isBatchPrepareSQL(sql); ok {
		return c.prepareNamedBatchTx(ctx, tx, name, sql, normalizedSQL)
	}

	st, err := c.internalPrepareForConnection(ctx, tx, name, sql)
	if err != nil {
		return nil, err
	}
	return st.Description, nil
}

// PrepareBatch creates a connection-scoped batch for INSERT, UPDATE, or DELETE
// without result sets.
//
// The returned [PreparedBatch] uses the connection's internal transaction.
// Always call [PreparedBatch.Close] when the batch is no longer needed.
func (c *Conn) PrepareBatch(ctx context.Context, sql string) (*PreparedBatch, error) {
	return c.prepareBatchTx(ctx, nil, sql, BatchOptions{})
}

// PrepareBatchWithOptions creates a connection-scoped batch with additional
// batch options.
//
// The returned [PreparedBatch] uses the connection's internal transaction.
// Always call [PreparedBatch.Close] when the batch is no longer needed.
func (c *Conn) PrepareBatchWithOptions(ctx context.Context, sql string, opts BatchOptions) (*PreparedBatch, error) {
	return c.prepareBatchTx(ctx, nil, sql, opts)
}

func (c *Conn) prepareBatchTx(ctx context.Context, tx *dbTx, sql string, opts BatchOptions) (*PreparedBatch, error) {
	releaseInternalTx := c.shouldReleaseBatchInternalTx(tx)
	st, err := c.internalPrepare(ctx, tx, "", sql)
	if err != nil {
		return nil, errors.Join(err, c.rollbackBatchInternalTx(ctx, releaseInternalTx))
	}
	if err := validateBatchStatement(st.Description); err != nil {
		var joinedErr = err
		if deallocErr := st.Deallocate(ctx); deallocErr != nil {
			joinedErr = errors.Join(joinedErr, deallocErr)
		}
		joinedErr = errors.Join(joinedErr, c.rollbackBatchInternalTx(ctx, releaseInternalTx))
		return nil, joinedErr
	}

	pb := &PreparedBatch{
		conn:           c,
		tx:             tx,
		sql:            sql,
		sd:             st.Description,
		autoFlushBytes: defaultPreparedBatchFlushSize,
		batchCfg:       normalizeBatchOptions(opts),
	}

	if tx != nil {
		pb.st = st
		return pb, nil
	}

	if err := errors.Join(st.Deallocate(ctx), c.rollbackBatchInternalTx(ctx, releaseInternalTx)); err != nil {
		return nil, err
	}

	return pb, nil
}

// Flush sends queued rows for a named batch without executing it.
//
// Rows are queued for a named batch by calling [Conn.Exec] with the prepared
// batch name. Flush is a no-op when the named batch has no locally queued rows.
func (c *Conn) Flush(ctx context.Context, name string) error {
	return c.flushTx(ctx, nil, name)
}

// ExecBatch flushes queued rows for a named batch, executes it, and returns the result.
//
// Like [PreparedBatch.Exec], ExecBatch is atomic in autocommit mode: any
// per-row error rolls the whole batch back (no rows applied), in both
// stop-on-first and BatchOptions.ContinueOnError modes.
func (c *Conn) ExecBatch(ctx context.Context, name string) (BatchResult, error) {
	return c.execBatchTx(ctx, nil, name)
}

// CancelBatch discards every row queued for a named batch — both the local
// buffer and any rows already flushed to the server. The prepared batch
// statement is kept and can be reused via [Conn.Exec], [Conn.Flush], and
// [Conn.ExecBatch].
//
// CancelBatch does not deallocate the named batch. Use [Conn.Deallocate] when
// the named batch is no longer needed.
func (c *Conn) CancelBatch(ctx context.Context, name string) error {
	return c.cancelBatchTx(ctx, nil, name)
}

func preparedStatementNameAndKey(name, sql string) (psName, psKey string) {
	if name == sql {
		digest := sha256.Sum256([]byte(sql))
		return "stmt_" + hex.EncodeToString(digest[0:24]), sql
	}
	return name, name
}

func (c *Conn) prepareNamedBatchTx(ctx context.Context, tx *dbTx, name, originalSQL, normalizedSQL string) (*fbconn.StatementDescription, error) {
	psName, psKey := preparedStatementNameAndKey(name, originalSQL)

	if existing := c.preparedBatches[psKey]; existing != nil {
		if existing.sql != normalizedSQL {
			return nil, fmt.Errorf("prepared query '%s' already exists", name)
		}
		if err := c.ensurePreparedBatchOwner(tx, psKey, existing); err != nil {
			return nil, err
		}
		return existing.sd, nil
	}
	if _, exists := c.preparedStatements[psKey]; exists {
		return nil, fmt.Errorf("prepared query '%s' already exists", name)
	}

	pb, err := c.prepareBatchTx(ctx, tx, normalizedSQL, BatchOptions{})
	if err != nil {
		return nil, err
	}
	pb.name = psName
	pb.autoFlushBytes = defaultPreparedBatchFlushSize
	pb.sd.Name = psName
	if pb.st != nil {
		pb.st.Description.Name = psName
	}
	if psKey != "" {
		c.preparedBatches[psKey] = pb
	}
	return pb.sd, nil
}

type statementScope uint8

const (
	statementScopeTransaction statementScope = iota
	statementScopeConnection
)

func (c *Conn) internalPrepare(ctx context.Context, tx *dbTx, name, sql string) (*fbconn.Statement, error) {
	return c.internalPrepareWithScope(ctx, tx, name, sql, statementScopeTransaction)
}

func (c *Conn) internalPrepareForConnection(ctx context.Context, tx *dbTx, name, sql string) (*fbconn.Statement, error) {
	return c.internalPrepareWithScope(ctx, tx, name, sql, statementScopeConnection)
}

func (c *Conn) internalPrepareWithScope(
	ctx context.Context,
	tx *dbTx,
	name string,
	sql string,
	scope statementScope,
) (sd *fbconn.Statement, err error) {
	if c.prepareTracer != nil {
		ctx = c.prepareTracer.TracePrepareStart(ctx, c, TracePrepareStartData{Name: name, SQL: sql})
	}

	var st *fbconn.Statement
	if name != "" {
		if _, ok := c.preparedBatches[name]; ok {
			return nil, fmt.Errorf("prepared query '%s' already exists", name)
		}
		var ok bool
		if st, ok = c.preparedStatements[name]; ok {
			if sql != c.preparedStatements[name].Description.SQL {
				return nil, fmt.Errorf("prepared query '%s' already exists", name)
			}
			if c.prepareTracer != nil {
				c.prepareTracer.TracePrepareEnd(ctx, c, TracePrepareEndData{AlreadyPrepared: true})
			}
			return st, nil
		}
	}

	if c.prepareTracer != nil {
		defer func() {
			c.prepareTracer.TracePrepareEnd(ctx, c, TracePrepareEndData{Err: err})
		}()
	}

	psName, psKey := preparedStatementNameAndKey(name, sql)

	if tx != nil {
		if scope == statementScopeConnection {
			st, err = c.fbConn.PrepareTxForConnection(ctx, tx.tx, sql)
		} else {
			st, err = c.fbConn.PrepareTx(ctx, tx.tx, sql)
		}
	} else {
		st, err = c.fbConn.Prepare(ctx, sql)
	}
	if err != nil {
		return nil, err
	}

	if psKey != "" {
		c.preparedStatements[psKey] = st
	}
	st.Description.Name = psName
	return st, nil
}

// Deallocate releases a prepared statement, an automatically cached statement,
// or a named batch. Automatically cached statements can be identified by their
// SQL text.
// If name identifies both a prepared statement and cached SQL, Deallocate
// releases both statements.
// Calling Deallocate on a non-existent object will succeed.
func (c *Conn) Deallocate(ctx context.Context, name string) error {
	if pb := c.preparedBatches[name]; pb != nil {
		err := pb.Close(ctx)
		if err != nil {
			return err
		}
		delete(c.preparedBatches, name)
		return nil
	}

	preparedStatement := c.preparedStatements[name]
	var cachedStatement *fbconn.Statement
	if c.statementCache != nil {
		cachedStatement = c.statementCache.Get(name)
		if cachedStatement == nil {
			cachedStatement = c.statementCache.GetInvalidatedBySQL(name)
		}
	}

	var err error
	if preparedStatement != nil {
		err = c.deallocatePreparedStatement(ctx, preparedStatement)
	}
	if cachedStatement != nil && cachedStatement != preparedStatement {
		err = errors.Join(err, c.deallocatePreparedStatement(ctx, cachedStatement))
	}

	return err
}

// DeallocateStatement releases the prepared statement identified by sd.
// It does nothing if sd is nil or no longer belongs to the connection.
func (c *Conn) DeallocateStatement(ctx context.Context, sd *fbconn.StatementDescription) error {
	if sd == nil {
		return nil
	}

	for _, st := range c.preparedStatements {
		if st.Description == sd {
			return c.deallocatePreparedStatement(ctx, st)
		}
	}

	return nil
}

func (c *Conn) deallocatePreparedStatement(ctx context.Context, st *fbconn.Statement) error {
	if err := st.Deallocate(ctx); err != nil {
		return err
	}

	if c.statementCache != nil {
		c.statementCache.Remove(st)
	}
	removed := false
	if c.preparedStatements[st.Description.SQL] == st {
		delete(c.preparedStatements, st.Description.SQL)
		removed = true
	}
	if c.preparedStatements[st.Description.Name] == st {
		delete(c.preparedStatements, st.Description.Name)
		removed = true
	}
	if !removed {
		for name, preparedStatement := range c.preparedStatements {
			if preparedStatement == st {
				delete(c.preparedStatements, name)
			}
		}
	}

	return nil
}

// DeallocateAll releases all prepared statements and named batches.
// It resets the statement cache after all deallocations succeed. On error,
// statements that have not been released remain registered for a later retry.
func (c *Conn) DeallocateAll(ctx context.Context) error {
	for name := range c.preparedStatements {
		if err := c.Deallocate(ctx, name); err != nil {
			return err
		}
	}
	for name := range c.preparedBatches {
		if err := c.Deallocate(ctx, name); err != nil {
			return err
		}
	}
	c.preparedStatements = map[string]*fbconn.Statement{}
	c.preparedBatches = map[string]*PreparedBatch{}
	if c.config.StatementCacheCapacity > 0 {
		c.statementCache = stmtcache.NewLRUCache(c.config.StatementCacheCapacity)
	}

	return nil
}

// IsClosed reports if the connection has been closed.
func (c *Conn) IsClosed() bool {
	return c.fbConn.IsClosed()
}

func (c *Conn) die() {
	if c.IsClosed() {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // force immediate hard cancel
	_ = c.fbConn.Close(ctx)
}

func quoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// Ping delegates to the underlying *fbconn.FbConn.Ping.
func (c *Conn) Ping(ctx context.Context) error {
	return c.fbConn.Ping(ctx)
}

// WireCompressionEnabled reports whether wire compression was negotiated with the server.
func (c *Conn) WireCompressionEnabled() bool {
	return c != nil && c.fbConn != nil && c.fbConn.WireCompressionEnabled()
}

// FbConn returns the underlying *fbconn.FbConn. This is an escape hatch method that allows lower level access to the
// Firebird connection than fbx exposes.
//
// It is strongly recommended that the connection be idle (no in-progress queries) before the underlying *fbconn.FbConn
// is used and the connection must be returned to the same state before any *fbx.Conn methods are again used.
func (c *Conn) FbConn() *fbconn.FbConn {
	return c.fbConn
}

// ServerVersion returns the server version discovered while establishing the
// connection. It does not perform a request to the server.
func (c *Conn) ServerVersion() ServerVersion {
	return c.fbConn.ServerVersion()
}

// TypeMap returns the connection info used for this connection.
func (c *Conn) TypeMap() *fbtype.Map {
	return c.typeMap
}

// Config returns a copy of config that was used to establish this connection.
func (c *Conn) Config() *ConnConfig {
	return c.config.Copy()
}

// Exec executes sql inside internal transaction. sql can be either a prepared statement name or an SQL string. arguments should be referenced
// positionally from the sql string as ?.
func (c *Conn) Exec(ctx context.Context, sql string, arguments ...any) (fbconn.QueryStatus, error) {
	return c.execTx(ctx, nil, sql, arguments...)
}

// ExecTx executes sql inside created transaction. sql can be either a prepared statement name or an SQL string. arguments should be referenced
// positionally from the sql string as ?.
func (c *Conn) execTx(ctx context.Context, tx *dbTx, sql string, arguments ...any) (fbconn.QueryStatus, error) {
	if c.queryTracer != nil {
		ctx = c.queryTracer.TraceQueryStart(ctx, c, TraceQueryStartData{SQL: sql, Args: arguments})
	}

	if err := c.deallocateInvalidatedCachedStatements(ctx); err != nil {
		return fbconn.QueryStatus{}, err
	}

	commandTag, err := c.exec(ctx, tx, sql, arguments...)

	if c.queryTracer != nil {
		c.queryTracer.TraceQueryEnd(ctx, c, TraceQueryEndData{QueryStatus: commandTag, Err: err})
	}

	return commandTag, err
}

func (c *Conn) exec(ctx context.Context, tx *dbTx, sql string, arguments ...any) (commandTag fbconn.QueryStatus, err error) {
	mode := c.config.DefaultQueryExecMode
	var queryRewriter QueryRewriter
	invalidateSQL := sql

optionLoop:
	for len(arguments) > 0 {
		switch arg := arguments[0].(type) {
		case QueryExecMode:
			mode = arg
			arguments = arguments[1:]
		case QueryRewriter:
			queryRewriter = arg
			arguments = arguments[1:]
		default:
			break optionLoop
		}
	}

	if queryRewriter != nil {
		sql, arguments, err = queryRewriter.RewriteQuery(ctx, c, sql, arguments)
		if err != nil {
			return fbconn.QueryStatus{}, fmt.Errorf("rewrite query failed: %w", err)
		}
		invalidateSQL = sql
	}

	if pb := c.preparedBatches[sql]; pb != nil {
		if err := c.ensurePreparedBatchOwner(tx, sql, pb); err != nil {
			return fbconn.QueryStatus{}, err
		}
		return c.execPreparedBatch(ctx, pb, arguments)
	}

	defer func() {
		if err != nil {
			if sc := c.statementCache; sc != nil {
				sc.Invalidate(invalidateSQL)
			}
		}
	}()

	st, temporary, err := c.resolveStatement(ctx, tx, sql, mode)
	if err != nil {
		return fbconn.QueryStatus{}, err
	}
	invalidateSQL = st.Description.SQL
	if temporary {
		defer func() {
			cleanupCtx, cancel := cleanupContext(ctx)
			defer cancel()
			if cleanupErr := st.Deallocate(cleanupCtx); cleanupErr != nil {
				err = errors.Join(err, cleanupErr)
			}
		}()
	}
	return c.execPrepared(ctx, tx, st, arguments)
}

// resolveStatement gives registered names precedence over the execution mode.
// temporary is true only when the caller must deallocate the returned statement:
// Exec does so before returning, while Query transfers ownership to Rows.
func (c *Conn) resolveStatement(ctx context.Context, tx *dbTx, sql string, mode QueryExecMode) (st *fbconn.Statement, temporary bool, err error) {
	if st = c.preparedStatements[sql]; st != nil {
		return st, false, nil
	}
	switch mode {
	case QueryExecModeCacheStatement:
		st, err = c.getCachedStatement(ctx, tx, sql)
		return st, false, err
	case QueryExecModeExec:
		st, err = c.internalPrepare(ctx, tx, "", sql)
		return st, true, err
	default:
		return nil, false, fmt.Errorf("unknown QueryExecMode: %v", mode)
	}
}

// getCachedStatement returns the connection-wide cached statement for sql.
// When tx is non-nil, a statement admitted to the cache is prepared in tx for
// metadata resolution. The statement remains connection-scoped:
// [fbconn.Statement.ExecTx] uses its explicit transaction, while
// [fbconn.Statement.Exec] uses the connection's current internal transaction.
//
// If the generated cache name already belongs to an explicitly prepared
// statement, that statement is returned without transferring its ownership to
// the cache.
func (c *Conn) getCachedStatement(ctx context.Context, tx *dbTx, sql string) (*fbconn.Statement, error) {
	if c.statementCache == nil {
		return nil, errDisabledStatementCache
	}

	st := c.statementCache.Get(sql)
	if st != nil {
		return st, nil
	}

	statementName := stmtcache.StatementName(sql)
	existingStatement := c.preparedStatements[statementName]
	st, err := c.internalPrepareForConnection(ctx, tx, statementName, sql)
	if err != nil {
		return nil, err
	}
	if existingStatement != nil {
		return st, nil
	}
	c.statementCache.Put(st)
	return st, nil
}

func (c *Conn) execPrepared(ctx context.Context, tx *dbTx, st *fbconn.Statement, arguments []any) (fbconn.QueryStatus, error) {
	err := c.eqb.Build(ctx, c, tx, c.typeMap, st.Description, arguments)
	if err != nil {
		return fbconn.QueryStatus{}, err
	}

	defer c.eqb.reset() // Allow c.eqb internal memory to be GC'ed as soon as possible.

	var containsSearchPathInSql bool
	var sp1, sp2 string

	if st.Description.StatementInfo.StatementType == fbconn.StatementDDL {
		containsSearchPathInSql = containsSearchPathChangingInSql(st.Description.SQL)
		if containsSearchPathInSql {
			err = c.QueryRow(ctx, "select rdb$get_context('SYSTEM', 'SEARCH_PATH') from rdb$database", QueryExecModeExec).Scan(&sp1)
			if err != nil {
				return fbconn.QueryStatus{}, err
			}
		}
	}

	var result *fbconn.Result
	if tx != nil {
		result = st.ExecTx(ctx, tx.tx, c.eqb.ParamValues).Read()
	} else {
		result = st.Exec(ctx, c.eqb.ParamValues).Read()
	}
	if result.Err == nil && containsSearchPathInSql {
		err = c.QueryRow(ctx, "select rdb$get_context('SYSTEM', 'SEARCH_PATH') from rdb$database", QueryExecModeExec).Scan(&sp2)
		if err != nil {
			return result.QueryStatus, errors.Join(fmt.Errorf("failed to clear the statement cache after changing the search_path"), err, c.Close(ctx))
		}
		if sp1 != sp2 {
			err = c.DeallocateAll(ctx)
			if err != nil {
				return result.QueryStatus, errors.Join(fmt.Errorf("failed to clear the statement cache after changing the search_path"), err, c.Close(ctx))
			}
		}
	}
	return result.QueryStatus, result.Err
}

func (c *Conn) execPreparedBatch(ctx context.Context, pb *PreparedBatch, arguments []any) (fbconn.QueryStatus, error) {
	if err := pb.addEncoded(ctx, arguments, true); err != nil {
		return fbconn.QueryStatus{}, err
	}
	return fbconn.QueryStatus{}, nil
}

func (c *Conn) flushTx(ctx context.Context, tx *dbTx, name string) error {
	pb := c.preparedBatches[name]
	if pb == nil {
		return fmt.Errorf("prepared batch %q not found", name)
	}
	if err := c.ensurePreparedBatchOwner(tx, name, pb); err != nil {
		return err
	}
	return pb.Flush(ctx)
}

func (c *Conn) execBatchTx(ctx context.Context, tx *dbTx, name string) (BatchResult, error) {
	pb := c.preparedBatches[name]
	if pb == nil {
		return BatchResult{}, fmt.Errorf("prepared batch %q not found", name)
	}
	if err := c.ensurePreparedBatchOwner(tx, name, pb); err != nil {
		return BatchResult{}, err
	}
	return pb.Exec(ctx)
}

func (c *Conn) cancelBatchTx(ctx context.Context, tx *dbTx, name string) error {
	pb := c.preparedBatches[name]
	if pb == nil {
		return fmt.Errorf("prepared batch %q not found", name)
	}
	if err := c.ensurePreparedBatchOwner(tx, name, pb); err != nil {
		return err
	}
	return pb.Cancel(ctx)
}

func (c *Conn) ensurePreparedBatchOwner(tx *dbTx, name string, pb *PreparedBatch) error {
	if pb == nil {
		return fmt.Errorf("prepared batch %q not found", name)
	}
	if pb.tx == nil {
		if tx != nil {
			return fmt.Errorf("prepared batch %q belongs to connection scope", name)
		}
		return nil
	}
	if pb.tx.closed {
		return ErrTxClosed
	}
	if tx == nil || tx != pb.tx {
		return fmt.Errorf("prepared batch %q belongs to a different transaction", name)
	}
	return nil
}

func (c *Conn) closePreparedBatchesForTx(ctx context.Context, tx *dbTx) error {
	var joinedErr error
	for name, pb := range c.preparedBatches {
		if pb == nil || pb.tx != tx {
			continue
		}
		if err := pb.Close(ctx); err != nil {
			joinedErr = errors.Join(joinedErr, err)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			delete(c.preparedBatches, name)
			continue
		}
		delete(c.preparedBatches, name)
	}
	return joinedErr
}

func (c *Conn) getRows(ctx context.Context, sql string, args []any) *baseRows {
	r := &baseRows{}

	r.ctx = ctx
	r.queryTracer = c.queryTracer
	r.typeMap = c.typeMap
	r.startTime = time.Now()
	r.sql = sql
	r.args = args
	r.conn = c

	return r
}

type QueryExecMode int32

const (
	_ QueryExecMode = iota

	// QueryExecModeCacheStatement Automatically prepare and cache statements. This uses the extended protocol. Queries are executed in a single round
	// trip after the statement is cached. This is the default. If the database schema is modified or the search_path is
	// changed after a statement is cached then the first execution of a previously cached query may fail. e.g. If the
	// number of columns returned by a "SELECT *" changes or the type of a column is changed.
	QueryExecModeCacheStatement

	// QueryExecModeExec prepares a temporary statement for SQL that is not a
	// registered statement name. The statement is deallocated after Exec or
	// when Query's Rows are closed. Registered names are reused in either mode.
	QueryExecModeExec
)

func (m QueryExecMode) String() string {
	switch m {
	case QueryExecModeCacheStatement:
		return "cache statement"
	case QueryExecModeExec:
		return "exec"
	default:
		return "invalid"
	}
}

// QueryRewriter rewrites a query when used as the first arguments to a query method.
type QueryRewriter interface {
	RewriteQuery(ctx context.Context, conn *Conn, sql string, args []any) (newSQL string, newArgs []any, err error)
}

// Query sends a query to the server and returns a Rows to read the results. Only errors encountered sending the query
// and initializing Rows will be returned. Err() on the returned Rows must be checked after the Rows is closed to
// determine if the query executed successfully.
//
// The returned Rows must be closed before the connection can be used again. It is safe to attempt to read from the
// returned Rows even if an error is returned. The error will be the available in rows.Err() after rows are closed. It
// is allowed to ignore the error returned from Query and handle it in Rows.
//
// It is possible for a call of FieldDescriptions on the returned Rows to return nil even if the Query call did not
// return an error.
//
// It is possible for a query to return one or more rows before encountering an error. In most cases the rows should be
// collected before processing rather than processed while receiving each row. This avoids the possibility of the
// application processing rows from a query that the server rejected. The CollectRows function is useful here.
//
// An implementor of QueryRewriter may be passed as the first element of args. It can rewrite the sql and change or
// replace args. For example, NamedArgs is QueryRewriter that implements named arguments.
func (c *Conn) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	return c.queryTx(ctx, nil, sql, args...)
}
func (c *Conn) queryTx(ctx context.Context, tx *dbTx, sql string, args ...any) (Rows, error) {
	if c.queryTracer != nil {
		ctx = c.queryTracer.TraceQueryStart(ctx, c, TraceQueryStartData{SQL: sql, Args: args})
	}

	if err := c.deallocateInvalidatedCachedStatements(ctx); err != nil {
		if c.queryTracer != nil {
			c.queryTracer.TraceQueryEnd(ctx, c, TraceQueryEndData{Err: err})
		}
		return &baseRows{err: err, closed: true}, err
	}

	mode := c.config.DefaultQueryExecMode
	var queryRewriter QueryRewriter

optionLoop:
	for len(args) > 0 {
		switch arg := args[0].(type) {
		case QueryExecMode:
			mode = arg
			args = args[1:]
		case QueryRewriter:
			queryRewriter = arg
			args = args[1:]
		default:
			break optionLoop
		}
	}

	if queryRewriter != nil {
		var err error
		originalSQL := sql
		originalArgs := args
		sql, args, err = queryRewriter.RewriteQuery(ctx, c, sql, args)
		if err != nil {
			rows := c.getRows(ctx, originalSQL, originalArgs)
			err = fmt.Errorf("rewrite query failed: %w", err)
			rows.fatal(err)
			return rows, err
		}
	}

	c.eqb.reset()
	rows := c.getRows(ctx, sql, args)

	if pb := c.preparedBatches[sql]; pb != nil {
		err := fmt.Errorf("prepared batch %q cannot be queried; use Exec/Flush/ExecBatch", sql)
		rows.fatal(err)
		return rows, err
	}

	st, temporary, err := c.resolveStatement(ctx, tx, sql, mode)
	if err != nil {
		rows.fatal(err)
		return rows, err
	}
	rows.sql = st.Description.SQL
	rows.boundStatement = st
	rows.ownsStatement = temporary

	if len(st.Description.Parameters) != len(args) {
		rows.fatal(fmt.Errorf("expected %d arguments, got %d", len(st.Description.Parameters), len(args)))
		return rows, rows.err
	}

	if st.Description.StatementInfo.StatementType == fbconn.StatementDDL &&
		containsSearchPathChangingInSql(st.Description.SQL) {
		if temporary {
			err = st.Deallocate(ctx)
			if err != nil {
				err = errors.Join(fmt.Errorf("failed to clear the statement cache after changing the search_path"), err, c.Close(ctx))
			}
		}
		err = errors.Join(fmt.Errorf(`"%s" not supported by Query method`, sql), err)
		if c.queryTracer != nil {
			c.queryTracer.TraceQueryEnd(ctx, c, TraceQueryEndData{Err: err})
		}
		return &baseRows{err: err, closed: true}, err
	}

	err = c.eqb.Build(ctx, c, tx, c.typeMap, st.Description, args)
	if err != nil {
		rows.fatal(err)
		return rows, rows.err
	}

	if tx != nil {
		rows.resultReader = st.ExecTx(ctx, tx.tx, c.eqb.ParamValues)
	} else {
		rows.resultReader = st.Exec(ctx, c.eqb.ParamValues)
	}

	if tx != nil {
		rows.boundTx = tx.tx
	}

	c.eqb.reset() // Allow c.eqb internal memory to be GC'ed as soon as possible.

	return rows, rows.err
}

// QueryRow is a convenience wrapper over Query. Any error that occurs while
// querying is deferred until calling Scan on the returned Row. That Row will
// error with ErrNoRows if no rows are returned.
func (c *Conn) QueryRow(ctx context.Context, sql string, args ...any) Row {
	rows, _ := c.Query(ctx, sql, args...)
	return (*connRow)(rows.(*baseRows))
}

func (c *Conn) deallocateInvalidatedCachedStatements(ctx context.Context) error {
	var invalidatedStatements []*fbconn.Statement
	if c.statementCache != nil {
		invalidatedStatements = append(invalidatedStatements, c.statementCache.GetInvalidated()...)
	}

	if len(invalidatedStatements) == 0 {
		return nil
	}

	for _, st := range invalidatedStatements {
		if err := c.deallocatePreparedStatement(ctx, st); err != nil {
			return err
		}
	}

	c.statementCache.RemoveInvalidated()

	return nil
}

// Commit commits internal transaction
func (c *Conn) Commit(ctx context.Context) error {
	return c.fbConn.Commit(ctx)
}

// Rollback rollbacks internal transaction
func (c *Conn) Rollback(ctx context.Context) error {
	return c.fbConn.Rollback(ctx)
}
