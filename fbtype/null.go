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

func (NullCodec) PlanEncode(m *Map, typeid uint32, subType int32, length uint32, scale int32, format int16, value any) EncodePlan {
	switch format {
	case BinaryFormatCode:
		return encodePlanNullValuer{}
	}

	return nil
}

type encodePlanNullValuer struct{}

func (encodePlanNullValuer) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return []byte{}, nil
}

func (NullCodec) PlanScan(m *Map, typeid uint32, subType int32, length uint32, scale int32, format int16, target any) ScanPlan {
	//Looks like there is no case where null type can be scanned, so no need to emit scanners
	return nil
}

func (c NullCodec) DecodeDatabaseSQLValue(m *Map, typeid uint32, subType int32, length uint32, scale int32, format int16, src []byte) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}
	return "", nil
}

func (c NullCodec) DecodeValue(m *Map, typeid uint32, subType int32, length uint32, scale int32, format int16, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}
	return []byte{}, nil
}
