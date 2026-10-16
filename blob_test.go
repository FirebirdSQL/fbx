package fbx_test

import (
	"bytes"
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestBlobReadBytesAndReader(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	runWithBlobStreamModes(ctx, t, func(ctx context.Context, t testing.TB, config *fbx.ConnConfig) {
		// Disable inline blob path so seek behavior is determined only by stream/segmented mode.
		config.MaxInlineBlobSize = 0
		config.MaxBlobCacheSize = 0
	}, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		table := createBlobTestTable(ctx, t, conn)
		dataA := makeBlobData(16384*2 + 17)
		dataB := makeBlobData(16384 + 9)
		dataC := makeBlobData(1024 + 123)
		textA := strings.Repeat("blob-text-a-", 300)
		textB := strings.Repeat("blob-text-b-", 200)
		textC := strings.Repeat("blob-text-c-", 100)

		_, err := conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob, txt_blob) values (?, ?, ?)", table), 1, dataA, textA)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob, txt_blob) values (?, ?, ?)", table), 2, bytes.NewReader(dataB), textB)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob, txt_blob) values (?, ?, ?)", table), 3, dataC, textC)
		require.NoError(t, err)

		rows, err := conn.Query(ctx, fmt.Sprintf("select id, bin_blob, txt_blob from %s order by id", table))
		require.NoError(t, err)
		defer rows.Close()

		// Row 1: scan binary blob into []byte, text blob into string.
		require.True(t, rows.Next())
		var id int
		var readBytes []byte
		var readText string
		err = rows.Scan(&id, &readBytes, &readText)
		require.NoError(t, err)
		require.Equal(t, 1, id)
		require.Equal(t, dataA, readBytes)
		require.Equal(t, textA, readText)

		// Row 2: scan binary blob into reader, text blob into string.
		require.True(t, rows.Next())
		var binReader io.ReadCloser
		err = rows.Scan(&id, &binReader, &readText)
		require.NoError(t, err)
		require.Equal(t, 2, id)
		require.NotNil(t, binReader)

		seeker, ok := binReader.(io.Seeker)
		require.True(t, ok)
		readOffset := 0
		_, err = seeker.Seek(10, io.SeekStart)
		if conn.Config().UseStreamBlobs {
			require.NoError(t, err)
			readOffset = 10
		} else {
			require.ErrorContains(t, err, "invalid BLOB type for operation")
		}
		buf := make([]byte, 25)
		n, err := binReader.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 25, n)
		require.Equal(t, dataB[readOffset:readOffset+25], buf)
		require.NoError(t, binReader.Close())
		_, err = binReader.Read(make([]byte, 1))
		require.EqualError(t, err, "blob is closed")
		require.Equal(t, textB, readText)

		// Row 3: scan binary blob into []byte, text blob into reader.
		require.True(t, rows.Next())
		var txtReader io.ReadCloser
		err = rows.Scan(&id, &readBytes, &txtReader)
		require.NoError(t, err)
		require.Equal(t, 3, id)
		require.Equal(t, dataC, readBytes)
		require.NotNil(t, txtReader)
		textBuf, err := io.ReadAll(txtReader)
		require.NoError(t, err)
		require.Equal(t, textC, string(textBuf))
		require.NoError(t, txtReader.Close())

		require.False(t, rows.Next())
		require.NoError(t, rows.Err())
	})
}

func TestClobReadString(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	runWithBlobStreamModes(ctx, t, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		table := createBlobTestTable(ctx, t, conn)
		textValue := strings.Repeat("clob-data-", 1000)

		_, err := conn.Exec(ctx, fmt.Sprintf("insert into %s (id, txt_blob) values (?, ?)", table), 1, textValue)
		require.NoError(t, err)

		var text string
		err = conn.QueryRow(ctx, fmt.Sprintf("select txt_blob from %s where id = ?", table), 1).Scan(&text)
		require.NoError(t, err)
		require.Equal(t, textValue, text)
	})
}

