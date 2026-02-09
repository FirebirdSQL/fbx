package stdlib

import (
	"testing"

	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestNumericPrecisionForField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fd        fbconn.FieldDescription
		wantPrec  int64
		wantFound bool
	}{
		{
			name:      "type short",
			fd:        fbconn.FieldDescription{Type: fbproto.SQL_TYPE_SHORT, Length: 16},
			wantPrec:  4,
			wantFound: true,
		},
		{
			name:      "type long",
			fd:        fbconn.FieldDescription{Type: fbproto.SQL_TYPE_LONG, Length: 16},
			wantPrec:  9,
			wantFound: true,
		},
		{
			name:      "type int64",
			fd:        fbconn.FieldDescription{Type: fbproto.SQL_TYPE_INT64, Length: 16},
			wantPrec:  18,
			wantFound: true,
		},
		{
			name:      "type int128",
			fd:        fbconn.FieldDescription{Type: fbproto.SQL_TYPE_INT128, Length: 2},
			wantPrec:  38,
			wantFound: true,
		},
		{
			name:      "type dec64",
			fd:        fbconn.FieldDescription{Type: fbproto.SQL_TYPE_DEC64, Length: 2},
			wantPrec:  16,
			wantFound: true,
		},
		{
			name:      "type dec128",
			fd:        fbconn.FieldDescription{Type: fbproto.SQL_TYPE_DEC128, Length: 2},
			wantPrec:  34,
			wantFound: true,
		},
		{
			name:      "unknown length",
			fd:        fbconn.FieldDescription{Type: fbproto.SQL_TYPE_NULL, Length: 3},
			wantPrec:  0,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotPrec, gotFound := numericPrecisionForField(tt.fd)
			if gotFound != tt.wantFound {
				t.Fatalf("found: got %v, want %v", gotFound, tt.wantFound)
			}
			if gotPrec != tt.wantPrec {
				t.Fatalf("precision: got %d, want %d", gotPrec, tt.wantPrec)
			}
		})
	}
}
