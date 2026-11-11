package zeronull

import (
	"database/sql/driver"

	"github.com/firebirdsql/fbx/fbtype"
)

// UUID represents a UUID whose zero value is SQL NULL.
type UUID [16]byte

// SkipUnderlyingTypePlan implements [fbtype.SkipUnderlyingTypePlanner].
func (UUID) SkipUnderlyingTypePlan() {}

// ScanUUID implements [fbtype.UUIDScanner].
func (u *UUID) ScanUUID(v fbtype.UUID) error {
	if !v.Valid {
		*u = UUID{}
		return nil
	}

	*u = UUID(v.Bytes)

	return nil
}

// UUIDValue implements [fbtype.UUIDValuer].
func (u UUID) UUIDValue() (fbtype.UUID, error) {
	if u == (UUID{}) {
		return fbtype.UUID{}, nil
	}
	return fbtype.UUID{Bytes: u, Valid: true}, nil
}

// Scan implements [database/sql.Scanner].
func (u *UUID) Scan(src any) error {
	*u = UUID{}

	if src == nil {
		return nil
	}

	var nullable fbtype.UUID
	err := nullable.Scan(src)
	if err != nil {
		return err
	}

	*u = UUID(nullable.Bytes)

	return nil
}

// Value implements [driver.Valuer]. It returns nil for the zero UUID and a
// canonical UUID string otherwise.
func (u UUID) Value() (driver.Value, error) {
	if u == (UUID{}) {
		return nil, nil
	}

	return fbtype.UUID{Bytes: u, Valid: true}.String(), nil
}
