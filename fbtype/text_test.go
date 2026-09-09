package fbtype_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

type someFmtStringer struct{}

const utf8CharsetID = 4

func (someFmtStringer) String() string {
	return "some fmt.Stringer"
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
func TestTextCodecBPChar(t *testing.T) {
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
