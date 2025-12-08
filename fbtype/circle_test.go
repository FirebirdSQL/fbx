package fbtype_test

import (
	"context"
	"testing"

	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestCircleTranscode(t *testing.T) {
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "circle", []fbxtest.ValueRoundTripTest{
		{Param: fbtype.Circle{P: fbtype.Vec2{1.234, 5.67890123}, R: 3.5, Valid: true}, Result: new(fbtype.Circle), Test: isExpectedEq(fbtype.Circle{P: fbtype.Vec2{1.234, 5.67890123}, R: 3.5, Valid: true})},
		{Param: fbtype.Circle{P: fbtype.Vec2{1.234, 5.67890123}, R: 3.5, Valid: true}, Result: new(fbtype.Circle), Test: isExpectedEq(fbtype.Circle{P: fbtype.Vec2{1.234, 5.67890123}, R: 3.5, Valid: true})},
		{Param: fbtype.Circle{}, Result: new(fbtype.Circle), Test: isExpectedEq(fbtype.Circle{})},
		{Param: nil, Result: new(fbtype.Circle), Test: isExpectedEq(fbtype.Circle{})},
	})
}
