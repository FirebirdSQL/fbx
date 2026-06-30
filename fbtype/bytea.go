package fbtype

import (
	"database/sql/driver"
)

type BytesScanner interface {
	// ScanBytes receives a byte slice of driver memory that is only valid until the next database method call.
	ScanBytes(v []byte) error
}

type BytesValuer interface {
	// BytesValue returns a byte slice of the byte data. The caller must not change the returned slice.
	BytesValue() ([]byte, error)
}

// DriverBytes is a byte slice that holds a reference to memory owned by the driver. It is only valid from the time it
// is scanned until Rows.Next or Rows.Close is called. It is never safe to use DriverBytes with QueryRow as Row.Scan
// internally calls Rows.Close before returning.
type DriverBytes []byte

func (b *DriverBytes) ScanBytes(v []byte) error {
	*b = v
	return nil
}

// PreallocBytes is a byte slice of preallocated memory that scanned bytes will be copied to. If it is too small a new
// slice will be allocated.
type PreallocBytes []byte

func (b *PreallocBytes) ScanBytes(v []byte) error {
	if v == nil {
		*b = nil
		return nil
	}

	if len(v) <= len(*b) {
		*b = (*b)[:len(v)]
	} else {
		*b = make(PreallocBytes, len(v))
	}
	copy(*b, v)
	return nil
}

// UndecodedBytes receives a copy of the raw Firebird bytes for any scanned
// value without codec decoding.
type UndecodedBytes []byte

type ByteaCodec struct{}

func (ByteaCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {

	switch value.(type) {
	case []byte:
		return encodePlanBytesCodecBinaryBytes{}
	case BytesValuer:
		return encodePlanBytesCodecBinaryBytesValuer{}
	case string:
		return encodePlanBytesCodecBinaryString{}
	}

	return nil
}

type encodePlanBytesCodecBinaryBytes struct{}

func (encodePlanBytesCodecBinaryBytes) Encode(value any, buf []byte) (newBuf []byte, err error) {
	b := value.([]byte)
	if b == nil {
		return nil, nil
	}

	return append(buf, b...), nil
}

type encodePlanBytesCodecBinaryBytesValuer struct{}

func (encodePlanBytesCodecBinaryBytesValuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	b, err := value.(BytesValuer).BytesValue()
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}

	return append(buf, b...), nil
}

type encodePlanBytesCodecBinaryString struct{}

func (encodePlanBytesCodecBinaryString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)
	return append(buf, s...), nil
}

func (ByteaCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {

	switch target.(type) {
	case *[]byte:
		return scanPlanBinaryBytesToBytes{}
	case BytesScanner:
		return scanPlanBinaryBytesToBytesScanner{}
	}

	return nil
}

type scanPlanBinaryBytesToBytes struct{}

func (scanPlanBinaryBytesToBytes) Scan(src []byte, dst any) error {
	dstBuf := dst.(*[]byte)
	if src == nil {
		*dstBuf = nil
		return nil
	}

	*dstBuf = make([]byte, len(src))
	copy(*dstBuf, src)
	return nil
}

type scanPlanBinaryBytesToBytesScanner struct{}

func (scanPlanBinaryBytesToBytesScanner) Scan(src []byte, dst any) error {
	scanner := (dst).(BytesScanner)
	return scanner.ScanBytes(src)
}

func (c ByteaCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	return c.DecodeValue(m, typeMetadata, src)
}

func (c ByteaCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var buf []byte
	err := codecScan(c, m, typeMetadata, src, &buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
