package zeronull

import (
	"rdb.red-soft.ru/fbx/fbtype"
)

// Register registers the zeronull types so they can be used in query exec modes that do not know the server TypeIDs.
func Register(m *fbtype.Map) {
	m.RegisterDefaultFbType(Float8(0), "double precision")
	m.RegisterDefaultFbType(Int2(0), "smallint")
	m.RegisterDefaultFbType(Int4(0), "int")
	m.RegisterDefaultFbType(Int8(0), "bigint")
	m.RegisterDefaultFbType(Text(""), "varchar")
	m.RegisterDefaultFbType(Timestamp{}, "timestamp")
	m.RegisterDefaultFbType(Timestamptz{}, "timestamptz")
	m.RegisterDefaultFbType(Int128{}, "int128")
	m.RegisterDefaultFbType(UUID{}, "octets")
	m.RegisterDefaultFbType(&UUID{}, "octets")
}
