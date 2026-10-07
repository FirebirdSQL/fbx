package fbservice

import (
	"context"
	"errors"
	"fmt"
	"io"

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

// LimboResolutionStatus describes the result of checking the limbo list after
// a service action. It does not identify whether a transaction committed or
// rolled back.
type LimboResolutionStatus uint8

const (
	// LimboResolutionUnknown means the action or the subsequent check failed.
	LimboResolutionUnknown LimboResolutionStatus = iota
	// LimboResolutionNotListed means the ID is absent from the returned limbo
	// list. A server may limit the list, so absence does not prove completion.
	LimboResolutionNotListed
	// LimboResolutionUnresolved means the ID is still in the database's limbo list.
	LimboResolutionUnresolved
)

// LimboResolution is the result of a checked limbo transaction service action.
// Transaction contains the remaining transaction details only when Status is
// LimboResolutionUnresolved. On error Status is LimboResolutionUnknown and
// Transaction is nil, even if the server may have completed the action.
type LimboResolution struct {
	Status      LimboResolutionStatus
	Transaction *LimboTransaction
}

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

// ResolveLimboTransactionChecked runs [Manager.ResolveLimboTransaction] once,
// then checks [Manager.LimboTransactionDetails] using a new service connection.
// A nil error means both calls completed; inspect Status to see whether the ID
// remains in limbo. An unresolved transaction is returned with its details.
// The check is a snapshot of the specified database's reported limbo list.
// Servers may limit that list without signalling truncation. NotListed therefore
// does not prove commit, rollback or completion in this or other databases.
//
// If the action or check fails, the result is unknown and the error is returned.
// An error does not undo server effects or authorize repeating the action.
// Errors preserve their causes but do not permit retrying the whole call.
// Both calls use ctx, and each service connection is closed before return.
// Output and cancellation behave as in [Manager.ResolveLimboTransaction].
func (m *Manager) ResolveLimboTransactionChecked(ctx context.Context, database string, id int64, action LimboTransactionAction, output io.Writer) (LimboResolution, error) {
	if err := m.ResolveLimboTransaction(ctx, database, id, action, output); err != nil {
		return LimboResolution{}, &limboResolutionError{err: err}
	}
	transactions, err := m.LimboTransactionDetails(ctx, database)
	if err != nil {
		return LimboResolution{}, &limboResolutionError{
			err: fmt.Errorf("check limbo transaction resolution: %w", err),
		}
	}
	for _, transaction := range transactions {
		if transaction.ID == id {
			return LimboResolution{
				Status:      LimboResolutionUnresolved,
				Transaction: &transaction,
			}, nil
		}
	}
	return LimboResolution{Status: LimboResolutionNotListed}, nil
}

// Resolve, cleanup and verification form one call. An unsent cleanup or list
// request cannot establish that the action is safe to repeat.
type limboResolutionError struct {
	err error
}

func (e *limboResolutionError) Error() string {
	return e.err.Error()
}

func (e *limboResolutionError) Unwrap() error {
	return e.err
}

func (e *limboResolutionError) SafeToRetry() bool {
	return false
}
