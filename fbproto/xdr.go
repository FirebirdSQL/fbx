package fbproto

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-appendix-xdr

import (
	"encoding/binary"
	"io"
)

type XDRReader struct {
	reader io.Reader
}

type XDRWriter struct {
	writer io.Writer
}

func NewXDRReader(r io.Reader) *XDRReader {
	return &XDRReader{reader: r}
}

func NewXDRWriter(w io.Writer) *XDRWriter {
	return &XDRWriter{writer: w}
}

func (x *XDRReader) SkipPadding(len int64) error {
	return x.SkipBytes((4 - len) & 3)
}

func (x *XDRReader) SkipBytes(n int64) error {
	switch r := x.reader.(type) {
	case io.Seeker:
		_, err := r.Seek(n, io.SeekCurrent)
		return err
	default:
		_, err := io.CopyN(io.Discard, r, n)
		return err
	}
}

func (x *XDRReader) ReadBufferLen(len int64) ([]byte, error) {
	if len == 0 {
		return []byte{}, nil
	}
	buf, err := x.ReadBytes(len)
	if err != nil {
		return buf, err
	}
	if err = x.SkipPadding(len); err != nil {
		return buf, err
	}
	return buf, nil
}

func (x *XDRReader) ReadBuffer() ([]byte, error) {
	n, err := x.ReadInt32()
	if err != nil {
		return nil, err
	}
	buf, err := x.ReadBytes(int64(n))
	if err != nil {
		return buf, err
	}
	if err = x.SkipPadding(int64(n)); err != nil {
		return buf, err
	}
	return buf, nil
}

func (x *XDRReader) ReadBytes(len int64) ([]byte, error) {
	buf := make([]byte, len)
	n, err := x.reader.Read(buf)
	if err != nil {
		return buf, err
	}
	if int64(n) < len {
		return buf, io.EOF
	}
	return buf, nil
}

func (x *XDRReader) ReadInt32() (int, error) {
	buf, err := x.ReadBytes(4)
	if err != nil {
		return 0, err
	}
	return int(int32(binary.BigEndian.Uint32(buf))), nil
}

func (x *XDRReader) ReadUInt32() (uint, error) {
	buf, err := x.ReadBytes(4)
	if err != nil {
		return 0, err
	}
	return uint(binary.BigEndian.Uint32(buf)), nil
}

func (x *XDRReader) ReadInt64() (int64, error) {
	buf, err := x.ReadBytes(8)
	if err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(buf)), nil
}

func (x *XDRReader) ReadString(encoding *Encoding) (string, error) {
	buf, err := x.ReadBuffer()
	if err != nil {
		return "", err
	}
	bytes, err := encoding.Decoder.Bytes(buf)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (x *XDRWriter) WriteBytes(buf []byte) error {
	n, err := x.writer.Write(buf)
	if err != nil {
		return err
	}
	if n < len(buf) {
		panic("not all buffer written")
	}
	return nil
}

func (x *XDRWriter) WriteInt32(i int) error {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(i))
	return x.WriteBytes(buf)
}

func (x *XDRWriter) WriteUInt32(i uint) error {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(i))
	return x.WriteBytes(buf)
}

func (x *XDRWriter) WriteInt64(i int64) error {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return x.WriteBytes(buf)
}

func (x *XDRWriter) WriteBuffer(buf []byte) error {
	if buf == nil || len(buf) == 0 {
		return x.WriteInt32(0)
	}
	n := len(buf)

	if err := x.WriteInt32(n); err != nil {
		return err
	}

	if err := x.WriteBytes(buf); err != nil {
		return err
	}

	return x.WritePadding((4 - n) & 3)
}

func (x *XDRWriter) WritePadding(len int) error {
	return x.WriteBytes(make([]byte, len))
}

func (x *XDRWriter) WriteString(s string, encoding *Encoding) error {

	if res, err := encoding.Encoder.String(s); err != nil {
		return err
	} else {
		return x.WriteBuffer([]byte(res))
	}
}
