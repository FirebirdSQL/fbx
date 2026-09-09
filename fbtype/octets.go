package fbtype

import (
	"database/sql/driver"
	"fmt"
)

// BytesScanner scans decoded text or raw Firebird OCTETS data as bytes.
type BytesScanner interface {
	// ScanBytes receives a byte slice of driver memory that is only valid until
	// the next database method call.
	ScanBytes(v []byte) error
}

// BytesValuer provides raw bytes for Firebird OCTETS encoding.
type BytesValuer interface {
	// BytesValue returns raw bytes. The caller must not change the returned
	// slice.
	BytesValue() ([]byte, error)
}

// DriverBytes implements [BytesScanner] by holding a reference to driver-owned
// memory. It is only valid until the next rows operation, and must not be used
// with QueryRow because Row.Scan closes rows before returning.
type DriverBytes []byte

// ScanBytes implements [BytesScanner] without copying v.
func (b *DriverBytes) ScanBytes(v []byte) error {
	*b = v
	return nil
}

// PreallocBytes implements [BytesScanner] by copying scanned bytes into
// preallocated memory. If it is too small a new slice will be allocated.
type PreallocBytes []byte

// ScanBytes implements [BytesScanner] by copying v into the receiver's
// available capacity when possible.
func (b *PreallocBytes) ScanBytes(v []byte) error {
	if v == nil {
		*b = nil
		return nil
	}

	if *b != nil && len(v) <= cap(*b) {
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

// OctetsCodec encodes and decodes Firebird CHAR and VARCHAR values declared
// with CHARACTER SET OCTETS.
type OctetsCodec struct{}

// PlanEncode returns an encode plan for supported Firebird CHAR and VARCHAR
// CHARACTER SET OCTETS input values.
func (OctetsCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case []byte:
		return encodePlanOctetsCodecBytes{}
	case BytesValuer:
		return encodePlanOctetsCodecBytesValuer{}
	case string:
		return encodePlanOctetsCodecString{}
	}

	return nil
}

type encodePlanOctetsCodecBytes struct{}

func (encodePlanOctetsCodecBytes) Encode(value any, buf []byte) (newBuf []byte, err error) {
	b := value.([]byte)
	if b == nil {
		return nil, nil
	}

	return appendNonNullBytes(buf, b), nil
}

type encodePlanOctetsCodecBytesValuer struct{}

func (encodePlanOctetsCodecBytesValuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	b, err := value.(BytesValuer).BytesValue()
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}

	return appendNonNullBytes(buf, b), nil
}

type encodePlanOctetsCodecString struct{}

func (encodePlanOctetsCodecString) Encode(value any, buf []byte) ([]byte, error) {
	s := value.(string)
	return appendNonNullBytes(buf, []byte(s)), nil
}

// PlanScan returns a scan plan for supported Firebird CHAR and VARCHAR
// CHARACTER SET OCTETS target values.
func (OctetsCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case *string:
		return scanPlanBinaryOctetsToString{}
	case *[]byte:
		return scanPlanBinaryOctetsToBytes{}
	case BytesScanner:
		return scanPlanBinaryOctetsToBytesScanner{}
	}

	return nil
}

type scanPlanBinaryOctetsToString struct{}

func (scanPlanBinaryOctetsToString) Scan(src []byte, dst any) error {
	p, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}

	*p = string(src)
	return nil
}

type scanPlanBinaryOctetsToBytes struct{}

func (scanPlanBinaryOctetsToBytes) Scan(src []byte, dst any) error {
	dstBuf, ok := dst.(*[]byte)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		*dstBuf = nil
		return nil
	}

	*dstBuf = make([]byte, len(src))
	copy(*dstBuf, src)
	return nil
}

type scanPlanBinaryOctetsToBytesScanner struct{}

func (scanPlanBinaryOctetsToBytesScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(BytesScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	return scanner.ScanBytes(src)
}

// DecodeDatabaseSQLValue decodes src into a database/sql driver value.
func (c OctetsCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	return c.DecodeValue(m, typeMetadata, src)
}

// DecodeValue decodes src into a copied byte slice.
func (c OctetsCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
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
