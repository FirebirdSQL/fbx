package fbtype_test

import (
	"context"
	"testing"

	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

type someFmtStringer struct{}

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
