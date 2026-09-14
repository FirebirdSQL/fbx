package fbtype

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// UUIDScanner scans a nullable UUID value.
type UUIDScanner interface {
	// ScanUUID scans v into the receiver.
	ScanUUID(v UUID) error
}

// UUIDValuer provides a nullable UUID value for encoding.
type UUIDValuer interface {
	// UUIDValue returns the UUID value to encode.
	UUIDValue() (UUID, error)
}

// UUID represents a nullable UUID encoded as 16 binary bytes. Firebird has no
// native UUID type; UUID values are commonly stored as CHAR(16) CHARACTER SET
// OCTETS. Valid is false for SQL NULL.
type UUID struct {
	Bytes [16]byte
	Valid bool
}

// ScanUUID implements [UUIDScanner].
func (b *UUID) ScanUUID(v UUID) error {
	*b = v
	return nil
}

// UUIDValue implements [UUIDValuer].
func (b UUID) UUIDValue() (UUID, error) {
	return b, nil
}

// parseUUID parses a canonical or compact UUID string.
func parseUUID(src string) (dst [16]byte, err error) {
	var encoded [32]byte

	switch len(src) {
	case 36:
		if src[8] != '-' || src[13] != '-' || src[18] != '-' || src[23] != '-' {
			return dst, fmt.Errorf("cannot parse UUID %q: invalid format", src)
		}
		copy(encoded[0:8], src[0:8])
		copy(encoded[8:12], src[9:13])
		copy(encoded[12:16], src[14:18])
		copy(encoded[16:20], src[19:23])
		copy(encoded[20:32], src[24:36])
	case 32:
		copy(encoded[:], src)
	default:
		return dst, fmt.Errorf("cannot parse UUID %q: invalid length %d", src, len(src))
	}

	if _, err := hex.Decode(dst[:], encoded[:]); err != nil {
		return dst, fmt.Errorf("cannot parse UUID %q: %w", src, err)
	}

	return dst, nil
}

// encodeUUID formats UUID bytes in canonical form.
func encodeUUID(src [16]byte) string {
	var buf [36]byte

	hex.Encode(buf[0:8], src[:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], src[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], src[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], src[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], src[10:])

	return string(buf[:])
}

// Scan implements [database/sql.Scanner]. It accepts a canonical or compact
// UUID string, a 16-byte slice, or nil.
func (dst *UUID) Scan(src any) error {
	*dst = UUID{}

	switch src := src.(type) {
	case nil:
		return nil
	case string:
		buf, err := parseUUID(src)
		if err != nil {
			return err
		}
		*dst = UUID{Bytes: buf, Valid: true}
		return nil
	case []byte:
		if len(src) != 16 {
			return fmt.Errorf("cannot parse UUID bytes: invalid length %d", len(src))
		}
		var buf [16]byte
		copy(buf[:], src)
		*dst = UUID{Bytes: buf, Valid: true}
		return nil
	}

	return fmt.Errorf("cannot scan %T into UUID", src)
}

// Value implements [driver.Valuer]. It returns nil for SQL NULL and a canonical
// UUID string otherwise.
func (src UUID) Value() (driver.Value, error) {
	if !src.Valid {
		return nil, nil
	}

	return encodeUUID(src.Bytes), nil
}

// String returns the canonical UUID representation, or an empty string for
// SQL NULL.
func (src UUID) String() string {
	if !src.Valid {
		return ""
	}

	return encodeUUID(src.Bytes)
}

// MarshalJSON encodes src as a canonical UUID string or null.
func (src UUID) MarshalJSON() ([]byte, error) {
	if !src.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(encodeUUID(src.Bytes))
}

// UnmarshalJSON decodes a canonical or compact UUID string, or null, into dst.
func (dst *UUID) UnmarshalJSON(src []byte) error {
	*dst = UUID{}

	var uuidText *string
	if err := json.Unmarshal(src, &uuidText); err != nil {
		return err
	}
	if uuidText == nil {
		return nil
	}

	buf, err := parseUUID(*uuidText)
	if err != nil {
		return err
	}
	*dst = UUID{Bytes: buf, Valid: true}
	return nil
}

// UUIDCodec encodes and decodes UUID values as 16 binary bytes. Firebird uses
// this representation with CHAR(16) CHARACTER SET OCTETS.
type UUIDCodec struct{}

// PlanEncode returns an encode plan for values implementing [UUIDValuer].
func (UUIDCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	if _, ok := value.(UUIDValuer); !ok {
		return nil
	}

	return encodePlanUUIDCodecBinaryUUIDValuer{}
}

type encodePlanUUIDCodecBinaryUUIDValuer struct{}

func (encodePlanUUIDCodecBinaryUUIDValuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	uuid, err := value.(UUIDValuer).UUIDValue()
	if err != nil {
		return nil, err
	}

	if !uuid.Valid {
		return nil, nil
	}

	return append(buf, uuid.Bytes[:]...), nil
}

// PlanScan returns a scan plan for supported UUID target values.
func (UUIDCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case UUIDScanner:
		return scanPlanBinaryUUIDToUUIDScanner{}
	case TextScanner:
		return scanPlanBinaryUUIDToTextScanner{}
	}

	return nil
}

type scanPlanBinaryUUIDToUUIDScanner struct{}

func (scanPlanBinaryUUIDToUUIDScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(UUIDScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanUUID(UUID{})
	}

	if len(src) != 16 {
		return fmt.Errorf("invalid length for UUID: %v", len(src))
	}

	uuid := UUID{Valid: true}
	copy(uuid.Bytes[:], src)

	return scanner.ScanUUID(uuid)
}

type scanPlanBinaryUUIDToTextScanner struct{}

func (scanPlanBinaryUUIDToTextScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(TextScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanText(Text{})
	}

	if len(src) != 16 {
		return fmt.Errorf("invalid length for UUID: %v", len(src))
	}

	var buf [16]byte
	copy(buf[:], src)

	return scanner.ScanText(Text{String: encodeUUID(buf), Valid: true})
}

// DecodeDatabaseSQLValue decodes src into a canonical UUID string.
func (c UUIDCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}

	var uuid UUID
	err := codecScan(c, m, typeMetadata, src, &uuid)
	if err != nil {
		return nil, err
	}

	return encodeUUID(uuid.Bytes), nil
}

// DecodeValue decodes src into a 16-byte UUID array.
func (c UUIDCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var uuid UUID
	err := codecScan(c, m, typeMetadata, src, &uuid)
	if err != nil {
		return nil, err
	}
	return uuid.Bytes, nil
}
