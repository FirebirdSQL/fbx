package fbtype_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/fbproto"
	"github.com/firebirdsql/fbx/fbtype"
	"github.com/firebirdsql/fbx/fbtype/zeronull"
	"github.com/firebirdsql/fbx/fbxtest"
	"github.com/stretchr/testify/require"
)

type pointerNullValuer struct {
	value driver.Value
}

func (v *pointerNullValuer) Value() (driver.Value, error) {
	return v.value, nil
}

type nilAwarePointerNullValuer struct{}

func (*nilAwarePointerNullValuer) Value() (driver.Value, error) {
	return "value", nil
}

type textValuerOnly struct{ fbtype.TextValuer }
type boolValuerOnly struct{ fbtype.BoolValuer }
type numericValuerOnly struct{ fbtype.NumericValuer }
type int128ValuerOnly struct{ fbtype.Int128Valuer }
type dec16ValuerOnly struct{ fbtype.Dec16Valuer }
type dec34ValuerOnly struct{ fbtype.Dec34Valuer }
type dateValuerOnly struct{ fbtype.DateValuer }
type timeValuerOnly struct{ fbtype.TimeValuer }
type timetzValuerOnly struct{ fbtype.TimetzValuer }
type timestampValuerOnly struct{ fbtype.TimestampValuer }
type timestamptzValuerOnly struct{ fbtype.TimestamptzValuer }
type uuidValuerOnly struct{ fbtype.UUIDValuer }
type bytesValuerOnly struct{ fbtype.BytesValuer }
type float64ValuerOnly struct{ fbtype.Float64Valuer }
type int64ValuerOnly struct{ fbtype.Int64Valuer }

type textValuerFunc func() (fbtype.Text, error)

func (f textValuerFunc) TextValue() (fbtype.Text, error) {
	return f()
}

type bytesValuerFunc func() ([]byte, error)

func (f bytesValuerFunc) BytesValue() ([]byte, error) {
	return f()
}

func TestNullCodecEncode(t *testing.T) {
	t.Parallel()

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_NULL, 0, 0, 0)
	var nilIntPointer *int32
	var nilTextPointer *fbtype.Text
	var nilAwareValuer *nilAwarePointerNullValuer
	pointerToNilIntPointer := &nilIntPointer
	number := int32(0)
	nonNilIntPointer := &number
	pointerToNonNilIntPointer := &nonNilIntPointer
	var nilIntSlice []int32
	emptyIntSlice := []int32{}
	var nilFlatArray fbtype.FlatArray[int32]
	emptyFlatArray := fbtype.FlatArray[int32]{}
	tests := []struct {
		name   string
		value  any
		isNull bool
	}{
		{name: "nil", value: nil, isNull: true},
		{name: "non-NULL value", value: struct{}{}},
		{name: "nil byte slice", value: []byte(nil), isNull: true},
		{name: "empty byte slice", value: []byte{}},
		{name: "nil integer slice", value: nilIntSlice, isNull: true},
		{name: "empty integer slice", value: emptyIntSlice},
		{name: "pointer to nil integer slice", value: &nilIntSlice, isNull: true},
		{name: "pointer to empty integer slice", value: &emptyIntSlice},
		{name: "nil flat array", value: nilFlatArray, isNull: true},
		{name: "empty flat array", value: emptyFlatArray},
		{name: "pointer to nil flat array", value: &nilFlatArray, isNull: true},
		{name: "pointer to empty flat array", value: &emptyFlatArray},
		{name: "pointer to nil pointer", value: pointerToNilIntPointer, isNull: true},
		{name: "pointer to non-nil pointer", value: pointerToNonNilIntPointer},
		{name: "nil pointer to value-receiver valuer", value: nilTextPointer, isNull: true},
		{name: "nil pointer-only valuer returning a value", value: nilAwareValuer},
		{name: "invalid fbtype text", value: fbtype.Text{}, isNull: true},
		{name: "valid empty fbtype text", value: fbtype.Text{Valid: true}},
		{name: "empty zeronull text", value: zeronull.Text(""), isNull: true},
		{name: "non-empty zeronull text", value: zeronull.Text("value")},
		{name: "invalid sql null string", value: sql.NullString{}, isNull: true},
		{name: "valid empty sql null string", value: sql.NullString{Valid: true}},
		{name: "pointer valuer returning nil", value: &pointerNullValuer{}, isNull: true},
		{name: "pointer valuer returning a value", value: &pointerNullValuer{value: "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := fbtype.NewMap()
			buf, err := m.Encode(typeMetadata, tt.value, nil)
			require.NoError(t, err)
			if tt.isNull {
				require.Nil(t, buf)
			} else {
				require.NotNil(t, buf)
				require.Empty(t, buf)
			}
		})
	}
}

