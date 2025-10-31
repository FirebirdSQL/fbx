package pgx

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

	"github.com/red-soft/fbx/internal/stmtcache"
	"github.com/red-soft/fbx/pgconn"
	"github.com/red-soft/fbx/pgtype"
)

// ConnConfig contains all the options used to establish a connection. It must be created by ParseConfig and
// then it can be modified. A manually initialized ConnConfig will cause ConnectConfig to panic.
type ConnConfig struct {
	pgconn.Config

	Tracer QueryTracer

	// Original connection string that was parsed into config.
	connString string

	// StatementCacheCapacity is maximum size of the statement cache used when executing a query with "cache_statement"
	// query exec mode.
	StatementCacheCapacity int

	// DescriptionCacheCapacity is the maximum size of the description cache used when executing a query with
	// "cache_describe" query exec mode.
	DescriptionCacheCapacity int

	// DefaultQueryExecMode controls the default mode for executing queries. By default pgx uses the extended protocol
	// and automatically prepares and caches prepared statements. However, this may be incompatible with proxies such as
	// PGBouncer. In this case it may be preferable to use QueryExecModeExec or QueryExecModeSimpleProtocol. The same
	// functionality can be controlled on a per query basis by passing a QueryExecMode as the first query argument.
	DefaultQueryExecMode QueryExecMode

	createdByParseConfig bool // Used to enforce created by ParseConfig rule.
	autocommit           bool
}

// ParseConfigOptions contains options that control how a config is built such as getsslpassword.
type ParseConfigOptions struct {
	pgconn.ParseConfigOptions
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

// ConnString returns the connection string as parsed by pgx.ParseConfig into pgx.ConnConfig.
func (cc *ConnConfig) ConnString() string {
	return cc.connString
}

// Conn is a PostgreSQL connection handle. It is not safe for concurrent usage. Use a connection pool to manage access
// to multiple database connections from multiple goroutines.
type Conn struct {
	pgConn             *pgconn.PgConn
	config             *ConnConfig // config used when establishing this connection
	preparedStatements map[string]*pgconn.Statement
	statementCache     stmtcache.Cache

	queryTracer    QueryTracer
	batchTracer    BatchTracer
	copyFromTracer CopyFromTracer
	prepareTracer  PrepareTracer

	notifications []*pgconn.Notification

	doneChan   chan struct{}
	closedChan chan error

	typeMap *pgtype.Map

	wbuf []byte
	eqb  ExtendedQueryBuilder

	autocommit bool
}

// Identifier a PostgreSQL identifier or name. Identifiers can be composed of
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

// Connect establishes a connection with a PostgreSQL server with a connection string. See
// pgconn.Connect for details.
func Connect(ctx context.Context, connString string) (*Conn, error) {
	connConfig, err := ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	return connect(ctx, connConfig)
}

// ConnectWithOptions behaves exactly like Connect with the addition of options. At the present options is only used to
// provide a GetSSLPassword function.
func ConnectWithOptions(ctx context.Context, connString string, options ParseConfigOptions) (*Conn, error) {
	connConfig, err := ParseConfigWithOptions(connString, options)
	if err != nil {
		return nil, err
	}
	return connect(ctx, connConfig)
}

// ConnectConfig establishes a connection with a PostgreSQL server with a configuration struct.
// connConfig must have been created by ParseConfig.
func ConnectConfig(ctx context.Context, connConfig *ConnConfig) (*Conn, error) {
	// In general this improves safety. In particular avoid the config.Config.OnNotification mutation from affecting other
	// connections with the same config. See https://github.com/jackc/pgx/issues/618.
	connConfig = connConfig.Copy()

	return connect(ctx, connConfig)
}

// ParseConfigWithOptions behaves exactly as ParseConfig does with the addition of options. At the present options is
// only used to provide a GetSSLPassword function.
func ParseConfigWithOptions(connString string, options ParseConfigOptions) (*ConnConfig, error) {
	config, err := pgconn.ParseConfigWithOptions(connString, options.ParseConfigOptions)
	if err != nil {
		return nil, err
	}

	statementCacheCapacity := 512
	if s, ok := config.RuntimeParams["statement_cache_capacity"]; ok {
		delete(config.RuntimeParams, "statement_cache_capacity")
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("cannot parse statement_cache_capacity: %w", err)
		}
		statementCacheCapacity = int(n)
	}

	descriptionCacheCapacity := 512
	if s, ok := config.RuntimeParams["description_cache_capacity"]; ok {
		delete(config.RuntimeParams, "description_cache_capacity")
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("cannot parse description_cache_capacity: %w", err)
		}
		descriptionCacheCapacity = int(n)
	}

	defaultQueryExecMode := QueryExecModeCacheStatement
	autocommit := true
	if s, ok := config.RuntimeParams["default_query_exec_mode"]; ok {
		delete(config.RuntimeParams, "default_query_exec_mode")
		switch s {
		case "cache_statement":
			defaultQueryExecMode = QueryExecModeCacheStatement
		case "exec":
			defaultQueryExecMode = QueryExecModeExec
		default:
			return nil, fmt.Errorf("invalid default_query_exec_mode: %s", s)
		}
	}

	if s, ok := config.RuntimeParams["noautocommit"]; ok {
		delete(config.RuntimeParams, "noautocommit")
		autocommit = true
		if res, err := strconv.ParseBool(s); err == nil {
			autocommit = !res
		}
	}

	connConfig := &ConnConfig{
		Config:                   *config,
		createdByParseConfig:     true,
		StatementCacheCapacity:   statementCacheCapacity,
		DescriptionCacheCapacity: descriptionCacheCapacity,
		DefaultQueryExecMode:     defaultQueryExecMode,
		connString:               connString,
		autocommit:               autocommit,
	}

	return connConfig, nil
}

