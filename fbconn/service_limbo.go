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

// LimboTransactionState is a participant state reported by the server.
type LimboTransactionState uint8

const (
	// LimboStateUnknown means the state was omitted or reported as unavailable.
	LimboStateUnknown LimboTransactionState = iota
	LimboStatePrepared
	LimboStateCommitted
	LimboStateRolledBack
)

// LimboTransactionAdvice is the server's recommendation for automatic recovery.
// It does not confirm that recovery has been performed.
type LimboTransactionAdvice uint8

const (
	// LimboAdviceUnknown means no commit or rollback recommendation is available.
	LimboAdviceUnknown LimboTransactionAdvice = iota
	LimboAdviceCommit
	LimboAdviceRollback
)

// LimboParticipant describes one database in a distributed transaction.
// A missing ID is zero; missing site names and paths are empty strings.
// State is LimboStateUnknown when the server does not report a state.
type LimboParticipant struct {
	ID           int64
	State        LimboTransactionState
	HostSite     string
	RemoteSite   string
	DatabasePath string
}

// LimboTransaction describes a top-level prepared transaction.
// Distributed distinguishes a multidatabase transaction from a single one.
// Participants and Advice contain only information reported by the server;
// missing states and advice remain unknown, without inferring an outcome.
type LimboTransaction struct {
	ID           int64
	Distributed  bool
	Participants []LimboParticipant
	Advice       LimboTransactionAdvice
}

// LimboTransactions returns the top-level IDs of prepared transactions, including
// 64-bit IDs. Participant IDs in distributed transactions are not returned as
// separate transactions. An error returns no list, even after partial output.
func (c *ServiceConn) LimboTransactions(ctx context.Context, database string) ([]int64, error) {
	transactions, err := c.LimboTransactionDetails(ctx, database)
	if err != nil || len(transactions) == 0 {
		return nil, err
	}
	ids := make([]int64, len(transactions))
	for i, transaction := range transactions {
		ids[i] = transaction.ID
	}
	return ids, nil
}

// LimboTransactionDetails returns details of all top-level prepared transactions,
// including participants and the server's recovery recommendation when available.
// IDs may be 32 or 64 bits. An error returns no list, even after partial output.
func (c *ServiceConn) LimboTransactionDetails(ctx context.Context, database string) ([]LimboTransaction, error) {
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
	return output.transactions, nil
}

// ResolveLimboTransaction finishes a prepared transaction. Small IDs use the
// 32-bit SPB tags for older servers; larger IDs require 64-bit transaction support.
// A nil error only confirms that the service action completed without an error.
// Automatic recovery can leave a transaction unresolved; inspect Output and
// query LimboTransactions again for unresolved IDs.
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
	pending          []byte
	transactions     []LimboTransaction
	participantPhase limboParticipantPhase
	adviceSeen       bool
}

// The server emits each participant's optional fields in this order. A return
// to an earlier field starts another participant, even if its ID is omitted.
type limboParticipantPhase uint8

const (
	limboParticipantNone limboParticipantPhase = iota
	limboParticipantHost
	limboParticipantID
	limboParticipantState
	limboParticipantRemote
	limboParticipantPath
)

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
		if err := w.readField(tag, w.pending[:length]); err != nil {
			return 0, err
		}
		w.pending = w.pending[length:]
	}
	return len(p), nil
}

func (w *limboTransactionWriter) readField(tag byte, field []byte) error {
	switch tag {
	case fbproto.Isc_spb_single_tra_id, fbproto.Isc_spb_multi_tra_id,
		fbproto.Isc_spb_single_tra_id_64, fbproto.Isc_spb_multi_tra_id_64:
		id, err := readLimboTransactionID(field)
		if err != nil {
			return err
		}
		w.transactions = append(w.transactions, LimboTransaction{
			ID:          id,
			Distributed: tag == fbproto.Isc_spb_multi_tra_id || tag == fbproto.Isc_spb_multi_tra_id_64,
		})
		w.participantPhase, w.adviceSeen = limboParticipantNone, false
		return nil
	}
	if len(w.transactions) == 0 || !w.transactions[len(w.transactions)-1].Distributed {
		return errors.New("limbo participant information without a distributed transaction")
	}
	if w.adviceSeen {
		return errors.New("limbo participant information after recovery advice")
	}
	switch tag {
	case fbproto.Isc_spb_tra_id, fbproto.Isc_spb_tra_id_64:
		id, err := readLimboTransactionID(field)
		if err != nil {
			return err
		}
		w.participant(limboParticipantID).ID = id
	case fbproto.Isc_spb_tra_host_site:
		w.participant(limboParticipantHost).HostSite = string(field[3:])
	case fbproto.Isc_spb_tra_remote_site:
		w.participant(limboParticipantRemote).RemoteSite = string(field[3:])
	case fbproto.Isc_spb_tra_db_path:
		w.participant(limboParticipantPath).DatabasePath = string(field[3:])
	case fbproto.Isc_spb_tra_state:
		var state LimboTransactionState
		switch field[1] {
		case fbproto.Isc_spb_tra_state_limbo:
			state = LimboStatePrepared
		case fbproto.Isc_spb_tra_state_commit:
			state = LimboStateCommitted
		case fbproto.Isc_spb_tra_state_rollback:
			state = LimboStateRolledBack
		case fbproto.Isc_spb_tra_state_unknown:
			state = LimboStateUnknown
		default:
			return fmt.Errorf("unknown limbo participant state %d", field[1])
		}
		w.participant(limboParticipantState).State = state
	case fbproto.Isc_spb_tra_advise:
		var advice LimboTransactionAdvice
		switch field[1] {
		case fbproto.Isc_spb_tra_advise_commit:
			advice = LimboAdviceCommit
		case fbproto.Isc_spb_tra_advise_rollback:
			advice = LimboAdviceRollback
		case fbproto.Isc_spb_tra_advise_unknown:
			advice = LimboAdviceUnknown
		default:
			return fmt.Errorf("unknown limbo recovery advice %d", field[1])
		}
		w.transactions[len(w.transactions)-1].Advice = advice
		w.adviceSeen = true
	}
	return nil
}

func (w *limboTransactionWriter) participant(phase limboParticipantPhase) *LimboParticipant {
	transaction := &w.transactions[len(w.transactions)-1]
	if w.participantPhase == limboParticipantNone || phase <= w.participantPhase {
		transaction.Participants = append(transaction.Participants, LimboParticipant{})
	}
	w.participantPhase = phase
	return &transaction.Participants[len(transaction.Participants)-1]
}

func readLimboTransactionID(field []byte) (int64, error) {
	if len(field) == 5 {
		return int64(binary.LittleEndian.Uint32(field[1:])), nil
	}
	id := binary.LittleEndian.Uint64(field[1:])
	if id > math.MaxInt64 {
		return 0, errors.New("limbo transaction ID exceeds int64")
	}
	return int64(id), nil
}
