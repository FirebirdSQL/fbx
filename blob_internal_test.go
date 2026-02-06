package fbx

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestBlobExplicitTxTextLengthAndEncoding(t *testing.T) {
	ctx := context.Background()
	runWithBlobStreamModesInternal(ctx, t, nil, func(ctx context.Context, t testing.TB, conn *Conn) {
		table := createBlobTestTableInternal(ctx, t, conn, "id integer not null primary key, txt_blob blob sub_type 1 character set utf8")

		text := "blob-text-ascii"
		if conn.fbConn != nil && conn.fbConn.Encoding != nil {
			switch conn.fbConn.Encoding.Charset {
			case "UTF8", "UNICODE_FSS":
				text = "blob-тест-utf"
			}
		}

		tx, err := conn.BeginTx(ctx, TxOptions{})
		require.NoError(t, err)
		dbtx, ok := tx.(*dbTx)
		require.True(t, ok, "unexpected tx type")

		_, err = tx.Exec(ctx, fmt.Sprintf("insert into %s (id, txt_blob) values (?, ?)", table), 1, text)
		require.NoError(t, err)

		rows, err := tx.Query(ctx, fmt.Sprintf("select txt_blob from %s where id = ?", table), 1)
		require.NoError(t, err)
		require.True(t, rows.Next())
		raw := append([]byte(nil), rows.RawValues()[0]...)
		fd := rows.FieldDescriptions()[0]
		rows.Close()
		require.NoError(t, rows.Err())

		rawID, err := fbconn.DecodeBlobID(raw)
		require.NoError(t, err)
		id := blobID(rawID)
		cfg, err := blobConfigFromField(conn, &fd)
		require.NoError(t, err)
		require.NotNil(t, cfg.encoding)

		blob := newInputBlob(conn, dbtx.tx, cfg, id)
		require.NoError(t, blob.openForRead(ctx))
		length, err := blob.Length(ctx)
		require.NoError(t, err)
		encoded, err := cfg.encoding.Encoder.Bytes([]byte(text))
		require.NoError(t, err)
		require.Equal(t, int64(len(encoded)), length)

		readText, err := blob.GetString(ctx)
		require.NoError(t, err)
		require.Equal(t, text, readText)

		require.NoError(t, tx.Commit(ctx))
	})
}

func TestInlineBlobExplicitTxSeekEOFAndCloseReleases(t *testing.T) {
	ctx := context.Background()
	runWithBlobStreamModesInternal(ctx, t, func(config *ConnConfig) {
		config.MaxInlineBlobSize = 65536
		config.MaxBlobCacheSize = 1 << 20
	}, func(ctx context.Context, t testing.TB, conn *Conn) {
		skipIfInlineBlobsUnsupported(ctx, t, conn)

		table := createBlobTestTableInternal(ctx, t, conn, "id integer not null primary key, bin_blob blob sub_type 0")
		data := makeBlobDataInternal(128)

		tx, err := conn.BeginTx(ctx, TxOptions{})
		require.NoError(t, err)
		_, ok := tx.(*dbTx)
		require.True(t, ok, "unexpected tx type")

		_, err = tx.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob) values (?, ?)", table), 1, data)
		require.NoError(t, err)

		var inlinePayloads [][]byte
		client := conn.fbConn.Client()
		client.SetInlineBlobHandler(func(op *fbproto.OpInlineBlob) {
			if op == nil {
				return
			}
			cp := append([]byte(nil), op.Data...)
			inlinePayloads = append(inlinePayloads, cp)
		})
		defer client.SetInlineBlobHandler(nil)

		var reader io.ReadCloser
		err = tx.QueryRow(ctx, fmt.Sprintf("select bin_blob from %s where id = ?", table), 1).Scan(&reader)
		require.NoError(t, err)

		seeker, ok := reader.(io.Seeker)
		require.True(t, ok)

		buf := make([]byte, 10)
		n, err := reader.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 10, n)
		require.Equal(t, data[:10], buf)

		pos, err := seeker.Seek(5, io.SeekStart)
		require.NoError(t, err)
		require.Equal(t, int64(5), pos)

		rest, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, data[5:], rest)

		n, err = reader.Read(make([]byte, 1))
		require.ErrorIs(t, err, io.EOF)
		require.Equal(t, 0, n)

		require.NotEmpty(t, inlinePayloads, "expected inline blob delivery")

		lazy, ok := reader.(*blobLazyReader)
		require.True(t, ok, "expected blob lazy reader")
		handle := lazy.blob.handle
		beforeLen, ok := inlineBlobDataLen(handle)
		require.True(t, ok, "expected inline blob handle")
		require.NotZero(t, beforeLen)
		require.NoError(t, reader.Close())
		afterLen, ok := inlineBlobDataLen(handle)
		require.True(t, ok, "expected inline blob handle")
		require.Equal(t, 0, afterLen)

		require.NoError(t, tx.Rollback(ctx))
	})
}

