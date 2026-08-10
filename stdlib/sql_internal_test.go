package stdlib

import (
	"context"
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
	"time"

	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
)

func TestConnectorBeforeConnectGetsIndependentConfig(t *testing.T) {
	t.Parallel()

	config, err := fbx.ParseConfig("firebird://user:password@primary/database")
	if err != nil {
		t.Fatal(err)
	}
	config.RuntimeParams["application_name"] = "original"
	config.Fallbacks = []*fbconn.FallbackConfig{{Host: "fallback", Port: 3050}}

	callbackErr := errors.New("stop before connect")
	connector := GetConnector(*config, OptionBeforeConnect(func(_ context.Context, candidate *fbx.ConnConfig) error {
		if got := candidate.RuntimeParams["application_name"]; got != "original" {
			return errors.New("connector did not preserve its configuration snapshot")
		}
		if got := candidate.Fallbacks[0].Host; got != "fallback" {
			return errors.New("connector did not preserve its fallback snapshot")
		}
		candidate.RuntimeParams["application_name"] = "changed"
		candidate.Fallbacks[0].Host = "changed"
		return callbackErr
	}))
	config.RuntimeParams["application_name"] = "outside change"
	config.Fallbacks[0].Host = "outside change"

	_, err = connector.Connect(context.Background())
	if !errors.Is(err, callbackErr) {
		t.Fatalf("Connect error = %v, want %v", err, callbackErr)
	}
	if got := config.RuntimeParams["application_name"]; got != "outside change" {
		t.Fatalf("RuntimeParams mutation leaked from BeforeConnect: got %q", got)
	}
	if got := config.Fallbacks[0].Host; got != "outside change" {
		t.Fatalf("Fallbacks mutation leaked from BeforeConnect: got %q", got)
	}
}

func TestDriverOpenConnectorParsesName(t *testing.T) {
	t.Parallel()

	d := &Driver{}
	if _, err := d.OpenConnector("firebird://%"); err == nil {
		t.Fatal("OpenConnector accepted an invalid connection string")
	}
}

func TestDriverRegisteredConfigIsSnapshot(t *testing.T) {
	t.Parallel()

	d := &Driver{}
	config, err := fbx.ParseConfig("firebird://user:password@original/database")
	if err != nil {
		t.Fatal(err)
	}
	config.RuntimeParams["application_name"] = "original"

	name := d.registerConnConfig(config)
	config.Host = "changed"
	config.RuntimeParams["application_name"] = "changed"

	connector, err := d.OpenConnector(name)
	if err != nil {
		t.Fatal(err)
	}
	dc := connector.(*driverConnector)
	if dc.config.Host != "original" {
		t.Fatalf("registered Host = %q, want %q", dc.config.Host, "original")
	}
	if got := dc.config.RuntimeParams["application_name"]; got != "original" {
		t.Fatalf("registered application_name = %q, want %q", got, "original")
	}

	dc.config.RuntimeParams["application_name"] = "connector"
	d.configMu.RLock()
	registeredApplicationName := d.configs[name].RuntimeParams["application_name"]
	d.configMu.RUnlock()
	if registeredApplicationName != "original" {
		t.Fatalf("connector mutation leaked to registered config: got %q", registeredApplicationName)
	}
	d.unregisterConnConfig(name)
}

func TestNilConnectorInputsPanic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func()
	}{
		{name: "pool", call: func() { GetPoolConnector(nil) }},
		{name: "registered config", call: func() { RegisterConnConfig(nil) }},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("call did not panic")
				}
			}()
			tt.call()
		})
	}
}

func TestNumericRowValueFunc(t *testing.T) {
	t.Parallel()

	encodeInt64 := func(v int64) []byte {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(v))
		return buf
	}

	valueFunc := numericRowValueFunc(
		fbtype.NewMap(),
		fbtype.NewTypeMetadata(fbproto.SQL_TYPE_INT64, 1, 8, -2),
	)
	tests := []struct {
		src  []byte
		want string
	}{
		{src: encodeInt64(12345), want: "123.45"},
		{src: encodeInt64(-67890), want: "-678.90"},
	}

	for _, tt := range tests {
		got, err := valueFunc(tt.src)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Fatalf("got %#v (%T), want %q", got, got, tt.want)
		}
	}
}

func TestRawBytesRowValueFuncReturnsCopy(t *testing.T) {
	t.Parallel()

	src := []byte{1, 2, 3, 4}
	value, err := rawBytesRowValueFunc(src)
	if err != nil {
		t.Fatal(err)
	}

	got, ok := value.([]byte)
	if !ok {
		t.Fatalf("got %T, want []byte", value)
	}
	src[0] = 9
	if got[0] != 1 {
		t.Fatalf("returned bytes alias source: got %v", got)
	}

	nullValue, err := rawBytesRowValueFunc(nil)
	if err != nil {
		t.Fatal(err)
	}
	if nullValue != nil {
		t.Fatalf("NULL: got %#v (%T), want nil", nullValue, nullValue)
	}
}

func TestColumnTypeScanType(t *testing.T) {
	t.Parallel()

	anyType := reflect.TypeOf((*any)(nil)).Elem()
	tests := []struct {
		name string
		fd   fbconn.FieldDescription
		want reflect.Type
	}{
		{name: "numeric", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_LONG, Subtype: 1}, want: reflect.TypeOf("")},
		{name: "blob", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_BLOB}, want: reflect.TypeOf(Blob{})},
		{name: "double", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_DOUBLE}, want: reflect.TypeOf(float64(0))},
		{name: "float", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_FLOAT}, want: reflect.TypeOf(float32(0))},
		{name: "bigint", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_INT64}, want: reflect.TypeOf(int64(0))},
		{name: "integer", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_LONG}, want: reflect.TypeOf(int32(0))},
		{name: "smallint", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_SHORT}, want: reflect.TypeOf(int16(0))},
		{name: "boolean", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_BOOLEAN}, want: reflect.TypeOf(false)},
		{name: "date", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_DATE}, want: reflect.TypeOf(time.Time{})},
		{name: "time", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_TIME}, want: reflect.TypeOf(time.Time{})},
		{name: "timestamp", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_TIMESTAMP}, want: reflect.TypeOf(time.Time{})},
		{name: "int128", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_INT128}, want: reflect.TypeOf("")},
		{name: "decfloat16", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_DEC64}, want: reflect.TypeOf("")},
		{name: "text", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_VARYING}, want: reflect.TypeOf("")},
		{name: "octets", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_VARYING, Subtype: fbproto.CS_BINARY}, want: reflect.TypeOf([]byte(nil))},
		{name: "array", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_ARRAY}, want: anyType},
		{name: "null", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_NULL}, want: anyType},
		{name: "legacy d float", fd: fbconn.FieldDescription{Type: fbproto.SQL_TYPE_D_FLOAT}, want: anyType},
		{name: "unknown", fd: fbconn.FieldDescription{Type: 65001}, want: anyType},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := columnTypeScanType(tt.fd); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

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
