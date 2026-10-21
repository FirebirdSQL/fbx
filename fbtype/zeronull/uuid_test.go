package zeronull_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/firebirdsql/fbx"
	"github.com/firebirdsql/fbx/fbtype"
	"github.com/firebirdsql/fbx/fbtype/zeronull"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUUIDValueEncoding(t *testing.T) {
	t.Parallel()

	want := zeronull.UUID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	uuidValue, err := want.UUIDValue()
	require.NoError(t, err)
	require.Equal(t, fbtype.UUID{Bytes: [16]byte(want), Valid: true}, uuidValue)

	databaseValue, err := want.Value()
	require.NoError(t, err)
	require.Equal(t, "00010203-0405-0607-0809-0a0b0c0d0e0f", databaseValue)

	zero := zeronull.UUID{}
	uuidValue, err = zero.UUIDValue()
	require.NoError(t, err)
	require.Equal(t, fbtype.UUID{}, uuidValue)

	databaseValue, err = zero.Value()
	require.NoError(t, err)
	require.Nil(t, databaseValue)
}

func TestRegisterMapsUUIDValuesToOctets(t *testing.T) {
	t.Parallel()

	want := zeronull.UUID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	metadata := fbtype.NewTypeMetadata(0, 0, 0, 0)
	m := fbtype.NewMap()
	zeronull.Register(m)

	for _, value := range []any{want, &want} {
		dataType, ok := m.TypeForValue(value)
		require.True(t, ok, "value type %T", value)
		require.Equal(t, "octets", dataType.Name, "value type %T", value)

		encoded, err := m.Encode(metadata, value, nil)
		require.NoError(t, err, "value type %T", value)
		require.Equal(t, want[:], encoded, "value type %T", value)
	}
}

func TestUUIDScanClearsValueOnError(t *testing.T) {
	t.Parallel()

	uuid := zeronull.UUID{0xff}
	err := uuid.Scan("not-a-uuid")
	require.Error(t, err)
	require.Equal(t, zeronull.UUID{}, uuid)
}

func TestUUIDtoFB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	defaultConnTestRunner.RunTest(ctx, t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		const tableName = "fbtype_zeronull_uuid_test"

		_, err := conn.Exec(ctx, "recreate table "+tableName+"(u char(16) character set octets)")
		require.NoError(t, err)
		defer func() {
			_, _ = conn.Exec(ctx, "drop table "+tableName)
		}()

		_, err = conn.Exec(ctx, "insert into "+tableName+" values(gen_uuid())")
		require.NoError(t, err)

		var u zeronull.UUID
		var s string
		err = conn.QueryRow(ctx, "select u, uuid_to_char(u) from "+tableName).Scan(&u, &s)
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
		err = conn.QueryRow(ctx, "select uuid_to_char(u), cast(? as varchar(36)) from "+tableName, &u).Scan(&s2, &u2)
		require.NoError(t, err)
		uv2, err := u2.UUIDValue()
		require.NoError(t, err)
		assert.Equal(t, uv2.String(), strings.ToLower(s2))
	})
}
