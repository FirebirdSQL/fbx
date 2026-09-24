package fbconn

import (
	"context"
	"errors"
	"fmt"

	"rdb.red-soft.ru/fbx/fbproto"
)

// serviceServerVersion queries the service attachment, without attaching to a
// database or starting an action. Sweep uses it to select version-specific SPB tags.
func (c *ServiceConn) serviceServerVersion(ctx context.Context) (ServerVersion, error) {
	if c == nil || c.conn == nil {
		return ServerVersion{}, errors.New("service connection is nil")
	}
	if c.actionActive {
		return ServerVersion{}, errServiceActionActive
	}
	response, err := c.requestService(ctx, &fbproto.OpServiceInfo{
		ServiceHandle: c.conn.attachmentHandle,
		ReceiveItems:  []byte{fbproto.Isc_info_svc_server_version},
		BufferLength:  1024,
	})
	if err != nil {
		return ServerVersion{}, fmt.Errorf("query service server version: %w", err)
	}
	version, err := parseServiceServerVersion(response.Data)
	if err != nil {
		return ServerVersion{}, fmt.Errorf("parse service server version: %w", err)
	}
	return version, nil
}

func parseServiceServerVersion(data []byte) (ServerVersion, error) {
	r := NewServerInfoBufferReader(data)
	item, err := r.ReadByte()
	if err != nil {
		return ServerVersion{}, err
	}
	if item != fbproto.Isc_info_svc_server_version {
		return ServerVersion{}, fmt.Errorf("unexpected server version item %d", item)
	}
	raw, err := r.ReadStringValue()
	if err != nil {
		return ServerVersion{}, err
	}
	end, err := r.ReadByte()
	if err != nil {
		return ServerVersion{}, err
	}
	if end != fbproto.Isc_info_end {
		return ServerVersion{}, fmt.Errorf("unexpected server version end marker %d", end)
	}
	return parseServerVersion(raw)
}
