package fbtype_test

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"testing"

	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestFloat8Codec(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "double precision", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Float8{Float64: -1, Valid: true}, Result: new(fbtype.Float8), Test: isExpectedEq(fbtype.Float8{Float64: -1, Valid: true})},
		{Param: fbtype.Float8{Float64: 0, Valid: true}, Result: new(fbtype.Float8), Test: isExpectedEq(fbtype.Float8{Float64: 0, Valid: true})},
		{Param: fbtype.Float8{Float64: 1, Valid: true}, Result: new(fbtype.Float8), Test: isExpectedEq(fbtype.Float8{Float64: 1, Valid: true})},
		{Param: float64(0.00001), Result: new(float64), Test: isExpectedEq(float64(0.00001))},
		{Param: float64(9999.99), Result: new(float64), Test: isExpectedEq(float64(9999.99))},
		{Param: fbtype.Float8{}, Result: new(fbtype.Float8), Test: isExpectedEq(fbtype.Float8{})},
		{Param: int64(1), Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: "1.23", Result: new(string), Test: isExpectedEq("1.23")},
		{Param: nil, Result: new(*float64), Test: isExpectedEq((*float64)(nil))},
	})
}

func TestFloat8MarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source fbtype.Float8
		result string
	}{
		{source: fbtype.Float8{Float64: 0}, result: "null"},
		{source: fbtype.Float8{Float64: 1.23, Valid: true}, result: "1.23"},
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

func TestFloat8UnmarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source string
		result fbtype.Float8
	}{
		{source: "null", result: fbtype.Float8{Float64: 0}},
		{source: "1.23", result: fbtype.Float8{Float64: 1.23, Valid: true}},
	}
	for i, tt := range successfulTests {
		var r fbtype.Float8
		err := r.UnmarshalJSON([]byte(tt.source))
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if r != tt.result {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, r)
		}
	}
}

func TestFloat8CodecDirectScanToString(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_DOUBLE, 0, 8, 0)
	src := make([]byte, 8)
	binary.BigEndian.PutUint64(src, math.Float64bits(1.25))

	var dst string
	plan := fbtype.Float8Codec{}.PlanScan(fbtype.NewMap(), typeMetadata, &dst)
	if plan == nil {
		t.Fatalf("PlanScan returned nil for *string")
	}

	if err := plan.Scan(src, &dst); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if dst != "1.25" {
		t.Fatalf("Scan returned %q, want 1.25", dst)
	}
}

func TestFloat8CodecDecodeValueReturnsFloat64(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_DOUBLE, 0, 8, 0)
	src := make([]byte, 8)
	binary.BigEndian.PutUint64(src, math.Float64bits(1.25))

	value, err := fbtype.Float8Codec{}.DecodeValue(fbtype.NewMap(), typeMetadata, src)
	if err != nil {
		t.Fatalf("DecodeValue failed: %v", err)
	}

	got, ok := value.(float64)
	if !ok {
		t.Fatalf("DecodeValue returned %T, want float64", value)
	}
	if got != 1.25 {
		t.Fatalf("DecodeValue returned %v, want 1.25", got)
	}

	value, err = fbtype.Float8Codec{}.DecodeValue(fbtype.NewMap(), typeMetadata, nil)
	if err != nil {
		t.Fatalf("DecodeValue NULL failed: %v", err)
	}
	if value != nil {
		t.Fatalf("DecodeValue NULL returned %v, want nil", value)
	}
}

func TestFloat8ScanPlansReportTargetTypeChanged(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_DOUBLE, 0, 8, 0)
	src := make([]byte, 8)
	binary.BigEndian.PutUint64(src, math.Float64bits(1.25))

	for _, tt := range []struct {
		name   string
		target any
	}{
		{name: "float64", target: new(float64)},
		{name: "Float64Scanner", target: new(fbtype.Float8)},
		{name: "Int64Scanner", target: new(fbtype.Int8)},
		{name: "TextScanner", target: new(fbtype.Text)},
		{name: "string", target: new(string)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			plan := fbtype.Float8Codec{}.PlanScan(fbtype.NewMap(), typeMetadata, tt.target)
			if plan == nil {
				t.Fatalf("expected scan plan, got nil")
			}

			var wrongTarget bool
			err := plan.Scan(src, &wrongTarget)
			if !errors.Is(err, fbtype.ErrScanTargetTypeChanged) {
				t.Fatalf("error = %v, want ErrScanTargetTypeChanged", err)
			}
		})
	}
}

func TestFloat8Int64ValueRejectsLossyConversions(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value fbtype.Float8
	}{
		{name: "fractional", value: fbtype.Float8{Float64: 1.5, Valid: true}},
		{name: "positive overflow", value: fbtype.Float8{Float64: math.MaxFloat64, Valid: true}},
		{name: "negative overflow", value: fbtype.Float8{Float64: -math.MaxFloat64, Valid: true}},
		{name: "nan", value: fbtype.Float8{Float64: math.NaN(), Valid: true}},
		{name: "infinity", value: fbtype.Float8{Float64: math.Inf(1), Valid: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.value.Int64Value()
			if err == nil || !strings.Contains(err.Error(), "cannot losslessly convert") {
				t.Fatalf("Int64Value error = %v, want lossless conversion error", err)
			}
		})
	}

	got, err := (fbtype.Float8{Float64: 42, Valid: true}).Int64Value()
	if err != nil {
		t.Fatalf("Int64Value exact integer failed: %v", err)
	}
	if got != (fbtype.Int8{Int64: 42, Valid: true}) {
		t.Fatalf("Int64Value = %v, want 42", got)
	}
}

func TestBuiltinFloatEncodeToIntegerRejectsLossyConversions(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT64, 0, 8, 0)

	for _, tt := range []struct {
		name  string
		value any
	}{
		{name: "float32 fractional", value: float32(1.5)},
		{name: "float64 fractional", value: float64(1.5)},
		{name: "float32 overflow", value: float32(1e20)},
		{name: "float64 overflow", value: math.MaxFloat64},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fbtype.NewMap().Encode(typeMetadata, tt.value, nil)
			if err == nil || !strings.Contains(err.Error(), "cannot losslessly convert") {
				t.Fatalf("Encode error = %v, want lossless conversion error", err)
			}
		})
	}

	buf, err := fbtype.NewMap().Encode(typeMetadata, float64(42), nil)
	if err != nil {
		t.Fatalf("Encode exact float64 integer failed: %v", err)
	}
	if got := int64(binary.BigEndian.Uint64(buf)); got != 42 {
		t.Fatalf("Encode bytes decode to %d, want 42", got)
	}
}

func TestFloat8ScanToInt64RejectsLossyConversions(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_DOUBLE, 0, 8, 0)
	for _, tt := range []struct {
		name  string
		value float64
	}{
		{name: "fractional", value: 1.5},
		{name: "overflow", value: math.MaxFloat64},
		{name: "nan", value: math.NaN()},
		{name: "infinity", value: math.Inf(1)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src := make([]byte, 8)
			binary.BigEndian.PutUint64(src, math.Float64bits(tt.value))

			var dst fbtype.Int8
			plan := fbtype.Float8Codec{}.PlanScan(fbtype.NewMap(), typeMetadata, &dst)
			if plan == nil {
				t.Fatalf("expected scan plan, got nil")
			}

			err := plan.Scan(src, &dst)
			if err == nil || !strings.Contains(err.Error(), "cannot losslessly convert") {
				t.Fatalf("Scan error = %v, want lossless conversion error", err)
			}
		})
	}
}
