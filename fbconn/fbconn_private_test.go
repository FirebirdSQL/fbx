package fbconn

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandTag(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		queryStatus QueryStatus
		isInsert    bool
		isUpdate    bool
		isDelete    bool
		isSelect    bool
		isDDL       bool
	}{
		{queryStatus: QueryStatus{statementType: StatementInsert}, isInsert: true},
		{queryStatus: QueryStatus{statementType: StatementSelect}, isSelect: true},
		{queryStatus: QueryStatus{statementType: StatementUpdate}, isUpdate: true},
		{queryStatus: QueryStatus{statementType: StatementDelete}, isDelete: true},
		{queryStatus: QueryStatus{statementType: StatementDDL}, isDDL: true},
	}

	for i, tt := range tests {
		ct := tt.queryStatus
		assert.Equalf(t, tt.isInsert, ct.Insert(), "%d. %v", i, tt.queryStatus)
		assert.Equalf(t, tt.isUpdate, ct.Update(), "%d. %v", i, tt.queryStatus)
		assert.Equalf(t, tt.isDelete, ct.Delete(), "%d. %v", i, tt.queryStatus)
		assert.Equalf(t, tt.isSelect, ct.Select(), "%d. %v", i, tt.queryStatus)
	}
}

func TestBatchCloseCanceledContextKeepsBatchOpen(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprintf("deadline=%t", deadline), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			if deadline {
				cancel()
				ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			}
			cancel()

			conn := &FbConn{status: connStatusIdle}
			st := &Statement{conn: conn}
			b := &Batch{st: st, state: batchStateCreated}
			st.activeBatch = b

			err := b.Close(ctx)

			require.ErrorIs(t, err, ctx.Err())
			require.True(t, SafeToRetry(err))
			require.False(t, b.isClosed())
			require.Same(t, b, st.activeBatch)
			require.False(t, conn.IsClosed())
		})
	}
}

func newBatchProtocolTest(t *testing.T, input io.Reader) (*Batch, *prepareCleanupTestConn) {
	t.Helper()
	transport := &prepareCleanupTestConn{}
	conn := newDetachTestConn(transport)
	conn.protocolVersion = fbproto.PROTOCOL_VERSION17
	encoding, err := fbproto.NewEncoding("UTF8")
	require.NoError(t, err)
	conn.client = fbproto.NewClient(input, conn.writer, encoding)
	st := &Statement{conn: conn, Description: &StatementDescription{
		StatementHandle: 42,
		StatementInfo: &StatementInfo{
			ParametersDescriptors: []fbproto.FieldDescriptor{{Type: fbproto.SQL_TYPE_LONG, Length: 4}},
		},
	}}
	b := st.NewBatch(BatchConfig{RecordCounts: true})
	b.state = batchStateCreated
	t.Cleanup(func() {
		conn.asyncClose()
		waitCleanupDone(t, conn)
	})
	return b, transport
}

