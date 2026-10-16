package fbtype_test

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
	"rdb.red-soft.ru/fbx/internal/fbio"
)

func TestFloat4Codec(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "float", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Float4{Float32: -1, Valid: true}, Result: new(fbtype.Float4), Test: isExpectedEq(fbtype.Float4{Float32: -1, Valid: true})},
		{Param: fbtype.Float4{Float32: 0, Valid: true}, Result: new(fbtype.Float4), Test: isExpectedEq(fbtype.Float4{Float32: 0, Valid: true})},
		{Param: fbtype.Float4{Float32: 1, Valid: true}, Result: new(fbtype.Float4), Test: isExpectedEq(fbtype.Float4{Float32: 1, Valid: true})},
		{Param: float32(0.00001), Result: new(float32), Test: isExpectedEq(float32(0.00001))},
		{Param: float32(9999.99), Result: new(float32), Test: isExpectedEq(float32(9999.99))},
		{Param: fbtype.Float4{}, Result: new(fbtype.Float4), Test: isExpectedEq(fbtype.Float4{})},
		{Param: int64(1), Result: new(int64), Test: isExpectedEq(int64(1))},
		{Param: "1.23", Result: new(string), Test: isExpectedEq("1.23")},
		{Param: nil, Result: new(*float32), Test: isExpectedEq((*float32)(nil))},
	})
}

func TestFloat4MarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source fbtype.Float4
		result string
	}{
		{source: fbtype.Float4{Float32: 0}, result: "null"},
		{source: fbtype.Float4{Float32: 1.23, Valid: true}, result: "1.23"},
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

func TestFloat4UnmarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source string
		result fbtype.Float4
	}{
		{source: "null", result: fbtype.Float4{Float32: 0}},
		{source: "1.23", result: fbtype.Float4{Float32: 1.23, Valid: true}},
	}
	for i, tt := range successfulTests {
		var r fbtype.Float4
		err := r.UnmarshalJSON([]byte(tt.source))
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if r != tt.result {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, r)
		}
	}
}

func TestFloat4CodecDecodeValueReturnsFloat32(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_FLOAT, 0, 4, 0)
	src := make([]byte, 4)
	binary.BigEndian.PutUint32(src, math.Float32bits(1.25))

	value, err := fbtype.Float4Codec{}.DecodeValue(fbtype.NewMap(), typeMetadata, src)
	if err != nil {
		t.Fatalf("DecodeValue failed: %v", err)
	}

	got, ok := value.(float32)
	if !ok {
		t.Fatalf("DecodeValue returned %T, want float32", value)
	}
	if got != float32(1.25) {
		t.Fatalf("DecodeValue returned %v, want 1.25", got)
	}

	value, err = fbtype.Float4Codec{}.DecodeValue(fbtype.NewMap(), typeMetadata, nil)
	if err != nil {
		t.Fatalf("DecodeValue NULL failed: %v", err)
	}
	if value != nil {
		t.Fatalf("DecodeValue NULL returned %v, want nil", value)
	}
}

