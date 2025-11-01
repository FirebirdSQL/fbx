//go:build nopgxregisterdefaulttypes

package fbtype

func registerDefaultPgTypeVariants[T any](m *Map, name string) {
}
