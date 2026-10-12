package fbconn

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"rdb.red-soft.ru/fbx/fbproto"
)

const (
	initialServiceOutputBufferLength = 1024
	maxServiceOutputBufferLength     = 16 * 1024 * 1024
)

func (c *ServiceConn) runServiceAction(ctx context.Context, actionSPB []byte, output io.Writer) error {
	if err := c.startServiceAction(ctx, actionSPB); err != nil {
		return err
	}
	if output == nil {
		output = io.Discard
	}
	return c.copyServiceOutput(ctx, output)
}

func (c *ServiceConn) copyServiceOutput(ctx context.Context, output io.Writer) error {
	return c.copyServiceData(ctx, output, fbproto.Isc_info_svc_to_eof)
}

func (c *ServiceConn) copyServiceData(ctx context.Context, output io.Writer, item byte) error {
	sendItems := buildServiceOutputSendItems()
	receiveItems := []byte{item}
	bufferLength := initialServiceOutputBufferLength

	for {
		response, err := c.queryService(ctx, sendItems, receiveItems, bufferLength)
		if err != nil {
			return err
		}

		chunk, state, err := parseServiceOutput(response, item)
		if err != nil {
			c.conn.asyncClose()
			return fmt.Errorf("parse service output: %w", err)
		}

		switch state {
		case serviceOutputData:
			if err := writeServiceOutput(ctx, output, chunk); err != nil {
				return err
			}
		case serviceOutputTruncated:
			if bufferLength >= maxServiceOutputBufferLength {
				return fmt.Errorf("service output exceeds the maximum response buffer size of %d bytes",
					maxServiceOutputBufferLength)
			}
			bufferLength *= 2
			if bufferLength > maxServiceOutputBufferLength {
				bufferLength = maxServiceOutputBufferLength
			}
		case serviceOutputComplete:
			c.actionActive = false
			return nil
		case serviceOutputTimeout:
			continue
		}
	}
}

func writeServiceOutput(ctx context.Context, output io.Writer, chunk []byte) error {
	select {
	case <-ctx.Done():
		return newContextAlreadyDoneError(ctx)
	default:
	}

	written, err := output.Write(chunk)
	if err != nil {
		return fmt.Errorf("write service output: %w", err)
	}
	if written != len(chunk) {
		return fmt.Errorf("write service output: %w", io.ErrShortWrite)
	}
	return nil
}

func buildServiceOutputSendItems() []byte {
	return fbproto.NewXPBWriter().
		AppendByte(fbproto.Isc_info_svc_timeout).
		AppendInt16(4).
		AppendInt32(1).
		AppendByte(fbproto.Isc_info_end).
		Bytes()
}

type serviceOutputState byte

const (
	serviceOutputData serviceOutputState = iota
	serviceOutputTimeout
	serviceOutputTruncated
	serviceOutputComplete
)

func parseServiceOutput(response []byte, item byte) ([]byte, serviceOutputState, error) {
	if len(response) == 0 {
		return nil, 0, errors.New("empty service output response")
	}

	switch response[0] {
	case item:
		if len(response) < 3 {
			return nil, 0, errors.New("truncated service output length")
		}
		dataLength := int(binary.LittleEndian.Uint16(response[1:3]))
		if dataLength > 0 {
			if len(response) < 3+dataLength {
				return nil, 0, fmt.Errorf(
					"service output declares %d bytes, but only %d are available",
					dataLength,
					len(response)-3,
				)
			}
			return response[3 : 3+dataLength], serviceOutputData, nil
		}
		if len(response) < 4 {
			return nil, 0, errors.New("service output response has no end marker")
		}
		switch response[3] {
		case fbproto.Isc_info_svc_timeout, fbproto.Isc_info_data_not_ready:
			return nil, serviceOutputTimeout, nil
		case fbproto.Isc_info_end:
			return nil, serviceOutputComplete, nil
		default:
			return nil, 0, fmt.Errorf("unexpected service output marker %d", response[3])
		}
	case fbproto.Isc_info_truncated:
		return nil, serviceOutputTruncated, nil
	case fbproto.Isc_info_end:
		return nil, serviceOutputComplete, nil
	default:
		return nil, 0, fmt.Errorf("unexpected service output item %d", response[0])
	}
}
