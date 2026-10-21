package fbconn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/firebirdsql/fbx/fbproto"
)

const maxTraceStartSPBLength = 1<<16 - 1

// TraceOptions controls output from the server's trace services.
type TraceOptions struct {
	// Output receives unmodified service output, including session IDs, trace
	// events and diagnostic messages. A nil value discards it. Writes are
	// synchronous; cancellation cannot interrupt a Write already in progress.
	// The writer belongs to the caller and is never closed by the service.
	Output io.Writer
}

// TraceStart starts a user trace session and copies its output until the session
// stops, ctx is canceled, or an error occurs. Name may be empty. Configuration
// is the trace configuration text, not a filename; its syntax is server specific.
// The encoded start parameter block, including name and configuration, must
// fit in 65535 bytes. An oversized request is rejected before sending it.
// Use another service connection to list, stop, suspend or resume sessions while
// this call runs; connect to the same server, since session IDs are server local.
// After an error, close the connection to release the session.
func (c *ServiceConn) TraceStart(ctx context.Context, name, configuration string, options TraceOptions) error {
	actionSPB, err := buildTraceStartActionSPB(name, configuration)
	if err != nil {
		return err
	}
	return c.runTraceAction(ctx, "start", actionSPB, options)
}

// TraceStop stops the session identified by sessionID and copies the server's
// diagnostic output. The session may have been started by another connection.
// A missing session or insufficient permissions may be reported as text in Output.
func (c *ServiceConn) TraceStop(ctx context.Context, sessionID int, options TraceOptions) error {
	return c.runTraceSessionAction(ctx, "stop", fbproto.Isc_action_svc_trace_stop, sessionID, options)
}

// TraceSuspend suspends the session identified by sessionID and copies the
// server's diagnostic output, with the same error semantics as [ServiceConn.TraceStop].
func (c *ServiceConn) TraceSuspend(ctx context.Context, sessionID int, options TraceOptions) error {
	return c.runTraceSessionAction(ctx, "suspend", fbproto.Isc_action_svc_trace_suspend, sessionID, options)
}

// TraceResume resumes the session identified by sessionID and copies the
// server's diagnostic output, with the same error semantics as [ServiceConn.TraceStop].
func (c *ServiceConn) TraceResume(ctx context.Context, sessionID int, options TraceOptions) error {
	return c.runTraceSessionAction(ctx, "resume", fbproto.Isc_action_svc_trace_resume, sessionID, options)
}

// TraceList copies the server's textual list of visible trace sessions to Output.
// The report includes session IDs, names, owners, start times and state flags.
func (c *ServiceConn) TraceList(ctx context.Context, options TraceOptions) error {
	return c.runTraceAction(ctx, "list", []byte{fbproto.Isc_action_svc_trace_list}, options)
}

func (c *ServiceConn) runTraceSessionAction(ctx context.Context, name string, action byte, sessionID int, options TraceOptions) error {
	if sessionID <= 0 || sessionID > maxServiceActionInt32 {
		return fmt.Errorf("trace session ID must be between 1 and %d", maxServiceActionInt32)
	}
	spb := fbproto.NewXPBWriterFromTag(action).
		AppendByte(fbproto.Isc_spb_trc_id).
		AppendInt32(int32(sessionID))
	return c.runTraceAction(ctx, name, spb.Bytes(), options)
}

func (c *ServiceConn) runTraceAction(ctx context.Context, name string, actionSPB []byte, options TraceOptions) error {
	if err := c.runServiceAction(ctx, actionSPB, options.Output); err != nil {
		// Servers may report canceled services as a connection shutdown rather
		// than a network timeout. Keep both the context and the server error.
		if ctxErr := ctx.Err(); ctxErr != nil && !errors.Is(err, ctxErr) {
			err = errors.Join(err, ctxErr)
		}
		return fmt.Errorf("trace %s: %w", name, err)
	}
	return nil
}

func buildTraceStartActionSPB(name, configuration string) ([]byte, error) {
	if configuration == "" {
		return nil, errors.New("trace configuration must not be empty")
	}
	spb := fbproto.NewXPBWriterFromTag(fbproto.Isc_action_svc_trace_start)
	for _, field := range []struct {
		name, value string
		tag         byte
	}{
		{"configuration", configuration, fbproto.Isc_spb_trc_cfg},
		{"name", name, fbproto.Isc_spb_trc_name},
	} {
		if strings.IndexByte(field.value, 0) >= 0 {
			return nil, fmt.Errorf("trace %s must not contain NUL", field.name)
		}
		if field.value != "" {
			if err := putServiceActionString(spb, field.tag, field.value); err != nil {
				return nil, fmt.Errorf("trace %s: %w", field.name, err)
			}
		}
	}
	// The server's service entry point accepts an unsigned 16-bit SPB length.
	actionSPB := spb.Bytes()
	if len(actionSPB) > maxTraceStartSPBLength {
		return nil, fmt.Errorf("trace start SPB is %d bytes; maximum is %d", len(actionSPB), maxTraceStartSPBLength)
	}
	return actionSPB, nil
}
