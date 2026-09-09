package fbtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"rdb.red-soft.ru/fbx/fbproto"
)

type TextScanner interface {
	ScanText(v Text) error
}

type TextValuer interface {
	TextValue() (Text, error)
}

type Text struct {
	String string
	Valid  bool
}

func (t *Text) ScanText(v Text) error {
	*t = v
	return nil
}

func (t Text) TextValue() (Text, error) {
	return t, nil
}

// Scan implements the database/sql Scanner interface.
func (dst *Text) Scan(src any) error {
	*dst = Text{}

	if src == nil {
		return nil
	}

	switch src := src.(type) {
	case string:
		*dst = Text{String: src, Valid: true}
		return nil
	case []byte:
		*dst = Text{String: string(src), Valid: true}
		return nil
	}

	return fmt.Errorf("cannot scan %T", src)
}

// Value implements the database/sql/driver Valuer interface.
func (src Text) Value() (driver.Value, error) {
	if !src.Valid {
		return nil, nil
	}
	return src.String, nil
}

func (src Text) MarshalJSON() ([]byte, error) {
	if !src.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(src.String)
}

func (dst *Text) UnmarshalJSON(b []byte) error {
	*dst = Text{}

	var s *string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	if s == nil {
		*dst = Text{}
	} else {
		*dst = Text{String: *s, Valid: true}
	}

	return nil
}

type TextCodec struct{}

func (TextCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch value.(type) {
	case string, []byte, TextValuer:
	case fmt.Stringer:
		if _, ok := value.(driver.Valuer); ok {
			return nil
		}
	default:
		return nil
	}

	encoding, err := textEncoding(typeMetadata)
	if err != nil {
		return encodePlanTextCodecError{err: err}
	}

	switch value.(type) {
	case string:
		return encodePlanTextCodecString{encoding: encoding, length: typeMetadata.Length}
	case []byte:
		return encodePlanTextCodecByteSlice{encoding: encoding, length: typeMetadata.Length}
	case TextValuer:
		return encodePlanTextCodecTextValuer{encoding: encoding, length: typeMetadata.Length}
	case fmt.Stringer:
		return encodePlanTextCodecStringer{encoding: encoding, length: typeMetadata.Length}
	}

	return nil
}

type encodePlanTextCodecError struct {
	err error
}

func (e encodePlanTextCodecError) Encode(value any, buf []byte) ([]byte, error) {
	return nil, e.err
}

type encodePlanTextCodecString struct {
	encoding *fbproto.Encoding
	length   uint32
}

func (e encodePlanTextCodecString) Encode(value any, buf []byte) (newBuf []byte, err error) {
	s := value.(string)
	res, err := encodeFBString(s, e.length, e.encoding)
	if err != nil {
		return nil, err
	}
	return appendNonNullBytes(buf, res), nil
}

type encodePlanTextCodecByteSlice struct {
	encoding *fbproto.Encoding
	length   uint32
}

func (e encodePlanTextCodecByteSlice) Encode(value any, buf []byte) (newBuf []byte, err error) {
	s := string(value.([]byte))
	res, err := encodeFBString(s, e.length, e.encoding)
	if err != nil {
		return nil, err
	}
	return appendNonNullBytes(buf, res), nil
}

type encodePlanTextCodecStringer struct {
	encoding *fbproto.Encoding
	length   uint32
}

func (e encodePlanTextCodecStringer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	s := value.(fmt.Stringer)
	res, err := encodeFBString(s.String(), e.length, e.encoding)
	if err != nil {
		return nil, err
	}
	return appendNonNullBytes(buf, res), nil
}

type encodePlanTextCodecTextValuer struct {
	encoding *fbproto.Encoding
	length   uint32
}

func (e encodePlanTextCodecTextValuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	text, err := value.(TextValuer).TextValue()
	if err != nil {
		return nil, err
	}

	if !text.Valid {
		return nil, nil
	}

	res, err := encodeFBString(text.String, e.length, e.encoding)
	if err != nil {
		return nil, err
	}
	return appendNonNullBytes(buf, res), nil
}

func (TextCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	switch target.(type) {
	case *string, *[]byte, BytesScanner, TextScanner:
	default:
		return nil
	}

	encoding, err := textEncoding(typeMetadata)
	if err != nil {
		return scanPlanTextCodecError{err: err}
	}

	switch target.(type) {
	case *string:
		return scanPlanTextAnyToString{encoding: encoding, length: typeMetadata.Length, typeId: typeMetadata.TypeID}
	case *[]byte:
		return scanPlanAnyToNewByteSlice{encoding: encoding, length: typeMetadata.Length, typeId: typeMetadata.TypeID}
	case BytesScanner:
		return scanPlanAnyToByteScanner{encoding: encoding, length: typeMetadata.Length, typeId: typeMetadata.TypeID}
	case TextScanner:
		return scanPlanTextAnyToTextScanner{encoding: encoding, length: typeMetadata.Length, typeId: typeMetadata.TypeID}
	}

	return nil
}

