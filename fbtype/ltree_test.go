package fbtype_test

import (
	"context"
	"testing"

	"github.com/red-soft/fbx/fbtype"
	"github.com/red-soft/fbx/pgxtest"
)

func TestLtreeCodec(t *testing.T) {
	skipCockroachDB(t, "Server does not support type ltree")

	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, pgxtest.KnownOIDQueryExecModes, "ltree", []pgxtest.ValueRoundTripTest{
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
