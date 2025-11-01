package fbtype_test

import (
	"context"
	"testing"
	"time"

	"github.com/red-soft/fbx/fbtype"
	"github.com/red-soft/fbx/fbxtest"
	"github.com/stretchr/testify/assert"
)

func isExpectedEqTime(a any) func(any) bool {
	return func(v any) bool {
		at := a.(time.Time)
		vt := v.(time.Time)

		return at.Equal(vt)
	}
}

func TestDateCodec(t *testing.T) {
	t.Skip("need implementation")
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "date", []fbxtest.ValueRoundTripTest{
		{Param: time.Date(-100, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(-100, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(-1, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(-1, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC))},
		{Param: time.Date(12200, 1, 2, 0, 0, 0, 0, time.UTC), Result: new(time.Time), Test: isExpectedEqTime(time.Date(12200, 1, 2, 0, 0, 0, 0, time.UTC))},
		{Param: fbtype.Date{InfinityModifier: fbtype.Infinity, Valid: true}, Result: new(fbtype.Date), Test: isExpectedEq(fbtype.Date{InfinityModifier: fbtype.Infinity, Valid: true})},
		{Param: fbtype.Date{InfinityModifier: fbtype.NegativeInfinity, Valid: true}, Result: new(fbtype.Date), Test: isExpectedEq(fbtype.Date{InfinityModifier: fbtype.NegativeInfinity, Valid: true})},
		{Param: fbtype.Date{}, Result: new(fbtype.Date), Test: isExpectedEq(fbtype.Date{})},
		{Param: nil, Result: new(*time.Time), Test: isExpectedEq((*time.Time)(nil))},
	})
}

func TestDateCodecTextEncode(t *testing.T) {
	m := fbtype.NewMap()

	successfulTests := []struct {
		source fbtype.Date
		result string
	}{
		{source: fbtype.Date{Time: time.Date(2012, 3, 29, 0, 0, 0, 0, time.UTC), Valid: true}, result: "2012-03-29"},
		{source: fbtype.Date{Time: time.Date(2012, 3, 29, 10, 5, 45, 0, time.FixedZone("", -6*60*60)), Valid: true}, result: "2012-03-29"},
		{source: fbtype.Date{Time: time.Date(2012, 3, 29, 10, 5, 45, 555*1000*1000, time.FixedZone("", -6*60*60)), Valid: true}, result: "2012-03-29"},
		{source: fbtype.Date{Time: time.Date(789, 1, 2, 0, 0, 0, 0, time.UTC), Valid: true}, result: "0789-01-02"},
		{source: fbtype.Date{Time: time.Date(89, 1, 2, 0, 0, 0, 0, time.UTC), Valid: true}, result: "0089-01-02"},
		{source: fbtype.Date{Time: time.Date(9, 1, 2, 0, 0, 0, 0, time.UTC), Valid: true}, result: "0009-01-02"},
		{source: fbtype.Date{Time: time.Date(12200, 1, 2, 0, 0, 0, 0, time.UTC), Valid: true}, result: "12200-01-02"},
		{source: fbtype.Date{InfinityModifier: fbtype.Infinity, Valid: true}, result: "infinity"},
		{source: fbtype.Date{InfinityModifier: fbtype.NegativeInfinity, Valid: true}, result: "-infinity"},
	}
	for i, tt := range successfulTests {
		buf, err := m.Encode(fbtype.DateOID, fbtype.TextFormatCode, tt.source, nil)
		assert.NoErrorf(t, err, "%d", i)
		assert.Equalf(t, tt.result, string(buf), "%d", i)
	}
}

func TestDateMarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source fbtype.Date
		result string
	}{
		{source: fbtype.Date{}, result: "null"},
		{source: fbtype.Date{Time: time.Date(2012, 3, 29, 0, 0, 0, 0, time.UTC), Valid: true}, result: "\"2012-03-29\""},
		{source: fbtype.Date{Time: time.Date(2012, 3, 29, 10, 5, 45, 0, time.FixedZone("", -6*60*60)), Valid: true}, result: "\"2012-03-29\""},
		{source: fbtype.Date{Time: time.Date(2012, 3, 29, 10, 5, 45, 555*1000*1000, time.FixedZone("", -6*60*60)), Valid: true}, result: "\"2012-03-29\""},
		{source: fbtype.Date{InfinityModifier: fbtype.Infinity, Valid: true}, result: "\"infinity\""},
		{source: fbtype.Date{InfinityModifier: fbtype.NegativeInfinity, Valid: true}, result: "\"-infinity\""},
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

func TestDateUnmarshalJSON(t *testing.T) {
	successfulTests := []struct {
		source string
		result fbtype.Date
	}{
		{source: "null", result: fbtype.Date{}},
		{source: "\"2012-03-29\"", result: fbtype.Date{Time: time.Date(2012, 3, 29, 0, 0, 0, 0, time.UTC), Valid: true}},
		{source: "\"2012-03-29\"", result: fbtype.Date{Time: time.Date(2012, 3, 29, 10, 5, 45, 0, time.FixedZone("", -6*60*60)), Valid: true}},
		{source: "\"2012-03-29\"", result: fbtype.Date{Time: time.Date(2012, 3, 29, 10, 5, 45, 555*1000*1000, time.FixedZone("", -6*60*60)), Valid: true}},
		{source: "\"infinity\"", result: fbtype.Date{InfinityModifier: fbtype.Infinity, Valid: true}},
		{source: "\"-infinity\"", result: fbtype.Date{InfinityModifier: fbtype.NegativeInfinity, Valid: true}},
	}
	for i, tt := range successfulTests {
		var r fbtype.Date
		err := r.UnmarshalJSON([]byte(tt.source))
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}

		if r.Time.Year() != tt.result.Time.Year() || r.Time.Month() != tt.result.Time.Month() || r.Time.Day() != tt.result.Time.Day() || r.Valid != tt.result.Valid || r.InfinityModifier != tt.result.InfinityModifier {
			t.Errorf("%d: expected %v to convert to %v, but it was %v", i, tt.source, tt.result, r)
		}
	}
}
