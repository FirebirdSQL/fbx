// Package tracelog provides a tracer that acts as a traditional logger.
package tracelog

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
	"unicode/utf8"

	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbxpool"
)

// LogLevel represents the fbx logging level. See LogLevel* constants for
// possible values.
type LogLevel int

// The values for log levels are chosen such that the zero value means that no
// log level was specified.
const (
	LogLevelTrace = LogLevel(6)
	LogLevelDebug = LogLevel(5)
	LogLevelInfo  = LogLevel(4)
	LogLevelWarn  = LogLevel(3)
	LogLevelError = LogLevel(2)
	LogLevelNone  = LogLevel(1)
)

func (ll LogLevel) String() string {
	switch ll {
	case LogLevelTrace:
		return "trace"
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	case LogLevelNone:
		return "none"
	default:
		return fmt.Sprintf("invalid level %d", ll)
	}
}

// Logger is the interface used to get log output from fbx.
type Logger interface {
	// Log a message at the given level with data key/value pairs. data may be nil.
	Log(ctx context.Context, level LogLevel, msg string, data map[string]any)
}

// LoggerFunc is a wrapper around a function to satisfy the fbx.Logger interface
type LoggerFunc func(ctx context.Context, level LogLevel, msg string, data map[string]interface{})

// Log delegates the logging request to the wrapped function
func (f LoggerFunc) Log(ctx context.Context, level LogLevel, msg string, data map[string]interface{}) {
	f(ctx, level, msg, data)
}

// LogLevelFromString converts log level string to constant
//
// Valid levels:
//
//	trace
//	debug
//	info
//	warn
//	error
//	none
func LogLevelFromString(s string) (LogLevel, error) {
	switch s {
	case "trace":
		return LogLevelTrace, nil
	case "debug":
		return LogLevelDebug, nil
	case "info":
		return LogLevelInfo, nil
	case "warn":
		return LogLevelWarn, nil
	case "error":
		return LogLevelError, nil
	case "none":
		return LogLevelNone, nil
	default:
		return 0, errors.New("invalid log level")
	}
}

func logQueryArgs(args []any) []any {
	logArgs := make([]any, 0, len(args))

	for _, a := range args {
		switch v := a.(type) {
		case []byte:
			if len(v) < 64 {
				a = hex.EncodeToString(v)
			} else {
				a = fmt.Sprintf("%x (truncated %d bytes)", v[:64], len(v)-64)
			}
		case string:
			if len(v) > 64 {
				var l = 0
				for w := 0; l < 64; l += w {
					_, w = utf8.DecodeRuneInString(v[l:])
				}
				if len(v) > l {
					a = fmt.Sprintf("%s (truncated %d bytes)", v[:l], len(v)-l)
				}
			}
		}
		logArgs = append(logArgs, a)
	}

	return logArgs
}

// TraceLogConfig holds the configuration for key names
type TraceLogConfig struct {
	TimeKey string
}

// DefaultTraceLogConfig returns the default configuration for TraceLog
func DefaultTraceLogConfig() *TraceLogConfig {
	return &TraceLogConfig{
		TimeKey: "time",
	}
}

// TraceLog implements fbx.QueryTracer, fbx.BatchTracer, fbx.ConnectTracer, fbx.CopyFromTracer, fbxpool.AcquireTracer,
// and fbxpool.ReleaseTracer. Logger and LogLevel are required. Config will be automatically initialized on the
// first use if nil.
type TraceLog struct {
	Logger   Logger
	LogLevel LogLevel

	Config           *TraceLogConfig
	ensureConfigOnce sync.Once
}

// ensureConfig initializes the Config field with default values if it is nil.
func (tl *TraceLog) ensureConfig() {
	tl.ensureConfigOnce.Do(
		func() {
			if tl.Config == nil {
				tl.Config = DefaultTraceLogConfig()
			}
		},
	)
}

type ctxKey int

const (
	_ ctxKey = iota
	tracelogQueryCtxKey
	tracelogBatchCtxKey
	tracelogCopyFromCtxKey
	tracelogConnectCtxKey
	tracelogPrepareCtxKey
	tracelogAcquireCtxKey
)

type traceQueryData struct {
	startTime time.Time
	sql       string
	args      []any
}

func (tl *TraceLog) TraceQueryStart(ctx context.Context, conn *fbx.Conn, data fbx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, tracelogQueryCtxKey, &traceQueryData{
		startTime: time.Now(),
		sql:       data.SQL,
		args:      data.Args,
	})
}

func (tl *TraceLog) TraceQueryEnd(ctx context.Context, conn *fbx.Conn, data fbx.TraceQueryEndData) {
	tl.ensureConfig()
	queryData := ctx.Value(tracelogQueryCtxKey).(*traceQueryData)

	endTime := time.Now()
	interval := endTime.Sub(queryData.startTime)

	if data.Err != nil {
		if tl.shouldLog(LogLevelError) {
			tl.log(ctx, conn, LogLevelError, "Query", map[string]any{"sql": queryData.sql, "args": logQueryArgs(queryData.args), "err": data.Err, tl.Config.TimeKey: interval})
		}
		return
	}

	if tl.shouldLog(LogLevelInfo) {
		tl.log(ctx, conn, LogLevelInfo, "Query", map[string]any{"sql": queryData.sql, "args": logQueryArgs(queryData.args), tl.Config.TimeKey: interval, "queryStatus": data.QueryStatus.String()})
	}
}