type scanPlanTextCodecError struct {
	err error
}

func (s scanPlanTextCodecError) Scan(src []byte, dst any) error {
	return s.err
}

func (c TextCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	return c.DecodeValue(m, typeMetadata, src)
}

func (c TextCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	encoding, err := textEncoding(typeMetadata)
	if err != nil {
		return nil, err
	}

	var res string
	if err = decodeFBString(src, typeMetadata.TypeID == fbproto.SQL_TYPE_VARYING, typeMetadata.Length, encoding, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func textEncoding(typeMetadata TypeMetadata) (*fbproto.Encoding, error) {
	charsetID := CharacterSetForType(typeMetadata.TypeID, typeMetadata.SubType, typeMetadata.Scale)
	name, err := fbproto.CharacterSetIdIntoName(charsetID)
	if err != nil {
		return nil, err
	}

	return fbproto.NewEncoding(name)
}

type scanPlanTextAnyToString struct {
	typeId   uint32
	length   uint32
	encoding *fbproto.Encoding
}

func (s scanPlanTextAnyToString) Scan(src []byte, dst any) error {
	p, ok := dst.(*string)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}
	return decodeFBString(src, s.typeId == fbproto.SQL_TYPE_VARYING, s.length, s.encoding, p)
}

type scanPlanAnyToNewByteSlice struct {
	typeId   uint32
	length   uint32
	encoding *fbproto.Encoding
}

func (s scanPlanAnyToNewByteSlice) Scan(src []byte, dst any) error {
	p, ok := dst.(*[]byte)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		*p = nil
	} else {
		var res string
		if err := decodeFBString(src, s.typeId == fbproto.SQL_TYPE_VARYING, s.length, s.encoding, &res); err != nil {
			return err
		}
		*p = []byte(res)
	}

	return nil
}

type scanPlanAnyToByteScanner struct {
	typeId   uint32
	length   uint32
	encoding *fbproto.Encoding
}

func (s scanPlanAnyToByteScanner) Scan(src []byte, dst any) error {
	p, ok := dst.(BytesScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return p.ScanBytes(nil)
	}

	var res string
	if err := decodeFBString(src, s.typeId == fbproto.SQL_TYPE_VARYING, s.length, s.encoding, &res); err != nil {
		return err
	}
	return p.ScanBytes([]byte(res))
}

type scanPlanTextAnyToTextScanner struct {
	typeId   uint32
	length   uint32
	encoding *fbproto.Encoding
}

func (s scanPlanTextAnyToTextScanner) Scan(src []byte, dst any) error {
	scanner, ok := dst.(TextScanner)
	if !ok {
		return ErrScanTargetTypeChanged
	}

	if src == nil {
		return scanner.ScanText(Text{})
	}

	var res string
	if err := decodeFBString(src, s.typeId == fbproto.SQL_TYPE_VARYING, s.length, s.encoding, &res); err != nil {
		return err
	}

	return scanner.ScanText(Text{String: res, Valid: true})
}

func decodeFBString(src []byte, varying bool, length uint32, enc *fbproto.Encoding, dst *string) error {
	if enc == nil || enc.Decoder == nil {
		return fmt.Errorf("decodeFBString: encoding is not initialized")
	}

	if varying {
		buf, err := enc.Decoder.Bytes(src)
		if err != nil {
			return err
		}
		*dst = string(buf)
		return nil
	}

	if uint64(len(src)) < uint64(length) {
		return fmt.Errorf("fixed text payload is too short: got %d bytes, need at least %d", len(src), length)
	}

	// Array elements may include XDR alignment bytes after the fixed value.
	// Decode only the length declared by the server.
	fixedSrc := src[:int(length)]
	buf, err := enc.Decoder.Bytes(fixedSrc)
	if err != nil {
		return err
	}

	tmpStr := string(buf)
	maxBytesPerChar := enc.MaxBytesPerChar
	if maxBytesPerChar == 0 {
		return fmt.Errorf("invalid maximum bytes per character for charset %s: 0", enc.Charset)
	}
	if maxBytesPerChar == 1 {
		*dst = tmpStr
		return nil
	}

	if length%maxBytesPerChar != 0 {
		return fmt.Errorf(
			"fixed text length %d is not divisible by %d bytes per character for charset %s",
			length,
			maxBytesPerChar,
			enc.Charset,
		)
	}

	characterLength := length / maxBytesPerChar
	runes := []rune(tmpStr)
	if uint64(len(runes)) < uint64(characterLength) {
		return fmt.Errorf(
			"fixed text payload contains %d characters, need at least %d for charset %s",
			len(runes),
			characterLength,
			enc.Charset,
		)
	}

	*dst = string(runes[:int(characterLength)])

	return nil
}

func encodeFBString(src string, length uint32, enc *fbproto.Encoding) ([]byte, error) {
	var err error
	res, err := enc.Encoder.String(src)
	if err != nil {
		return nil, err
	}

	if len(res) > int(length) {
		return nil, fmt.Errorf("string truncation")
	}

	return []byte(res), nil
}