func fetchEngineVersion(ctx context.Context, t testing.TB, conn *Conn) string {
	t.Helper()
	var version string
	err := conn.QueryRow(ctx, "select rdb$get_context('SYSTEM','ENGINE_VERSION') from rdb$database").Scan(&version)
	require.NoError(t, err)
	return version
}

func skipIfInlineBlobsUnsupported(ctx context.Context, t testing.TB, conn *Conn) {
	t.Helper()
	engineVersion := fetchEngineVersion(ctx, t, conn)
	if !engineVersionSupportsInlineBlobs(engineVersion) {
		t.Skipf("inline blobs not supported by server engine version %q", engineVersion)
	}
}

func engineVersionSupportsInlineBlobs(version string) bool {
	major, ok := parseEngineMajorVersion(version)
	return ok && major >= 4
}

func parseEngineMajorVersion(version string) (int, bool) {
	match := regexp.MustCompile(`\d+`).FindString(version)
	if match == "" {
		return 0, false
	}
	major, err := strconv.Atoi(match)
	if err != nil {
		return 0, false
	}
	return major, true
}

func runWithBlobStreamModesInternal(
	ctx context.Context,
	t *testing.T,
	configure func(*ConnConfig),
	f func(context.Context, testing.TB, *Conn),
) {
	t.Helper()

	connString := os.Getenv("PGX_TEST_DATABASE")
	if connString == "" {
		t.Skipf("Skipping due to missing environment variable %v", "PGX_TEST_DATABASE")
	}

	for _, useStreamBlobs := range []bool{true, false} {
		t.Run(fmt.Sprintf("use_stream_blobs=%t", useStreamBlobs), func(t *testing.T) {
			config := mustParseConfig(t, connString)
			config.UseStreamBlobs = useStreamBlobs
			if configure != nil {
				configure(config)
			}
			conn := mustConnect(t, config)
			defer func() {
				require.NoError(t, conn.Close(context.Background()))
			}()
			f(ctx, t, conn)
		})
	}
}

func createBlobTestTableInternal(ctx context.Context, t testing.TB, conn *Conn, columns string) string {
	t.Helper()
	table := fmt.Sprintf("FBX_BLOB_TX_TEST_%d", time.Now().UnixNano())
	_, err := conn.Exec(ctx, fmt.Sprintf("create table %s (%s)", table, columns))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = conn.Exec(ctx, "drop table "+table)
	})
	return table
}

func makeBlobDataInternal(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte((i % 251) + 1)
	}
	return data
}

func inlineBlobDataLen(handle any) (int, bool) {
	if handle == nil {
		return 0, false
	}
	val := reflect.ValueOf(handle)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return 0, false
		}
		val = val.Elem()
	}
	if !val.IsValid() || val.Kind() != reflect.Struct {
		return 0, false
	}
	data := val.FieldByName("data")
	if !data.IsValid() || data.Kind() != reflect.Slice {
		return 0, false
	}
	return data.Len(), true
}