type traceConnectData struct {
	startTime  time.Time
	connConfig *fbx.ConnConfig
}

func (tl *TraceLog) TraceConnectStart(ctx context.Context, data fbx.TraceConnectStartData) context.Context {
	return context.WithValue(ctx, tracelogConnectCtxKey, &traceConnectData{
		startTime:  time.Now(),
		connConfig: data.ConnConfig,
	})
}

func (tl *TraceLog) TraceConnectEnd(ctx context.Context, data fbx.TraceConnectEndData) {
	tl.ensureConfig()
	connectData := ctx.Value(tracelogConnectCtxKey).(*traceConnectData)

	endTime := time.Now()
	interval := endTime.Sub(connectData.startTime)

	if data.Err != nil {
		if tl.shouldLog(LogLevelError) {
			tl.Logger.Log(ctx, LogLevelError, "Connect", map[string]any{
				"host":            connectData.connConfig.Host,
				"port":            connectData.connConfig.Port,
				"database":        connectData.connConfig.Database,
				tl.Config.TimeKey: interval,
				"err":             data.Err,
			})
		}
		return
	}

	if data.Conn != nil {
		if tl.shouldLog(LogLevelInfo) {
			tl.log(ctx, data.Conn, LogLevelInfo, "Connect", map[string]any{
				"host":            connectData.connConfig.Host,
				"port":            connectData.connConfig.Port,
				"database":        connectData.connConfig.Database,
				tl.Config.TimeKey: interval,
			})
		}
	}
}

type tracePrepareData struct {
	startTime time.Time
	name      string
	sql       string
}

func (tl *TraceLog) TracePrepareStart(ctx context.Context, _ *fbx.Conn, data fbx.TracePrepareStartData) context.Context {
	return context.WithValue(ctx, tracelogPrepareCtxKey, &tracePrepareData{
		startTime: time.Now(),
		name:      data.Name,
		sql:       data.SQL,
	})
}

func (tl *TraceLog) TracePrepareEnd(ctx context.Context, conn *fbx.Conn, data fbx.TracePrepareEndData) {
	tl.ensureConfig()
	prepareData := ctx.Value(tracelogPrepareCtxKey).(*tracePrepareData)

	endTime := time.Now()
	interval := endTime.Sub(prepareData.startTime)

	if data.Err != nil {
		if tl.shouldLog(LogLevelError) {
			tl.log(ctx, conn, LogLevelError, "Prepare", map[string]any{"name": prepareData.name, "sql": prepareData.sql, "err": data.Err, tl.Config.TimeKey: interval})
		}
		return
	}

	if tl.shouldLog(LogLevelInfo) {
		tl.log(ctx, conn, LogLevelInfo, "Prepare", map[string]any{"name": prepareData.name, "sql": prepareData.sql, tl.Config.TimeKey: interval, "alreadyPrepared": data.AlreadyPrepared})
	}
}

type traceAcquireData struct {
	startTime time.Time
}

func (tl *TraceLog) TraceAcquireStart(ctx context.Context, _ *fbxpool.Pool, _ fbxpool.TraceAcquireStartData) context.Context {
	return context.WithValue(ctx, tracelogAcquireCtxKey, &traceAcquireData{
		startTime: time.Now(),
	})
}

func (tl *TraceLog) TraceAcquireEnd(ctx context.Context, _ *fbxpool.Pool, data fbxpool.TraceAcquireEndData) {
	tl.ensureConfig()
	acquireData := ctx.Value(tracelogAcquireCtxKey).(*traceAcquireData)

	endTime := time.Now()
	interval := endTime.Sub(acquireData.startTime)

	if data.Err != nil {
		if tl.shouldLog(LogLevelError) {
			tl.Logger.Log(ctx, LogLevelError, "Acquire", map[string]any{"err": data.Err, tl.Config.TimeKey: interval})
		}
		return
	}

	if data.Conn != nil {
		if tl.shouldLog(LogLevelDebug) {
			tl.log(ctx, data.Conn, LogLevelDebug, "Acquire", map[string]any{tl.Config.TimeKey: interval})
		}
	}
}

func (tl *TraceLog) TraceRelease(_ *fbxpool.Pool, data fbxpool.TraceReleaseData) {
	if tl.shouldLog(LogLevelDebug) {
		// there is no context on the TraceRelease callback
		tl.log(context.Background(), data.Conn, LogLevelDebug, "Release", map[string]any{})
	}
}

func (tl *TraceLog) shouldLog(lvl LogLevel) bool {
	return tl.LogLevel >= lvl
}

func (tl *TraceLog) log(ctx context.Context, conn *fbx.Conn, lvl LogLevel, msg string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}

	fbConn := conn.FbConn()
	if fbConn != nil {
		attachmentID := fbConn.AttachmentID()
		if attachmentID != 0 {
			data["attachment_id"] = attachmentID
		}
	}

	tl.Logger.Log(ctx, lvl, msg, data)
}
