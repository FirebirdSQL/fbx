package fbtype_test

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestInt2Codec(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "smallint", []fbxtest.ValueRoundTripTest{
		{Param: int8(1), Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: int16(1), Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: int32(1), Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: int64(1), Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: uint8(1), Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: uint16(1), Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: uint32(1), Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: uint64(1), Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: int(1), Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: uint(1), Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: fbtype.Int2{Int16: 1, Valid: true}, Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: int32(-1), Result: new(fbtype.Int2), Test: isExpectedEq(fbtype.Int2{Int16: -1, Valid: true})},
		{Param: 1, Result: new(int8), Test: isExpectedEq(int8(1))},
		{Param: 1, Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: 1, Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: 1, Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: 1, Result: new(uint8), Test: isExpectedEq(uint8(1))},
		{Param: 1, Result: new(uint16), Test: isExpectedEq(uint16(1))},
		{Param: 1, Result: new(uint32), Test: isExpectedEq(uint32(1))},
		{Param: 1, Result: new(uint64), Test: isExpectedEq(uint64(1))},
		{Param: 1, Result: new(int), Test: isExpectedEq(int(1))},
		{Param: 1, Result: new(uint), Test: isExpectedEq(uint(1))},
		{Param: -1, Result: new(int8), Test: isExpectedEq(int8(-1))},
		{Param: -1, Result: new(int16), Test: isExpectedEq(int16(-1))},
		{Param: -1, Result: new(int32), Test: isExpectedEq(int32(-1))},
		{Param: -1, Result: new(int64), Test: isExpectedEq(int64(-1))},
		{Param: -1, Result: new(int), Test: isExpectedEq(int(-1))},
		{Param: math.MinInt16, Result: new(int16), Test: isExpectedEq(int16(math.MinInt16))},
		{Param: -1, Result: new(int16), Test: isExpectedEq(int16(-1))},
		{Param: 0, Result: new(int16), Test: isExpectedEq(int16(0))},
		{Param: 1, Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: math.MaxInt16, Result: new(int16), Test: isExpectedEq(int16(math.MaxInt16))},
		{Param: 1, Result: new(fbtype.Int2), Test: isExpectedEq(fbtype.Int2{Int16: 1, Valid: true})},
		{Param: fbtype.Int2{}, Result: new(fbtype.Int2), Test: isExpectedEq(fbtype.Int2{})},
		{Param: nil, Result: new(*int16), Test: isExpectedEq((*int16)(nil))},
		{Param: "11", Result: new(int16), Test: isExpectedEq(int16(11))},
		{Param: "-5", Result: new(int16), Test: isExpectedEq(int16(-5))},
		{Param: "0", Result: new(int16), Test: isExpectedEq(int16(0))},
		{Param: "32767", Result: new(int16), Test: isExpectedEq(int16(math.MaxInt16))},
		{Param: "-32768", Result: new(int16), Test: isExpectedEq(int16(math.MinInt16))},
	})
}

func TestInt2MarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source fbtype.Int2
		result string
	}{
		{source: fbtype.Int2{Int16: 0}, result: "null"},
		{source: fbtype.Int2{Int16: 1, Valid: true}, result: "1"},
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

func TestInt2UnmarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source string
		result fbtype.Int2
	}{
		{source: "null", result: fbtype.Int2{Int16: 0}},
		{source: "1", result: fbtype.Int2{Int16: 1, Valid: true}},
	}
	for i, tt := range successfulTests {
		var r fbtype.Int2
		err := r.UnmarshalJSON([]byte(tt.source))
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if r != tt.result {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, r)
		}
	}
}

func TestInt4Codec(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "int", []fbxtest.ValueRoundTripTest{
		{Param: int8(1), Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: int16(1), Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: int32(1), Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: int64(1), Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: uint8(1), Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: uint16(1), Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: uint32(1), Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: uint64(1), Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: int(1), Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: uint(1), Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: fbtype.Int4{Int32: 1, Valid: true}, Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: int32(-1), Result: new(fbtype.Int4), Test: isExpectedEq(fbtype.Int4{Int32: -1, Valid: true})},
		{Param: 1, Result: new(int8), Test: isExpectedEq(int8(1))},
		{Param: 1, Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: 1, Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: 1, Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: 1, Result: new(uint8), Test: isExpectedEq(uint8(1))},
		{Param: 1, Result: new(uint16), Test: isExpectedEq(uint16(1))},
		{Param: 1, Result: new(uint32), Test: isExpectedEq(uint32(1))},
		{Param: 1, Result: new(uint64), Test: isExpectedEq(uint64(1))},
		{Param: 1, Result: new(int), Test: isExpectedEq(int(1))},
		{Param: 1, Result: new(uint), Test: isExpectedEq(uint(1))},
		{Param: -1, Result: new(int8), Test: isExpectedEq(int8(-1))},
		{Param: -1, Result: new(int16), Test: isExpectedEq(int16(-1))},
		{Param: -1, Result: new(int32), Test: isExpectedEq(int32(-1))},
		{Param: -1, Result: new(int64), Test: isExpectedEq(int64(-1))},
		{Param: -1, Result: new(int), Test: isExpectedEq(int(-1))},
		{Param: math.MinInt32, Result: new(int32), Test: isExpectedEq(int32(math.MinInt32))},
		{Param: -1, Result: new(int32), Test: isExpectedEq(int32(-1))},
		{Param: 0, Result: new(int32), Test: isExpectedEq(int32(0))},
		{Param: 1, Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: math.MaxInt32, Result: new(int32), Test: isExpectedEq(int32(math.MaxInt32))},
		{Param: 1, Result: new(fbtype.Int4), Test: isExpectedEq(fbtype.Int4{Int32: 1, Valid: true})},
		{Param: fbtype.Int4{}, Result: new(fbtype.Int4), Test: isExpectedEq(fbtype.Int4{})},
		{Param: nil, Result: new(*int32), Test: isExpectedEq((*int32)(nil))},
		{Param: "123", Result: new(int32), Test: isExpectedEq(int32(123))},
		{Param: "-456", Result: new(int32), Test: isExpectedEq(int32(-456))},
		{Param: "0", Result: new(int32), Test: isExpectedEq(int32(0))},
		{Param: "2147483647", Result: new(int32), Test: isExpectedEq(int32(math.MaxInt32))},
		{Param: "-2147483648", Result: new(int32), Test: isExpectedEq(int32(math.MinInt32))},
	})
}