func TestNullCodecEncodeNativeValuers(t *testing.T) {
	t.Parallel()

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_NULL, 0, 0, 0)
	tests := []struct {
		name         string
		nullValue    any
		nonNullValue any
	}{
		{name: "TextValuer", nullValue: textValuerOnly{fbtype.Text{}}, nonNullValue: textValuerOnly{fbtype.Text{Valid: true}}},
		{name: "BoolValuer", nullValue: boolValuerOnly{fbtype.Bool{}}, nonNullValue: boolValuerOnly{fbtype.Bool{Valid: true}}},
		{name: "NumericValuer", nullValue: numericValuerOnly{fbtype.Numeric{}}, nonNullValue: numericValuerOnly{fbtype.Numeric{Valid: true}}},
		{name: "Int128Valuer", nullValue: int128ValuerOnly{fbtype.Int128{}}, nonNullValue: int128ValuerOnly{fbtype.Int128{Valid: true}}},
		{name: "Dec16Valuer", nullValue: dec16ValuerOnly{fbtype.DecFloat16{}}, nonNullValue: dec16ValuerOnly{fbtype.DecFloat16{Valid: true}}},
		{name: "Dec34Valuer", nullValue: dec34ValuerOnly{fbtype.DecFloat34{}}, nonNullValue: dec34ValuerOnly{fbtype.DecFloat34{Valid: true}}},
		{name: "DateValuer", nullValue: dateValuerOnly{fbtype.Date{}}, nonNullValue: dateValuerOnly{fbtype.Date{Valid: true}}},
		{name: "TimeValuer", nullValue: timeValuerOnly{fbtype.Time{}}, nonNullValue: timeValuerOnly{fbtype.Time{Valid: true}}},
		{name: "TimetzValuer", nullValue: timetzValuerOnly{fbtype.Timetz{}}, nonNullValue: timetzValuerOnly{fbtype.Timetz{Valid: true}}},
		{name: "TimestampValuer", nullValue: timestampValuerOnly{fbtype.Timestamp{}}, nonNullValue: timestampValuerOnly{fbtype.Timestamp{Valid: true}}},
		{name: "TimestamptzValuer", nullValue: timestamptzValuerOnly{fbtype.Timestamptz{}}, nonNullValue: timestamptzValuerOnly{fbtype.Timestamptz{Valid: true}}},
		{name: "UUIDValuer", nullValue: uuidValuerOnly{fbtype.UUID{}}, nonNullValue: uuidValuerOnly{fbtype.UUID{Valid: true}}},
		{name: "BytesValuer", nullValue: bytesValuerOnly{bytesValuerFunc(func() ([]byte, error) { return nil, nil })}, nonNullValue: bytesValuerOnly{bytesValuerFunc(func() ([]byte, error) { return []byte{}, nil })}},
		{name: "Float64Valuer", nullValue: float64ValuerOnly{fbtype.Float8{}}, nonNullValue: float64ValuerOnly{fbtype.Float8{Valid: true}}},
		{name: "Int64Valuer", nullValue: int64ValuerOnly{fbtype.Int8{}}, nonNullValue: int64ValuerOnly{fbtype.Int8{Valid: true}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := fbtype.NewMap()
			buf, err := m.Encode(typeMetadata, tt.nullValue, nil)
			require.NoError(t, err)
			require.Nil(t, buf)

			buf, err = m.Encode(typeMetadata, tt.nonNullValue, nil)
			require.NoError(t, err)
			require.NotNil(t, buf)
			require.Empty(t, buf)
		})
	}
}