func TestFloat4ScanPlansReportTargetTypeChanged(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_FLOAT, 0, 4, 0)
	src := make([]byte, 4)
	binary.BigEndian.PutUint32(src, math.Float32bits(1.25))

	for _, tt := range []struct {
		name   string
		target any
	}{
		{name: "float32", target: new(float32)},
		{name: "Float64Scanner", target: new(fbtype.Float4)},
		{name: "Int64Scanner", target: new(fbtype.Int8)},
		{name: "TextScanner", target: new(fbtype.Text)},
		{name: "string", target: new(string)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			plan := fbtype.Float4Codec{}.PlanScan(fbtype.NewMap(), typeMetadata, tt.target)
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

func TestFloat4CodecScanBinary(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_FLOAT, 0, 4, 0)
	src := []byte{0xc2, 0xf6, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff}
	floatTarget := float32(42)
	stringTarget := "unchanged"

	for _, tt := range []struct {
		name     string
		target   any
		want     any
		wantNull any
	}{
		{"float32", &floatTarget, float32(-123), nil},
		{"Float64Scanner", &fbtype.Float4{Float32: 42, Valid: true}, fbtype.Float4{Float32: -123, Valid: true}, fbtype.Float4{}},
		{"Int64Scanner", &fbtype.Int8{Int64: 42, Valid: true}, fbtype.Int8{Int64: -123, Valid: true}, fbtype.Int8{}},
		{"TextScanner", &fbtype.Text{String: "unchanged", Valid: true}, fbtype.Text{String: "-123", Valid: true}, fbtype.Text{}},
		{"string", &stringTarget, "-123", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			plan := fbtype.Float4Codec{}.PlanScan(fbtype.NewMap(), typeMetadata, tt.target)
			if plan == nil {
				t.Fatal("expected scan plan, got nil")
			}

			before := reflect.ValueOf(tt.target).Elem().Interface()
			for _, n := range []int{0, 1, 2, 3, 5, 8} {
				err := plan.Scan(src[:n], tt.target)
				if !errors.Is(err, fbio.ErrInvalidLength) {
					t.Fatalf("Scan(%d bytes) error = %v, want ErrInvalidLength", n, err)
				}
				if got := reflect.ValueOf(tt.target).Elem().Interface(); !reflect.DeepEqual(got, before) {
					t.Fatalf("Scan(%d bytes) changed destination to %v, want %v", n, got, before)
				}
			}

			if err := plan.Scan(src[:4], tt.target); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}
			if got := reflect.ValueOf(tt.target).Elem().Interface(); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Scan = %v, want %v", got, tt.want)
			}

			err := plan.Scan(nil, tt.target)
			want := tt.wantNull
			if tt.wantNull == nil {
				if err == nil {
					t.Fatal("Scan(NULL) succeeded for a non-nullable destination")
				}
				want = tt.want
			} else if err != nil {
				t.Fatalf("Scan(NULL) failed: %v", err)
			}
			if got := reflect.ValueOf(tt.target).Elem().Interface(); !reflect.DeepEqual(got, want) {
				t.Fatalf("Scan(NULL) destination = %v, want %v", got, want)
			}
		})
	}
}

func TestFloat4CodecDecodeDatabaseSQLValue(t *testing.T) {
	c := fbtype.Float4Codec{}
	m := fbtype.NewMap()
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_FLOAT, 0, 4, 0)
	src := []byte{0x3f, 0xa0, 0x00, 0x00}

	value, err := c.DecodeDatabaseSQLValue(m, typeMetadata, src)
	if err != nil || value != float64(1.25) {
		t.Fatalf("DecodeDatabaseSQLValue = %v (%T), %v, want float64(1.25), nil", value, value, err)
	}
	value, err = c.DecodeDatabaseSQLValue(m, typeMetadata, nil)
	if err != nil || value != nil {
		t.Fatalf("DecodeDatabaseSQLValue(NULL) = %v, %v, want nil, nil", value, err)
	}

	for _, n := range []int{0, 1, 2, 3, 5, 8} {
		src := make([]byte, n)
		value, err := c.DecodeDatabaseSQLValue(m, typeMetadata, src)
		if value != nil || !errors.Is(err, fbio.ErrInvalidLength) {
			t.Errorf("DecodeDatabaseSQLValue(%d bytes) = %v, %v, want nil, ErrInvalidLength", n, value, err)
		}
		value, err = c.DecodeValue(m, typeMetadata, src)
		if value != nil || !errors.Is(err, fbio.ErrInvalidLength) {
			t.Errorf("DecodeValue(%d bytes) = %v, %v, want nil, ErrInvalidLength", n, value, err)
		}
	}
}