func TestBlobExecParamAutoCreate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	runWithBlobStreamModes(ctx, t, func(ctx context.Context, t testing.TB, config *fbx.ConnConfig) {
		config.DefaultQueryExecMode = fbx.QueryExecModeCacheStatement
	}, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		table := createBlobTestTable(ctx, t, conn)
		dataA := makeBlobData(4096 + 17)
		dataB := makeBlobData(2048 + 7)
		textA := "text-blob-a"
		textB := "text-blob-b"

		_, err := conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob, txt_blob) values (?, ?, ?)", table), 1, dataA, textA)
		require.NoError(t, err)

		reader := bytes.NewReader(dataB)
		_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob, txt_blob) values (?, ?, ?)", table), 2, reader, textB)
		require.NoError(t, err)

		var readBytes []byte
		var readText string
		err = conn.QueryRow(ctx, fmt.Sprintf("select bin_blob, txt_blob from %s where id = ?", table), 1).Scan(&readBytes, &readText)
		require.NoError(t, err)
		require.Equal(t, dataA, readBytes)
		require.Equal(t, textA, readText)

		err = conn.QueryRow(ctx, fmt.Sprintf("select bin_blob, txt_blob from %s where id = ?", table), 2).Scan(&readBytes, &readText)
		require.NoError(t, err)
		require.Equal(t, dataB, readBytes)
		require.Equal(t, textB, readText)
	})
}

func TestBlobExecParamExplicitTxCommit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	runWithBlobStreamModes(ctx, t, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		table := createBlobTestTable(ctx, t, conn)
		data := makeBlobData(2048 + 3)
		text := "explicit-tx-blob"

		tx, err := conn.BeginTx(ctx, fbx.TxOptions{})
		require.NoError(t, err)

		_, err = tx.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob, txt_blob) values (?, ?, ?)", table), 1, data, text)
		require.NoError(t, err)

		var binReader io.ReadCloser
		var readText string
		err = tx.QueryRow(ctx, fmt.Sprintf("select bin_blob, txt_blob from %s where id = ?", table), 1).Scan(&binReader, &readText)
		require.NoError(t, err)
		readBytes, err := io.ReadAll(binReader)
		require.NoError(t, err)
		require.NoError(t, binReader.Close())
		require.Equal(t, data, readBytes)
		require.Equal(t, text, readText)

		require.NoError(t, tx.Commit(ctx))

		var persisted []byte
		err = conn.QueryRow(ctx, fmt.Sprintf("select bin_blob from %s where id = ?", table), 1).Scan(&persisted)
		require.NoError(t, err)
		require.Equal(t, data, persisted)
	})
}

func TestBlobExecParamExplicitTxRollback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	runWithBlobStreamModes(ctx, t, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		table := createBlobTestTable(ctx, t, conn)
		data := makeBlobData(1024 + 7)

		tx, err := conn.BeginTx(ctx, fbx.TxOptions{})
		require.NoError(t, err)

		_, err = tx.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob) values (?, ?)", table), 1, bytes.NewReader(data))
		require.NoError(t, err)
		require.NoError(t, tx.Rollback(ctx))

		var count int
		err = conn.QueryRow(ctx, fmt.Sprintf("select count(*) from %s where id = ?", table), 1).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})
}

func TestInlineBlobHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	runWithBlobStreamModes(ctx, t, func(ctx context.Context, t testing.TB, config *fbx.ConnConfig) {
		config.MaxInlineBlobSize = 65536
		config.MaxBlobCacheSize = 1 << 20
	}, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		skipIfInlineBlobsUnsupported(ctx, t, conn)

		table := createBlobTestTable(ctx, t, conn)
		data := makeBlobData(128)

		_, err := conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob) values (?, ?)", table), 1, data)
		require.NoError(t, err)

		var inlinePayloads [][]byte
		client := conn.FbConn().Client()
		client.SetInlineBlobHandler(func(op *fbproto.OpInlineBlob) {
			if op == nil {
				return
			}
			cp := append([]byte(nil), op.Data...)
			inlinePayloads = append(inlinePayloads, cp)
		})
		defer client.SetInlineBlobHandler(nil)

		var readBytes []byte
		err = conn.QueryRow(ctx, fmt.Sprintf("select bin_blob from %s where id = ?", table), 1).Scan(&readBytes)
		require.NoError(t, err)
		require.Equal(t, data, readBytes)

		require.NotEmpty(t, inlinePayloads, "expected inline blob delivery")
		matched := false
		for _, payload := range inlinePayloads {
			if bytes.Equal(payload, data) {
				matched = true
				break
			}
		}
		require.True(t, matched, "inline blob payload mismatch")
	})
}

