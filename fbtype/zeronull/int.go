package zeronull

import (
	"database/sql/driver"
	"fmt"
	"math"

	"rdb.red-soft.ru/fbx/fbtype"
)

type Int2 int16

func (Int2) SkipUnderlyingTypePlan() {}

// ScanInt64 implements the Int64Scanner interface.
func (dst *Int2) ScanInt64(n fbtype.Int8) error {
	if !n.Valid {
		*dst = 0
		return nil
	}

	if n.Int64 < math.MinInt16 {
		return fmt.Errorf("%d is less than minimum value for Int2", n.Int64)
	}
	if n.Int64 > math.MaxInt16 {
		return fmt.Errorf("%d is greater than maximum value for Int2", n.Int64)
	}
	*dst = Int2(n.Int64)

	return nil
}

// Int64Value implements the [fbtype.Int64Valuer] interface.
func (src Int2) Int64Value() (fbtype.Int8, error) {
	if src == 0 {
		return fbtype.Int8{}, nil
	}
	return fbtype.Int8{Int64: int64(src), Valid: true}, nil
}

// Scan implements the database/sql Scanner interface.
func (dst *Int2) Scan(src any) error {
	if src == nil {
		*dst = 0
		return nil
	}

	var nullable fbtype.Int2
	err := nullable.Scan(src)
	if err != nil {
		return err
	}

	*dst = Int2(nullable.Int16)

	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (src Int2) Value() (driver.Value, error) {
	if src == 0 {
		return nil, nil
	}
	return int64(src), nil
}

type Int4 int32

func (Int4) SkipUnderlyingTypePlan() {}

// ScanInt64 implements the Int64Scanner interface.
func (dst *Int4) ScanInt64(n fbtype.Int8) error {
	if !n.Valid {
		*dst = 0
		return nil
	}

	if n.Int64 < math.MinInt32 {
		return fmt.Errorf("%d is less than minimum value for Int4", n.Int64)
	}
	if n.Int64 > math.MaxInt32 {
		return fmt.Errorf("%d is greater than maximum value for Int4", n.Int64)
	}
	*dst = Int4(n.Int64)

	return nil
}

// Int64Value implements the [fbtype.Int64Valuer] interface.
func (src Int4) Int64Value() (fbtype.Int8, error) {
	if src == 0 {
		return fbtype.Int8{}, nil
	}
	return fbtype.Int8{Int64: int64(src), Valid: true}, nil
}

// Scan implements the database/sql Scanner interface.
func (dst *Int4) Scan(src any) error {
	if src == nil {
		*dst = 0
		return nil
	}

	var nullable fbtype.Int4
	err := nullable.Scan(src)
	if err != nil {
		return err
	}

	*dst = Int4(nullable.Int32)

	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (src Int4) Value() (driver.Value, error) {
	if src == 0 {
		return nil, nil
	}
	return int64(src), nil
}

type Int8 int64

func (Int8) SkipUnderlyingTypePlan() {}

// ScanInt64 implements the Int64Scanner interface.
func (dst *Int8) ScanInt64(n fbtype.Int8) error {
	if !n.Valid {
		*dst = 0
		return nil
	}

	*dst = Int8(n.Int64)

	return nil
}

// Int64Value implements the [fbtype.Int64Valuer] interface.
func (src Int8) Int64Value() (fbtype.Int8, error) {
	if src == 0 {
		return fbtype.Int8{}, nil
	}
	return fbtype.Int8{Int64: int64(src), Valid: true}, nil
}

// Scan implements the database/sql Scanner interface.
func (dst *Int8) Scan(src any) error {
	if src == nil {
		*dst = 0
		return nil
	}

	var nullable fbtype.Int8
	err := nullable.Scan(src)
	if err != nil {
		return err
	}

	*dst = Int8(nullable.Int64)

	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (src Int8) Value() (driver.Value, error) {
	if src == 0 {
		return nil, nil
	}
	return int64(src), nil
}
