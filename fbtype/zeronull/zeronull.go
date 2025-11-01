package zeronull

import (
	"github.com/red-soft/fbx/fbtype"
)

// Register registers the zeronull types so they can be used in query exec modes that do not know the server OIDs.
func Register(m *fbtype.Map) {
	m.RegisterDefaultFbType(Float8(0), "float8")
	m.RegisterDefaultFbType(Int2(0), "int2")
	m.RegisterDefaultFbType(Int4(0), "int4")
	m.RegisterDefaultFbType(Int8(0), "int8")
	m.RegisterDefaultFbType(Text(""), "text")
	m.RegisterDefaultFbType(Timestamp{}, "timestamp")
	m.RegisterDefaultFbType(Timestamptz{}, "timestamptz")
	m.RegisterDefaultFbType(UUID{}, "uuid")
}