func skipIfInlineBlobsUnsupported(ctx context.Context, t testing.TB, conn *fbx.Conn) {
	t.Helper()
	version := conn.ServerVersion()
	if version.Compare(5, 0, 3) < 0 || version.Compare(5, 0, 8) >= 0 {
		t.Skipf("inline blobs not supported by server engine version %q", version)
	}
}

func TestBlobExecParamFromFile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	runWithBlobStreamModes(ctx, t, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		table := createBlobTestTable(ctx, t, conn)
		data := makeBlobData(5 * 1024 * 1024)

		file, err := os.CreateTemp(t.TempDir(), "fbx-blob-file-*")
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = file.Close()
			_ = os.Remove(file.Name())
		})

		_, err = file.Write(data)
		require.NoError(t, err)
		_, err = file.Seek(0, io.SeekStart)
		require.NoError(t, err)

		_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob) values (?, ?)", table), 1, file)
		require.NoError(t, err)

		var readBytes []byte
		err = conn.QueryRow(ctx, fmt.Sprintf("select bin_blob from %s where id = ?", table), 1).Scan(&readBytes)
		require.NoError(t, err)
		require.Equal(t, data, readBytes)
	})
}

type byteValuer struct {
}

func (byteValuer) Value() (driver.Value, error) {
	return makeBlobData(2048 + 7), nil
}

type stringValuer struct {
}

func (stringValuer) Value() (driver.Value, error) {
	return "text-blob-a", nil
}

type nilValuer struct {
}

func (nilValuer) Value() (driver.Value, error) {
	return nil, nil
}

type nilPointerValuer struct{}

func (*nilPointerValuer) Value() (driver.Value, error) {
	return []byte("nil-pointer-value"), nil
}

type valueReceiverValuer string

func (v valueReceiverValuer) Value() (driver.Value, error) {
	return string(v), nil
}

type nilSliceValuer []byte

func (nilSliceValuer) Value() (driver.Value, error) {
	return []byte("nil-slice-value"), nil
}

type nilMapValuer map[string]string

func (nilMapValuer) Value() (driver.Value, error) {
	return []byte("nil-map-value"), nil
}

type nilFuncValuer func()

func (nilFuncValuer) Value() (driver.Value, error) {
	return []byte("nil-func-value"), nil
}

type nilChanValuer chan struct{}

func (nilChanValuer) Value() (driver.Value, error) {
	return []byte("nil-chan-value"), nil
}

type nilFuncErrorValuer func()

func (nilFuncErrorValuer) Value() (driver.Value, error) {
	return nil, errNilFuncValuer
}

var errNilFuncValuer = errors.New("nil func valuer failure")

type readerValuer struct {
	*bytes.Reader
	value driver.Value
	err   error
	calls int
}

func (v *readerValuer) Value() (driver.Value, error) {
	v.calls++
	return v.value, v.err
}