func TestBatchSyncResponses(t *testing.T) {
	for _, mode := range []string{"server error", "sync error", "broken response"} {
		t.Run(mode, func(t *testing.T) {
			var input bytes.Buffer
			w := fbproto.NewXDRWriter(&input)
			broken := mode == "broken response"
			if broken {
				require.NoError(t, w.WriteInt32(fbproto.Op_sql_response))
				require.NoError(t, w.WriteInt32(0))
			} else if mode == "server error" {
				require.NoError(t, retryTestResponse(w, fbproto.Isc_deadlock))
			} else {
				require.NoError(t, writeProbeResponse(w, 0))
			}
			if mode == "sync error" {
				require.NoError(t, retryTestResponse(w, fbproto.Isc_deadlock))
			} else {
				require.NoError(t, writeProbeResponse(w, 0))
			}
			require.NoError(t, writeProbeResponse(w, 77))
			b, _ := newBatchProtocolTest(t, &input)
			conn := b.st.conn
			var resets []int
			h := &CancelRequestContextWatcherHandler{Conn: conn, sendCancelOperationFunc: func(_ context.Context, kind int) error {
				resets = append(resets, kind)
				return nil
			}}
			h.pendingCancelReset.Store(true)
			conn.contextHandler = h

			err := b.Send(context.Background(), [][]byte{{0, 0, 0, 1}})
			require.Error(t, err)
			require.True(t, b.NeedsReset())
			require.True(t, b.HasPending())
			require.False(t, SafeToRetry(err))
			require.Equal(t, []int{fbproto.Fb_cancel_disable, fbproto.Fb_cancel_enable}, resets)
			require.Equal(t, broken, conn.IsClosed())
			if broken {
				require.Equal(t, 2*24, input.Len(), "do not read another response after closing the connection")
				return
			}
			var fbErr *FbError
			require.ErrorAs(t, err, &fbErr)
			require.Contains(t, fbErr.GdsCodes, fbproto.Isc_deadlock)
			msg, err := conn.receiveMessage()
			require.NoError(t, err)
			require.Equal(t, 77, msg.(*fbproto.OpResponse).Object, "the sync response must be consumed")

			require.NoError(t, retryTestResponse(w, fbproto.Isc_cancelled))
			require.NoError(t, writeProbeResponse(w, 0))
			require.Error(t, b.Cancel(context.Background()))
			require.True(t, b.NeedsReset())
			require.Same(t, b, b.st.activeBatch, "failed cleanup must retain the handle")
			require.NoError(t, writeProbeResponse(w, 0))
			require.NoError(t, writeProbeResponse(w, 0))
			require.NoError(t, b.Cancel(context.Background()))
			require.False(t, b.HasPending())
			require.False(t, b.NeedsReset())
			require.False(t, b.isClosed())
		})
	}
}

func TestBatchWriteFailureDisallowsRetry(t *testing.T) {
	t.Run("write error", func(t *testing.T) {
		b, transport := newBatchProtocolTest(t, bytes.NewReader(nil))
		transport.writeErr = &fbconnError{msg: "write failed", safeToRetry: true}

		err := b.Send(context.Background(), [][]byte{{0, 0, 0, 1}})

		require.ErrorIs(t, err, transport.writeErr)
		require.False(t, SafeToRetry(err))
		require.True(t, b.st.conn.IsClosed())
		b.st.conn.writeMu.Lock()
		failed := b.st.conn.writeFailed
		b.st.conn.writeMu.Unlock()
		require.True(t, failed)
	})

	t.Run("canceled during write", func(t *testing.T) {
		client, server := net.Pipe()
		defer func() { _ = server.Close() }()
		conn := newDetachTestConn(client)
		conn.protocolVersion = fbproto.PROTOCOL_VERSION17
		t.Cleanup(func() {
			conn.asyncClose()
			waitCleanupDone(t, conn)
		})
		st := &Statement{conn: conn, Description: &StatementDescription{StatementHandle: 42}}
		b := st.NewBatch(BatchConfig{})
		b.state = batchStateCreated
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- b.Send(ctx, [][]byte{{0, 0, 0, 1}, {0, 0, 0, 2}}) }()

		// Read the header and first row, leaving the second row's write blocked.
		require.NoError(t, server.SetReadDeadline(time.Now().Add(3*time.Second)))
		r := fbproto.NewXDRReader(server)
		for _, want := range []int{fbproto.Op_batch_msg, 42, 2, 1} {
			got, err := r.ReadInt32()
			require.NoError(t, err)
			require.Equal(t, want, got)
		}
		cancel()

		select {
		case err := <-done:
			require.ErrorIs(t, err, context.Canceled)
			require.False(t, SafeToRetry(err))
		case <-time.After(3 * time.Second):
			t.Fatal("batch write did not stop after cancellation")
		}
		require.True(t, conn.IsClosed())
		conn.writeMu.Lock()
		failed := conn.writeFailed
		conn.writeMu.Unlock()
		require.True(t, failed)
	})
}

