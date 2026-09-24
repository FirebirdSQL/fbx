package fbconn

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"rdb.red-soft.ru/fbx/fbproto"
)

// LimboTransactionAction selects how to finish a prepared transaction.
type LimboTransactionAction uint8

const (
	LimboCommit LimboTransactionAction = iota + 1
	LimboRollback
	// LimboRecover uses the states of the other participants of a two-phase commit.
	LimboRecover
)

// LimboTransactions returns the top-level IDs of prepared transactions, including
// 64-bit IDs. Participant IDs in distributed transactions are not returned as
// separate transactions. An error returns no list, even after partial output.
func (c *ServiceConn) LimboTransactions(ctx context.Context, database string) ([]int64, error) {
	spb, err := newMaintenanceSPB(fbproto.Isc_action_svc_repair, database)
	if err != nil {
		return nil, err
	}
	spb.AppendByte(fbproto.Isc_spb_options).AppendInt32(fbproto.Isc_spb_rpr_list_limbo_trans)
	if err := c.startServiceAction(ctx, spb.Bytes()); err != nil {
		return nil, fmt.Errorf("list limbo transactions: %w", err)
	}
	var output limboTransactionWriter
	if err := c.copyServiceData(ctx, &output, fbproto.Isc_info_svc_limbo_trans); err != nil {
		c.conn.asyncClose()
		return nil, fmt.Errorf("list limbo transactions: %w", err)
	}
	if len(output.pending) != 0 {
		c.conn.asyncClose()
		return nil, errors.New("list limbo transactions: truncated transaction information")
	}
	return output.ids, nil
}

// ResolveLimboTransaction finishes a prepared transaction. Small IDs use the
// 32-bit SPB tags for older servers; larger IDs require 64-bit transaction support.
// A successful service request does not guarantee the outcome of automatic
// recovery; inspect Output and query LimboTransactions again for unresolved IDs.
// Output and cancellation behave as in Repair.
func (c *ServiceConn) ResolveLimboTransaction(ctx context.Context, database string, id int64, action LimboTransactionAction, output io.Writer) error {
	spb, err := buildResolveLimboActionSPB(database, id, action)
	if err != nil {
		return err
	}
	if err := c.runServiceAction(ctx, spb, output); err != nil {
		return fmt.Errorf("resolve limbo transaction: %w", err)
	}
	return nil
}

func buildResolveLimboActionSPB(database string, id int64, action LimboTransactionAction) ([]byte, error) {
	spb, err := newMaintenanceSPB(fbproto.Isc_action_svc_repair, database)
	if err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, errors.New("limbo transaction ID must be positive")
	}
	var tag32, tag64 byte
	switch action {
	case LimboCommit:
		tag32, tag64 = fbproto.Isc_spb_rpr_commit_trans, fbproto.Isc_spb_rpr_commit_trans_64
	case LimboRollback:
		tag32, tag64 = fbproto.Isc_spb_rpr_rollback_trans, fbproto.Isc_spb_rpr_rollback_trans_64
	case LimboRecover:
		tag32, tag64 = fbproto.Isc_spb_rpr_recover_two_phase, fbproto.Isc_spb_rpr_recover_two_phase_64
	default:
		return nil, errors.New("invalid limbo transaction action")
	}
	// The server renders 32-bit clumplets as signed integers in gfix arguments.
	if id <= math.MaxInt32 {
		spb.AppendByte(tag32).AppendInt32(int32(id))
	} else {
		spb.AppendByte(tag64).PutInt64(id)
	}
	return spb.Bytes(), nil
}

// Each info field can span multiple service query replies. Keep at most one
// incomplete field, rather than buffering the entire binary service output.
type limboTransactionWriter struct {
	pending []byte
	ids     []int64
}

func (w *limboTransactionWriter) Write(p []byte) (int, error) {
	w.pending = append(w.pending, p...)
	for len(w.pending) > 0 {
		tag := w.pending[0]
		length := 0
		switch tag {
		case fbproto.Isc_spb_single_tra_id, fbproto.Isc_spb_multi_tra_id, fbproto.Isc_spb_tra_id:
			length = 5
		case fbproto.Isc_spb_single_tra_id_64, fbproto.Isc_spb_multi_tra_id_64, fbproto.Isc_spb_tra_id_64:
			length = 9
		case fbproto.Isc_spb_tra_state, fbproto.Isc_spb_tra_advise:
			length = 2
		case fbproto.Isc_spb_tra_host_site, fbproto.Isc_spb_tra_remote_site, fbproto.Isc_spb_tra_db_path:
			if len(w.pending) < 3 {
				return len(p), nil
			}
			length = 3 + int(binary.LittleEndian.Uint16(w.pending[1:3]))
		default:
			return 0, fmt.Errorf("unknown limbo information tag %d", tag)
		}
		if len(w.pending) < length {
			return len(p), nil
		}
		switch tag {
		case fbproto.Isc_spb_single_tra_id, fbproto.Isc_spb_multi_tra_id:
			w.ids = append(w.ids, int64(binary.LittleEndian.Uint32(w.pending[1:5])))
		case fbproto.Isc_spb_single_tra_id_64, fbproto.Isc_spb_multi_tra_id_64:
			id := binary.LittleEndian.Uint64(w.pending[1:9])
			if id > math.MaxInt64 {
				return 0, errors.New("limbo transaction ID exceeds int64")
			}
			w.ids = append(w.ids, int64(id))
		}
		w.pending = w.pending[length:]
	}
	return len(p), nil
}
