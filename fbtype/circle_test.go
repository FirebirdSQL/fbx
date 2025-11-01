package fbtype_test

import (
	"context"
	"testing"

	"github.com/red-soft/fbx/fbtype"
	"github.com/red-soft/fbx/pgxtest"
)

func TestCircleTranscode(t *testing.T) {
	skipCockroachDB(t, "Server does not support box type")

	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "circle", []pgxtest.ValueRoundTripTest{
		{Param: fbtype.Circle{P: fbtype.Vec2{1.234, 5.67890123}, R: 3.5, Valid: true}, Result: new(fbtype.Circle), Test: isExpectedEq(fbtype.Circle{P: fbtype.Vec2{1.234, 5.67890123}, R: 3.5, Valid: true})},
		{Param: fbtype.Circle{P: fbtype.Vec2{1.234, 5.67890123}, R: 3.5, Valid: true}, Result: new(fbtype.Circle), Test: isExpectedEq(fbtype.Circle{P: fbtype.Vec2{1.234, 5.67890123}, R: 3.5, Valid: true})},
		{Param: fbtype.Circle{}, Result: new(fbtype.Circle), Test: isExpectedEq(fbtype.Circle{})},
		{Param: nil, Result: new(fbtype.Circle), Test: isExpectedEq(fbtype.Circle{})},
	})
}
