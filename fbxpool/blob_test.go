package fbxpool_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbxpool"
	"github.com/stretchr/testify/require"
)

func TestPoolBlobBinary(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	t.Cleanup(conn.Release)

	table := createPoolBlobTable(t, ctx, conn, "id integer primary key, data blob sub_type 0")

	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte((i % 251) + 1)
	}

	_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s(id, data) values (?, ?)", table), 1, bytes.NewReader(data))
	require.NoError(t, err)

	var got []byte
	err = conn.QueryRow(ctx, fmt.Sprintf("select data from %s where id = ?", table), 1).Scan(&got)
	require.NoError(t, err)
	require.Equal(t, data, got)
}

func TestPoolBlobTextCharsets(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := fbxpool.New(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	t.Cleanup(conn.Release)

	tests := []struct {
		name    string
		charset string
		text    string
	}{
		{name: "utf8", charset: "UTF8", text: "pool-utf-" + string([]rune{0x20AC, 0x00A9})},
		{name: "unicode_fss", charset: "UNICODE_FSS", text: "pool-fss-" + string([]rune{0x0422, 0x0435, 0x0441, 0x0442})},
		{name: "ascii", charset: "ASCII", text: "pool-ascii"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			table := fmt.Sprintf("FBXPB_%x", time.Now().UnixNano())
			_, err := conn.Exec(ctx, fmt.Sprintf("recreate table %s(id integer primary key, data blob sub_type 1 character set %s)", table, tc.charset))
			if err != nil {
				t.Skipf("charset %s not supported: %v", tc.charset, err)
			}
			t.Cleanup(func() {
				_, _ = conn.Exec(context.Background(), "drop table "+table)
			})

			_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s(id, data) values (?, ?)", table), 1, tc.text)
			require.NoError(t, err)

			var got string
			err = conn.QueryRow(ctx, fmt.Sprintf("select data from %s where id = ?", table), 1).Scan(&got)
			require.NoError(t, err)
			require.Equal(t, tc.text, got)
		})
	}
}

func createPoolBlobTable(t testing.TB, ctx context.Context, conn *fbxpool.Conn, columns string) string {
	t.Helper()
	table := fmt.Sprintf("FBXPB_%x", time.Now().UnixNano())
	_, err := conn.Exec(ctx, fmt.Sprintf("recreate table %s (%s)", table, columns))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), "drop table "+table)
	})
	return table
}
