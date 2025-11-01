package zeronull

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/red-soft/fbx/fbtype"
)

type Timestamp time.Time

func (Timestamp) SkipUnderlyingTypePlan() {}

func (ts *Timestamp) ScanTimestamp(v fbtype.Timestamp) error {
	if !v.Valid {
		*ts = Timestamp{}
		return nil
	}

	switch v.InfinityModifier {
	case fbtype.Finite:
		*ts = Timestamp(v.Time)
		return nil
	case fbtype.Infinity:
		return fmt.Errorf("cannot scan Infinity into *time.Time")
	case fbtype.NegativeInfinity:
		return fmt.Errorf("cannot scan -Infinity into *time.Time")
	default:
		return fmt.Errorf("invalid InfinityModifier: %v", v.InfinityModifier)
	}
}

func (ts Timestamp) TimestampValue() (fbtype.Timestamp, error) {
	if time.Time(ts).IsZero() {
		return fbtype.Timestamp{}, nil
	}

	return fbtype.Timestamp{Time: time.Time(ts), Valid: true}, nil
}

// Scan implements the database/sql Scanner interface.
func (ts *Timestamp) Scan(src any) error {
	if src == nil {
		*ts = Timestamp{}
		return nil
	}

	var nullable fbtype.Timestamp
	err := nullable.Scan(src)
	if err != nil {
		return err
	}

	*ts = Timestamp(nullable.Time)

	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (ts Timestamp) Value() (driver.Value, error) {
	if time.Time(ts).IsZero() {
		return nil, nil
	}

	return time.Time(ts), nil
}
