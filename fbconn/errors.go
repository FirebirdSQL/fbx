package fbconn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strings"
	"syscall"

	"rdb.red-soft.ru/fbx/fbproto"
)

// SafeToRetry checks if the err is guaranteed to have occurred before sending any data to the server.
func SafeToRetry(err error) bool {
	var retryableErr interface{ SafeToRetry() bool }
	if errors.As(err, &retryableErr) {
		return retryableErr.SafeToRetry()
	}
	return false
}

// Timeout checks if err was caused by a timeout. To be specific, it is true if err was caused within fbconn by a
// context.DeadlineExceeded or an implementer of net.Error where Timeout() is true.
func Timeout(err error) bool {
	var timeoutErr *errTimeout
	return errors.As(err, &timeoutErr)
}

type FbError struct {
	Object   int
	BlobId   int64
	GdsCodes []int
	SqlErr   int
	Code     string
	Message  string
	Line     int
	Position int
}

// IsConnectionFatal reports whether a server error makes the connection unsafe for further use.
// Transport and protocol errors are handled separately and do not produce an [FbError].
func IsConnectionFatal(err *FbError) bool {
	if err == nil {
		return false
	}

	// readStatusVector supplies a fallback SQLSTATE when the server omits it. A non-empty malformed value therefore
	// indicates an unexpected server or protocol condition.
	if err.Code != "" && len(err.Code) != 5 {
		return true
	}

	if len(err.Code) == 5 && err.Code[:2] == "08" {
		return true
	}

	switch err.Code {
	case "2E000", // Invalid connection name.
		"HY001", // Memory allocation error.
		"HYT00", // Timeout expired.
		"HYT01": // Connection timeout expired.
		return true
	}

	for _, code := range err.GdsCodes {
		if isConnectionFatalGDSCode(code) {
			return true
		}
	}

	return false
}

func isConnectionFatalGDSCode(code int) bool {
	switch code {
	case fbproto.Isc_bad_db_handle,
		fbproto.Isc_network_error,
		fbproto.Isc_net_read_err,
		fbproto.Isc_net_write_err,
		fbproto.Isc_lost_db_connection,
		fbproto.Isc_io_error,
		fbproto.Isc_req_sync,
		fbproto.Isc_unavailable,
		fbproto.Isc_shutdown,
		fbproto.Isc_att_shutdown,
		fbproto.Isc_net_server_shutdown,
		fbproto.Isc_att_shut_killed,
		fbproto.Isc_att_shut_idle,
		fbproto.Isc_att_shut_db_down,
		fbproto.Isc_att_shut_engine,
		fbproto.Isc_ses_reset_failed,
		fbproto.Isc_exception_stack_overflow:
		return true
	default:
		return false
	}
}

func (e *FbError) SQLState() string {
	return e.Code
}

func (e *FbError) Error() string {
	return e.Message + "SQLSTATE " + e.Code + ""
}

func ErrorResponseToFbError(errorResponse *fbproto.ErrorResponse) *FbError {
	gdsCodes := append([]int(nil), errorResponse.GdsCodes...)
	return &FbError{
		Object:   errorResponse.Object,
		BlobId:   errorResponse.BlobId,
		GdsCodes: gdsCodes,
		SqlErr:   errorResponse.SqlErr,
		Code:     errorResponse.SqlState,
		Message:  errorResponse.Message,
		Line:     errorResponse.Line,
		Position: errorResponse.Position,
	}
}

// ConnectError is the error returned when a connection attempt fails.
type ConnectError struct {
	Config *Config // The configuration that was used in the connection attempt.
	err    error
}

func (e *ConnectError) Error() string {
	prefix := fmt.Sprintf("failed to connect to `user=%s database=%s`:", e.Config.User, e.Config.Database)
	details := e.err.Error()
	if strings.Contains(details, "\n") {
		return prefix + "\n\t" + strings.ReplaceAll(details, "\n", "\n\t")
	} else {
		return prefix + " " + details
	}
}

func (e *ConnectError) Unwrap() error {
	return e.err
}

type perDialConnectError struct {
	address          string
	originalHostname string
	err              error
}

func (e *perDialConnectError) Error() string {
	return fmt.Sprintf("%s (%s): %s", e.address, e.originalHostname, e.err.Error())
}

func (e *perDialConnectError) Unwrap() error {
	return e.err
}

type connLockError struct {
	status string
}

// ErrConnClosed is returned, possibly wrapped, when an operation is attempted on a closed connection.
var ErrConnClosed = errors.New("conn closed")

func (e *connLockError) SafeToRetry() bool {
	return true // a lock failure by definition happens before the connection is used.
}

func (e *connLockError) Error() string {
	return e.status
}

func (e *connLockError) Unwrap() error {
	if e.status == "conn closed" {
		return ErrConnClosed
	}
	return nil
}

