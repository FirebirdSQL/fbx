package fbservice_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/fbservice"
	"github.com/stretchr/testify/require"
)

func TestManagerResolveLimboTransactionChecked(t *testing.T) {
	remaining := fbservice.LimboTransaction{
		ID: math.MaxInt64, Distributed: true,
		Participants: []fbservice.LimboParticipant{
			{ID: 7, State: fbservice.LimboStatePrepared, DatabasePath: "/data/one.fdb"},
			{ID: 8, DatabasePath: "/data/unavailable.fdb"},
		},
	}
	for name, tt := range map[string]struct {
		id           int64
		transactions []fbservice.LimboTransaction
		want         fbservice.LimboResolution
	}{
		"empty list": {id: 42, want: fbservice.LimboResolution{Status: fbservice.LimboResolutionNotListed}},
		"participant ID is not a top-level ID": {
			id: 42,
			transactions: []fbservice.LimboTransaction{{
				ID: 100, Distributed: true,
				Participants: []fbservice.LimboParticipant{{ID: 42, State: fbservice.LimboStatePrepared}},
			}},
			want: fbservice.LimboResolution{Status: fbservice.LimboResolutionNotListed},
		},
		"single transaction remains": {
			id: 42, transactions: []fbservice.LimboTransaction{{ID: 1}, {ID: 42}},
			want: fbservice.LimboResolution{Status: fbservice.LimboResolutionUnresolved, Transaction: &fbservice.LimboTransaction{ID: 42}},
		},
		"distributed transaction with unknown state remains": {
			id: math.MaxInt64, transactions: []fbservice.LimboTransaction{{ID: 1}, remaining, {ID: 2}},
			want: fbservice.LimboResolution{Status: fbservice.LimboResolutionUnresolved, Transaction: &remaining},
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, action := range []fbservice.LimboTransactionAction{fbservice.LimboCommit, fbservice.LimboRollback, fbservice.LimboRecover} {
				manager, operation, check, events := newLimboResolutionManager(t, nil)
				check.limboDetails = tt.transactions
				var output bytes.Buffer
				ctx := context.WithValue(context.Background(), limboResolutionContextKey{}, name)
				got, err := manager.ResolveLimboTransactionChecked(ctx, "db", tt.id, action, &output)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
				require.Equal(t, tt.id, operation.transactionID)
				require.Equal(t, fbconn.LimboTransactionAction(action), operation.limboAction)
				require.Same(t, &output, operation.limboOutput)
				require.Equal(t, "service action completed\n", output.String())
				require.Equal(t, ctx, operation.operationContext)
				require.Equal(t, ctx, check.operationContext)
				require.Equal(t, "db", check.database)
				require.Equal(t, []string{"open action", "resolve", "close action", "open check", "check", "close check"}, *events)
			}
		})
	}

	t.Run("optional output", func(t *testing.T) {
		for _, action := range []fbservice.LimboTransactionAction{fbservice.LimboCommit, fbservice.LimboRollback} {
			manager, operation, _, _ := newLimboResolutionManager(t, nil)
			result, err := manager.ResolveLimboTransactionChecked(context.Background(), "db", 42, action, nil)
			require.NoError(t, err)
			require.Equal(t, fbservice.LimboResolutionNotListed, result.Status)
			require.Nil(t, operation.limboOutput)
		}
	})
	t.Run("explicitly discarded recovery report", func(t *testing.T) {
		manager, operation, _, events := newLimboResolutionManager(t, nil)
		result, err := manager.ResolveLimboTransactionChecked(context.Background(), "db", 42, fbservice.LimboRecover, io.Discard)
		require.NoError(t, err)
		require.Equal(t, fbservice.LimboResolutionNotListed, result.Status)
		require.Equal(t, io.Discard, operation.limboOutput)
		require.Equal(t, []string{"open action", "resolve", "close action", "open check", "check", "close check"}, *events)
	})
	t.Run("missing recovery report", func(t *testing.T) {
		manager, _, _, events := newLimboResolutionManager(t, nil)
		result, err := manager.ResolveLimboTransactionChecked(context.Background(), "db", 42, fbservice.LimboRecover, nil)
		require.ErrorContains(t, err, "output must not be nil")
		require.Equal(t, fbservice.LimboResolution{}, result)
		require.Empty(t, *events, "neither the action nor the check should open a connection")
	})
}

