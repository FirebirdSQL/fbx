package fbservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"rdb.red-soft.ru/fbx/fbconn"
)

const maxTraceStartSPBLength = 1<<16 - 1

// TraceOptions controls output from the server's trace services.
type TraceOptions struct {
	// Output receives unmodified service output, including session IDs, trace
	// events and diagnostic messages. TraceStart, TraceStartSession and TraceList
	// require Output;
	// use io.Discard to explicitly discard it. Other trace operations allow nil.
	// Writes are synchronous and must not block indefinitely. Shared writers
	// must be concurrency safe. The manager never closes Output; on error, it
	// may contain partial output. Cancellation cannot interrupt an ongoing Write.
	Output io.Writer
}

// TraceStart starts a user trace session with an optional name and configuration
// text (not a filename). It streams output until the session stops, ctx is
// canceled, or an error occurs, then closes its service connection. Closing the
// connection ends the user trace session. Exactly one options value with a
// non-nil Output is required. The configuration syntax depends on the server.
// The encoded start parameter block must fit in 65535 bytes: configuration plus
// 4 bytes, and, when name is nonempty, name plus another 3 bytes.
// All trace operations require a configuration with a single host and port;
// fallback servers are rejected before connecting because session IDs are
// server local. The host must identify the same server for every operation.
//
// Use [Manager.TraceStartSession] for background streaming with a session ID,
// or run TraceStart in a goroutine to manage the session concurrently through
// [Manager.TraceList], [Manager.TraceStop], [Manager.TraceSuspend] and
// [Manager.TraceResume]. The server reports the session ID in the output.
// Requests are validated before connecting; a valid request with an already
// canceled ctx returns its error without connecting.
func (m *Manager) TraceStart(ctx context.Context, name, configuration string, options ...TraceOptions) error {
	if configuration == "" {
		return errors.New("trace configuration must not be empty")
	}
	for _, field := range []struct{ name, value string }{
		{"name", name}, {"configuration", configuration},
	} {
		if strings.IndexByte(field.value, 0) >= 0 {
			return fmt.Errorf("trace %s must not contain NUL", field.name)
		}
		if len(field.value) > maxServiceActionStringLength {
			return fmt.Errorf("trace %s is %d bytes; maximum is %d", field.name, len(field.value), maxServiceActionStringLength)
		}
	}
	spbLength := 4 + len(configuration) // Action tag, configuration tag and uint16 length.
	if name != "" {
		spbLength += 3 + len(name) // Name tag and uint16 length.
	}
	if spbLength > maxTraceStartSPBLength {
		return fmt.Errorf("trace start SPB is %d bytes; maximum is %d", spbLength, maxTraceStartSPBLength)
	}
	return m.runTrace(ctx, true, options, func(conn serviceConnection, option fbconn.TraceOptions) error {
		return conn.TraceStart(ctx, name, configuration, option)
	})
}

// TraceStop stops a trace session by its positive session ID. It uses a separate
// service connection, which is closed before returning. Output is optional.
// Server diagnostics such as a missing session or insufficient permissions may
// be returned as text in Output rather than as an error. Use TraceList to check
// session state. At most one options value is accepted.
// The manager must target the same single server that created the session;
// fallback servers are rejected before connecting.
func (m *Manager) TraceStop(ctx context.Context, sessionID int, options ...TraceOptions) error {
	return m.runTraceSession(ctx, sessionID, options, func(conn serviceConnection, option fbconn.TraceOptions) error {
		return conn.TraceStop(ctx, sessionID, option)
	})
}

// TraceSuspend pauses a trace session without closing its output stream.
// It has the same connection, output and error semantics as [Manager.TraceStop].
func (m *Manager) TraceSuspend(ctx context.Context, sessionID int, options ...TraceOptions) error {
	return m.runTraceSession(ctx, sessionID, options, func(conn serviceConnection, option fbconn.TraceOptions) error {
		return conn.TraceSuspend(ctx, sessionID, option)
	})
}

// TraceResume resumes a suspended trace session.
// It has the same connection, output and error semantics as [Manager.TraceStop].
func (m *Manager) TraceResume(ctx context.Context, sessionID int, options ...TraceOptions) error {
	return m.runTraceSession(ctx, sessionID, options, func(conn serviceConnection, option fbconn.TraceOptions) error {
		return conn.TraceResume(ctx, sessionID, option)
	})
}

// TraceList writes the server's textual list of visible trace sessions, including
// IDs, names, owners, start times and state flags. Exactly one options value with
// a non-nil Output is required. It validates the request before connecting and
// closes its service connection before returning.
// Fallback servers are rejected; the manager must target a single server.
func (m *Manager) TraceList(ctx context.Context, options ...TraceOptions) error {
	return m.runTrace(ctx, true, options, func(conn serviceConnection, option fbconn.TraceOptions) error {
		return conn.TraceList(ctx, option)
	})
}

func (m *Manager) runTraceSession(ctx context.Context, sessionID int, options []TraceOptions,
	run func(serviceConnection, fbconn.TraceOptions) error,
) error {
	if sessionID <= 0 || sessionID > maxServiceActionInt32 {
		return fmt.Errorf("trace session ID must be between 1 and %d", maxServiceActionInt32)
	}
	return m.runTrace(ctx, false, options, run)
}

func (m *Manager) runTrace(ctx context.Context, requireOutput bool, options []TraceOptions,
	run func(serviceConnection, fbconn.TraceOptions) error,
) (err error) {
	if len(options) > 1 {
		return errors.New("trace accepts at most one TraceOptions value")
	}
	var option TraceOptions
	if len(options) == 1 {
		option = options[0]
	}
	if requireOutput && option.Output == nil {
		return errors.New("trace output must not be nil; use io.Discard to discard the output")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if m == nil {
		return errors.New("service manager config must not be nil")
	}
	config, err := m.config.toServiceConfig("")
	if err != nil {
		return err
	}
	// A fresh connection can choose a different fallback after server recovery,
	// but the same session ID may refer to an unrelated session there.
	if len(config.ConnectionConfig.Fallbacks) > 0 || strings.Contains(config.ConnectionConfig.Host, ",") {
		return errors.New("trace operations require a single server; use a connection string with one host and port")
	}
	conn, err := m.connect(ctx, config)
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() { err = errors.Join(err, closeService(conn)) }()
	return run(conn, fbconn.TraceOptions{Output: option.Output})
}
