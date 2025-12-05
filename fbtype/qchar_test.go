package fbtype_test

import (
	"context"
	"math"
	"testing"

	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestQcharTranscode(t *testing.T) {
	skipCockroachDB(t, "Server does not support qchar")

	var tests []fbxtest.ValueRoundTripTest
	for i := 0; i <= math.MaxUint8; i++ {
		tests = append(tests, fbxtest.ValueRoundTripTest{Param: rune(i), Result: new(rune), Test: isExpectedEq(rune(i))})
		tests = append(tests, fbxtest.ValueRoundTripTest{Param: byte(i), Result: new(byte), Test: isExpectedEq(byte(i))})
	}
	tests = append(tests, fbxtest.ValueRoundTripTest{Param: nil, Result: new(*rune), Test: isExpectedEq((*rune)(nil))})
	tests = append(tests, fbxtest.ValueRoundTripTest{Param: nil, Result: new(*byte), Test: isExpectedEq((*byte)(nil))})

	// Can only test with known OIDs as rune and byte would be considered numbers.
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, fbxtest.KnownOIDQueryExecModes, `"char"`, tests)
}
