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
		{Param: 1, Result: new(any), Test: isExpectedEq(int16(1))},
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

func TestIntegerUnmarshalJSONClearsDestinationOnError(t *testing.T) {
	t.Run("Int2", func(t *testing.T) {
		got := fbtype.Int2{Int16: 42, Valid: true}
		if err := got.UnmarshalJSON([]byte(`"bad"`)); err == nil {
			t.Fatal("expected error, got nil")
		}
		if got != (fbtype.Int2{}) {
			t.Fatalf("got = %#v, want zero value", got)
		}
	})

	t.Run("Int4", func(t *testing.T) {
		got := fbtype.Int4{Int32: 42, Valid: true}
		if err := got.UnmarshalJSON([]byte(`"bad"`)); err == nil {
			t.Fatal("expected error, got nil")
		}
		if got != (fbtype.Int4{}) {
			t.Fatalf("got = %#v, want zero value", got)
		}
	})

	t.Run("Int8", func(t *testing.T) {
		got := fbtype.Int8{Int64: 42, Valid: true}
		if err := got.UnmarshalJSON([]byte(`"bad"`)); err == nil {
			t.Fatal("expected error, got nil")
		}
		if got != (fbtype.Int8{}) {
			t.Fatalf("got = %#v, want zero value", got)
		}
	})
}

func TestSmallintEncodeRangeCheck(t *testing.T) {
	m := fbtype.NewMap()
	smallint := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 0, 2, 0)
	int4 := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 0, 4, 0)

	for _, v := range []any{
		int32(99999), int(99999), int64(99999), "99999",
		int32(-99999), int64(-99999), "-99999",
		int32(math.MaxInt32),
	} {
		if _, err := m.Encode(smallint, v, nil); err == nil {
			t.Errorf("smallint encode of %#v (out of int16 range): expected error, got nil", v)
		}
	}

	for _, v := range []any{
		int16(math.MaxInt16), int16(math.MinInt16),
		int32(32767), int(-32768), int64(100), "32767", "-32768", "0",
	} {
		if _, err := m.Encode(smallint, v, nil); err != nil {
			t.Errorf("smallint encode of %#v (in int16 range): expected success, got %v", v, err)
		}
	}

	for _, v := range []any{int32(99999), int(99999), int64(99999), "99999"} {
		if _, err := m.Encode(int4, v, nil); err != nil {
			t.Errorf("int encode of %#v (in int32 range): expected success, got %v", v, err)
		}
	}
}

func TestIntegerScanClearsDestinationOnError(t *testing.T) {
	t.Run("Int2 ScanInt64 range error", func(t *testing.T) {
		got := fbtype.Int2{Int16: 1, Valid: true}
		err := got.ScanInt64(fbtype.Int8{Int64: math.MaxInt16 + 1, Valid: true})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if got != (fbtype.Int2{}) {
			t.Fatalf("got = %#v, want zero value", got)
		}
	})

	t.Run("Int2 parse error", func(t *testing.T) {
		got := fbtype.Int2{Int16: 1, Valid: true}
		if err := got.Scan("bad"); err == nil {
			t.Fatalf("expected error, got nil")
		}
		if got != (fbtype.Int2{}) {
			t.Fatalf("got = %#v, want zero value", got)
		}
	})

	t.Run("Int2 range error", func(t *testing.T) {
		got := fbtype.Int2{Int16: 1, Valid: true}
		if err := got.Scan(int64(math.MaxInt16 + 1)); err == nil {
			t.Fatalf("expected error, got nil")
		}
		if got != (fbtype.Int2{}) {
			t.Fatalf("got = %#v, want zero value", got)
		}
	})

	t.Run("Int4 ScanInt64 range error", func(t *testing.T) {
		got := fbtype.Int4{Int32: 1, Valid: true}
		err := got.ScanInt64(fbtype.Int8{Int64: math.MaxInt32 + 1, Valid: true})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if got != (fbtype.Int4{}) {
			t.Fatalf("got = %#v, want zero value", got)
		}
	})

	t.Run("Int4 parse error", func(t *testing.T) {
		got := fbtype.Int4{Int32: 1, Valid: true}
		if err := got.Scan([]byte("bad")); err == nil {
			t.Fatalf("expected error, got nil")
		}
		if got != (fbtype.Int4{}) {
			t.Fatalf("got = %#v, want zero value", got)
		}
	})

	t.Run("Int4 range error", func(t *testing.T) {
		got := fbtype.Int4{Int32: 1, Valid: true}
		if err := got.Scan(int64(math.MaxInt32 + 1)); err == nil {
			t.Fatalf("expected error, got nil")
		}
		if got != (fbtype.Int4{}) {
			t.Fatalf("got = %#v, want zero value", got)
		}
	})

	t.Run("Int8 parse error", func(t *testing.T) {
		got := fbtype.Int8{Int64: 1, Valid: true}
		if err := got.Scan("bad"); err == nil {
			t.Fatalf("expected error, got nil")
		}
		if got != (fbtype.Int8{}) {
			t.Fatalf("got = %#v, want zero value", got)
		}
	})

	t.Run("unsupported source type", func(t *testing.T) {
		got := fbtype.Int8{Int64: 1, Valid: true}
		if err := got.Scan(1); err == nil {
			t.Fatalf("expected error, got nil")
		}
		if got != (fbtype.Int8{}) {
			t.Fatalf("got = %#v, want zero value", got)
		}
	})
}

