package zeronull_test

import (
	"context"
	"testing"

	"github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/fbtype/zeronull"
	"github.com/firebirdsql/fbx/fbxtest"
)

func mustInt128(t testing.TB, s string) zeronull.Int128 {
	t.Helper()

	var z zeronull.Int128
	if _, ok := z.Int.SetString(s, 10); !ok {
		t.Fatalf("invalid int128 literal: %q", s)
	}
	return z
}

func isExpectedEqInt128(want zeronull.Int128) func(any) bool {
	return func(got any) bool {
		switch v := got.(type) {
		case zeronull.Int128:
			return v.Int.Cmp(&want.Int) == 0
		case *zeronull.Int128:
			if v == nil {
				return false
			}
			return v.Int.Cmp(&want.Int) == 0
		default:
			return false
		}
	}
}

func TestInt128Transcode(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		if getServerMajorVersionForConn(ctx, tb, conn) <= 3 {
			t.Skip("int128 is not supported by server engine version <= 3")
		}
	})

	val := mustInt128(t, "1")
	valBig := mustInt128(t, "123456789012345678901234567890")
	valNeg := mustInt128(t, "-123456789012345678901234567890")

	valMax := mustInt128(t, "170141183460469231731687303715884105727")
	valMin := mustInt128(t, "-170141183460469231731687303715884105728")

	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "int128", []fbxtest.ValueRoundTripTest{
		{
			Param:  val,
			Result: new(zeronull.Int128),
			Test:   isExpectedEqInt128(val),
		},
		{
			Param:  valBig,
			Result: new(zeronull.Int128),
			Test:   isExpectedEqInt128(valBig),
		},
		{
			Param:  valNeg,
			Result: new(zeronull.Int128),
			Test:   isExpectedEqInt128(valNeg),
		},
		{
			Param:  valMax,
			Result: new(zeronull.Int128),
			Test:   isExpectedEqInt128(valMax),
		},
		{
			Param:  valMin,
			Result: new(zeronull.Int128),
			Test:   isExpectedEqInt128(valMin),
		},
		{
			Param:  nil,
			Result: new(zeronull.Int128),
			Test:   isExpectedEqInt128(zeronull.Int128{}),
		},
		{
			Param:  zeronull.Int128{},
			Result: new(any),
			Test:   isExpectedEq(nil),
		},
	})
}