func TestFloat4RejectsFiniteFloat64Overflow(t *testing.T) {
	var dst fbtype.Float4
	if err := dst.Scan(math.MaxFloat64); err == nil || !strings.Contains(err.Error(), "overflows float32") {
		t.Fatalf("Scan error = %v, want float32 overflow", err)
	}

	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_FLOAT, 0, 4, 0)
	_, err := fbtype.NewMap().Encode(typeMetadata, math.MaxFloat64, nil)
	if err == nil || !strings.Contains(err.Error(), "overflows float32") {
		t.Fatalf("Encode error = %v, want float32 overflow", err)
	}

	buf, err := fbtype.NewMap().Encode(typeMetadata, math.Inf(1), nil)
	if err != nil {
		t.Fatalf("Encode +Inf failed: %v", err)
	}
	if got, want := binary.BigEndian.Uint32(buf), math.Float32bits(float32(math.Inf(1))); got != want {
		t.Fatalf("Encode +Inf bytes = %08x, want %08x", got, want)
	}
}

func TestFloat4UnderflowUsesIEEE754Rounding(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_FLOAT, 0, 4, 0)
	smallest := float64(math.SmallestNonzeroFloat32)
	tests := []struct {
		name     string
		value    float64
		wantBits uint32
	}{
		{name: "positive rounds to zero", value: smallest / 4, wantBits: 0},
		{name: "negative rounds to zero", value: -smallest / 4, wantBits: 1 << 31},
		{name: "positive rounds to smallest subnormal", value: smallest * 3 / 4, wantBits: 1},
		{name: "negative rounds to smallest subnormal", value: -smallest * 3 / 4, wantBits: 1<<31 | 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var scanned fbtype.Float4
			if err := scanned.Scan(tt.value); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}
			if !scanned.Valid || math.Float32bits(scanned.Float32) != tt.wantBits {
				t.Fatalf("Scan bits = %08x valid=%t, want %08x", math.Float32bits(scanned.Float32), scanned.Valid, tt.wantBits)
			}

			var scannedValue fbtype.Float4
			if err := scannedValue.ScanFloat64(fbtype.Float8{Float64: tt.value, Valid: true}); err != nil {
				t.Fatalf("ScanFloat64 failed: %v", err)
			}
			if got := math.Float32bits(scannedValue.Float32); got != tt.wantBits {
				t.Fatalf("ScanFloat64 bits = %08x, want %08x", got, tt.wantBits)
			}

			for _, input := range []any{tt.value, fbtype.Float8{Float64: tt.value, Valid: true}} {
				buf, err := fbtype.NewMap().Encode(typeMetadata, input, nil)
				if err != nil {
					t.Fatalf("Encode(%T) failed: %v", input, err)
				}
				if got := binary.BigEndian.Uint32(buf); got != tt.wantBits {
					t.Fatalf("Encode(%T) bits = %08x, want %08x", input, got, tt.wantBits)
				}
			}
		})
	}

	for _, tt := range []struct {
		text     string
		wantBits uint32
	}{
		{text: "1e-100", wantBits: 0},
		{text: "-1e-100", wantBits: 1 << 31},
	} {
		t.Run("text "+tt.text, func(t *testing.T) {
			var scanned fbtype.Float4
			if err := scanned.Scan(tt.text); err != nil {
				t.Fatalf("Scan failed: %v", err)
			}
			if !scanned.Valid || math.Float32bits(scanned.Float32) != tt.wantBits {
				t.Fatalf("Scan bits = %08x valid=%t, want %08x", math.Float32bits(scanned.Float32), scanned.Valid, tt.wantBits)
			}

			var decoded fbtype.Float4
			if err := decoded.UnmarshalJSON([]byte(tt.text)); err != nil {
				t.Fatalf("UnmarshalJSON failed: %v", err)
			}
			if !decoded.Valid || math.Float32bits(decoded.Float32) != tt.wantBits {
				t.Fatalf("UnmarshalJSON bits = %08x valid=%t, want %08x", math.Float32bits(decoded.Float32), decoded.Valid, tt.wantBits)
			}

			buf, err := fbtype.NewMap().Encode(typeMetadata, tt.text, nil)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			if got := binary.BigEndian.Uint32(buf); got != tt.wantBits {
				t.Fatalf("Encode bits = %08x, want %08x", got, tt.wantBits)
			}
		})
	}
}

