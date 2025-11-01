package fbtype_test

import (
	"context"
	"testing"

	"github.com/red-soft/fbx/fbtype"
	"github.com/red-soft/fbx/fbxtest"
)

func TestBoolCodec(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "bool", []fbxtest.ValueRoundTripTest{
		{Param: true, Result: new(bool), Test: isExpectedEq(true)},
		{Param: false, Result: new(bool), Test: isExpectedEq(false)},
		{Param: true, Result: new(fbtype.Bool), Test: isExpectedEq(fbtype.Bool{Bool: true, Valid: true})},
		{Param: fbtype.Bool{}, Result: new(fbtype.Bool), Test: isExpectedEq(fbtype.Bool{})},
		{Param: nil, Result: new(*bool), Test: isExpectedEq((*bool)(nil))},
	})
}

func TestBoolMarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source fbtype.Bool
		result string
	}{
		{source: fbtype.Bool{}, result: "null"},
		{source: fbtype.Bool{Bool: true, Valid: true}, result: "true"},
		{source: fbtype.Bool{Bool: false, Valid: true}, result: "false"},
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

func TestBoolUnmarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source string
		result fbtype.Bool
	}{
		{source: "null", result: fbtype.Bool{}},
		{source: "true", result: fbtype.Bool{Bool: true, Valid: true}},
		{source: "false", result: fbtype.Bool{Bool: false, Valid: true}},
	}
	for i, tt := range successfulTests {
		var r fbtype.Bool
		err := r.UnmarshalJSON([]byte(tt.source))
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if r != tt.result {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, r)
		}
	}
}