func TestInt4MarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source fbtype.Int4
		result string
	}{
		{source: fbtype.Int4{Int32: 0}, result: "null"},
		{source: fbtype.Int4{Int32: 1, Valid: true}, result: "1"},
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

func TestInt4UnmarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source string
		result fbtype.Int4
	}{
		{source: "null", result: fbtype.Int4{Int32: 0}},
		{source: "1", result: fbtype.Int4{Int32: 1, Valid: true}},
	}
	for i, tt := range successfulTests {
		var r fbtype.Int4
		err := r.UnmarshalJSON([]byte(tt.source))
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if r != tt.result {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, r)
		}
	}
}

func TestInt8Codec(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "bigint", []fbxtest.ValueRoundTripTest{
		{Param: int8(1), Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: int16(1), Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: int32(1), Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: int64(1), Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: uint8(1), Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: uint16(1), Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: uint32(1), Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: uint64(1), Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: int(1), Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: uint(1), Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: fbtype.Int8{Int64: 1, Valid: true}, Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: int32(-1), Result: new(fbtype.Int8), Test: isExpectedEq(fbtype.Int8{Int64: -1, Valid: true})},
		{Param: 1, Result: new(int8), Test: isExpectedEq(int8(1))},
		{Param: 1, Result: new(int16), Test: isExpectedEq(int16(1))},
		{Param: 1, Result: new(int32), Test: isExpectedEq(int32(1))},
		{Param: 1, Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: 1, Result: new(uint8), Test: isExpectedEq(uint8(1))},
		{Param: 1, Result: new(uint16), Test: isExpectedEq(uint16(1))},
		{Param: 1, Result: new(uint32), Test: isExpectedEq(uint32(1))},
		{Param: 1, Result: new(uint64), Test: isExpectedEq(uint64(1))},
		{Param: 1, Result: new(int), Test: isExpectedEq(int(1))},
		{Param: 1, Result: new(uint), Test: isExpectedEq(uint(1))},
		{Param: -1, Result: new(int8), Test: isExpectedEq(int8(-1))},
		{Param: -1, Result: new(int16), Test: isExpectedEq(int16(-1))},
		{Param: -1, Result: new(int32), Test: isExpectedEq(int32(-1))},
		{Param: -1, Result: new(int64), Test: isExpectedEq(int64(-1))},
		{Param: -1, Result: new(int), Test: isExpectedEq(int(-1))},
		{Param: math.MinInt64, Result: new(int64), Test: isExpectedEq(int64(math.MinInt64))},
		{Param: -1, Result: new(int64), Test: isExpectedEq(int64(-1))},
		{Param: 0, Result: new(int64), Test: isExpectedEq(int64(0))},
		{Param: 1, Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: math.MaxInt64, Result: new(int64), Test: isExpectedEq(int64(math.MaxInt64))},
		{Param: 1, Result: new(fbtype.Int8), Test: isExpectedEq(fbtype.Int8{Int64: 1, Valid: true})},
		{Param: fbtype.Int8{}, Result: new(fbtype.Int8), Test: isExpectedEq(fbtype.Int8{})},
		{Param: nil, Result: new(*int64), Test: isExpectedEq((*int64)(nil))},
		{Param: "999", Result: new(int64), Test: isExpectedEq(int64(999))},
		{Param: "-999", Result: new(int64), Test: isExpectedEq(int64(-999))},
		{Param: "0", Result: new(int64), Test: isExpectedEq(int64(0))},
		{Param: "9223372036854775807", Result: new(int64), Test: isExpectedEq(int64(math.MaxInt64))},
		{Param: "-9223372036854775808", Result: new(int64), Test: isExpectedEq(int64(math.MinInt64))},
	})
}

func TestInt8MarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source fbtype.Int8
		result string
	}{
		{source: fbtype.Int8{Int64: 0}, result: "null"},
		{source: fbtype.Int8{Int64: 1, Valid: true}, result: "1"},
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

func TestInt8UnmarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source string
		result fbtype.Int8
	}{
		{source: "null", result: fbtype.Int8{Int64: 0}},
		{source: "1", result: fbtype.Int8{Int64: 1, Valid: true}},
	}
	for i, tt := range successfulTests {
		var r fbtype.Int8
		err := r.UnmarshalJSON([]byte(tt.source))
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if r != tt.result {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, r)
		}
	}
}

func TestInt2CodecStringErrors(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"non-numeric", "abc"},
		{"too large", "99999"},
		{"too small", "-99999"},
		{"float", "123.45"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeMetadata := fbtype.NewTypeMetadata(0, 0, 0, 0)
			plan := fbtype.Int2Codec{}.PlanEncode(nil, typeMetadata, tt.value)
			require.NotNil(t, plan)

			_, err := plan.Encode(tt.value, nil)
			require.Error(t, err, "expected error for value: %s", tt.value)
		})
	}
}