func TestBlobExecWithDriverValuer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	runWithBlobStreamModes(ctx, t, func(ctx context.Context, t testing.TB, config *fbx.ConnConfig) {
		config.DefaultQueryExecMode = fbx.QueryExecModeCacheStatement
	}, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		table := createBlobTestTable(ctx, t, conn)
		dataA := byteValuer{}
		textA := stringValuer{}
		dataB := nilValuer{}
		textB := nilValuer{}

		_, err := conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob, txt_blob) values (?, ?, ?)", table), 1, dataA, textA)
		require.NoError(t, err)

		_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob, txt_blob) values (?, ?, ?)", table), 2, dataB, textB)
		require.NoError(t, err)

		var readBytes []byte
		var readText string
		err = conn.QueryRow(ctx, fmt.Sprintf("select bin_blob, txt_blob from %s where id = ?", table), 1).Scan(&readBytes, &readText)
		require.NoError(t, err)
		valDataA, _ := dataA.Value()
		valTextA, _ := textA.Value()
		require.Equal(t, valDataA, readBytes)
		require.Equal(t, valTextA, readText)

		var readNilBytes []byte
		var readNilText []byte
		err = conn.QueryRow(ctx, fmt.Sprintf("select bin_blob, txt_blob from %s where id = ?", table), 2).Scan(&readNilBytes, &readNilText)
		require.NoError(t, err)
		require.Nil(t, readNilBytes)
		require.Nil(t, readNilText)
	})
}

func TestBlobExecNilDriverValuer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	runWithBlobStreamModes(ctx, t, func(ctx context.Context, t testing.TB, config *fbx.ConnConfig) {
		config.DefaultQueryExecMode = fbx.QueryExecModeCacheStatement
	}, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		table := createBlobTestTable(ctx, t, conn)

		_, err := conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob) values (?, ?)", table), 1, (*nilPointerValuer)(nil))
		require.NoError(t, err)
		var got []byte
		require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf("select bin_blob from %s where id = ?", table), 1).Scan(&got))
		require.Equal(t, []byte("nil-pointer-value"), got)

		_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob) values (?, ?)", table), 2, nilSliceValuer(nil))
		require.NoError(t, err)
		require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf("select bin_blob from %s where id = ?", table), 2).Scan(&got))
		require.Equal(t, []byte("nil-slice-value"), got)

		_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob) values (?, ?)", table), 3, nilMapValuer(nil))
		require.NoError(t, err)
		require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf("select bin_blob from %s where id = ?", table), 3).Scan(&got))
		require.Equal(t, []byte("nil-map-value"), got)

		_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob) values (?, ?)", table), 4, nilFuncValuer(nil))
		require.NoError(t, err)
		require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf("select bin_blob from %s where id = ?", table), 4).Scan(&got))
		require.Equal(t, []byte("nil-func-value"), got)

		_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob) values (?, ?)", table), 5, nilChanValuer(nil))
		require.NoError(t, err)
		require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf("select bin_blob from %s where id = ?", table), 5).Scan(&got))
		require.Equal(t, []byte("nil-chan-value"), got)

		_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob) values (?, ?)", table), 6, nilFuncErrorValuer(nil))
		require.ErrorIs(t, err, errNilFuncValuer)

		_, err = conn.Exec(ctx, fmt.Sprintf("insert into %s (id, txt_blob) values (?, ?)", table), 7, (*valueReceiverValuer)(nil))
		require.NoError(t, err)
		require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf("select txt_blob from %s where id = ?", table), 7).Scan(&got))
		require.Nil(t, got)
	})
}

func TestBlobExecReaderPreferredOverDriverValuer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	runWithBlobStreamModes(ctx, t, func(ctx context.Context, t testing.TB, config *fbx.ConnConfig) {
		config.DefaultQueryExecMode = fbx.QueryExecModeCacheStatement
	}, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		table := createBlobTestTable(ctx, t, conn)
		v := &readerValuer{
			Reader: bytes.NewReader([]byte("reader-data")),
			value:  []byte("valuer-data"),
			err:    errors.New("valuer should not be called"),
		}

		_, err := conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob) values (?, ?)", table), 1, v)
		require.NoError(t, err)
		require.Zero(t, v.calls)

		var got []byte
		require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf("select bin_blob from %s where id = ?", table), 1).Scan(&got))
		require.Equal(t, []byte("reader-data"), got)
	})
}