func TestIntegerScanPlansReportTargetTypeChangedBeforeReadingSource(t *testing.T) {
	m := fbtype.NewMap()

	int4Buf := make([]byte, 4)
	binary.BigEndian.PutUint32(int4Buf, 42)

	int8Buf := make([]byte, 8)
	binary.BigEndian.PutUint64(int8Buf, 42)

	codecs := []struct {
		name         string
		typeMetadata fbtype.TypeMetadata
		validSrc     []byte
	}{
		{
			name:         "int4",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 0, 4, 0),
			validSrc:     int4Buf,
		},
		{
			name:         "int8",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT64, 0, 8, 0),
			validSrc:     int8Buf,
		},
	}

	targets := []struct {
		name   string
		target any
	}{
		{name: "int8", target: new(int8)},
		{name: "uint8", target: new(uint8)},
		{name: "int16", target: new(int16)},
		{name: "uint16", target: new(uint16)},
		{name: "int32", target: new(int32)},
		{name: "uint32", target: new(uint32)},
		{name: "int64", target: new(int64)},
		{name: "uint64", target: new(uint64)},
		{name: "int", target: new(int)},
		{name: "uint", target: new(uint)},
		{name: "Int64Scanner", target: new(fbtype.Int8)},
		{name: "string", target: new(string)},
		{name: "TextScanner", target: new(fbtype.Text)},
	}

	for _, codec := range codecs {
		t.Run(codec.name, func(t *testing.T) {
			for _, target := range targets {
				t.Run(target.name, func(t *testing.T) {
					plan := m.PlanScan(codec.typeMetadata, target.target)
					if plan == nil {
						t.Fatal("expected scan plan, got nil")
					}

					for _, src := range [][]byte{codec.validSrc, nil, {1, 2, 3}} {
						err := plan.Scan(src, new(float64))
						if !errors.Is(err, fbtype.ErrScanTargetTypeChanged) {
							t.Fatalf("source %v: error = %v, want ErrScanTargetTypeChanged", src, err)
						}
					}
				})
			}
		})
	}
}

func TestInt4DecodeDatabaseSQLValueChecksSmallintRange(t *testing.T) {
	m := fbtype.NewMap()
	metadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_SHORT, 0, 2, 0)
	src := make([]byte, 4)
	binary.BigEndian.PutUint32(src, uint32(65536))

	value, err := (fbtype.Int4Codec{}).DecodeDatabaseSQLValue(m, metadata, src)
	if err == nil {
		t.Fatalf("value = %v, expected range error", value)
	}
	if !strings.Contains(err.Error(), "greater than maximum value for int16") {
		t.Fatalf("error = %q, want SMALLINT range error", err)
	}
}

func TestIntegerScanInvalidLengthErrorUsesFirebirdTypeName(t *testing.T) {
	m := fbtype.NewMap()

	for _, tt := range []struct {
		name         string
		typeMetadata fbtype.TypeMetadata
		target       any
		want         string
	}{
		{
			name:         "int4 to uint8",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_LONG, 0, 4, 0),
			target:       new(uint8),
			want:         "int4: invalid length",
		},
		{
			name:         "int8 to uint8",
			typeMetadata: fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT64, 0, 8, 0),
			target:       new(uint8),
			want:         "int8: invalid length",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := m.Scan(tt.typeMetadata, []byte{1, 2, 3}, tt.target)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}
