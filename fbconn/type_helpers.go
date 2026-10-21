package fbconn

import "github.com/firebirdsql/fbx/fbproto"

// IsBlobType reports whether the type id represents a blob.
func IsBlobType(typeId uint32) bool {
	return typeId == fbproto.SQL_TYPE_BLOB
}

// IsBlob reports whether the field description represents a blob.
func (fd FieldDescription) IsBlob() bool {
	return IsBlobType(fd.Type)
}

func IsArrayType(typeId uint32) bool {
	return typeId == fbproto.SQL_TYPE_ARRAY
}

func (fd FieldDescription) IsArray() bool {
	return IsArrayType(fd.Type)
}
