package fbproto

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type failOnRead struct {
	t *testing.T
}

func (r failOnRead) Read([]byte) (int, error) {
	r.t.Fatal("compression initialization must not read from the transport")
	return 0, io.ErrUnexpectedEOF
}

type xorStreamCodec struct {
	key []byte
	pos int
}

func (c *xorStreamCodec) Encrypt(dst, src []byte) {
	c.apply(dst, src)
}

func (c *xorStreamCodec) Decrypt(dst, src []byte) {
	c.apply(dst, src)
}

func (c *xorStreamCodec) apply(dst, src []byte) {
	for i := range src {
		dst[i] = src[i] ^ c.key[(c.pos+i)%len(c.key)]
	}
	c.pos += len(src)
}

func TestEnableCompressionDoesNotReadTransport(t *testing.T) {
	client := NewClient(failOnRead{t: t}, io.Discard, nil)

	require.NoError(t, client.EnableCompression())
	require.True(t, client.CompressionEnabled())
}

func TestCompressionRoundTripAcrossFlushes(t *testing.T) {
	var wire bytes.Buffer
	writer := NewClient(nil, &wire, nil)
	require.NoError(t, writer.EnableCompression())

	first := bytes.Repeat([]byte("first payload "), 128)
	second := bytes.Repeat([]byte("second payload "), 256)
	_, err := writer.Write(first)
	require.NoError(t, err)
	require.NoError(t, writer.Flush())
	_, err = writer.Write(second)
	require.NoError(t, err)
	require.NoError(t, writer.Flush())

	reader := NewClient(bytes.NewReader(wire.Bytes()), io.Discard, nil)
	require.NoError(t, reader.EnableCompression())
	got := make([]byte, len(first)+len(second))
	_, err = io.ReadFull(reader, got)
	require.NoError(t, err)
	require.Equal(t, append(first, second...), got)
}

func TestCompressionWrapsEncryption(t *testing.T) {
	payload := bytes.Repeat([]byte("compress before encrypt "), 256)
	key := []byte("test-key")

	var wire bytes.Buffer
	writer := NewClient(nil, &wire, nil)
	require.NoError(t, writer.EnableCompression())
	writer.SetCodec("test", &xorStreamCodec{key: key})
	_, err := writer.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Flush())

	decrypted := append([]byte(nil), wire.Bytes()...)
	(&xorStreamCodec{key: key}).Decrypt(decrypted, decrypted)
	zr, err := zlib.NewReader(bytes.NewReader(decrypted))
	require.NoError(t, err)
	plain := make([]byte, len(payload))
	_, err = io.ReadFull(zr, plain)
	require.NoError(t, err)
	require.NoError(t, zr.Close())
	require.Equal(t, payload, plain)

	reader := NewClient(bytes.NewReader(wire.Bytes()), io.Discard, nil)
	require.NoError(t, reader.EnableCompression())
	reader.SetCodec("test", &xorStreamCodec{key: key})
	got := make([]byte, len(payload))
	_, err = io.ReadFull(reader, got)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestCompressionPreservesStreamAcrossEncryptionAndRebind(t *testing.T) {
	beforeCrypt := bytes.Repeat([]byte("op_crypt "), 32)
	afterCrypt := bytes.Repeat([]byte("encrypted response "), 64)
	key := []byte("boundary-key")

	var plainWire bytes.Buffer
	writer := NewClient(nil, &plainWire, nil)
	require.NoError(t, writer.EnableCompression())
	_, err := writer.Write(beforeCrypt)
	require.NoError(t, err)
	require.NoError(t, writer.flushCompression())

	var encryptedWire bytes.Buffer
	writer.Rebind(nil, &encryptedWire)
	writer.SetCodec("test", &xorStreamCodec{key: key})
	_, err = writer.Write(afterCrypt)
	require.NoError(t, err)
	require.NoError(t, writer.Flush())

	reader := NewClient(bytes.NewReader(plainWire.Bytes()), io.Discard, nil)
	require.NoError(t, reader.EnableCompression())
	gotBefore := make([]byte, len(beforeCrypt))
	_, err = io.ReadFull(reader, gotBefore)
	require.NoError(t, err)
	require.Equal(t, beforeCrypt, gotBefore)

	reader.Rebind(bytes.NewReader(encryptedWire.Bytes()), io.Discard)
	reader.SetCodec("test", &xorStreamCodec{key: key})
	gotAfter := make([]byte, len(afterCrypt))
	_, err = io.ReadFull(reader, gotAfter)
	require.NoError(t, err)
	require.Equal(t, afterCrypt, gotAfter)
}

func TestCompressionDoesNotEmitEmptyFlushAcrossEncryptionBoundary(t *testing.T) {
	var wire bytes.Buffer
	client := NewClient(nil, &wire, nil)
	require.NoError(t, client.EnableCompression())
	_, err := client.Write([]byte("op_crypt"))
	require.NoError(t, err)
	require.NoError(t, client.Flush())
	plainLength := wire.Len()

	client.SetCodec("test", &xorStreamCodec{key: []byte("key")})
	require.NoError(t, client.Flush())
	require.Equal(t, plainLength, wire.Len())
}

func TestCloseCompressionFinalizesStreamAndFlushesWriter(t *testing.T) {
	payload := bytes.Repeat([]byte("final payload "), 64)
	var wire bytes.Buffer
	bufferedWire := bufio.NewWriter(&wire)
	client := NewClient(nil, bufferedWire, nil)
	require.NoError(t, client.EnableCompression())

	_, err := client.Write(payload)
	require.NoError(t, err)
	require.NoError(t, client.CloseCompression())
	require.False(t, client.CompressionEnabled())
	require.NoError(t, client.CloseCompression())

	zr, err := zlib.NewReader(bytes.NewReader(wire.Bytes()))
	require.NoError(t, err)
	decoded, err := io.ReadAll(zr)
	require.NoError(t, err)
	require.NoError(t, zr.Close())
	require.Equal(t, payload, decoded)
}
