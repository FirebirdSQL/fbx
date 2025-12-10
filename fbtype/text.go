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
	if src == nil {
		*dst = Text{}
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

func (TextCodec) FormatSupported(format int16) bool {
	return format == TextFormatCode || format == BinaryFormatCode
}

func (TextCodec) PreferredFormat() int16 {
	return TextFormatCode
}

func (TextCodec) PlanEncode(m *Map, typeId uint32, subType int32, length uint32, scale int32, format int16, value any) EncodePlan {
	charsetId := CharacterSetForType(typeId, subType, scale)
	name, err := fbproto.CharacterSetIdIntoName(charsetId)
	if err != nil {
		return nil
	}
	encoding, err := fbproto.NewEncoding(name)
	if err != nil {
		return nil
	}

	switch format {
	case TextFormatCode, BinaryFormatCode:
		switch value.(type) {
		case string:
			return encodePlanTextCodecString{encoding: encoding, length: length}
		case []byte:
			return encodePlanTextCodecByteSlice{encoding: encoding, length: length}
		case fmt.Stringer:
			return encodePlanTextCodecStringer{encoding: encoding, length: length}
		case TextValuer:
			return encodePlanTextCodecTextValuer{encoding: encoding, length: length}
		}
	}

	return nil
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
	buf = append(buf, res...)
	return buf, nil
}

type encodePlanTextCodecByteSlice struct {
	encoding *fbproto.Encoding
	length   uint32
}

func (e encodePlanTextCodecByteSlice) Encode(value any, buf []byte) (newBuf []byte, err error) {
	s := string(value.([]byte))
	res, err := encodeFBString(s, e.length, e.encoding)
	buf = append(buf, res...)
	return buf, nil
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
	buf = append(buf, res...)
	return buf, nil
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
	buf = append(buf, res...)

	return buf, nil
}

func (TextCodec) PlanScan(m *Map, typeId uint32, subType int32, length uint32, scale int32, format int16, target any) ScanPlan {
	charsetId := CharacterSetForType(typeId, subType, scale)
	name, err := fbproto.CharacterSetIdIntoName(charsetId)
	if err != nil {
		return nil
	}
	encoding, err := fbproto.NewEncoding(name)
	if err != nil {
		return nil
	}

	switch format {
	case TextFormatCode, BinaryFormatCode:
		switch target.(type) {
		case *string:
			return scanPlanTextAnyToString{encoding: encoding, length: length, typeId: typeId}
		case *[]byte:
			return scanPlanAnyToNewByteSlice{encoding: encoding, length: length, typeId: typeId}
		case BytesScanner:
			return scanPlanAnyToByteScanner{encoding: encoding, length: length, typeId: typeId}
		case TextScanner:
			return scanPlanTextAnyToTextScanner{encoding: encoding, length: length, typeId: typeId}
		}
	}

	return nil
}

func (c TextCodec) DecodeDatabaseSQLValue(m *Map, oid uint32, subType int32, length uint32, scale int32, format int16, src []byte) (driver.Value, error) {
	return c.DecodeValue(m, oid, subType, length, scale, format, src)
}

func (c TextCodec) DecodeValue(m *Map, typeId uint32, subType int32, length uint32, scale int32, format int16, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	charsetId := CharacterSetForType(typeId, subType, scale)
	name, err := fbproto.CharacterSetIdIntoName(charsetId)
	if err != nil {
		return nil, err
	}
	encoding, err := fbproto.NewEncoding(name)
	if err != nil {
		return nil, err
	}

	var res string
	if err = decodeFBString(src, typeId == fbproto.SQL_TYPE_VARYING, length, encoding, &res); err != nil {
		return nil, err
	}

	return res, nil
}

type scanPlanTextAnyToString struct {
	typeId   uint32
	length   uint32
	encoding *fbproto.Encoding
}

func (s scanPlanTextAnyToString) Scan(src []byte, dst any) error {
	if src == nil {
		return fmt.Errorf("cannot scan NULL into %T", dst)
	}
	return decodeFBString(src, s.typeId == fbproto.SQL_TYPE_VARYING, s.length, s.encoding, dst)
}

type scanPlanAnyToNewByteSlice struct {
	typeId   uint32
	length   uint32
	encoding *fbproto.Encoding
}

func (s scanPlanAnyToNewByteSlice) Scan(src []byte, dst any) error {
	p := (dst).(*[]byte)
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
	p := (dst).(BytesScanner)
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
	scanner := (dst).(TextScanner)

	if src == nil {
		return scanner.ScanText(Text{})
	}

	var res string
	if err := decodeFBString(src, s.typeId == fbproto.SQL_TYPE_VARYING, s.length, s.encoding, &res); err != nil {
		return err
	}

	return scanner.ScanText(Text{String: res, Valid: true})
}

func decodeFBString(src []byte, varying bool, length uint32, enc *fbproto.Encoding, dst any) error {
	var err error
	buf, err := enc.Decoder.Bytes(src)
	if err != nil {
		return err
	}

	p, ok := (dst).(*string)
	if !ok {
		return fmt.Errorf("decodeFBString: dst must be *string, got %T", dst)
	}

	tmpStr := string(buf)

	if varying {
		*p = tmpStr
		return nil
	}

	maxBytesPerChar := enc.MaxBytesPerChar
	possibleCharLength := length
	if maxBytesPerChar > 1 && length%maxBytesPerChar == 0 {
		possibleCharLength = length / maxBytesPerChar
	}

	*p = string([]rune(tmpStr)[:possibleCharLength])

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
