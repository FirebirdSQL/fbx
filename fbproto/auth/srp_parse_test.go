package auth

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSrpServerData_ShortSalt(t *testing.T) {
	salt := make([]byte, 32)
	for i := range salt {
		salt[i] = byte(i + 1)
	}
	serverHex := "DEADBEEFCAFEBABE"

	var data []byte
	data = append(data, 0, 0)
	binary.LittleEndian.PutUint16(data[len(data)-2:], uint16(len(salt)))
	data = append(data, salt...)
	data = append(data, 0, 0)
	data = append(data, []byte(serverHex)...)

	gotSalt, gotHex, err := parseSrpServerData(data)
	require.NoError(t, err)
	require.Equal(t, salt, gotSalt)
	require.Equal(t, serverHex, gotHex)
}

func TestParseSrpServerData_ExtendedSalt(t *testing.T) {
	salt := []byte{1, 2, 3, 4, 5, 6, 7}
	serverHex := "ABCDEF012345"

	var data []byte
	data = append(data, 0, 0)
	binary.LittleEndian.PutUint16(data[len(data)-2:], uint16(1))
	sectionLen := 1 + len(salt)
	data = append(data, 0, 0)
	binary.LittleEndian.PutUint16(data[len(data)-2:], uint16(sectionLen))
	data = append(data, byte(1))
	data = append(data, salt...)
	data = append(data, []byte(serverHex)...)

	gotSalt, gotHex, err := parseSrpServerData(data)
	require.NoError(t, err)
	require.Equal(t, salt, gotSalt)
	require.Equal(t, serverHex, gotHex)
}