// ParseConfig creates a ConnConfig from a connection string. ParseConfig handles all options that [pgconn.ParseConfig]
// does. In addition, it accepts the following options:
//
//   - default_query_exec_mode.
//     Possible values: "cache_statement", "cache_describe", "describe_exec", "exec", and "simple_protocol". See
//     QueryExecMode constant documentation for the meaning of these values. Default: "cache_statement".
//
//   - statement_cache_capacity.
//     The maximum size of the statement cache used when executing a query with "cache_statement" query exec mode.
//     Default: 512.
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
		typeMap:     pgtype.NewMap(),
		queryTracer: config.Tracer,
	}

	if t, ok := c.queryTracer.(BatchTracer); ok {
		c.batchTracer = t
	}
	if t, ok := c.queryTracer.(CopyFromTracer); ok {
		c.copyFromTracer = t
	}
	if t, ok := c.queryTracer.(PrepareTracer); ok {
		c.prepareTracer = t
	}

	// Only install pgx notification system if no other callback handler is present.
	if config.Config.OnNotification == nil {
		config.Config.OnNotification = c.bufferNotifications
	}

	c.pgConn, err = pgconn.ConnectConfig(ctx, &config.Config)
	if err != nil {
		return nil, err
	}

	c.preparedStatements = make(map[string]*pgconn.Statement)
	c.doneChan = make(chan struct{})
	c.closedChan = make(chan error)
	c.wbuf = make([]byte, 0, 1024)

	if c.config.StatementCacheCapacity > 0 {
		c.statementCache = stmtcache.NewLRUCache(c.config.StatementCacheCapacity)
	}
	c.autocommit = c.config.autocommit

	return c, nil
}

// Close closes a connection. It is safe to call Close on an already closed
// connection.
func (c *Conn) Close(ctx context.Context) error {
	if c.IsClosed() {
		return nil
	}

	err := c.pgConn.Close(ctx)
	return err
}

// Prepare creates a prepared statement with name and sql. sql can contain placeholders for bound parameters. These
// placeholders are referenced positionally as $1, $2, etc. name can be used instead of sql with Query, QueryRow, and
// Exec to execute the statement. It can also be used with Batch.Queue.
//
// The underlying PostgreSQL identifier for the prepared statement will be name if name != sql or a digest of sql if
// name == sql.
//
// Prepare is idempotent; i.e. it is safe to call Prepare multiple times with the same name and sql arguments. This
// allows a code path to Prepare and Query/Exec without concern for if the statement has already been prepared.
func (c *Conn) Prepare(ctx context.Context, name, sql string) (sd *pgconn.StatementDescription, err error) {
	return c.prepareTx(ctx, nil, name, sql)
}

func (c *Conn) prepareTx(ctx context.Context, tx *dbTx, name string, sql string) (*pgconn.StatementDescription, error) {
	st, err := c.internalPrepare(ctx, tx, name, sql)
	if err != nil {
		return nil, err
	}
	return st.Description, nil
}

