package fbtype

import (
	"database/sql/driver"
)

type NullCodec struct{}

func (NullCodec) FormatSupported(format int16) bool {
	return format == BinaryFormatCode
}

func (NullCodec) PreferredFormat() int16 {
	return BinaryFormatCode
}

func (NullCodec) PlanEncode(m *Map, typeMetadata TypeMetadata, value any) EncodePlan {
	switch typeMetadata.FormatCode {
	case BinaryFormatCode:
		return encodePlanNullValuer{}
	}

	return nil
}

type encodePlanNullValuer struct{}

func (encodePlanNullValuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return buf, nil
}

func (NullCodec) PlanScan(m *Map, typeMetadata TypeMetadata, target any) ScanPlan {
	//Looks like there is no case where null type can be scanned, so no need to emit scanners
	return nil
}

func (c NullCodec) DecodeDatabaseSQLValue(m *Map, typeMetadata TypeMetadata, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}
	return "", nil
}

func (c NullCodec) DecodeValue(m *Map, typeMetadata TypeMetadata, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}
	return []byte{}, nil
}