func TestNullCodecPlanEncodeHandlesNilAtExecution(t *testing.T) {
	t.Parallel()

	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_NULL, 0, 0, 0)
	plan := m.PlanEncode(typeMetadata, []byte(nil))
	require.NotNil(t, plan)

	buf, err := plan.Encode([]byte(nil), nil)
	require.NoError(t, err)
	require.Nil(t, buf)

	buf, err = plan.Encode([]byte{}, nil)
	require.NoError(t, err)
	require.NotNil(t, buf)
	require.Empty(t, buf)
}

func TestNullCodecPlanEncodeUsesCurrentInterfaceValue(t *testing.T) {
	t.Parallel()

	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_NULL, 0, 0, 0)
	var value any = "initial value"
	plan := m.PlanEncode(typeMetadata, &value)
	require.NotNil(t, plan)
	wantErr := errors.New("valuer failed")

	tests := []struct {
		name   string
		value  any
		isNull bool
		err    error
	}{
		{name: "nil interface", value: nil, isNull: true},
		{name: "invalid Text", value: fbtype.Text{}, isNull: true},
		{name: "valid empty Text", value: fbtype.Text{Valid: true}},
		{name: "nil byte slice", value: []byte(nil), isNull: true},
		{name: "empty byte slice", value: []byte{}},
		{name: "native valuer NULL", value: textValuerOnly{fbtype.Text{}}, isNull: true},
		{name: "native valuer error", value: textValuerFunc(func() (fbtype.Text, error) { return fbtype.Text{}, wantErr }), err: wantErr},
		{name: "driver valuer NULL", value: driverValuerFunc(func() (driver.Value, error) { return nil, nil }), isNull: true},
		{name: "driver valuer error", value: driverValuerFunc(func() (driver.Value, error) { return nil, wantErr }), err: wantErr},
		{name: "nil pointer-only driver valuer", value: (*nilAwarePointerNullValuer)(nil)},
		{name: "nil value-receiver driver valuer", value: (*fbtype.Text)(nil), isNull: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value = tt.value
			buf, err := plan.Encode(&value, nil)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				require.Nil(t, buf)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.isNull, buf == nil)
		})
	}
}

func TestNullCodecEncodeRejectsCyclicInterfacePointer(t *testing.T) {
	t.Parallel()

	var value any
	value = &value
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_NULL, 0, 0, 0)

	m := fbtype.NewMap()
	buf, err := m.Encode(typeMetadata, &value, nil)

	require.ErrorContains(t, err, "maximum indirection depth")
	require.Nil(t, buf)

	plan := m.PlanEncode(typeMetadata, &value)
	require.NotNil(t, plan)
	buf, err = plan.Encode(&value, nil)
	require.ErrorContains(t, err, "maximum indirection depth")
	require.Nil(t, buf)
}

func TestNullCodecEncodeAppendsNoData(t *testing.T) {
	t.Parallel()

	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_NULL, 0, 0, 0)
	prefix := []byte{1, 2, 3}

	buf, err := m.Encode(typeMetadata, "non-NULL", prefix)

	require.NoError(t, err)
	require.Equal(t, prefix, buf)
	require.NotNil(t, buf[len(prefix):])
}

func TestNullCodecEncodePropagatesDriverValuerError(t *testing.T) {
	t.Parallel()

	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_NULL, 0, 0, 0)
	wantErr := errors.New("valuer failed")

	buf, err := m.Encode(typeMetadata, driverValuerFunc(func() (driver.Value, error) {
		return nil, wantErr
	}), nil)

	require.ErrorIs(t, err, wantErr)
	require.Nil(t, buf)
}

