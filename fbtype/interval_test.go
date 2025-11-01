package fbtype_test

import (
	"context"
	"testing"
	"time"

	"github.com/red-soft/fbx/fbtype"
	"github.com/red-soft/fbx/pgxtest"
	"github.com/stretchr/testify/assert"
)

func TestIntervalCodec(t *testing.T) {
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "interval", []pgxtest.ValueRoundTripTest{
		{Param: fbtype.Interval{Microseconds: 1, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Microseconds: 1, Valid: true})},
		{Param: fbtype.Interval{Microseconds: 1000000, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Microseconds: 1000000, Valid: true})},
		{Param: fbtype.Interval{Microseconds: 1000001, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Microseconds: 1000001, Valid: true})},
		{Param: fbtype.Interval{Microseconds: 123202800000000, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Microseconds: 123202800000000, Valid: true})},
		{Param: fbtype.Interval{Days: 1, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Days: 1, Valid: true})},
		{Param: fbtype.Interval{Months: 1, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Months: 1, Valid: true})},
		{Param: fbtype.Interval{Months: 12, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Months: 12, Valid: true})},
		{Param: fbtype.Interval{Months: 13, Days: 15, Microseconds: 1000001, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Months: 13, Days: 15, Microseconds: 1000001, Valid: true})},
		{Param: fbtype.Interval{Microseconds: -1, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Microseconds: -1, Valid: true})},
		{Param: fbtype.Interval{Microseconds: -1000000, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Microseconds: -1000000, Valid: true})},
		{Param: fbtype.Interval{Microseconds: -1000001, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Microseconds: -1000001, Valid: true})},
		{Param: fbtype.Interval{Microseconds: -123202800000000, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Microseconds: -123202800000000, Valid: true})},
		{Param: fbtype.Interval{Days: -1, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Days: -1, Valid: true})},
		{Param: fbtype.Interval{Months: -1, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Months: -1, Valid: true})},
		{Param: fbtype.Interval{Months: -12, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Months: -12, Valid: true})},
		{Param: fbtype.Interval{Months: -13, Days: -15, Microseconds: -1000001, Valid: true}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Months: -13, Days: -15, Microseconds: -1000001, Valid: true})},
		{Param: "1 second", Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Microseconds: 1000000, Valid: true})},
		{Param: "1.000001 second", Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Microseconds: 1000001, Valid: true})},
		{Param: "34223 hours", Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Microseconds: 123202800000000, Valid: true})},
		{Param: "1 day", Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Days: 1, Valid: true})},
		{Param: "1 month", Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Months: 1, Valid: true})},
		{Param: "1 year", Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Months: 12, Valid: true})},
		{Param: "-13 mon", Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{Months: -13, Valid: true})},
		{Param: time.Hour, Result: new(time.Duration), Test: isExpectedEq(time.Hour)},
		{Param: fbtype.Interval{Months: 1, Days: 1, Valid: true}, Result: new(time.Duration), Test: isExpectedEq(time.Duration(2678400000000000))},
		{Param: fbtype.Interval{}, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{})},
		{Param: nil, Result: new(fbtype.Interval), Test: isExpectedEq(fbtype.Interval{})},
	})
}

func TestIntervalTextEncode(t *testing.T) {
	m := fbtype.NewMap()

	successfulTests := []struct {
		source fbtype.Interval
		result string
	}{
		{source: fbtype.Interval{Months: 2, Days: 1, Microseconds: 0, Valid: true}, result: "2 mon 1 day 00:00:00"},
		{source: fbtype.Interval{Months: 0, Days: 0, Microseconds: 0, Valid: true}, result: "00:00:00"},
		{source: fbtype.Interval{Months: 0, Days: 0, Microseconds: 6 * 60 * 1000000, Valid: true}, result: "00:06:00"},
		{source: fbtype.Interval{Months: 0, Days: 1, Microseconds: 6*60*1000000 + 30, Valid: true}, result: "1 day 00:06:00.000030"},
	}
	for i, tt := range successfulTests {
		buf, err := m.Encode(fbtype.DateOID, fbtype.TextFormatCode, tt.source, nil)
		assert.NoErrorf(t, err, "%d", i)
		assert.Equalf(t, tt.result, string(buf), "%d", i)
	}
}
