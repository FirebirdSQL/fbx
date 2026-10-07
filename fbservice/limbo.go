package fbservice

import (
	"context"
	"errors"
	"fmt"

	"rdb.red-soft.ru/fbx/fbconn"
)

// LimboTransaction describes a top-level prepared transaction and its participants.
type LimboTransaction = fbconn.LimboTransaction

// LimboParticipant contains the ID, state, sites and database path reported for a
// participant. Missing states are unknown; missing IDs and strings are zero values.
type LimboParticipant = fbconn.LimboParticipant

// LimboTransactionState is a participant state reported by the server.
type LimboTransactionState = fbconn.LimboTransactionState

const (
	// LimboStateUnknown means the state was omitted or reported as unavailable.
	LimboStateUnknown    = fbconn.LimboStateUnknown
	LimboStatePrepared   = fbconn.LimboStatePrepared
	LimboStateCommitted  = fbconn.LimboStateCommitted
	LimboStateRolledBack = fbconn.LimboStateRolledBack
)

// LimboTransactionAdvice is the server's recommendation for automatic recovery.
// It does not confirm that recovery has been performed.
type LimboTransactionAdvice = fbconn.LimboTransactionAdvice

const (
	// LimboAdviceUnknown means no commit or rollback recommendation is available.
	LimboAdviceUnknown  = fbconn.LimboAdviceUnknown
	LimboAdviceCommit   = fbconn.LimboAdviceCommit
	LimboAdviceRollback = fbconn.LimboAdviceRollback
)

// LimboTransactionDetails returns details of all transactions in limbo in database.
// Distributed transactions include participants and recovery advice when reported
// by the server. Missing participant states and advice remain unknown; the driver
// does not infer commit or rollback. IDs may be 32 or 64 bits.
// On error no list is returned. The service connection is closed before return.
func (m *Manager) LimboTransactionDetails(ctx context.Context, database string) (transactions []LimboTransaction, err error) {
	if err := validateMaintenanceDatabase(database); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	conn, err := m.open(ctx, database)
	if err != nil {
		return nil, fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() {
		err = errors.Join(err, closeService(conn))
		if err != nil {
			transactions = nil
		}
	}()
	return conn.LimboTransactionDetails(ctx, database)
}
