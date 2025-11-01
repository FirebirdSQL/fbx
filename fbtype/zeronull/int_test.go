// Code generated from pgtype/zeronull/int_test.go.erb. DO NOT EDIT.

package zeronull_test

import (
	"context"
	"testing"

	"github.com/red-soft/fbx/fbtype/zeronull"
	"github.com/red-soft/fbx/pgxtest"
)

func TestInt2Transcode(t *testing.T) {
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "int2", []pgxtest.ValueRoundTripTest{
		{
			Param:  (zeronull.Int2)(1),
			Result: new(zeronull.Int2),
			Test:   isExpectedEq((zeronull.Int2)(1)),
		},
		{
			Param:  nil,
			Result: new(zeronull.Int2),
			Test:   isExpectedEq((zeronull.Int2)(0)),
		},
		{
			Param:  (zeronull.Int2)(0),
			Result: new(any),
			Test:   isExpectedEq(nil),
		},
	})
}

func TestInt4Transcode(t *testing.T) {
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "int4", []pgxtest.ValueRoundTripTest{
		{
			Param:  (zeronull.Int4)(1),
			Result: new(zeronull.Int4),
			Test:   isExpectedEq((zeronull.Int4)(1)),
		},
		{
			Param:  nil,
			Result: new(zeronull.Int4),
			Test:   isExpectedEq((zeronull.Int4)(0)),
		},
		{
			Param:  (zeronull.Int4)(0),
			Result: new(any),
			Test:   isExpectedEq(nil),
		},
	})
}

func TestInt8Transcode(t *testing.T) {
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "int8", []pgxtest.ValueRoundTripTest{
		{
			Param:  (zeronull.Int8)(1),
			Result: new(zeronull.Int8),
			Test:   isExpectedEq((zeronull.Int8)(1)),
		},
		{
			Param:  nil,
			Result: new(zeronull.Int8),
			Test:   isExpectedEq((zeronull.Int8)(0)),
		},
		{
			Param:  (zeronull.Int8)(0),
			Result: new(any),
			Test:   isExpectedEq(nil),
		},
	})
}