func TestManagerResolveLimboTransactionCheckedErrors(t *testing.T) {
	actionErr, closeErr := errors.New("action failed"), errors.New("detach failed")
	checkErr := retryableLimboServiceError{error: errors.New("check was not sent")}
	cleanupErr := retryableLimboServiceError{error: errors.New("detach was not sent")}
	require.True(t, fbconn.SafeToRetry(checkErr))
	require.True(t, fbconn.SafeToRetry(cleanupErr))
	for name, tt := range map[string]struct {
		actionErr   error
		actionClose error
		checkOpen   error
		checkErr    error
		checkClose  error
		emptyList   bool
		wantEvents  []string
	}{
		"action error and cleanup error": {
			actionErr: actionErr, actionClose: closeErr,
			wantEvents: []string{"open action", "resolve", "close action"},
		},
		"action cleanup error": {
			actionClose: cleanupErr,
			wantEvents:  []string{"open action", "resolve", "close action"},
		},
		"check connection error": {
			checkOpen:  checkErr,
			wantEvents: []string{"open action", "resolve", "close action", "open check"},
		},
		"check error and cleanup error": {
			checkErr: checkErr, checkClose: closeErr,
			wantEvents: []string{"open action", "resolve", "close action", "open check", "check", "close check"},
		},
		"remaining details with cleanup error": {
			checkClose: closeErr,
			wantEvents: []string{"open action", "resolve", "close action", "open check", "check", "close check"},
		},
		"empty list with cleanup error": {
			checkClose: closeErr, emptyList: true,
			wantEvents: []string{"open action", "resolve", "close action", "open check", "check", "close check"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			manager, operation, check, events := newLimboResolutionManager(t, tt.checkOpen)
			operation.maintenanceErr, operation.closeErr = tt.actionErr, tt.actionClose
			check.maintenanceErr, check.closeErr = tt.checkErr, tt.checkClose
			if !tt.emptyList {
				check.limboDetails = []fbservice.LimboTransaction{{ID: 42}}
			}
			result, err := manager.ResolveLimboTransactionChecked(context.Background(), "db", 42, fbservice.LimboRecover, io.Discard)
			require.Error(t, err)
			require.Equal(t, fbservice.LimboResolution{}, result)
			require.Equal(t, fbservice.LimboResolutionUnknown, result.Status)
			for _, cause := range []error{tt.actionErr, tt.actionClose, tt.checkOpen, tt.checkErr, tt.checkClose} {
				if cause != nil {
					require.ErrorIs(t, err, cause)
				}
			}
			require.False(t, fbconn.SafeToRetry(err), "a cleanup or check error must not permit repeating the action")
			require.Equal(t, tt.wantEvents, *events, "the action must not be retried after an uncertain result")
		})
	}
}

func TestManagerResolveLimboTransactionCheckedCancellation(t *testing.T) {
	for _, phase := range []string{"before action", "after action", "during check"} {
		t.Run(phase, func(t *testing.T) {
			manager, operation, check, events := newLimboResolutionManager(t, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var wantEvents []string
			switch phase {
			case "before action":
				cancel()
			case "after action":
				operation.cancelOnCall = cancel
				wantEvents = []string{"open action", "resolve", "close action"}
			case "during check":
				check.cancelOnCall = cancel
				wantEvents = []string{"open action", "resolve", "close action", "open check", "check", "close check"}
			}
			result, err := manager.ResolveLimboTransactionChecked(ctx, "db", 42, fbservice.LimboRecover, io.Discard)
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, fbservice.LimboResolution{}, result)
			require.Equal(t, wantEvents, *events)
			for _, connection := range []*limboResolutionConnection{operation, check} {
				if connection.closed {
					require.NoError(t, connection.closeContextErr)
					require.True(t, connection.closeHasDeadline)
				}
			}
		})
	}
}

type limboResolutionContextKey struct{}

type retryableLimboServiceError struct {
	error
}

func (retryableLimboServiceError) SafeToRetry() bool {
	return true
}

type limboResolutionConnection struct {
	fakeServiceConnection
	name             string
	events           *[]string
	operationContext context.Context
	cancelOnCall     context.CancelFunc
	closeContextErr  error
	closeHasDeadline bool
}

func newLimboResolutionManager(t *testing.T, checkOpenErr error) (*fbservice.Manager, *limboResolutionConnection, *limboResolutionConnection, *[]string) {
	t.Helper()
	manager, _ := newTestManager(t)
	var events []string
	operation := &limboResolutionConnection{name: "action", events: &events}
	check := &limboResolutionConnection{name: "check", events: &events}
	opened := 0
	fbservice.SetConnectForTest(manager, func(_ context.Context, config *fbconn.ServiceConfig) (fbservice.ServiceConnection, error) {
		opened++
		require.Equal(t, "db", config.ExpectedDatabase)
		switch opened {
		case 1:
			events = append(events, "open action")
			return operation, nil
		case 2:
			require.True(t, operation.closed, "close the action connection before checking")
			events = append(events, "open check")
			if checkOpenErr != nil {
				return nil, checkOpenErr
			}
			return check, nil
		default:
			t.Fatal("unexpected retry or extra service connection")
			return nil, errors.New("unexpected service connection")
		}
	})
	return manager, operation, check, &events
}

func (c *limboResolutionConnection) ResolveLimboTransaction(ctx context.Context, database string, id int64, action fbconn.LimboTransactionAction, output io.Writer) error {
	*c.events = append(*c.events, "resolve")
	c.operationContext = ctx
	err := c.fakeServiceConnection.ResolveLimboTransaction(ctx, database, id, action, output)
	if c.cancelOnCall != nil {
		c.cancelOnCall()
	}
	if err == nil && output != nil {
		_, err = io.WriteString(output, "service action completed\n")
	}
	return err
}

func (c *limboResolutionConnection) LimboTransactionDetails(ctx context.Context, database string) ([]fbconn.LimboTransaction, error) {
	*c.events = append(*c.events, "check")
	c.operationContext = ctx
	if c.cancelOnCall != nil {
		c.cancelOnCall()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return c.fakeServiceConnection.LimboTransactionDetails(ctx, database)
}

func (c *limboResolutionConnection) Close(ctx context.Context) error {
	*c.events = append(*c.events, "close "+c.name)
	c.closeContextErr = ctx.Err()
	_, c.closeHasDeadline = ctx.Deadline()
	return c.fakeServiceConnection.Close(ctx)
}
