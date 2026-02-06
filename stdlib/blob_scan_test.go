package stdlib_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/stdlib"
)

type trackingReadCloser struct {
	reader *bytes.Reader
	closed bool
}

func newTrackingReadCloser(s string) *trackingReadCloser {
	return &trackingReadCloser{reader: bytes.NewReader([]byte(s))}
}

func (t *trackingReadCloser) Read(p []byte) (int, error) {
	return t.reader.Read(p)
}

func (t *trackingReadCloser) Close() error {
	t.closed = true
	return nil
}

func TestBlobScanNil(t *testing.T) {
	var b stdlib.Blob
	require.NoError(t, b.Scan(nil))

	data, err := io.ReadAll(&b)
	require.NoError(t, err)
	require.Len(t, data, 0)
	require.NoError(t, b.Close())
}

func TestBlobScanBlob(t *testing.T) {
	rc := newTrackingReadCloser("blob")
	var src stdlib.Blob
	require.NoError(t, src.Scan(rc))
	src.SubType = fbproto.BLOB_SUB_TYPE_TEXT
	src.Charset = "UTF8"

	var dst stdlib.Blob
	require.NoError(t, dst.Scan(src))
	require.Equal(t, src.SubType, dst.SubType)
	require.Equal(t, src.Charset, dst.Charset)

	data, err := io.ReadAll(&dst)
	require.NoError(t, err)
	require.Equal(t, "blob", string(data))
	require.False(t, rc.closed)
	require.NoError(t, dst.Close())
	require.True(t, rc.closed)
}

func TestBlobScanBlobPtr(t *testing.T) {
	rc := newTrackingReadCloser("ptr")
	var src stdlib.Blob
	require.NoError(t, src.Scan(rc))
	src.SubType = fbproto.BLOB_SUB_TYPE_TEXT
	src.Charset = "ASCII"

	var dst stdlib.Blob
	require.NoError(t, dst.Scan(&src))
	require.Equal(t, src.SubType, dst.SubType)
	require.Equal(t, src.Charset, dst.Charset)

	data, err := io.ReadAll(&dst)
	require.NoError(t, err)
	require.Equal(t, "ptr", string(data))
	require.False(t, rc.closed)
	require.NoError(t, dst.Close())
	require.True(t, rc.closed)
}

func TestBlobScanNilBlobPtr(t *testing.T) {
	var dst stdlib.Blob
	var src *stdlib.Blob
	require.NoError(t, dst.Scan(src))

	data, err := io.ReadAll(&dst)
	require.NoError(t, err)
	require.Len(t, data, 0)
}

func TestBlobScanReadCloser(t *testing.T) {
	rc := newTrackingReadCloser("read")
	var b stdlib.Blob
	require.NoError(t, b.Scan(rc))

	data, err := io.ReadAll(&b)
	require.NoError(t, err)
	require.Equal(t, "read", string(data))
	require.False(t, rc.closed)
	require.NoError(t, b.Close())
	require.True(t, rc.closed)
}

func TestBlobScanBytes(t *testing.T) {
	var b stdlib.Blob
	require.NoError(t, b.Scan([]byte("bytes")))

	data, err := b.Bytes()
	require.NoError(t, err)
	require.Equal(t, "bytes", string(data))
}

func TestBlobScanString(t *testing.T) {
	var b stdlib.Blob
	require.NoError(t, b.Scan("string"))

	text, err := b.Text()
	require.NoError(t, err)
	require.Equal(t, "string", text)
}

func TestBlobScanUnsupportedType(t *testing.T) {
	var b stdlib.Blob
	err := b.Scan(123)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot scan int")
}
