package auth

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSrpPluginsClientDataMatchesPad(t *testing.T) {
	constructors := []func() *SrpAuthPlugin{
		NewSrpAuthPlugin,
		NewSrp256AuthPlugin,
		NewSrp224AuthPlugin,
		NewSrp384AuthPlugin,
		NewSrp512AuthPlugin,
	}
	for _, ctor := range constructors {
		p := ctor()
		require.NotNil(t, p)
		// Now GetClientData returns hex-encoded A
		require.Equal(t, []byte(hex.EncodeToString(srpPad(p.ClientPublic))), p.GetClientData())
		// Plugin name is not empty
		require.NotEmpty(t, p.GetName())
	}
}

func TestBigIntToBytesUsesPad(t *testing.T) {
	v := big.NewInt(1234567890)
	require.Equal(t, srpPad(v), BigIntToBytes(v))
	require.Nil(t, BigIntToBytes(nil))
}