func TestBatchExecuteLostCommitReplyPreservesCompletion(t *testing.T) {
	var input bytes.Buffer
	w := fbproto.NewXDRWriter(&input)
	// Execution succeeded for both rows, but there is no reply to the following commit.
	for _, n := range []int{fbproto.Op_batch_cs, 42, 2, 2, 0, 0, 1, 1} {
		require.NoError(t, w.WriteInt32(n))
	}
	b, transport := newBatchProtocolTest(t, &input)
	b.bufferState = batchBufferPending
	conn := b.st.conn
	conn.config.Autocommit = true
	tx := &FBTransaction{Connection: conn, Handle: 42}
	conn.internalTransaction = tx
	conn.activeTransactions = map[*FBTransaction]struct{}{tx: {}}

	completion, err := b.Execute(context.Background(), nil)

	var fbErr *FbError
	require.ErrorAs(t, err, &fbErr)
	require.Equal(t, "08006", fbErr.Code)
	require.False(t, SafeToRetry(err))
	require.True(t, conn.IsClosed())
	require.NotNil(t, completion)
	require.EqualValues(t, 2, completion.TotalRecords)
	require.Equal(t, []int64{1, 1}, completion.UpdateCounts)
	require.Empty(t, completion.Errors)
	require.False(t, b.HasPending(), "the completion confirms that the server consumed the queue")

	r := fbproto.NewXDRReader(&transport.written)
	for _, want := range []int{fbproto.Op_batch_exec, 42, 42, fbproto.Op_commit, 42} {
		got, err := r.ReadInt32()
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	require.Zero(t, transport.written.Len(), "do not send rollback after losing the connection")
}

func TestBatchAcknowledgementSurvivesSyncError(t *testing.T) {
	for _, create := range []bool{true, false} {
		t.Run(fmt.Sprintf("create=%t", create), func(t *testing.T) {
			var input bytes.Buffer
			w := fbproto.NewXDRWriter(&input)
			require.NoError(t, writeProbeResponse(w, 0))
			require.NoError(t, retryTestResponse(w, fbproto.Isc_cancelled))
			b, _ := newBatchProtocolTest(t, &input)
			var err error
			if create {
				b.state = batchStateInit
				err = b.Create(context.Background())
				require.Equal(t, batchStateCreated, b.state, "Close must release the acknowledged batch")
			} else {
				err = b.Close(context.Background())
				require.True(t, b.isClosed(), "release already succeeded")
				require.Nil(t, b.st.activeBatch)
			}
			require.Error(t, err)
			require.False(t, b.st.conn.IsClosed())
			require.Zero(t, input.Len())
		})
	}
}

type batchCancelReader struct {
	io.Reader
	cancel context.CancelFunc
}

func (r *batchCancelReader) Read(p []byte) (int, error) {
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	return r.Reader.Read(p)
}

func TestBatchExecuteCancellation(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		for _, rowError := range []bool{false, true} {
			t.Run(fmt.Sprintf("explicit=%t/rowError=%t", explicit, rowError), func(t *testing.T) {
				var input bytes.Buffer
				w := fbproto.NewXDRWriter(&input)
				if rowError {
					for _, n := range []int{fbproto.Op_batch_cs, 42, 2, 2, 1, 0, 1, -1, 1, fbproto.Isc_arg_gds, fbproto.Isc_cancelled, fbproto.Isc_arg_end} {
						require.NoError(t, w.WriteInt32(n))
					}
				} else {
					require.NoError(t, retryTestResponse(w, fbproto.Isc_cancelled))
				}
				if !explicit {
					require.NoError(t, writeProbeResponse(w, 0))
				}
				if !rowError {
					require.NoError(t, writeProbeResponse(w, 0))
					require.NoError(t, writeProbeResponse(w, 0))
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				b, transport := newBatchProtocolTest(t, &batchCancelReader{Reader: &input, cancel: cancel})
				conn := b.st.conn
				conn.config.Autocommit = true
				conn.explicitTransaction = explicit
				tx := &FBTransaction{Connection: conn, Handle: 42}
				conn.internalTransaction = tx
				conn.activeTransactions = map[*FBTransaction]struct{}{tx: {}}

				completion, err := b.Execute(ctx, tx)

				require.ErrorIs(t, err, context.Canceled)
				require.False(t, SafeToRetry(err))
				require.False(t, conn.IsClosed())
				require.Equal(t, explicit, conn.ActiveTx())
				require.Equal(t, !rowError, b.NeedsReset(), "only a completion confirms that execution consumed the queue")
				require.Equal(t, !rowError, b.HasPending())
				if rowError {
					require.NotNil(t, completion)
					require.Equal(t, []int64{1, -1}, completion.UpdateCounts)
					require.Len(t, completion.Errors, 1)
				} else {
					require.Nil(t, completion)
					var fbErr *FbError
					require.ErrorAs(t, err, &fbErr)
					require.Contains(t, fbErr.GdsCodes, fbproto.Isc_cancelled)
				}
				if explicit {
					require.Equal(t, 12, transport.written.Len(), "must not roll back a caller-owned transaction")
				} else {
					require.Equal(t, 20, transport.written.Len())
					require.EqualValues(t, fbproto.Op_rollback, binary.BigEndian.Uint32(transport.written.Bytes()[12:]))
				}
				if !rowError {
					written := transport.written.Len()
					require.ErrorContains(t, b.Send(context.Background(), [][]byte{{0, 0, 0, 1}}), "requires Cancel")
					require.Equal(t, written, transport.written.Len(), "do not append to an uncertain server queue")
					require.NoError(t, b.Cancel(context.Background()))
					require.False(t, b.NeedsReset())
					require.False(t, b.HasPending())
					require.False(t, b.isClosed())
				}
				require.Zero(t, input.Len())
			})
		}
	}
}

func TestBatchPendingSurvivesCanceledOperation(t *testing.T) {
	for _, execute := range []bool{false, true} {
		t.Run(fmt.Sprintf("execute=%t", execute), func(t *testing.T) {
			var input bytes.Buffer
			w := fbproto.NewXDRWriter(&input)
			require.NoError(t, writeProbeResponse(w, 0))
			require.NoError(t, writeProbeResponse(w, 0))
			b, transport := newBatchProtocolTest(t, &input)
			rows := [][]byte{{0, 0, 0, 1}}
			require.NoError(t, b.Send(context.Background(), rows))
			written := transport.written.Len()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			var err error
			if execute {
				_, err = b.Execute(ctx, &FBTransaction{Connection: b.st.conn, Handle: 42})
			} else {
				err = b.Send(ctx, rows)
			}
			require.ErrorIs(t, err, context.Canceled)
			require.True(t, SafeToRetry(err))
			require.True(t, b.HasPending())
			require.False(t, b.NeedsReset())
			require.Equal(t, written, transport.written.Len())
		})
	}
}

func TestBatchExecuteRejectsInvalidReply(t *testing.T) {
	for _, reply := range [][]int{
		{fbproto.Op_batch_cs, 43, 0, 0, 0, 0},
		{fbproto.Op_batch_cs, 42, 1, -1, 0, 0},
		{fbproto.Op_batch_cs, 42, 1},
		{fbproto.Op_response, 0, 0, 0, 0, fbproto.Isc_arg_end},
	} {
		t.Run(fmt.Sprint(reply), func(t *testing.T) {
			var input bytes.Buffer
			for _, n := range reply {
				require.NoError(t, fbproto.NewXDRWriter(&input).WriteInt32(n))
			}
			b, _ := newBatchProtocolTest(t, &input)
			completion, err := b.Execute(context.Background(), &FBTransaction{Connection: b.st.conn, Handle: 42})
			require.Nil(t, completion)
			require.Error(t, err)
			require.False(t, SafeToRetry(err))
			require.True(t, b.st.conn.IsClosed())
		})
	}
}

func TestStatementDeallocateCanceledContextKeepsStatementOpen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	conn := &FbConn{status: connStatusIdle}
	st := &Statement{
		conn: conn,
		Description: &StatementDescription{
			StatementHandle: 1,
		},
	}

	err := st.Deallocate(ctx)

	require.Error(t, err)
	require.False(t, st.closed)
	require.False(t, conn.IsClosed())
}

