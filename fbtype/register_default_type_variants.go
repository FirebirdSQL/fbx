package fbtype

func registerDefaultFbTypeVariants[T any](m *Map, name string) {
	arrayName := "_" + name

	var value T
	m.RegisterDefaultFbType(value, name)  // T
	m.RegisterDefaultFbType(&value, name) // *T

	var sliceT []T
	m.RegisterDefaultFbType(sliceT, arrayName)  // []T
	m.RegisterDefaultFbType(&sliceT, arrayName) // *[]T

	var slicePtrT []*T
	m.RegisterDefaultFbType(slicePtrT, arrayName)  // []*T
	m.RegisterDefaultFbType(&slicePtrT, arrayName) // *[]*T

	var arrayOfT Array[T]
	m.RegisterDefaultFbType(arrayOfT, arrayName)  // Array[T]
	m.RegisterDefaultFbType(&arrayOfT, arrayName) // *Array[T]

	var arrayOfPtrT Array[*T]
	m.RegisterDefaultFbType(arrayOfPtrT, arrayName)  // Array[*T]
	m.RegisterDefaultFbType(&arrayOfPtrT, arrayName) // *Array[*T]

	var flatArrayOfT FlatArray[T]
	m.RegisterDefaultFbType(flatArrayOfT, arrayName)  // FlatArray[T]
	m.RegisterDefaultFbType(&flatArrayOfT, arrayName) // *FlatArray[T]

	var flatArrayOfPtrT FlatArray[*T]
	m.RegisterDefaultFbType(flatArrayOfPtrT, arrayName)  // FlatArray[*T]
	m.RegisterDefaultFbType(&flatArrayOfPtrT, arrayName) // *FlatArray[*T]
}
