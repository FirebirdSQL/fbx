package fbtype_test

import (
	"context"
	"testing"

	"github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/fbproto"
	"github.com/firebirdsql/fbx/fbtype"
	"github.com/firebirdsql/fbx/fbxtest"
	"github.com/stretchr/testify/require"
)

type someFmtStringer struct{}

type emptyFmtStringer struct{}

const utf8CharsetID = 4

func (someFmtStringer) String() string {
	return "some fmt.Stringer"
}

func (emptyFmtStringer) String() string {
	return ""
}

func TestTextCodec(t *testing.T) {
	for _, fbTypeName := range []string{"varchar(255)"} {
		fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, fbTypeName, []fbxtest.ValueRoundTripTest{
			{Param: fbtype.Text{String: "", Valid: true}, Result: new(fbtype.Text), Test: isExpectedEq(fbtype.Text{String: "", Valid: true})},
			{Param: fbtype.Text{String: "foo", Valid: true}, Result: new(fbtype.Text), Test: isExpectedEq(fbtype.Text{String: "foo", Valid: true})},
			{Param: nil, Result: new(fbtype.Text), Test: isExpectedEq(fbtype.Text{})},
			{Param: "foo", Result: new(string), Test: isExpectedEq("foo")},
			{Param: someFmtStringer{}, Result: new(string), Test: isExpectedEq("some fmt.Stringer")},
		},
		)
	}
}

// Test fixed length char types like char(3)
func TestTextCodecFixedCharRoundTrip(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "char(3)", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Text{String: "a  ", Valid: true}, Result: new(fbtype.Text), Test: isExpectedEq(fbtype.Text{String: "a  ", Valid: true})},
		{Param: nil, Result: new(fbtype.Text), Test: isExpectedEq(fbtype.Text{})},
		{Param: "   ", Result: new(string), Test: isExpectedEq("   ")},
		{Param: "", Result: new(string), Test: isExpectedEq("   ")},
		{Param: " 嗨 ", Result: new(string), Test: isExpectedEq(" 嗨 ")},
	})
}

func TestTextCodecCharNonePreservesBytes(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "char(3) character set none", []fbxtest.ValueRoundTripTest{
		{Param: "嗨", Result: new(string), Test: isExpectedEq("嗨")},
		{Param: "嗨", Result: new([]byte), Test: isExpectedEqBytes([]byte("嗨"))},
		{Param: "嗨", Result: new(fbtype.Text), Test: isExpectedEq(fbtype.Text{String: "嗨", Valid: true})},
	})
}

func TestTextCodecFixedCharDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata fbtype.TypeMetadata
		src      []byte
		want     string
	}{
		{
			name:     "none preserves bytes",
			metadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, fbproto.CS_NONE, 3, 0),
			src:      []byte("嗨"),
			want:     "嗨",
		},
		{
			name:     "none ignores array alignment",
			metadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, fbproto.CS_NONE, 3, 0),
			src:      append([]byte("嗨"), 0),
			want:     "嗨",
		},
		{
			name:     "utf8 keeps declared character count",
			metadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, utf8CharsetID, 12, 0),
			src:      append([]byte("嗨"), []byte("         ")...),
			want:     "嗨  ",
		},
		{
			name:     "varchar uses actual length",
			metadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_VARYING, utf8CharsetID, 12, 0),
			src:      []byte("嗨"),
			want:     "嗨",
		},
	}

	m := fbtype.NewMap()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			err := m.Scan(tt.metadata, tt.src, &got)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTextCodecFixedCharDecodeInvalidMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata fbtype.TypeMetadata
		src      []byte
		wantErr  string
	}{
		{
			name:     "payload shorter than metadata",
			metadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, fbproto.CS_NONE, 3, 0),
			src:      []byte("a"),
			wantErr:  "fixed text payload is too short",
		},
		{
			name:     "multibyte length is not divisible",
			metadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, utf8CharsetID, 5, 0),
			src:      []byte("abcde"),
			wantErr:  "fixed text length 5 is not divisible by 4",
		},
		{
			name:     "large inconsistent length does not panic",
			metadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, fbproto.CS_NONE, 1024, 0),
			src:      []byte("a"),
			wantErr:  "fixed text payload is too short",
		},
	}

	m := fbtype.NewMap()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			require.NotPanics(t, func() {
				err := m.Scan(tt.metadata, tt.src, &got)
				require.ErrorContains(t, err, tt.wantErr)
			})
		})
	}
}