func TestTransactionEndCanceledContextKeepsTransactionOpen(t *testing.T) {
	for _, tt := range []struct {
		name string
		end  func(context.Context, *FBTransaction) error
	}{
		{name: "commit", end: func(ctx context.Context, tx *FBTransaction) error { return tx.Commit(ctx) }},
		{name: "rollback", end: func(ctx context.Context, tx *FBTransaction) error { return tx.Rollback(ctx) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			conn := &FbConn{
				status:             connStatusIdle,
				activeTransactions: make(map[*FBTransaction]struct{}),
			}
			tx := &FBTransaction{Connection: conn, Handle: 1}
			conn.activeTransactions[tx] = struct{}{}

			err := tt.end(ctx, tx)

			require.Error(t, err)
			require.False(t, tx.Closed)
			require.Contains(t, conn.activeTransactions, tx)
			require.False(t, conn.IsClosed())
		})
	}
}

func TestActiveExplicitTxIgnoresInternalTransaction(t *testing.T) {
	internalTx := &FBTransaction{}
	conn := &FbConn{
		internalTransaction: internalTx,
		explicitTransaction: false,
		activeTransactions:  map[*FBTransaction]struct{}{internalTx: {}},
	}

	require.True(t, conn.ActiveTx())
	require.False(t, conn.ActiveExplicitTx())

	conn.explicitTransaction = true
	require.True(t, conn.ActiveExplicitTx())

	internalTx.Closed = true
	require.False(t, conn.ActiveExplicitTx())
}