// ParseConfigError is the error returned when a connection string cannot be parsed.
type ParseConfigError struct {
	ConnString string // The connection string that could not be parsed.
	msg        string
	err        error
}

func NewParseConfigError(conn, msg string, err error) error {
	return &ParseConfigError{
		ConnString: conn,
		msg:        msg,
		err:        err,
	}
}

func (e *ParseConfigError) Error() string {
	// Now that ParseConfigError is public and ConnString is available to the developer, perhaps it would be better only
	// return a static string. That would ensure that the error message cannot leak a password. The ConnString field would
	// allow access to the original string if desired and Unwrap would allow access to the underlying error.
	connString := redactPW(e.ConnString)
	if e.err == nil {
		return fmt.Sprintf("cannot parse `%s`: %s", connString, e.msg)
	}
	return fmt.Sprintf("cannot parse `%s`: %s (%s)", connString, e.msg, e.err.Error())
}

func (e *ParseConfigError) Unwrap() error {
	return e.err
}

func normalizeTimeoutError(ctx context.Context, err error) error {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		switch ctx.Err() {
		case context.Canceled:
			// Since the timeout was caused by a context cancellation, the actual error is context.Canceled not the timeout error.
			return context.Canceled
		case context.DeadlineExceeded:
			return &errTimeout{err: ctx.Err()}
		default:
			return &errTimeout{err: err}
		}
	}
	return err
}

// normalizeConnClosedError converts transport-level connection breakage (EOF / reset / closed socket)
// into an FbError with SQLSTATE 08006 so callers don't have to special-case io.EOF.
//
// It intentionally does not convert timeout errors; those are handled by normalizeTimeoutError.
func normalizeConnClosedError(err error) error {
	if err == nil {
		return nil
	}

	// Keep timeout errors intact.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return err
	}

	if errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNABORTED) {
		return &FbError{
			Message: fmt.Sprintf("Connection failure: %s\n", err.Error()),
			Code:    "08006",
		}
	}

	return err
}

type fbconnError struct {
	msg         string
	err         error
	safeToRetry bool
}

func (e *fbconnError) Error() string {
	if e.msg == "" {
		return e.err.Error()
	}
	if e.err == nil {
		return e.msg
	}
	return fmt.Sprintf("%s: %s", e.msg, e.err.Error())
}

func (e *fbconnError) SafeToRetry() bool {
	return e.safeToRetry
}

func (e *fbconnError) Unwrap() error {
	return e.err
}

// errTimeout occurs when an error was caused by a timeout. Specifically, it wraps an error which is
// context.Canceled, context.DeadlineExceeded, or an implementer of net.Error where Timeout() is true.
type errTimeout struct {
	err error
}

func (e *errTimeout) Error() string {
	return fmt.Sprintf("timeout: %s", e.err.Error())
}

func (e *errTimeout) SafeToRetry() bool {
	return SafeToRetry(e.err)
}

func (e *errTimeout) Unwrap() error {
	return e.err
}

type contextAlreadyDoneError struct {
	err error
}

func (e *contextAlreadyDoneError) Error() string {
	return fmt.Sprintf("context already done: %s", e.err.Error())
}

func (e *contextAlreadyDoneError) SafeToRetry() bool {
	return true
}

func (e *contextAlreadyDoneError) Unwrap() error {
	return e.err
}

// newContextAlreadyDoneError double-wraps a context error in `contextAlreadyDoneError` and `errTimeout`.
func newContextAlreadyDoneError(ctx context.Context) (err error) {
	return &errTimeout{&contextAlreadyDoneError{err: ctx.Err()}}
}

func redactPW(connString string) string {
	if strings.HasPrefix(connString, "firebird://") || strings.HasPrefix(connString, "firebirdsql://") {
		if u, err := url.Parse(connString); err == nil {
			return redactURL(u)
		}
	}
	quotedKV := regexp.MustCompile(`password='[^']*'`)
	connString = quotedKV.ReplaceAllLiteralString(connString, "password=xxxxx")
	plainKV := regexp.MustCompile(`password=[^ ]*`)
	connString = plainKV.ReplaceAllLiteralString(connString, "password=xxxxx")
	brokenURL := regexp.MustCompile(`:[^:@]+?@`)
	connString = brokenURL.ReplaceAllLiteralString(connString, ":xxxxx@")
	return connString
}

func redactURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	if _, pwSet := u.User.Password(); pwSet {
		u.User = url.UserPassword(u.User.Username(), "xxxxx")
	}
	return u.String()
}

type NotPreferredError struct {
	err         error
	safeToRetry bool
}

func (e *NotPreferredError) Error() string {
	return fmt.Sprintf("standby server not found: %s", e.err.Error())
}

func (e *NotPreferredError) SafeToRetry() bool {
	return e.safeToRetry
}

func (e *NotPreferredError) Unwrap() error {
	return e.err
}