func TestFloat32ScanFromDoubleRejectsFiniteOverflow(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_DOUBLE, 0, 8, 0)
	src := make([]byte, 8)
	binary.BigEndian.PutUint64(src, math.Float64bits(math.MaxFloat64))

	var dst float32
	plan := fbtype.NewMap().PlanScan(typeMetadata, &dst)
	if plan == nil {
		t.Fatalf("expected scan plan, got nil")
	}

	err := plan.Scan(src, &dst)
	if err == nil || !strings.Contains(err.Error(), "overflows float32") {
		t.Fatalf("Scan error = %v, want float32 overflow", err)
	}

	binary.BigEndian.PutUint64(src, math.Float64bits(float64(math.SmallestNonzeroFloat32)/4))
	err = plan.Scan(src, &dst)
	if err != nil {
		t.Fatalf("Scan underflow failed: %v", err)
	}
	if got := math.Float32bits(dst); got != 0 {
		t.Fatalf("Scan underflow bits = %08x, want positive zero", got)
	}
}

func TestFloat4ScanAndUnmarshalJSONClearDestinationOnError(t *testing.T) {
	for _, input := range []any{"not-a-number", math.MaxFloat64, struct{}{}} {
		got := fbtype.Float4{Float32: 42, Valid: true}
		if err := got.Scan(input); err == nil {
			t.Fatalf("Scan(%T) succeeded", input)
		}
		if got != (fbtype.Float4{}) {
			t.Fatalf("Scan(%T) left stale value: %+v", input, got)
		}
	}

	for _, input := range [][]byte{[]byte(`"not-a-number"`), []byte(`1e100`)} {
		got := fbtype.Float4{Float32: 42, Valid: true}
		if err := got.UnmarshalJSON(input); err == nil {
			t.Fatalf("UnmarshalJSON(%s) succeeded", input)
		}
		if got != (fbtype.Float4{}) {
			t.Fatalf("UnmarshalJSON(%s) left stale value: %+v", input, got)
		}
	}
}

func TestFloat4Int64ValueRejectsLossyConversions(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value fbtype.Float4
	}{
		{name: "fractional", value: fbtype.Float4{Float32: 1.5, Valid: true}},
		{name: "positive overflow", value: fbtype.Float4{Float32: 1e20, Valid: true}},
		{name: "negative overflow", value: fbtype.Float4{Float32: -1e20, Valid: true}},
		{name: "nan", value: fbtype.Float4{Float32: float32(math.NaN()), Valid: true}},
		{name: "infinity", value: fbtype.Float4{Float32: float32(math.Inf(1)), Valid: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.value.Int64Value()
			if err == nil || !strings.Contains(err.Error(), "cannot losslessly convert") {
				t.Fatalf("Int64Value error = %v, want lossless conversion error", err)
			}
		})
	}

	got, err := (fbtype.Float4{Float32: 42, Valid: true}).Int64Value()
	if err != nil {
		t.Fatalf("Int64Value exact integer failed: %v", err)
	}
	if got != (fbtype.Int8{Int64: 42, Valid: true}) {
		t.Fatalf("Int64Value = %v, want 42", got)
	}
}

func TestFloat4ScanToInt64RejectsLossyConversions(t *testing.T) {
	typeMetadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_FLOAT, 0, 4, 0)
	for _, tt := range []struct {
		name  string
		value float32
	}{
		{name: "fractional", value: 1.5},
		{name: "overflow", value: 1e20},
		{name: "nan", value: float32(math.NaN())},
		{name: "infinity", value: float32(math.Inf(1))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src := make([]byte, 4)
			binary.BigEndian.PutUint32(src, math.Float32bits(tt.value))

			var dst fbtype.Int8
			plan := fbtype.Float4Codec{}.PlanScan(fbtype.NewMap(), typeMetadata, &dst)
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
