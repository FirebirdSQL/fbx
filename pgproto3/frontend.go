package pgproto3

import "io"

// Frontend acts as a client for the PostgreSQL wire protocol version 3.
type Frontend struct {
}

func NewFrontend(r io.Reader, w io.Writer) *Frontend {
	panic("implement me")
}
