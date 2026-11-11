package fbservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// TraceSession represents a user trace started by [Manager.TraceStartSession].
// Its methods are safe for concurrent use. A TraceSession must not be copied;
// its zero value is not usable. Cancel the session or stop it on the server when
// finished, then call Wait to await connection cleanup and check the result.
type TraceSession struct {
	id     int
	cancel context.CancelFunc
	done   chan struct{}
	err    error // Published by closing done.
}

// ID returns the server-assigned session ID. Use it with the originating
// manager's TraceStop, TraceSuspend and TraceResume methods while the session
// is active. IDs are local to a server and may be reused after a session ends.
func (s *TraceSession) ID() int { return s.id }

// Done returns a channel closed after output reading and connection cleanup
// finish. Call Wait to retrieve the result.
func (s *TraceSession) Done() <-chan struct{} { return s.done }

// Wait waits for output reading and connection cleanup, then returns their
// errors. It may be called multiple times, including concurrently. Cancellation
// is reported as a context error; successful server-side stopping returns nil.
func (s *TraceSession) Wait() error {
	<-s.done
	return s.err
}

// Cancel requests cancellation of the stream, which closes its service
// connection and ends the user trace session. It is safe to call repeatedly.
// Cancel does not wait for cleanup; use Wait or Done to await completion.
// Cancellation cannot interrupt a blocked Output.Write.
func (s *TraceSession) Cancel() { s.cancel() }

// TraceStartSession starts a user trace and reads its output in a background
// goroutine. It returns after the server's "Trace session ID <id> started"
// line has been received and written to Output. An unrecognized or missing
// startup line returns an error after connection cleanup. Use [Manager.TraceStart]
// for synchronous streaming without interpreting server messages.
//
// The arguments, validation and output ownership are the same as TraceStart.
// Exactly one options value with a non-nil Output is required. Output receives
// all bytes unchanged, including the startup line, and remains open. It must
// support concurrent access if the caller uses it while the session runs.
// The context controls the entire session lifetime, including after this method
// returns. Cancel the session or stop it on the server when finished, and call
// [TraceSession.Wait] to collect background errors and await cleanup. An error
// after the startup line is reported by Wait, even if it occurs before this
// method returns. Cancellation cannot interrupt a blocked Output.Write.
func (m *Manager) TraceStartSession(ctx context.Context, name, configuration string, options ...TraceOptions) (*TraceSession, error) {
	if len(options) > 1 {
		return nil, errors.New("trace accepts at most one TraceOptions value")
	}
	if len(options) == 0 || options[0].Output == nil {
		return nil, errors.New("trace output must not be nil; use io.Discard to discard the output")
	}
	ctx, cancel := context.WithCancel(ctx)
	session := &TraceSession{cancel: cancel, done: make(chan struct{})}
	ready := make(chan int, 1)
	output := &traceSessionOutput{output: options[0].Output, ready: ready}
	option := options[0]
	option.Output = output
	go func() {
		defer cancel()
		session.err = m.TraceStart(ctx, name, configuration, option)
		if !output.started && session.err == nil {
			session.err = errors.New("trace ended without a session ID")
		}
		close(session.done)
	}()
	select {
	case session.id = <-ready:
		return session, nil
	case <-session.done:
		// The stream may finish before the caller observes the startup line.
		select {
		case session.id = <-ready:
			return session, nil
		default:
			return nil, session.err
		}
	}
}

// traceSessionOutput only interprets the first line. Its buffer is bounded even
// when the server never sends a newline; subsequent events are passed through.
type traceSessionOutput struct {
	output  io.Writer
	ready   chan<- int
	header  []byte
	started bool
}

func (w *traceSessionOutput) Write(p []byte) (int, error) {
	n, err := w.output.Write(p)
	if err != nil {
		return n, err
	}
	if n != len(p) {
		return n, io.ErrShortWrite
	}
	if w.started {
		return n, nil
	}
	lineEnd := bytes.IndexByte(p, '\n')
	fragment := p
	if lineEnd >= 0 {
		fragment = p[:lineEnd]
	}
	const maxStartupLine = 4096
	if len(w.header)+len(fragment) > maxStartupLine {
		return n, errors.New("trace startup line exceeds 4096 bytes")
	}
	w.header = append(w.header, fragment...)
	if lineEnd < 0 {
		return n, nil
	}
	line := strings.TrimSuffix(string(w.header), "\r")
	idText, prefixOK := strings.CutPrefix(line, "Trace session ID ")
	idText, suffixOK := strings.CutSuffix(idText, " started")
	id, err := strconv.ParseUint(idText, 10, 31)
	if !prefixOK || !suffixOK || err != nil || id == 0 {
		return n, fmt.Errorf("unrecognized trace startup line: %q", line)
	}
	w.started = true
	w.header = nil
	w.ready <- int(id)
	return n, nil
}
