package pgproto3

import "io"

// Client acts as a client for the Firebird wire protocol
type Client struct {
}

type Response struct {
}

func NewClient(r io.Reader, w io.Writer) *Client {
	return &Client{}
}
