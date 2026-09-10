package zeronull_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbtype"
	"rdb.red-soft.ru/fbx/fbtype/zeronull"
)

func TestUUIDValueEncoding(t *testing.T) {
	t.Parallel()

	want := zeronull.UUID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	uuidValue, err := want.UUIDValue()
	require.NoError(t, err)
	require.Equal(t, fbtype.UUID{Bytes: [16]byte(want), Valid: true}, uuidValue)

	binaryValue, err := want.BytesValue()
	require.NoError(t, err)
	require.Equal(t, want[:], binaryValue)

	databaseValue, err := want.Value()
	require.NoError(t, err)
	require.Equal(t, "00010203-0405-0607-0809-0a0b0c0d0e0f", databaseValue)

	zero := zeronull.UUID{}
	uuidValue, err = zero.UUIDValue()
	require.NoError(t, err)
	require.Equal(t, fbtype.UUID{}, uuidValue)

	binaryValue, err = zero.BytesValue()
	require.NoError(t, err)
	require.Nil(t, binaryValue)

	databaseValue, err = zero.Value()
	require.NoError(t, err)
	require.Nil(t, databaseValue)
}

func TestUUIDtoFB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	defaultConnTestRunner.RunTest(ctx, t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		_, err := conn.Exec(ctx, "recreate table u(u char(16) character set octets)")
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "insert into u values(gen_uuid())")
		require.NoError(t, err)

		var u zeronull.UUID
		var s string
		err = conn.QueryRow(ctx, "select u, uuid_to_char(u) from u").Scan(&u, &s)
		require.NoError(t, err)
		uv, err := u.UUIDValue()
		require.NoError(t, err)
		assert.Equal(t, uv.String(), strings.ToLower(s))

		var binaryRoundTrip zeronull.UUID
		err = conn.QueryRow(ctx,
			"select cast(? as char(16) character set octets) from rdb$database",
			u,
		).Scan(&binaryRoundTrip)
		require.NoError(t, err)
		require.Equal(t, u, binaryRoundTrip)

		var u2 zeronull.UUID
		var s2 string
		err = conn.QueryRow(ctx, "select uuid_to_char(u), cast(? as varchar(36)) from u", &u).Scan(&s2, &u2)
		require.NoError(t, err)
		uv2, err := u2.UUIDValue()
		require.NoError(t, err)
		assert.Equal(t, uv2.String(), strings.ToLower(s2))
	})
}
