package zeronull

import (
	"database/sql/driver"
	"math/big"

	"github.com/firebirdsql/fbx/fbtype"
)

type Int128 struct {
	Int big.Int
}

func (Int128) SkipUnderlyingTypePlan() {}

func (dst *Int128) ScanInt128(n fbtype.Int128) error {
	if !n.Valid {
		dst.Int = big.Int{}
		return nil
	}
	dst.Int.Set(&n.Int)
	return nil
}

func (src Int128) Int128Value() (fbtype.Int128, error) {
	if src.Int.Sign() == 0 {
		return fbtype.Int128{}, nil
	}
	var tmp big.Int
	tmp.Set(&src.Int)
	return fbtype.Int128{Int: tmp, Valid: true}, nil
}

func (dst *Int128) Scan(src any) error {
	if src == nil {
		dst.Int = big.Int{}
		return nil
	}

	var n fbtype.Int128
	if err := n.Scan(src); err != nil {
		return err
	}
	if !n.Valid {
		dst.Int = big.Int{}
		return nil
	}

	dst.Int.Set(&n.Int)
	return nil
}

func (src Int128) Value() (driver.Value, error) {
	if src.Int.Sign() == 0 {
		return nil, nil
	}
	return src.Int.String(), nil
}
