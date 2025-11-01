package fbtype_test

import (
	"context"
	"testing"

	"github.com/red-soft/fbx/fbtype"
	"github.com/red-soft/fbx/fbxtest"
)

func TestLtreeCodec(t *testing.T) {
	skipCockroachDB(t, "Server does not support type ltree")

	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, fbxtest.KnownOIDQueryExecModes, "ltree", []fbxtest.ValueRoundTripTest{
		{
			Param:  "A.B.C",
			Result: new(string),
			Test:   isExpectedEq("A.B.C"),
		},
		{
			Param:  fbtype.Text{String: "", Valid: true},
			Result: new(fbtype.Text),
			Test:   isExpectedEq(fbtype.Text{String: "", Valid: true}),
		},
	})
}
