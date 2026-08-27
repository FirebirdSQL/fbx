package fbtype

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
)

type UUIDScanner interface {
	ScanUUID(v UUID) error
}

type UUIDValuer interface {
	UUIDValue() (UUID, error)
}

type UUID struct {
	Bytes [16]byte
	Valid bool
}

func (b *UUID) ScanUUID(v UUID) error {
	*b = v
	return nil
}

func (b UUID) UUIDValue() (UUID, error) {
	return b, nil
}

// parseUUID converts a string UUID in standard form to a byte array.
func parseUUID(src string) (dst [16]byte, err error) {
	switch len(src) {
	case 36:
		src = src[0:8] + src[9:13] + src[14:18] + src[19:23] + src[24:]
	case 32:
		// dashes already stripped, assume valid
	default:
		// assume invalid.
		return dst, fmt.Errorf("cannot parse UUID %v", src)
	}

	buf, err := hex.DecodeString(src)
	if err != nil {
		return dst, err
	}

	copy(dst[:], buf)
	return dst, err
}

// encodeUUID converts a uuid byte array to UUID standard string form.
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

// Scan implements the database/sql Scanner interface.
func (dst *UUID) Scan(src any) error {
	if src == nil {
		*dst = UUID{}
		return nil
	}

	switch src := src.(type) {
	case string:
		buf, err := parseUUID(src)
		if err != nil {
			return err
		}
		*dst = UUID{Bytes: buf, Valid: true}
		return nil
	case []byte:
		if len(src) != 16 {
			return fmt.Errorf("cannot parse UUID %b", src)
		}
		*dst = UUID{Bytes: [16]byte(src), Valid: true}
		return nil
	}

	return fmt.Errorf("cannot scan %T", src)
}

// Value implements the database/sql/driver Valuer interface.
func (src UUID) Value() (driver.Value, error) {
	if !src.Valid {
		return nil, nil
	}

	return encodeUUID(src.Bytes), nil
}

func (src UUID) String() string {
	if !src.Valid {
		return ""
	}

	return encodeUUID(src.Bytes)
}

func (src UUID) MarshalJSON() ([]byte, error) {
	if !src.Valid {
		return []byte("null"), nil
	}

	var buff bytes.Buffer
	buff.WriteByte('"')
	buff.WriteString(encodeUUID(src.Bytes))
	buff.WriteByte('"')
	return buff.Bytes(), nil
}

func (dst *UUID) UnmarshalJSON(src []byte) error {
	if bytes.Equal(src, []byte("null")) {
		*dst = UUID{}
		return nil
	}
	if len(src) != 38 {
		return fmt.Errorf("invalid length for UUID: %v", len(src))
	}
	buf, err := parseUUID(string(src[1 : len(src)-1]))
	if err != nil {
		return err
	}
	*dst = UUID{Bytes: buf, Valid: true}
	return nil
}

type UUIDCodec struct{}

func (UUIDCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	return encodePlanUUIDCodecTextUUIDValuer{}
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

type encodePlanUUIDCodecBinaryString struct{}

func (encodePlanUUIDCodecBinaryString) Encode(value any, buf []byte) ([]byte, error) {
	str := value.(string)
	uuid, err := parseUUID(str)
	if err != nil {
		return nil, err
	}
	return append(buf, uuid[:]...), nil
}

type encodePlanUUIDCodecTextUUIDValuer struct{}

func (encodePlanUUIDCodecTextUUIDValuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	uuid, err := value.(UUIDValuer).UUIDValue()
	if err != nil {
		return nil, err
	}

	if !uuid.Valid {
		return nil, nil
	}

	return append(buf, encodeUUID(uuid.Bytes)...), nil
}

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
	scanner := (dst).(UUIDScanner)

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
	scanner := (dst).(TextScanner)

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
