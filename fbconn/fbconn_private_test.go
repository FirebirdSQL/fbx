package fbconn

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
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
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	conn := &FbConn{status: connStatusIdle}
	st := &Statement{conn: conn}
	b := &Batch{st: st, state: batchStateCreated}
	st.activeBatch = b

	err := b.Close(ctx)

	require.Error(t, err)
	require.False(t, b.isClosed())
	require.Same(t, b, st.activeBatch)
	require.False(t, conn.IsClosed())
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