func TestNullCodecEncodePropagatesNativeValuerErrors(t *testing.T) {
	t.Parallel()

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_NULL, 0, 0, 0)
	wantErr := errors.New("valuer failed")
	tests := []struct {
		name  string
		value any
	}{
		{
			name: "native valuer",
			value: textValuerFunc(func() (fbtype.Text, error) {
				return fbtype.Text{}, wantErr
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := fbtype.NewMap().Encode(typeMetadata, tt.value, nil)
			require.ErrorIs(t, err, wantErr)
			require.Nil(t, buf)
		})
	}
}

func TestNullCodecQueryParameter(t *testing.T) {
	fbxtest.RunWithQueryExecModes(context.Background(), t, defaultConnTestRunner, nil, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		var nilIntPointer *int32
		var nilIntSlice []int32
		emptyIntSlice := []int32{}
		var nilFlatArray fbtype.FlatArray[int32]
		emptyFlatArray := fbtype.FlatArray[int32]{}
		var hiddenNull any = fbtype.Text{}
		var hiddenValue any = fbtype.Text{Valid: true}
		tests := []struct {
			name   string
			value  any
			isNull int32
		}{
			{name: "nil", value: nil, isNull: 1},
			{name: "non-NULL string", value: "value"},
			{name: "invalid fbtype text", value: fbtype.Text{}, isNull: 1},
			{name: "valid empty fbtype text", value: fbtype.Text{Valid: true}},
			{name: "empty zeronull text", value: zeronull.Text(""), isNull: 1},
			{name: "non-empty zeronull text", value: zeronull.Text("value")},
			{name: "invalid sql null string", value: sql.NullString{}, isNull: 1},
			{name: "valid empty sql null string", value: sql.NullString{Valid: true}},
			{name: "pointer valuer returning nil", value: &pointerNullValuer{}, isNull: 1},
			{name: "pointer valuer returning a value", value: &pointerNullValuer{value: "value"}},
			{name: "nil byte slice", value: []byte(nil), isNull: 1},
			{name: "empty byte slice", value: []byte{}},
			{name: "nil integer slice", value: nilIntSlice, isNull: 1},
			{name: "empty integer slice", value: emptyIntSlice},
			{name: "nil flat array", value: nilFlatArray, isNull: 1},
			{name: "empty flat array", value: emptyFlatArray},
			{name: "pointer to nil pointer", value: &nilIntPointer, isNull: 1},
			{name: "pointer to nil slice", value: &nilIntSlice, isNull: 1},
			{name: "pointer to empty slice", value: &emptyIntSlice},
			{name: "interface pointer to NULL", value: &hiddenNull, isNull: 1},
			{name: "interface pointer to value", value: &hiddenValue},
			{name: "native TextValuer", value: textValuerOnly{fbtype.Text{}}, isNull: 1},
			{name: "native BytesValuer", value: bytesValuerOnly{bytesValuerFunc(func() ([]byte, error) { return nil, nil })}, isNull: 1},
		}

		for _, tt := range tests {
			var got int32
			err := conn.QueryRow(
				ctx,
				"select case when ? is null then 1 else 0 end from rdb$database",
				tt.value,
			).Scan(&got)
			require.NoError(tb, err, tt.name)
			require.Equal(tb, tt.isNull, got, tt.name)
		}

		var hiddenError any = driverValuerFunc(func() (driver.Value, error) {
			return nil, errors.New("valuer failed")
		})
		err := conn.QueryRow(
			ctx,
			"select case when ? is null then 1 else 0 end from rdb$database",
			&hiddenError,
		).Scan(new(int32))
		require.ErrorContains(tb, err, "valuer failed")

		var got int32
		err = conn.QueryRow(ctx, "select 1 from rdb$database").Scan(&got)
		require.NoError(tb, err)
		require.Equal(tb, int32(1), got)
	})
}
