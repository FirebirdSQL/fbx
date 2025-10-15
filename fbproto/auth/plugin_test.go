package auth

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSrpAuthPluginsClientDataMatchesPad(t *testing.T) {
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
		require.Equal(t, []byte(hex.EncodeToString(srpPad(p.ClientPublic))), p.InitialClientData())
		require.NotEmpty(t, p.Name())
	}
}

func TestBigIntToBytesUsesPad(t *testing.T) {
	v := big.NewInt(1234567890)
	require.Equal(t, srpPad(v), BigIntToBytes(v))
	require.Nil(t, BigIntToBytes(nil))
}
