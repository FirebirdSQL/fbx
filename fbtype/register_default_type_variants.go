package fbtype

func registerDefaultFbTypeVariants[T any](m *Map, name string) {
	var value T
	m.RegisterDefaultFbType(value, name)  // T
	m.RegisterDefaultFbType(&value, name) // *T
}
