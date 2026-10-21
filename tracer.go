package fbx

import (
	"context"

	"github.com/firebirdsql/fbx/fbconn"
)

// QueryTracer traces Query, QueryRow, and Exec.
type QueryTracer interface {
	// TraceQueryStart is called at the beginning of Query, QueryRow, and Exec calls. The returned context is used for the
	// rest of the call and will be passed to TraceQueryEnd.
	TraceQueryStart(ctx context.Context, conn *Conn, data TraceQueryStartData) context.Context

	TraceQueryEnd(ctx context.Context, conn *Conn, data TraceQueryEndData)
}

type TraceQueryStartData struct {
	SQL  string
	Args []any
}

type TraceQueryEndData struct {
	QueryStatus fbconn.QueryStatus
	Err         error
}

// PrepareTracer traces Prepare.
type PrepareTracer interface {
	// TracePrepareStart is called at the beginning of Prepare calls. The returned context is used for the
	// rest of the call and will be passed to TracePrepareEnd.
	TracePrepareStart(ctx context.Context, conn *Conn, data TracePrepareStartData) context.Context

	TracePrepareEnd(ctx context.Context, conn *Conn, data TracePrepareEndData)
}

type TracePrepareStartData struct {
	Name string
	SQL  string
}

type TracePrepareEndData struct {
	AlreadyPrepared bool
	Err             error
}

// ConnectTracer traces Connect and ConnectConfig.
type ConnectTracer interface {
	// TraceConnectStart is called at the beginning of Connect and ConnectConfig calls. The returned context is used for
	// the rest of the call and will be passed to TraceConnectEnd.
	TraceConnectStart(ctx context.Context, data TraceConnectStartData) context.Context

	TraceConnectEnd(ctx context.Context, data TraceConnectEndData)
}

type TraceConnectStartData struct {
	ConnConfig *ConnConfig
}

type TraceConnectEndData struct {
	Conn *Conn
	Err  error
}
