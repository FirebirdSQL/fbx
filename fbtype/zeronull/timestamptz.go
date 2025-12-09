package zeronull

import (
	"database/sql/driver"
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

	*ts = Timestamptz(v.Time)
	return nil
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