func TestTextCodecBytesScannerPreservesNull(t *testing.T) {
	t.Parallel()

	metadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_VARYING, fbproto.CS_NONE, 10, 0)
	plan := fbtype.TextCodec{}.PlanScan(fbtype.NewMap(), metadata, new(fbtype.PreallocBytes))
	require.NotNil(t, plan)

	value := fbtype.PreallocBytes{1, 2, 3}
	require.NoError(t, plan.Scan(nil, &value))
	require.Nil(t, value)

	value = fbtype.PreallocBytes{1, 2, 3}
	require.NoError(t, plan.Scan([]byte{}, &value))
	require.NotNil(t, value)
	require.Empty(t, value)
}

func TestTextCodecBytesScannerRoundTrip(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		tb.Helper()

		tests := []struct {
			name  string
			query string
			want  []byte
		}{
			{name: "null", query: "select cast(null as varchar(10)) from rdb$database", want: nil},
			{name: "empty", query: "select cast('' as varchar(10)) from rdb$database", want: []byte{}},
			{name: "text", query: "select cast('hello' as varchar(10)) from rdb$database", want: []byte("hello")},
		}

		for _, tt := range tests {
			got := fbtype.PreallocBytes{1, 2, 3}
			err := conn.QueryRow(ctx, tt.query).Scan(&got)
			require.NoError(tb, err, "case=%s", tt.name)
			if tt.want == nil {
				require.Nil(tb, got, "case=%s", tt.name)
			} else {
				require.NotNil(tb, got, "case=%s", tt.name)
				require.Equal(tb, tt.want, []byte(got), "case=%s", tt.name)
			}
		}
	})
}

func TestTextCodecScanPlansRejectChangedTarget(t *testing.T) {
	t.Parallel()

	metadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_VARYING, fbproto.CS_NONE, 10, 0)
	tests := []struct {
		name   string
		target any
		other  any
	}{
		{name: "string", target: new(string), other: new([]byte)},
		{name: "bytes", target: new([]byte), other: new(string)},
		{name: "bytes scanner", target: new(fbtype.PreallocBytes), other: new(string)},
		{name: "text scanner", target: new(fbtype.Text), other: new(string)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := fbtype.TextCodec{}.PlanScan(fbtype.NewMap(), metadata, tt.target)
			require.NotNil(t, plan)
			require.ErrorIs(t, plan.Scan([]byte("value"), tt.other), fbtype.ErrScanTargetTypeChanged)
		})
	}
}

func TestTextCodecEncodePreservesEmptyValue(t *testing.T) {
	t.Parallel()

	metadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_VARYING, utf8CharsetID, 16, 0)
	tests := []struct {
		name  string
		value any
	}{
		{name: "string", value: ""},
		{name: "bytes", value: []byte{}},
		{name: "Text", value: fbtype.Text{String: "", Valid: true}},
		{name: "fmt.Stringer", value: emptyFmtStringer{}},
	}

	m := fbtype.NewMap()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := m.Encode(metadata, tt.value, nil)
			require.NoError(t, err)
			require.NotNil(t, buf)
			require.Empty(t, buf)
		})
	}

	buf, err := m.Encode(metadata, fbtype.Text{}, nil)
	require.NoError(t, err)
	require.Nil(t, buf)
}

func TestTextCodecPreservesCharsetErrors(t *testing.T) {
	t.Parallel()

	const unsupportedCharsetID = 52
	metadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_VARYING, unsupportedCharsetID, 16, 0)
	m := fbtype.NewMap()

	encodePlan := fbtype.TextCodec{}.PlanEncode(m, metadata, "value")
	require.NotNil(t, encodePlan)
	_, err := encodePlan.Encode("value", nil)
	require.ErrorContains(t, err, "unknown or unsupported charset id: 52")

	_, err = m.Encode(metadata, "value", nil)
	require.ErrorContains(t, err, "unknown or unsupported charset id: 52")
	require.NotContains(t, err.Error(), "cannot find encode plan")

	scanPlan := fbtype.TextCodec{}.PlanScan(m, metadata, new(string))
	require.NotNil(t, scanPlan)
	err = scanPlan.Scan([]byte("value"), new(string))
	require.ErrorContains(t, err, "unknown or unsupported charset id: 52")

	var got string
	err = m.Scan(metadata, []byte("value"), &got)
	require.ErrorContains(t, err, "unknown or unsupported charset id: 52")

	_, err = fbtype.TextCodec{}.DecodeValue(m, metadata, []byte("value"))
	require.ErrorContains(t, err, "unknown or unsupported charset id: 52")
}