func TestBlobExecNilReaderValuer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	runWithBlobStreamModes(ctx, t, func(ctx context.Context, t testing.TB, config *fbx.ConnConfig) {
		config.DefaultQueryExecMode = fbx.QueryExecModeCacheStatement
	}, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		table := createBlobTestTable(ctx, t, conn)

		_, err := conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob) values (?, ?)", table), 1, (*readerValuer)(nil))
		require.NoError(t, err)

		var got []byte
		require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf("select bin_blob from %s where id = ?", table), 1).Scan(&got))
		require.Nil(t, got)
	})
}

type nilSliceReaderValuer []byte

func (nilSliceReaderValuer) Value() (driver.Value, error) {
	return []byte("slice-reader-valuer"), nil
}

func (s nilSliceReaderValuer) Read(p []byte) (int, error) {
	if len(s) == 0 {
		return 0, io.EOF
	}
	n := copy(p, s)
	return n, io.EOF
}

type nilMapReaderValuer map[string]string

func (nilMapReaderValuer) Value() (driver.Value, error) {
	return []byte("map-reader-valuer"), nil
}

func (nilMapReaderValuer) Read([]byte) (int, error) {
	return 0, io.EOF
}

type nilFuncReaderValuer func()

func (nilFuncReaderValuer) Value() (driver.Value, error) {
	return []byte("func-reader-valuer"), nil
}

func (nilFuncReaderValuer) Read([]byte) (int, error) {
	return 0, io.EOF
}

type nilChanReaderValuer chan struct{}

func (nilChanReaderValuer) Value() (driver.Value, error) {
	return []byte("chan-reader-valuer"), nil
}

func (nilChanReaderValuer) Read([]byte) (int, error) {
	return 0, io.EOF
}

func TestBlobExecNilReaderValuerKinds(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	runWithBlobStreamModes(ctx, t, func(ctx context.Context, t testing.TB, config *fbx.ConnConfig) {
		config.DefaultQueryExecMode = fbx.QueryExecModeCacheStatement
	}, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		table := createBlobTestTable(ctx, t, conn)
		for i, arg := range []any{
			nilSliceReaderValuer(nil),
			nilMapReaderValuer(nil),
			nilFuncReaderValuer(nil),
			nilChanReaderValuer(nil),
		} {
			_, err := conn.Exec(ctx, fmt.Sprintf("insert into %s (id, bin_blob) values (?, ?)", table), i+1, arg)
			require.NoError(t, err)
			var got []byte
			require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf("select bin_blob from %s where id = ?", table), i+1).Scan(&got))
			require.Nil(t, got, "%T", arg)
		}
	})
}

func runWithBlobStreamModes(
	ctx context.Context,
	t *testing.T,
	configure func(context.Context, testing.TB, *fbx.ConnConfig),
	f func(context.Context, testing.TB, *fbx.Conn),
) {
	t.Helper()

	for _, useStreamBlobs := range []bool{true, false} {
		t.Run(fmt.Sprintf("use_stream_blobs=%t", useStreamBlobs), func(t *testing.T) {
			ctr := defaultConnTestRunner
			baseCreateConfig := ctr.CreateConfig
			ctr.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
				config := baseCreateConfig(ctx, t)
				config.UseStreamBlobs = useStreamBlobs
				if configure != nil {
					configure(ctx, t, config)
				}
				return config
			}
			ctr.RunTest(ctx, t, f)
		})
	}
}

func createBlobTestTable(ctx context.Context, t testing.TB, conn *fbx.Conn) string {
	t.Helper()
	table := fmt.Sprintf("FBXB_%x", time.Now().UnixNano())
	_, err := conn.Exec(ctx, fmt.Sprintf(
		"create table %s (id integer not null primary key, bin_blob blob sub_type 0, txt_blob blob sub_type 1)",
		table,
	))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = conn.Exec(ctx, "drop table "+table)
	})
	return table
}

func makeBlobData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte((i % 251) + 1)
	}
	return data
}