func TestAttachArrayMetadataRequiresMetadataForEveryArray(t *testing.T) {
	declared := []map[string]int{{"Lower": 1, "Upper": 4}}
	available := &Array{
		Dimensions:        declared,
		CurrentDimensions: []map[string]int{{"Lower": 1, "Upper": 2}},
	}
	descriptions := []FieldDescription{
		{Type: fbproto.SQL_TYPE_LONG},
		{
			Type:              fbproto.SQL_TYPE_ARRAY,
			OriginalTableName: "T",
			OriginalName:      "KNOWN",
		},
		{
			Type:              fbproto.SQL_TYPE_ARRAY,
			OriginalTableName: "T",
			OriginalName:      "MISSING",
		},
	}
	arrays := map[metadataMapKey]*Array{
		{table: "T", field: "KNOWN"}: available,
	}

	err := attachArrayMetadata(descriptions, arrays, "parameter")

	require.ErrorContains(t, err, "cannot resolve array metadata for parameter 3")
	assert.Nil(t, descriptions[0].ArrayData)
	require.NotNil(t, descriptions[1].ArrayData)
	assert.NotSame(t, available, descriptions[1].ArrayData)
	assert.Equal(t, declared, descriptions[1].ArrayData.Dimensions)
	assert.Equal(t, declared, descriptions[1].ArrayData.CurrentDimensions)
	assert.Nil(t, descriptions[2].ArrayData)

	descriptions[1].ArrayData.Dimensions[0]["Lower"] = 2
	assert.Equal(t, 1, available.Dimensions[0]["Lower"])
}

func TestAttachArrayMetadataCreatesIndependentCopies(t *testing.T) {
	dimensions := []map[string]int{{"Lower": 1, "Upper": 4}}
	descriptions := []FieldDescription{
		{Type: fbproto.SQL_TYPE_ARRAY, OriginalTableName: "T", OriginalName: "A"},
		{Type: fbproto.SQL_TYPE_ARRAY, OriginalTableName: "T", OriginalName: "A"},
	}
	arrays := map[metadataMapKey]*Array{
		{table: "T", field: "A"}: {Dimensions: dimensions, CurrentDimensions: dimensions},
	}

	require.NoError(t, attachArrayMetadata(descriptions, arrays, "parameter"))
	require.NotSame(t, descriptions[0].ArrayData, descriptions[1].ArrayData)

	descriptions[0].ArrayData.CurrentDimensions = []map[string]int{{"Lower": 1, "Upper": 2}}
	assert.Equal(t, dimensions, descriptions[1].ArrayData.CurrentDimensions)
}

