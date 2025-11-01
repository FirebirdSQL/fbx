package fbtype_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/red-soft/fbx/fbtype"
	"github.com/red-soft/fbx/pgxtest"
)

func isExpectedEqBits(a any) func(any) bool {
	return func(v any) bool {
		ab := a.(fbtype.Bits)
		vb := v.(fbtype.Bits)
		return bytes.Equal(ab.Bytes, vb.Bytes) && ab.Len == vb.Len && ab.Valid == vb.Valid
	}
}

func TestBitsCodecBit(t *testing.T) {
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "bit(40)", []pgxtest.ValueRoundTripTest{
		{Param: fbtype.Bits{Bytes: []byte{0, 0, 0, 0, 0}, Len: 40, Valid: true}, Result: new(fbtype.Bits), Test: isExpectedEqBits(fbtype.Bits{Bytes: []byte{0, 0, 0, 0, 0}, Len: 40, Valid: true})},
		{Param: fbtype.Bits{Bytes: []byte{0, 1, 128, 254, 255}, Len: 40, Valid: true}, Result: new(fbtype.Bits), Test: isExpectedEqBits(fbtype.Bits{Bytes: []byte{0, 1, 128, 254, 255}, Len: 40, Valid: true})},
		{Param: fbtype.Bits{}, Result: new(fbtype.Bits), Test: isExpectedEqBits(fbtype.Bits{})},
		{Param: nil, Result: new(fbtype.Bits), Test: isExpectedEqBits(fbtype.Bits{})},
	})
}

func TestBitsCodecVarbit(t *testing.T) {
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "varbit", []pgxtest.ValueRoundTripTest{
		{Param: fbtype.Bits{Bytes: []byte{}, Len: 0, Valid: true}, Result: new(fbtype.Bits), Test: isExpectedEqBits(fbtype.Bits{Bytes: []byte{}, Len: 0, Valid: true})},
		{Param: fbtype.Bits{Bytes: []byte{0, 1, 128, 254, 255}, Len: 40, Valid: true}, Result: new(fbtype.Bits), Test: isExpectedEqBits(fbtype.Bits{Bytes: []byte{0, 1, 128, 254, 255}, Len: 40, Valid: true})},
		{Param: fbtype.Bits{Bytes: []byte{0, 1, 128, 254, 128}, Len: 33, Valid: true}, Result: new(fbtype.Bits), Test: isExpectedEqBits(fbtype.Bits{Bytes: []byte{0, 1, 128, 254, 128}, Len: 33, Valid: true})},
		{Param: fbtype.Bits{}, Result: new(fbtype.Bits), Test: isExpectedEqBits(fbtype.Bits{})},
		{Param: nil, Result: new(fbtype.Bits), Test: isExpectedEqBits(fbtype.Bits{})},
	})
}
