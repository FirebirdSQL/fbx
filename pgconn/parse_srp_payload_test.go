package pgconn

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSrpPayload_ShortSalt(t *testing.T) {
	// salt length > 8 to avoid being misinterpreted as extended section count
	salt := make([]byte, 32)
	for i := range salt {
		salt[i] = byte(i + 1)
	}
	serverHex := "DEADBEEFCAFEBABE"

	var data []byte
	// salt length (LE)
	data = append(data, 0, 0)
	binary.LittleEndian.PutUint16(data[len(data)-2:], uint16(len(salt)))
	// salt bytes
	data = append(data, salt...)
	// 2 reserved bytes
	data = append(data, 0, 0)
	// server public hex string bytes
	data = append(data, []byte(serverHex)...)

	gotSalt, gotHex, err := parseSrpPayload(data)
	require.NoError(t, err)
	require.Equal(t, salt, gotSalt)
	require.Equal(t, serverHex, gotHex)
}

func TestParseSrpPayload_ExtendedSalt(t *testing.T) {
	salt := []byte{1, 2, 3, 4, 5, 6, 7}
	serverHex := "ABCDEF012345"

	var data []byte
	// section count = 1 (LE)
	data = append(data, 0, 0)
	binary.LittleEndian.PutUint16(data[len(data)-2:], uint16(1))
	// one section: length = 1 (header) + saltLen
	sectionLen := 1 + len(salt)
	data = append(data, 0, 0)
	binary.LittleEndian.PutUint16(data[len(data)-2:], uint16(sectionLen))
	// section data: leading header/tag < 32, then salt
	data = append(data, byte(1))
	data = append(data, salt...)
	// server public hex string bytes
	data = append(data, []byte(serverHex)...)

	gotSalt, gotHex, err := parseSrpPayload(data)
	require.NoError(t, err)
	require.Equal(t, salt, gotSalt)
	require.Equal(t, serverHex, gotHex)
}