func TestGetArrayDimensionsWithNoSources(t *testing.T) {
	dimensions, err := (&FbConn{}).getArrayDimensions(context.Background(), nil, 5, nil)

	require.NoError(t, err)
	assert.Empty(t, dimensions)
}

func TestArraySliceLengthValidatesBoundsAndOverflow(t *testing.T) {
	tests := []struct {
		name              string
		currentDimensions []map[string]int
		length            uint32
		want              int
		wantError         string
	}{
		{
			name:              "negative lower bound",
			currentDimensions: []map[string]int{{"Lower": -1, "Upper": 1}},
			length:            4,
			want:              12,
		},
		{
			name:              "empty slice",
			currentDimensions: []map[string]int{{"Lower": 1, "Upper": 0}},
			length:            4,
		},
		{
			name:              "invalid empty range",
			currentDimensions: []map[string]int{{"Lower": 2, "Upper": 0}},
			length:            4,
			wantError:         "invalid bounds 2:0",
		},
		{
			name:              "byte length exceeds protocol limit",
			currentDimensions: []map[string]int{{"Lower": 1, "Upper": 1 << 30}},
			length:            4,
			wantError:         "byte length exceeds protocol limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			array := Array{CurrentDimensions: tt.currentDimensions, Length: tt.length}
			got, err := array.getSliceLength(tt.currentDimensions)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPutArrayRestoresDimensionsAfterValidationError(t *testing.T) {
	declared := []map[string]int{{"Lower": 1, "Upper": 3}}
	array := &Array{
		Dimensions:        declared,
		CurrentDimensions: []map[string]int{{"Lower": 2, "Upper": 0}},
		Length:            4,
	}

	_, err := putArray(nil, context.Background(), nil, array, nil)
	require.ErrorContains(t, err, "invalid bounds")
	assert.Equal(t, declared, array.CurrentDimensions)
}

func TestArrayGenerateSDLIncludesOctetsCharacterSet(t *testing.T) {
	for _, tt := range []struct {
		name       string
		typeID     uint32
		blrTypeID  uint32
		wantBLRTag byte
	}{
		{name: "fixed", typeID: fbproto.SQL_TYPE_TEXT, blrTypeID: fbproto.Isc_blr_text, wantBLRTag: fbproto.Isc_blr_text2},
		{name: "varying", typeID: fbproto.SQL_TYPE_VARYING, blrTypeID: fbproto.Isc_blr_varying, wantBLRTag: fbproto.Isc_blr_varying2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			array := Array{
				TableName:         "T",
				FieldName:         "F",
				CurrentDimensions: []map[string]int{{"Lower": 1, "Upper": 2}},
				Length:            10,
				BlrTypeId:         tt.blrTypeID,
				TypeId:            tt.typeID,
				SubType:           fbproto.CS_BINARY,
			}

			sdl := array.generateSDL(array.CurrentDimensions)

			require.GreaterOrEqual(t, len(sdl), 9)
			assert.Equal(t, tt.wantBLRTag, sdl[3])
			assert.Equal(t, byte(fbproto.CS_BINARY), sdl[4])
			assert.Equal(t, byte(0), sdl[5])
			assert.Equal(t, []byte{10, 0}, sdl[6:8])
		})
	}
}

func TestArrayGenerateSDLIncludesInt128Scale(t *testing.T) {
	for _, scale := range []int32{0, -2} {
		t.Run(fmt.Sprintf("scale_%d", scale), func(t *testing.T) {
			array := Array{
				TableName: "T",
				FieldName: "F",
				Length:    16,
				Scale:     scale,
				BlrTypeId: fbproto.Isc_blr_int128,
				TypeId:    fbproto.SQL_TYPE_INT128,
			}
			dimensions := []map[string]int{{"Lower": 1, "Upper": 2}}

			sdl := array.generateSDL(dimensions)

			require.GreaterOrEqual(t, len(sdl), 7)
			assert.Equal(t, byte(fbproto.Isc_blr_int128), sdl[3])
			assert.Equal(t, byte(scale), sdl[4])
			assert.Equal(t, byte(fbproto.Isc_sdl_relation), sdl[5])
		})
	}
}