func (c *Conn) internalPrepare(ctx context.Context, tx *dbTx, name, sql string) (sd *pgconn.Statement, err error) {
	if c.prepareTracer != nil {
		ctx = c.prepareTracer.TracePrepareStart(ctx, c, TracePrepareStartData{Name: name, SQL: sql})
	}

	var st *pgconn.Statement
	if name != "" {
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

	var psName, psKey string
	if name == sql {
		digest := sha256.Sum256([]byte(sql))
		psName = "stmt_" + hex.EncodeToString(digest[0:24])
		psKey = sql
	} else {
		psName = name
		psKey = name
	}

	if tx != nil {
		st, err = c.pgConn.PrepareTx(ctx, tx.tx, sql)
	} else {
		st, err = c.pgConn.Prepare(ctx, sql)
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

// Deallocate releases a prepared statement. Calling Deallocate on a non-existent prepared statement will succeed.
func (c *Conn) Deallocate(ctx context.Context, name string) error {
	st := c.preparedStatements[name]

	if st != nil {
		err := st.Deallocate(ctx)

		if err != nil {
			return err
		}

		delete(c.preparedStatements, name)
	}

	return nil
}

// DeallocateAll releases all previously prepared statements from the server and client, where it also resets the statement and description cache.
func (c *Conn) DeallocateAll(ctx context.Context) error {
	c.preparedStatements = map[string]*pgconn.Statement{}
	if c.config.StatementCacheCapacity > 0 {
		c.statementCache = stmtcache.NewLRUCache(c.config.StatementCacheCapacity)
	}

	for name, _ := range c.preparedStatements {
		if err := c.Deallocate(ctx, name); err != nil {
			return err
		}
	}

	return nil
}

func (c *Conn) bufferNotifications(_ *pgconn.PgConn, n *pgconn.Notification) {
	c.notifications = append(c.notifications, n)
}

// WaitForNotification waits for a PostgreSQL notification. It wraps the underlying pgconn notification system in a
// slightly more convenient form.
func (c *Conn) WaitForNotification(ctx context.Context) (*pgconn.Notification, error) {
	var n *pgconn.Notification

	// Return already received notification immediately
	if len(c.notifications) > 0 {
		n = c.notifications[0]
		c.notifications = c.notifications[1:]
		return n, nil
	}

	err := c.pgConn.WaitForNotification(ctx)
	if len(c.notifications) > 0 {
		n = c.notifications[0]
		c.notifications = c.notifications[1:]
	}
	return n, err
}

// IsClosed reports if the connection has been closed.
func (c *Conn) IsClosed() bool {
	return c.pgConn.IsClosed()
}

func (c *Conn) die() {
	if c.IsClosed() {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // force immediate hard cancel
	c.pgConn.Close(ctx)
}

func quoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// Ping delegates to the underlying *pgconn.PgConn.Ping.
func (c *Conn) Ping(ctx context.Context) error {
	return c.pgConn.Ping(ctx)
}

// PgConn returns the underlying *pgconn.PgConn. This is an escape hatch method that allows lower level access to the
// PostgreSQL connection than pgx exposes.
//
// It is strongly recommended that the connection be idle (no in-progress queries) before the underlying *pgconn.PgConn
// is used and the connection must be returned to the same state before any *pgx.Conn methods are again used.
func (c *Conn) PgConn() *pgconn.PgConn {
	return c.pgConn
}

// TypeMap returns the connection info used for this connection.
func (c *Conn) TypeMap() *pgtype.Map {
	return c.typeMap
}

// Config returns a copy of config that was used to establish this connection.
func (c *Conn) Config() *ConnConfig {
	return c.config.Copy()
}

// Exec executes sql inside internal transaction. sql can be either a prepared statement name or an SQL string. arguments should be referenced
// positionally from the sql string as $1, $2, etc.
func (c *Conn) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.QueryStatus, error) {
	return c.execTx(ctx, nil, sql, arguments...)
}

// ExecTx executes sql inside created transaction. sql can be either a prepared statement name or an SQL string. arguments should be referenced
// positionally from the sql string as $1, $2, etc.
func (c *Conn) execTx(ctx context.Context, tx *dbTx, sql string, arguments ...any) (pgconn.QueryStatus, error) {
	if c.queryTracer != nil {
		ctx = c.queryTracer.TraceQueryStart(ctx, c, TraceQueryStartData{SQL: sql, Args: arguments})
	}

	if err := c.deallocateInvalidatedCachedStatements(ctx); err != nil {
		return pgconn.QueryStatus{}, err
	}

	commandTag, err := c.exec(ctx, tx, sql, arguments...)

	if c.queryTracer != nil {
		c.queryTracer.TraceQueryEnd(ctx, c, TraceQueryEndData{CommandTag: commandTag, Err: err})
	}

	if err == nil && tx == nil && c.autocommit {
		err = c.pgConn.Commit(ctx)
	}

	return commandTag, err
}

func (c *Conn) exec(ctx context.Context, tx *dbTx, sql string, arguments ...any) (commandTag pgconn.QueryStatus, err error) {
	mode := c.config.DefaultQueryExecMode
	var queryRewriter QueryRewriter

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
			return pgconn.QueryStatus{}, fmt.Errorf("rewrite query failed: %w", err)
		}
	}

	if sd, ok := c.preparedStatements[sql]; ok {
		return c.execPrepared(ctx, tx, sd, arguments)
	}

	switch mode {
	case QueryExecModeCacheStatement:
		if c.statementCache == nil {
			return pgconn.QueryStatus{}, errDisabledStatementCache
		}
		st := c.statementCache.Get(sql)
		if st == nil {
			st, err = c.internalPrepare(ctx, tx, stmtcache.StatementName(sql), sql)
			if err != nil {
				return pgconn.QueryStatus{}, err
			}
			c.statementCache.Put(st)
		}

		return c.execPrepared(ctx, tx, st, arguments)
	case QueryExecModeExec:
		st, err := c.internalPrepare(ctx, tx, stmtcache.StatementName(sql), sql)
		if err != nil {
			return pgconn.QueryStatus{}, err
		}
		defer st.Deallocate(ctx)
		return c.execPrepared(ctx, tx, st, arguments)
	default:
		return pgconn.QueryStatus{}, fmt.Errorf("unknown QueryExecMode: %v", mode)
	}
}

func (c *Conn) execParams(ctx context.Context, tx *dbTx, sd *pgconn.StatementDescription, arguments []any) (pgconn.QueryStatus, error) {
	err := c.eqb.Build(c.typeMap, sd, arguments)
	if err != nil {
		return pgconn.QueryStatus{}, err
	}

	var result *pgconn.Result
	if tx != nil {
		result = c.pgConn.ExecParamsTx(ctx, tx.tx, sd.SQL, c.eqb.ParamValues).Read()
	} else {
		result = c.pgConn.ExecParams(ctx, sd.SQL, c.eqb.ParamValues).Read()
	}
	c.eqb.reset() // Allow c.eqb internal memory to be GC'ed as soon as possible.
	return result.QueryStatus, result.Err
}

func (c *Conn) execPrepared(ctx context.Context, tx *dbTx, st *pgconn.Statement, arguments []any) (pgconn.QueryStatus, error) {
	err := c.eqb.Build(c.typeMap, st.Description, arguments)
	if err != nil {
		return pgconn.QueryStatus{}, err
	}

	var result *pgconn.Result
	if tx != nil {
		result = st.ExecTx(ctx, tx.tx, c.eqb.ParamValues).Read()
	} else {
		result = st.Exec(ctx, c.eqb.ParamValues).Read()
	}
	c.eqb.reset() // Allow c.eqb internal memory to be GC'ed as soon as possible.
	return result.QueryStatus, result.Err
}

func (c *Conn) execSQLParams(ctx context.Context, tx *dbTx, sql string, args []any) (pgconn.QueryStatus, error) {
	err := c.eqb.Build(c.typeMap, nil, args)
	if err != nil {
		return pgconn.QueryStatus{}, err
	}

	var result *pgconn.Result
	if tx != nil {
		result = c.pgConn.ExecParamsTx(ctx, tx.tx, sql, c.eqb.ParamValues).Read()
	} else {
		result = c.pgConn.ExecParams(ctx, sql, c.eqb.ParamValues).Read()
	}
	c.eqb.reset() // Allow c.eqb internal memory to be GC'ed as soon as possible.
	return result.QueryStatus, result.Err
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

	// Automatically prepare and cache statements. This uses the extended protocol. Queries are executed in a single round
	// trip after the statement is cached. This is the default. If the database schema is modified or the search_path is
	// changed after a statement is cached then the first execution of a previously cached query may fail. e.g. If the
	// number of columns returned by a "SELECT *" changes or the type of a column is changed.
	QueryExecModeCacheStatement

	// Assume the PostgreSQL query parameter types based on the Go type of the arguments. This uses the extended protocol
	// with text formatted parameters and results. Queries are executed in a single round trip. Type mappings can be
	// registered with pgtype.Map.RegisterDefaultPgType. Queries will be rejected that have arguments that are
	// unregistered or ambiguous. e.g. A map[string]string may have the PostgreSQL type json or hstore. Modes that know
	// the PostgreSQL type can use a map[string]string directly as an argument. This mode cannot.
	//
	// On rare occasions user defined types may behave differently when encoded in the text format instead of the binary
	// format. For example, this could happen if a "type RomanNumeral int32" implements fmt.Stringer to format integers as
	// Roman numerals (e.g. 7 is VII). The binary format would properly encode the integer 7 as the binary value for 7.
	// But the text format would encode the integer 7 as the string "VII". As QueryExecModeExec uses the text format, it
	// is possible that changing query mode from another mode to QueryExecModeExec could change the behavior of the query.
	// This should not occur with types pgx supports directly and can be avoided by registering the types with
	// pgtype.Map.RegisterDefaultPgType and implementing the appropriate type interfaces. In the cas of RomanNumeral, it
	// should implement pgtype.Int64Valuer.
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

// QueryResultFormats controls the result format (text=0, binary=1) of a query by result column position.
type QueryResultFormats []int16

// QueryResultFormatsByOID controls the result format (text=0, binary=1) of a query by the result column OID.
type QueryResultFormatsByOID map[uint32]int16

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
//
// For extra control over how the query is executed, the types QueryExecMode, QueryResultFormats, and
// QueryResultFormatsByOID may be used as the first args to control exactly how the query is executed. This is rarely
// needed. See the documentation for those types for details.
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
		case QueryResultFormats:
			args = args[1:]
		case QueryResultFormatsByOID:
			args = args[1:]
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

	var err error
	st, _ := c.preparedStatements[sql]
	if st != nil || mode == QueryExecModeCacheStatement {
		if st == nil {
			st, err = c.internalPrepare(ctx, tx, sql, sql)
			if err != nil {
				rows.fatal(err)
				return rows, err
			}
		}

		if len(st.Description.Parameters) != len(args) {
			rows.fatal(fmt.Errorf("expected %d arguments, got %d", len(st.Description.Parameters), len(args)))
			return rows, rows.err
		}

		rows.sql = st.Description.SQL
		rows.boundStatement = st

		err = c.eqb.Build(c.typeMap, st.Description, args)
		if err != nil {
			rows.fatal(err)
			return rows, rows.err
		}

		if tx != nil {
			rows.resultReader = st.ExecTx(ctx, tx.tx, c.eqb.ParamValues)
		} else {
			rows.resultReader = st.Exec(ctx, c.eqb.ParamValues)
		}
	} else if mode == QueryExecModeExec {
		st, err = c.internalPrepare(ctx, tx, sql, sql)
		if err != nil {
			rows.fatal(err)
			return rows, err
		}

		err = c.eqb.Build(c.typeMap, st.Description, args)
		if err != nil {
			st.Deallocate(ctx)
			rows.fatal(err)
			return rows, rows.err
		}

		rows.boundStatement = st
		if tx != nil {
			rows.resultReader = st.ExecTx(ctx, tx.tx, c.eqb.ParamValues)
		} else {
			rows.resultReader = st.Exec(ctx, c.eqb.ParamValues)
		}
	} else {
		err = fmt.Errorf("unknown QueryExecMode: %v", mode)
		rows.fatal(err)
		return rows, rows.err
	}

	c.eqb.reset() // Allow c.eqb internal memory to be GC'ed as soon as possible.

	return rows, rows.err
}

// getStatementDescription returns the statement description of the sql query
// according to the given mode.
//
// If the mode is one that doesn't require to know the param and result OIDs
// then nil is returned without error.
func (c *Conn) getStatementDescription(
	ctx context.Context,
	mode QueryExecMode,
	sql string,
) (sd *pgconn.StatementDescription, err error) {
	//switch mode {
	//case QueryExecModeCacheStatement:
	//	if c.statementCache == nil {
	//		return nil, errDisabledStatementCache
	//	}
	//	sd = c.statementCache.Get(sql)
	//	if sd == nil {
	//		sd, err = c.Prepare(ctx, stmtcache.StatementName(sql), sql)
	//		if err != nil {
	//			return nil, err
	//		}
	//		c.statementCache.Put(sd)
	//	}
	//case QueryExecModeCacheDescribe:
	//	if c.descriptionCache == nil {
	//		return nil, errDisabledDescriptionCache
	//	}
	//	sd = c.descriptionCache.Get(sql)
	//	if sd == nil {
	//		sd, err = c.Prepare(ctx, "", sql)
	//		if err != nil {
	//			return nil, err
	//		}
	//		c.descriptionCache.Put(sd)
	//	}
	//case QueryExecModeDescribeExec:
	//	return c.Prepare(ctx, "", sql)
	//}
	//return sd, err
	panic("implement me")
}

// QueryRow is a convenience wrapper over Query. Any error that occurs while
// querying is deferred until calling Scan on the returned Row. That Row will
// error with ErrNoRows if no rows are returned.
func (c *Conn) QueryRow(ctx context.Context, sql string, args ...any) Row {
	rows, _ := c.Query(ctx, sql, args...)
	return (*connRow)(rows.(*baseRows))
	//panic("implement me")
}

// SendBatch sends all queued queries to the server at once. All queries are run in an implicit transaction unless
// explicit transaction control statements are executed. The returned BatchResults must be closed before the connection
// is used again.
//
// Depending on the QueryExecMode, all queries may be prepared before any are executed. This means that creating a table
// and using it in a subsequent query in the same batch can fail.
func (c *Conn) SendBatch(ctx context.Context, b *Batch) (br BatchResults) {
	//if c.batchTracer != nil {
	//	ctx = c.batchTracer.TraceBatchStart(ctx, c, TraceBatchStartData{Batch: b})
	//	defer func() {
	//		err := br.(interface{ earlyError() error }).earlyError()
	//		if err != nil {
	//			c.batchTracer.TraceBatchEnd(ctx, c, TraceBatchEndData{Err: err})
	//		}
	//	}()
	//}
	//
	//if err := c.deallocateInvalidatedCachedStatements(ctx); err != nil {
	//	return &batchResults{ctx: ctx, conn: c, err: err}
	//}
	//
	//for _, bi := range b.QueuedQueries {
	//	var queryRewriter QueryRewriter
	//	sql := bi.SQL
	//	arguments := bi.Arguments
	//
	//optionLoop:
	//	for len(arguments) > 0 {
	//		// Update Batch.Queue function comment when additional options are implemented
	//		switch arg := arguments[0].(type) {
	//		case QueryRewriter:
	//			queryRewriter = arg
	//			arguments = arguments[1:]
	//		default:
	//			break optionLoop
	//		}
	//	}
	//
	//	if queryRewriter != nil {
	//		var err error
	//		sql, arguments, err = queryRewriter.RewriteQuery(ctx, c, sql, arguments)
	//		if err != nil {
	//			return &batchResults{ctx: ctx, conn: c, err: fmt.Errorf("rewrite query failed: %w", err)}
	//		}
	//	}
	//
	//	bi.SQL = sql
	//	bi.Arguments = arguments
	//}
	//
	//// TODO: changing mode per batch? Update Batch.Queue function comment when implemented
	//mode := c.config.DefaultQueryExecMode
	//if mode == QueryExecModeSimpleProtocol {
	//	return c.sendBatchQueryExecModeSimpleProtocol(ctx, b)
	//}
	//
	//// All other modes use extended protocol and thus can use prepared statements.
	//for _, bi := range b.QueuedQueries {
	//	if sd, ok := c.preparedStatements[bi.SQL]; ok {
	//		bi.sd = sd
	//	}
	//}
	//
	//switch mode {
	//case QueryExecModeExec:
	//	return c.sendBatchQueryExecModeExec(ctx, b)
	//case QueryExecModeCacheStatement:
	//	return c.sendBatchQueryExecModeCacheStatement(ctx, b)
	//case QueryExecModeCacheDescribe:
	//	return c.sendBatchQueryExecModeCacheDescribe(ctx, b)
	//case QueryExecModeDescribeExec:
	//	return c.sendBatchQueryExecModeDescribeExec(ctx, b)
	//default:
	//	panic("unknown QueryExecMode")
	//}
	panic("implement me")
}

func (c *Conn) sendBatchQueryExecModeSimpleProtocol(ctx context.Context, b *Batch) *batchResults {
	//var sb strings.Builder
	//for i, bi := range b.QueuedQueries {
	//	if i > 0 {
	//		sb.WriteByte(';')
	//	}
	//	sql, err := c.sanitizeForSimpleQuery(bi.SQL, bi.Arguments...)
	//	if err != nil {
	//		return &batchResults{ctx: ctx, conn: c, err: err}
	//	}
	//	sb.WriteString(sql)
	//}
	//mrr := c.pgConn.Exec(ctx, sb.String())
	//return &batchResults{
	//	ctx:   ctx,
	//	conn:  c,
	//	mrr:   mrr,
	//	b:     b,
	//	qqIdx: 0,
	//}
	panic("implement me")
}

func (c *Conn) sendBatchQueryExecModeExec(ctx context.Context, b *Batch) *batchResults {
	//batch := &pgconn.Batch{}
	//
	//for _, bi := range b.QueuedQueries {
	//	sd := bi.sd
	//	if sd != nil {
	//		err := c.eqb.Build(c.typeMap, sd, bi.Arguments)
	//		if err != nil {
	//			return &batchResults{ctx: ctx, conn: c, err: err}
	//		}
	//
	//		batch.ExecPrepared(sd.Name, c.eqb.ParamValues, c.eqb.ParamFormats, c.eqb.ResultFormats)
	//	} else {
	//		err := c.eqb.Build(c.typeMap, nil, bi.Arguments)
	//		if err != nil {
	//			return &batchResults{ctx: ctx, conn: c, err: err}
	//		}
	//		batch.ExecParams(bi.SQL, c.eqb.ParamValues, nil, c.eqb.ParamFormats, c.eqb.ResultFormats)
	//	}
	//}
	//
	//c.eqb.reset() // Allow c.eqb internal memory to be GC'ed as soon as possible.
	//
	//mrr := c.pgConn.ExecBatch(ctx, batch)
	//
	//return &batchResults{
	//	ctx:   ctx,
	//	conn:  c,
	//	mrr:   mrr,
	//	b:     b,
	//	qqIdx: 0,
	//}
	panic("implement me")
}

func (c *Conn) sendBatchQueryExecModeCacheStatement(ctx context.Context, b *Batch) (pbr *pipelineBatchResults) {
	//if c.statementCache == nil {
	//	return &pipelineBatchResults{ctx: ctx, conn: c, err: errDisabledStatementCache, closed: true}
	//}
	//
	//distinctNewQueries := []*pgconn.StatementDescription{}
	//distinctNewQueriesIdxMap := make(map[string]int)
	//
	//for _, bi := range b.QueuedQueries {
	//	if bi.sd == nil {
	//		sd := c.statementCache.Get(bi.SQL)
	//		if sd != nil {
	//			bi.sd = sd
	//		} else {
	//			if idx, present := distinctNewQueriesIdxMap[bi.SQL]; present {
	//				bi.sd = distinctNewQueries[idx]
	//			} else {
	//				sd = &pgconn.StatementDescription{
	//					Name: stmtcache.StatementName(bi.SQL),
	//					SQL:  bi.SQL,
	//				}
	//				distinctNewQueriesIdxMap[sd.SQL] = len(distinctNewQueries)
	//				distinctNewQueries = append(distinctNewQueries, sd)
	//				bi.sd = sd
	//			}
	//		}
	//	}
	//}
	//
	//return c.sendBatchExtendedWithDescription(ctx, b, distinctNewQueries, c.statementCache)
	panic("implement me")
}

func (c *Conn) sendBatchQueryExecModeCacheDescribe(ctx context.Context, b *Batch) (pbr *pipelineBatchResults) {
	//if c.descriptionCache == nil {
	//	return &pipelineBatchResults{ctx: ctx, conn: c, err: errDisabledDescriptionCache, closed: true}
	//}
	//
	//distinctNewQueries := []*pgconn.StatementDescription{}
	//distinctNewQueriesIdxMap := make(map[string]int)
	//
	//for _, bi := range b.QueuedQueries {
	//	if bi.sd == nil {
	//		sd := c.descriptionCache.Get(bi.SQL)
	//		if sd != nil {
	//			bi.sd = sd
	//		} else {
	//			if idx, present := distinctNewQueriesIdxMap[bi.SQL]; present {
	//				bi.sd = distinctNewQueries[idx]
	//			} else {
	//				sd = &pgconn.StatementDescription{
	//					SQL: bi.SQL,
	//				}
	//				distinctNewQueriesIdxMap[sd.SQL] = len(distinctNewQueries)
	//				distinctNewQueries = append(distinctNewQueries, sd)
	//				bi.sd = sd
	//			}
	//		}
	//	}
	//}
	//
	//return c.sendBatchExtendedWithDescription(ctx, b, distinctNewQueries, c.descriptionCache)
	panic("implement me")
}

func (c *Conn) sendBatchQueryExecModeDescribeExec(ctx context.Context, b *Batch) (pbr *pipelineBatchResults) {
	//distinctNewQueries := []*pgconn.StatementDescription{}
	//distinctNewQueriesIdxMap := make(map[string]int)
	//
	//for _, bi := range b.QueuedQueries {
	//	if bi.sd == nil {
	//		if idx, present := distinctNewQueriesIdxMap[bi.SQL]; present {
	//			bi.sd = distinctNewQueries[idx]
	//		} else {
	//			sd := &pgconn.StatementDescription{
	//				SQL: bi.SQL,
	//			}
	//			distinctNewQueriesIdxMap[sd.SQL] = len(distinctNewQueries)
	//			distinctNewQueries = append(distinctNewQueries, sd)
	//			bi.sd = sd
	//		}
	//	}
	//}
	//
	//return c.sendBatchExtendedWithDescription(ctx, b, distinctNewQueries, nil)
	panic("implement me")
}

func (c *Conn) sendBatchExtendedWithDescription(ctx context.Context, b *Batch, distinctNewQueries []*pgconn.StatementDescription, sdCache stmtcache.Cache) (pbr *pipelineBatchResults) {
	//pipeline := c.pgConn.StartPipeline(ctx)
	//defer func() {
	//	if pbr != nil && pbr.err != nil {
	//		pipeline.Close()
	//	}
	//}()
	//
	//// Prepare any needed queries
	//if len(distinctNewQueries) > 0 {
	//	err := func() (err error) {
	//		for _, sd := range distinctNewQueries {
	//			pipeline.SendPrepare(sd.Name, sd.SQL, nil)
	//		}
	//
	//		// Store all statements we are preparing into the cache. It's fine if it overflows because HandleInvalidated will
	//		// clean them up later.
	//		if sdCache != nil {
	//			for _, sd := range distinctNewQueries {
	//				sdCache.Put(sd)
	//			}
	//		}
	//
	//		// If something goes wrong preparing the statements, we need to invalidate the cache entries we just added.
	//		defer func() {
	//			if err != nil && sdCache != nil {
	//				for _, sd := range distinctNewQueries {
	//					sdCache.Invalidate(sd.SQL)
	//				}
	//			}
	//		}()
	//
	//		err = pipeline.Sync()
	//		if err != nil {
	//			return err
	//		}
	//
	//		for _, sd := range distinctNewQueries {
	//			results, err := pipeline.GetResults()
	//			if err != nil {
	//				return err
	//			}
	//
	//			resultSD, ok := results.(*pgconn.StatementDescription)
	//			if !ok {
	//				return fmt.Errorf("expected statement description, got %T", results)
	//			}
	//
	//			// Fill in the previously empty / pending statement descriptions.
	//			sd.ParamOIDs = resultSD.ParamOIDs
	//			sd.Fields = resultSD.Fields
	//		}
	//
	//		results, err := pipeline.GetResults()
	//		if err != nil {
	//			return err
	//		}
	//
	//		_, ok := results.(*pgconn.PipelineSync)
	//		if !ok {
	//			return fmt.Errorf("expected sync, got %T", results)
	//		}
	//
	//		return nil
	//	}()
	//	if err != nil {
	//		return &pipelineBatchResults{ctx: ctx, conn: c, err: err, closed: true}
	//	}
	//}
	//
	//// Queue the queries.
	//for _, bi := range b.QueuedQueries {
	//	err := c.eqb.Build(c.typeMap, bi.sd, bi.Arguments)
	//	if err != nil {
	//		// we wrap the error so we the user can understand which query failed inside the batch
	//		err = fmt.Errorf("error building query %s: %w", bi.SQL, err)
	//		return &pipelineBatchResults{ctx: ctx, conn: c, err: err, closed: true}
	//	}
	//
	//	if bi.sd.Name == "" {
	//		pipeline.SendQueryParams(bi.sd.SQL, c.eqb.ParamValues, bi.sd.ParamOIDs, c.eqb.ParamFormats, c.eqb.ResultFormats)
	//	} else {
	//		pipeline.SendQueryPrepared(bi.sd.Name, c.eqb.ParamValues, c.eqb.ParamFormats, c.eqb.ResultFormats)
	//	}
	//}
	//
	//err := pipeline.Sync()
	//if err != nil {
	//	return &pipelineBatchResults{ctx: ctx, conn: c, err: err, closed: true}
	//}
	//
	//return &pipelineBatchResults{
	//	ctx:      ctx,
	//	conn:     c,
	//	pipeline: pipeline,
	//	b:        b,
	//}
	panic("implement me")
}

// LoadType inspects the database for typeName and produces a pgtype.Type suitable for registration. typeName must be
// the name of a type where the underlying type(s) is already understood by pgx. It is for derived types. In particular,
// typeName must be one of the following:
//   - An array type name of a type that is already registered. e.g. "_foo" when "foo" is registered.
//   - A composite type name where all field types are already registered.
//   - A domain type name where the base type is already registered.
//   - An enum type name.
//   - A range type name where the element type is already registered.
//   - A multirange type name where the element type is already registered.
func (c *Conn) LoadType(ctx context.Context, typeName string) (*pgtype.Type, error) {
	//var oid uint32
	//
	//err := c.QueryRow(ctx, "select $1::text::regtype::oid;", typeName).Scan(&oid)
	//if err != nil {
	//	return nil, err
	//}
	//
	//var typtype string
	//var typbasetype uint32
	//
	//err = c.QueryRow(ctx, "select typtype::text, typbasetype from pg_type where oid=$1", oid).Scan(&typtype, &typbasetype)
	//if err != nil {
	//	return nil, err
	//}
	//
	//switch typtype {
	//case "b": // array
	//	elementOID, err := c.getArrayElementOID(ctx, oid)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	dt, ok := c.TypeMap().TypeForOID(elementOID)
	//	if !ok {
	//		return nil, errors.New("array element OID not registered")
	//	}
	//
	//	return &pgtype.Type{Name: typeName, OID: oid, Codec: &pgtype.ArrayCodec{ElementType: dt}}, nil
	//case "c": // composite
	//	fields, err := c.getCompositeFields(ctx, oid)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	return &pgtype.Type{Name: typeName, OID: oid, Codec: &pgtype.CompositeCodec{Fields: fields}}, nil
	//case "d": // domain
	//	dt, ok := c.TypeMap().TypeForOID(typbasetype)
	//	if !ok {
	//		return nil, errors.New("domain base type OID not registered")
	//	}
	//
	//	return &pgtype.Type{Name: typeName, OID: oid, Codec: dt.Codec}, nil
	//case "e": // enum
	//	return &pgtype.Type{Name: typeName, OID: oid, Codec: &pgtype.EnumCodec{}}, nil
	//case "r": // range
	//	elementOID, err := c.getRangeElementOID(ctx, oid)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	dt, ok := c.TypeMap().TypeForOID(elementOID)
	//	if !ok {
	//		return nil, errors.New("range element OID not registered")
	//	}
	//
	//	return &pgtype.Type{Name: typeName, OID: oid, Codec: &pgtype.RangeCodec{ElementType: dt}}, nil
	//case "m": // multirange
	//	elementOID, err := c.getMultiRangeElementOID(ctx, oid)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	dt, ok := c.TypeMap().TypeForOID(elementOID)
	//	if !ok {
	//		return nil, errors.New("multirange element OID not registered")
	//	}
	//
	//	return &pgtype.Type{Name: typeName, OID: oid, Codec: &pgtype.MultirangeCodec{ElementType: dt}}, nil
	//default:
	//	return &pgtype.Type{}, errors.New("unknown typtype")
	//}
	panic("implement me")
}

func (c *Conn) getArrayElementOID(ctx context.Context, oid uint32) (uint32, error) {
	//var typelem uint32
	//
	//err := c.QueryRow(ctx, "select typelem from pg_type where oid=$1", oid).Scan(&typelem)
	//if err != nil {
	//	return 0, err
	//}
	//
	//return typelem, nil
	panic("implement me")
}

func (c *Conn) getRangeElementOID(ctx context.Context, oid uint32) (uint32, error) {
	//var typelem uint32
	//
	//err := c.QueryRow(ctx, "select rngsubtype from pg_range where rngtypid=$1", oid).Scan(&typelem)
	//if err != nil {
	//	return 0, err
	//}
	//
	//return typelem, nil
	panic("implement me")
}

func (c *Conn) getMultiRangeElementOID(ctx context.Context, oid uint32) (uint32, error) {
	//var typelem uint32
	//
	//err := c.QueryRow(ctx, "select rngtypid from pg_range where rngmultitypid=$1", oid).Scan(&typelem)
	//if err != nil {
	//	return 0, err
	//}
	//
	//return typelem, nil
	panic("implement me")
}

func (c *Conn) getCompositeFields(ctx context.Context, oid uint32) ([]pgtype.CompositeCodecField, error) {
	//	var typrelid uint32
	//
	//	err := c.QueryRow(ctx, "select typrelid from pg_type where oid=$1", oid).Scan(&typrelid)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	var fields []pgtype.CompositeCodecField
	//	var fieldName string
	//	var fieldOID uint32
	//	rows, _ := c.Query(ctx, `select attname, atttypid
	//from pg_attribute
	//where attrelid=$1
	//	and not attisdropped
	//	and attnum > 0
	//order by attnum`,
	//		typrelid,
	//	)
	//	_, err = ForEachRow(rows, []any{&fieldName, &fieldOID}, func() error {
	//		dt, ok := c.TypeMap().TypeForOID(fieldOID)
	//		if !ok {
	//			return fmt.Errorf("unknown composite type field OID: %v", fieldOID)
	//		}
	//		fields = append(fields, pgtype.CompositeCodecField{Name: fieldName, Type: dt})
	//		return nil
	//	})
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	return fields, nil
	panic("implement me")
}

func (c *Conn) deallocateInvalidatedCachedStatements(ctx context.Context) error {
	//FIXME: fbx need to check state of transaction bound to statement?
	//	if txStatus := c.pgConn.TxStatus(); txStatus != 'I' && txStatus != 'T' {
	//		return nil
	//	}

	var invalidatedStatements []*pgconn.Statement
	if c.statementCache != nil {
		invalidatedStatements = c.statementCache.GetInvalidated()
	}

	if len(invalidatedStatements) == 0 {
		return nil
	}

	pipeline := c.pgConn.StartPipeline(ctx)
	defer pipeline.Close()

	for _, st := range invalidatedStatements {
		pipeline.SendDeallocate(st.Description.Name)
	}

	err := pipeline.Sync()
	if err != nil {
		return fmt.Errorf("failed to deallocate cached statement(s): %w", err)
	}

	err = pipeline.Close()
	if err != nil {
		return fmt.Errorf("failed to deallocate cached statement(s): %w", err)
	}

	c.statementCache.RemoveInvalidated()
	for _, st := range invalidatedStatements {
		delete(c.preparedStatements, st.Description.Name)
	}

	return nil
}

// Commit commits internal transaction
func (c *Conn) Commit(ctx context.Context) error {
	return c.pgConn.Commit(ctx)
}

func (c *Conn) SetAutocommit(autocommit bool) {
	c.autocommit = autocommit
}
