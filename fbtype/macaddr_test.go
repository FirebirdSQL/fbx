package fbtype_test

import (
	"bytes"
	"context"
	"net"
	"testing"

	"rdb.red-soft.ru/fbx/fbxtest"
)

func isExpectedEqHardwareAddr(a any) func(any) bool {
	return func(v any) bool {
		aa := a.(net.HardwareAddr)
		vv := v.(net.HardwareAddr)

		if (aa == nil) != (vv == nil) {
			return false
		}

		if aa == nil {
			return true
		}

		return bytes.Equal(aa, vv)
	}
}

func TestMacaddrCodec(t *testing.T) {
	// Only testing known OID query exec modes as net.HardwareAddr could map to macaddr or macaddr8.
	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, fbxtest.KnownOIDQueryExecModes, "macaddr", []fbxtest.ValueRoundTripTest{
		{Param: mustParseMacaddr(t, "01:23:45:67:89:ab"), Result: new(net.HardwareAddr), Test: isExpectedEqHardwareAddr(mustParseMacaddr(t, "01:23:45:67:89:ab"))},
		{Param: "01:23:45:67:89:ab", Result: new(net.HardwareAddr), Test: isExpectedEqHardwareAddr(mustParseMacaddr(t, "01:23:45:67:89:ab"))},
		{Param: mustParseMacaddr(t, "01:23:45:67:89:ab"), Result: new(string), Test: isExpectedEq("01:23:45:67:89:ab")},
		{Param: nil, Result: new(*net.HardwareAddr), Test: isExpectedEq((*net.HardwareAddr)(nil))},
	})

	fbxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, fbxtest.KnownOIDQueryExecModes, "macaddr8", []fbxtest.ValueRoundTripTest{
		{Param: mustParseMacaddr(t, "01:23:45:67:89:ab:01:08"), Result: new(net.HardwareAddr), Test: isExpectedEqHardwareAddr(mustParseMacaddr(t, "01:23:45:67:89:ab:01:08"))},
		{Param: "01:23:45:67:89:ab:01:08", Result: new(net.HardwareAddr), Test: isExpectedEqHardwareAddr(mustParseMacaddr(t, "01:23:45:67:89:ab:01:08"))},
		{Param: mustParseMacaddr(t, "01:23:45:67:89:ab:01:08"), Result: new(string), Test: isExpectedEq("01:23:45:67:89:ab:01:08")},
		{Param: nil, Result: new(*net.HardwareAddr), Test: isExpectedEq((*net.HardwareAddr)(nil))},
	})
}