func TestTextCodecUnsupportedCharsetAllowsNull(t *testing.T) {
	t.Parallel()

	const unsupportedCharsetID = 52
	tests := []struct {
		name   string
		typeID uint32
	}{
		{name: "char", typeID: fbproto.SQL_TYPE_TEXT},
		{name: "varchar", typeID: fbproto.SQL_TYPE_VARYING},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := fbtype.NewTypeMetadata(tt.typeID, unsupportedCharsetID, 16, 0)
			m := fbtype.NewMap()

			encoded, err := m.Encode(metadata, fbtype.Text{}, nil)
			require.NoError(t, err)
			require.Nil(t, encoded)

			_, err = m.Encode(metadata, fbtype.Text{String: "value", Valid: true}, nil)
			require.ErrorContains(t, err, "unknown or unsupported charset id: 52")

			text := fbtype.Text{String: "old", Valid: true}
			require.NoError(t, m.Scan(metadata, nil, &text))
			require.Equal(t, fbtype.Text{}, text)

			bytes := []byte("old")
			require.NoError(t, m.Scan(metadata, nil, &bytes))
			require.Nil(t, bytes)

			prealloc := fbtype.PreallocBytes("old")
			require.NoError(t, m.Scan(metadata, nil, &prealloc))
			require.Nil(t, prealloc)

			var stringValue string
			err = m.Scan(metadata, nil, &stringValue)
			require.ErrorContains(t, err, "cannot scan NULL into *string")
			require.NotContains(t, err.Error(), "unknown or unsupported charset")

			err = m.Scan(metadata, []byte("value"), &text)
			require.ErrorContains(t, err, "unknown or unsupported charset id: 52")
		})
	}
}

func TestTextCodecUnsupportedCharsetNullRoundTrip(t *testing.T) {
	runner := defaultConnTestRunner
	createConfig := runner.CreateConfig
	runner.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config := createConfig(ctx, t)
		config.Encoding = "NONE"
		return config
	}

	fbxtest.RunWithQueryExecModes(context.Background(), t, runner, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		for _, typeName := range []string{"char(10)", "varchar(10)"} {
			value := fbtype.Text{String: "old", Valid: true}
			err := conn.QueryRow(ctx,
				"select cast(null as "+typeName+" character set win1251) from rdb$database",
			).Scan(&value)
			require.NoError(t, err, "type=%s", typeName)
			require.Equal(t, fbtype.Text{}, value, "type=%s", typeName)

			value = fbtype.Text{String: "old", Valid: true}
			err = conn.QueryRow(ctx,
				"select cast(? as "+typeName+" character set win1251) from rdb$database",
				fbtype.Text{},
			).Scan(&value)
			require.NoError(t, err, "type=%s", typeName)
			require.Equal(t, fbtype.Text{}, value, "type=%s", typeName)
		}
	})
}

func TestTextCodecUnsupportedTypesDoNotReturnCharsetErrorPlan(t *testing.T) {
	t.Parallel()

	metadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_VARYING, 52, 16, 0)
	codec := fbtype.TextCodec{}

	require.Nil(t, codec.PlanEncode(fbtype.NewMap(), metadata, int64(1)))
	require.Nil(t, codec.PlanScan(fbtype.NewMap(), metadata, new(int64)))
}

func TestTextScanClearsValueOnError(t *testing.T) {
	t.Parallel()

	value := fbtype.Text{String: "old", Valid: true}
	err := value.Scan(int64(1))
	require.Error(t, err)
	require.Equal(t, fbtype.Text{}, value)
}

func TestTextMarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source fbtype.Text
		result string
	}{
		{source: fbtype.Text{String: ""}, result: "null"},
		{source: fbtype.Text{String: "a", Valid: true}, result: "\"a\""},
	}
	for i, tt := range successfulTests {
		r, err := tt.source.MarshalJSON()
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if string(r) != tt.result {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, string(r))
		}
	}
}

func TestTextUnmarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source string
		result fbtype.Text
	}{
		{source: "null", result: fbtype.Text{String: ""}},
		{source: "\"a\"", result: fbtype.Text{String: "a", Valid: true}},
	}
	for i, tt := range successfulTests {
		var r fbtype.Text
		err := r.UnmarshalJSON([]byte(tt.source))
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if r != tt.result {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, r)
		}
	}
}

func TestTextUnmarshalJSONClearsValueOnError(t *testing.T) {
	t.Parallel()

	value := fbtype.Text{String: "old", Valid: true}
	err := value.UnmarshalJSON([]byte("{"))
	require.Error(t, err)
	require.Equal(t, fbtype.Text{}, value)
}
