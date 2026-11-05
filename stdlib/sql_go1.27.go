//go:build go1.27

package stdlib

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"time"

	"github.com/firebirdsql/fbx/fbtype"
)

// [Rows] implements [driver.RowsColumnScanner] as of Go 1.27.
var _ driver.RowsColumnScanner = (*Rows)(nil)

// NextRow implements the [driver.RowsColumnScanner] interface. It advances to the
// next row of data and returns [io.EOF] when there are no more rows.
func (r *Rows) NextRow() error {
	var more bool
	if r.skipNext {
		more = r.skipNextMore
		r.skipNext = false
	} else {
		more = r.rows.Next()
	}

	if !more {
		if err := r.rows.Err(); err != nil {
			return err
		}
		return io.EOF
	}

	r.initValueFuncs()

	var err error
	if r.hasBlobCols {
		r.blobValues, err = scanBlobValues(r.rows, r.blobColumns)
		if err != nil {
			return err
		}
	}

	return nil
}

// ScanColumn implements the [driver.RowsColumnScanner] interface. It preserves
// [database/sql] conversions for scalar destinations and [sql.Scanner] implementations.
// Other destinations, such as Go slices, fbtype.Array, are
// scanned directly using the fbx type map.
func (r *Rows) ScanColumn(scanCtx driver.ScanContext, index int, dest any) error {
	if dest == nil {
		return errors.New("destination not a pointer")
	}
	rv := reflect.ValueOf(dest)
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		return errors.New("destination pointer is nil")
	}

	fd := r.rows.FieldDescriptions()[index]
	typeMetadata := fbtype.NewTypeMetadata(
		fd.Type,
		fd.Subtype,
		fd.Length,
		fd.Scale,
	)

	arr := fd.ArrayData
	if arr != nil {
		arrayData, err := fbtype.NewArrayData(arr.TypeId, arr.SubType, arr.Length, arr.Scale, arr.Dimensions)
		if err != nil {
			return err
		}
		typeMetadata.SetArrayMetadata(arrayData)
	}
	src := r.rows.RawValues()[index]

	if fd.IsBlob() {
		if src == nil {
			return sql.ConvertAssign(scanCtx, dest, nil)
		}
		blobValue, err := buildBlobValue(r.blobValues[index], &r.rows.FieldDescriptions()[index])
		if err != nil {
			return err
		}

		return sql.ConvertAssign(scanCtx, dest, blobValue)
	}

	if isSQLScanDestination(rv.Type()) {
		var value driver.Value
		if src != nil {
			var err error
			value, err = r.valueFuncs[index](src)
			if err != nil {
				return err
			}
		}
		return sql.ConvertAssign(scanCtx, dest, value)
	}

	m := r.conn.conn.TypeMap()

	return m.Scan(typeMetadata, src, dest)
}

// isSQLScanDestination includes named scalar types and nullable pointers to
// [database/sql] destinations. Select the conversion before scanning: retrying a
// failed scan could call user code twice or discard its error.
func isSQLScanDestination(t reflect.Type) bool {
	var namedPointers map[reflect.Type]bool
	for {
		if t.Implements(reflect.TypeFor[sql.Scanner]()) {
			return true
		}
		// These interfaces distinguish native byte slices, such as
		// FlatArray[byte] and PreallocBytes, from ordinary []byte destinations.
		if t.Implements(reflect.TypeFor[fbtype.ArraySetter]()) || t.Implements(reflect.TypeFor[fbtype.BytesScanner]()) {
			return false
		}
		if t.Kind() != reflect.Pointer {
			break
		}
		// Only defined pointer types can form a cycle without reaching a
		// non-pointer type. Leave those to fbx's bounded scan planning.
		if t.Name() != "" {
			if namedPointers[t] {
				return false
			}
			if namedPointers == nil {
				namedPointers = make(map[reflect.Type]bool)
			}
			namedPointers[t] = true
		}
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String, reflect.Interface:
		return true
	case reflect.Slice:
		return t.Elem().Kind() == reflect.Uint8
	case reflect.Struct:
		return t.ConvertibleTo(reflect.TypeFor[time.Time]())
	default:
		return false
	}
}
