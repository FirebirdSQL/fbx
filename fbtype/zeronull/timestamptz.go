package zeronull

import (
	"database/sql/driver"
	"fmt"
	"time"

	"rdb.red-soft.ru/fbx/fbtype"
)

type Timestamptz time.Time

func (Timestamptz) SkipUnderlyingTypePlan() {}

func (ts *Timestamptz) ScanTimestamptz(v fbtype.Timestamptz) error {
	if !v.Valid {
		*ts = Timestamptz{}
		return nil
	}

	switch v.InfinityModifier {
	case fbtype.Finite:
		*ts = Timestamptz(v.Time)
		return nil
	case fbtype.Infinity:
		return fmt.Errorf("cannot scan Infinity into *time.Time")
	case fbtype.NegativeInfinity:
		return fmt.Errorf("cannot scan -Infinity into *time.Time")
	default:
		return fmt.Errorf("invalid InfinityModifier: %v", v.InfinityModifier)
	}
}

func (ts Timestamptz) TimestamptzValue() (fbtype.Timestamptz, error) {
	if time.Time(ts).IsZero() {
		return fbtype.Timestamptz{}, nil
	}

	return fbtype.Timestamptz{Time: time.Time(ts), Valid: true}, nil
}

// Scan implements the database/sql Scanner interface.
func (ts *Timestamptz) Scan(src any) error {
	if src == nil {
		*ts = Timestamptz{}
		return nil
	}

	var nullable fbtype.Timestamptz
	err := nullable.Scan(src)
	if err != nil {
		return err
	}

	*ts = Timestamptz(nullable.Time)

	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (ts Timestamptz) Value() (driver.Value, error) {
	if time.Time(ts).IsZero() {
		return nil, nil
	}

	return time.Time(ts), nil
}
