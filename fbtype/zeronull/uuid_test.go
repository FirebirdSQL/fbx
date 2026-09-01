package zeronull_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbtype/zeronull"
	"rdb.red-soft.ru/fbx/fbxtest"
)

func TestUUIDTranscode(t *testing.T) {
	t.Skip()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbxtest.RunValueRoundTripTests(ctx, t, defaultConnTestRunner, nil, "uuid", []fbxtest.ValueRoundTripTest{
		{
			Param:  (zeronull.UUID)([16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}),
			Result: new(zeronull.UUID),
			Test:   isExpectedEq((zeronull.UUID)([16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})),
		},
		{
			Param:  nil,
			Result: new(zeronull.UUID),
			Test:   isExpectedEq((zeronull.UUID)([16]byte{})),
		},
		{
			Param:  (zeronull.UUID)([16]byte{}),
			Result: new(any),
			Test:   isExpectedEq(nil),
		},
	})
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

		var u2 zeronull.UUID
		var s2 string
		err = conn.QueryRow(ctx, "select uuid_to_char(u), cast(? as varchar(36)) from u", &u).Scan(&s2, &u2)
		require.NoError(t, err)
		uv2, err := u2.UUIDValue()
		require.NoError(t, err)
		assert.Equal(t, uv2.String(), strings.ToLower(s2))
	})
}
