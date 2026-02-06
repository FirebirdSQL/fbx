package stdlib

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
)

// Blob is a lazy BLOB reader returned from database/sql scans when using stdlib.
// Read from it directly or call Bytes/Text to materialize the contents.
type Blob struct {
	rc      io.ReadCloser
	SubType int32
	Charset string

	encoding *fbproto.Encoding
}

func buildBlobValue(reader io.ReadCloser, fd *fbconn.FieldDescription, conn *Conn) (Blob, error) {
	blob := Blob{
		rc:      reader,
		SubType: fd.Subtype,
	}
	if fd.Subtype == fbproto.BLOB_SUB_TYPE_TEXT {
		encoding, charset, err := blobTextEncoding(fd, conn)
		if err != nil {
			return Blob{}, err
		}
		blob.encoding = encoding
		blob.Charset = charset
	}
	return blob, nil
}

func blobTextEncoding(fd *fbconn.FieldDescription, conn *Conn) (*fbproto.Encoding, string, error) {
	charsetID := fbtype.CharacterSetForType(fd.Type, fd.Subtype, fd.Scale)
	charset, err := fbproto.CharacterSetIdIntoName(charsetID)
	if err != nil {
		return nil, "", fmt.Errorf("blob text charset: %w", err)
	}
	encoding, err := fbproto.NewEncoding(charset)
	if err != nil {
		return nil, "", err
	}
	return encoding, charset, nil
}

// Read implements io.Reader.
func (b Blob) Read(p []byte) (int, error) {
	if b.rc == nil {
		return 0, io.EOF
	}
	return b.rc.Read(p)
}

// Close implements io.Closer.
func (b Blob) Close() error {
	if b.rc == nil {
		return nil
	}
	return b.rc.Close()
}

// Bytes reads the entire blob into memory and closes the underlying reader.
func (b Blob) Bytes() ([]byte, error) {
	if b.rc == nil {
		return nil, nil
	}
	data, err := io.ReadAll(b.rc)
	closeErr := b.rc.Close()
	if err == nil {
		err = closeErr
	}
	return data, err
}

// Text reads the entire blob, decodes it using the configured charset (if any),
// and closes the underlying reader.
func (b Blob) Text() (string, error) {
	data, err := b.Bytes()
	if err != nil {
		return "", err
	}
	if b.encoding == nil {
		return string(data), nil
	}
	decoded, err := b.encoding.Decoder.Bytes(data)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// Seek forwards seek requests when the underlying reader supports io.Seeker.
func (b Blob) Seek(offset int64, whence int) (int64, error) {
	if b.rc == nil {
		return 0, errors.New("blob is nil")
	}
	seeker, ok := b.rc.(io.Seeker)
	if !ok {
		return 0, errors.New("blob reader does not support seek")
	}
	return seeker.Seek(offset, whence)
}

// Scan implements sql.Scanner for lazy BLOB values.
func (b *Blob) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*b = Blob{}
		return nil
	case Blob:
		*b = v
		return nil
	case *Blob:
		if v == nil {
			*b = Blob{}
		} else {
			*b = *v
		}
		return nil
	case io.ReadCloser:
		*b = Blob{rc: v}
		return nil
	case []byte:
		*b = Blob{rc: io.NopCloser(bytes.NewReader(v))}
		return nil
	case string:
		*b = Blob{rc: io.NopCloser(strings.NewReader(v))}
		return nil
	default:
		return fmt.Errorf("cannot scan %T into Blob", src)
	}
}
