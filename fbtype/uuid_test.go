package fbtype_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
)

func TestUUIDCodec(t *testing.T) {
	t.Parallel()

	codec := fbtype.UUIDCodec{}
	metadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, fbproto.CS_BINARY, 16, 0)
	wantBytes := [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	wantText := "00010203-0405-0607-0809-0a0b0c0d0e0f"
	wantUUID := fbtype.UUID{Bytes: wantBytes, Valid: true}

	plan := codec.PlanEncode(nil, metadata, wantUUID)
	require.NotNil(t, plan)

	encoded, err := plan.Encode(wantUUID, []byte{0xff})
	require.NoError(t, err)
	require.Equal(t, append([]byte{0xff}, wantBytes[:]...), encoded)

	nullEncoded, err := plan.Encode(fbtype.UUID{}, nil)
	require.NoError(t, err)
	require.Nil(t, nullEncoded)
	require.Nil(t, codec.PlanEncode(nil, metadata, wantText))

	var scannedUUID fbtype.UUID
	uuidPlan := codec.PlanScan(nil, metadata, &scannedUUID)
	require.NotNil(t, uuidPlan)
	require.NoError(t, uuidPlan.Scan(wantBytes[:], &scannedUUID))
	require.Equal(t, wantUUID, scannedUUID)
	require.ErrorIs(t, uuidPlan.Scan(wantBytes[:], new(fbtype.Text)), fbtype.ErrScanTargetTypeChanged)
	require.ErrorContains(t, uuidPlan.Scan(wantBytes[:15], &scannedUUID), "invalid length for UUID: 15")
	require.NoError(t, uuidPlan.Scan(nil, &scannedUUID))
	require.Equal(t, fbtype.UUID{}, scannedUUID)

	var scannedText fbtype.Text
	textPlan := codec.PlanScan(nil, metadata, &scannedText)
	require.NotNil(t, textPlan)
	require.NoError(t, textPlan.Scan(wantBytes[:], &scannedText))
	require.Equal(t, fbtype.Text{String: wantText, Valid: true}, scannedText)
	require.ErrorIs(t, textPlan.Scan(wantBytes[:], new(fbtype.UUID)), fbtype.ErrScanTargetTypeChanged)
	require.NoError(t, textPlan.Scan(nil, &scannedText))
	require.Equal(t, fbtype.Text{}, scannedText)

	databaseValue, err := codec.DecodeDatabaseSQLValue(nil, metadata, wantBytes[:])
	require.NoError(t, err)
	require.Equal(t, wantText, databaseValue)

	decoded, err := codec.DecodeValue(nil, metadata, wantBytes[:])
	require.NoError(t, err)
	require.Equal(t, wantBytes, decoded)

	decoded, err = codec.DecodeValue(nil, metadata, wantBytes[:15])
	require.ErrorContains(t, err, "invalid length for UUID: 15")
	require.Nil(t, decoded)
}

func TestUUIDEncodesThroughOctetsCodec(t *testing.T) {
	t.Parallel()

	metadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_TEXT, fbproto.CS_BINARY, 16, 0)
	want := fbtype.UUID{
		Bytes: [16]byte{15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
		Valid: true,
	}

	encoded, err := fbtype.NewMap().Encode(metadata, want, nil)
	require.NoError(t, err)
	require.Equal(t, want.Bytes[:], encoded)

	encoded, err = fbtype.NewMap().Encode(metadata, fbtype.UUID{}, nil)
	require.NoError(t, err)
	require.Nil(t, encoded)
}

func TestUUID_String(t *testing.T) {
	tests := []struct {
		name string
		src  fbtype.UUID
		want string
	}{
		{
			name: "first",
			src: fbtype.UUID{
				Bytes: [16]byte{29, 72, 90, 122, 109, 24, 69, 153, 140, 108, 52, 66, 86, 22, 136, 122},
				Valid: true,
			},
			want: "1d485a7a-6d18-4599-8c6c-34425616887a",
		},
		{
			name: "third",
			src: fbtype.UUID{
				Bytes: [16]byte{},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.src.String()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUUID_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		src  fbtype.UUID
		want []byte
	}{
		{
			name: "first",
			src: fbtype.UUID{
				Bytes: [16]byte{29, 72, 90, 122, 109, 24, 69, 153, 140, 108, 52, 66, 86, 22, 136, 122},
				Valid: true,
			},
			want: []byte(`"1d485a7a-6d18-4599-8c6c-34425616887a"`),
		},
		{
			name: "third",
			src: fbtype.UUID{
				Bytes: [16]byte{},
			},
			want: []byte("null"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.src.MarshalJSON()
			require.NoError(t, err)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUUID_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		want    *fbtype.UUID
		src     []byte
		wantErr bool
	}{
		{
			name: "first",
			want: &fbtype.UUID{
				Bytes: [16]byte{29, 72, 90, 122, 109, 24, 69, 153, 140, 108, 52, 66, 86, 22, 136, 122},
				Valid: true,
			},
			src:     []byte(`"1d485a7a-6d18-4599-8c6c-34425616887a"`),
			wantErr: false,
		},
		{
			name: "second",
			want: &fbtype.UUID{
				Bytes: [16]byte{},
			},
			src:     []byte("null"),
			wantErr: false,
		},
		{
			name: "third",
			want: &fbtype.UUID{
				Bytes: [16]byte{},
				Valid: false,
			},
			src:     []byte("1d485a7a-6d18-4599-8c6c-34425616887a"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &fbtype.UUID{}
			if err := got.UnmarshalJSON(tt.src); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUUIDtoFB(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		const tableName = "fbtype_uuid_codec_test"

		_, err := conn.Exec(ctx, "recreate table "+tableName+"(id integer primary key, u char(16) character set octets)")
		require.NoError(t, err)
		defer func() {
			_, _ = conn.Exec(ctx, "drop table "+tableName)
		}()

		_, err = conn.Exec(ctx, "insert into "+tableName+" values(1, gen_uuid())")
		require.NoError(t, err)

		var generated fbtype.UUID
		var generatedText string
		err = conn.QueryRow(ctx, "select u, uuid_to_char(u) from "+tableName+" where id = 1").Scan(&generated, &generatedText)
		require.NoError(t, err)
		assert.Equal(t, generated.String(), strings.ToLower(generatedText))

		want := fbtype.UUID{
			Bytes: [16]byte{29, 72, 90, 122, 109, 24, 69, 153, 140, 108, 52, 66, 86, 22, 136, 122},
			Valid: true,
		}
		_, err = conn.Exec(ctx, "insert into "+tableName+" values(2, ?)", want)
		require.NoError(t, err)

		var got fbtype.UUID
		var gotText string
		err = conn.QueryRow(ctx, "select u, uuid_to_char(u) from "+tableName+" where id = 2").Scan(&got, &gotText)
		require.NoError(t, err)
		require.Equal(t, want, got)
		require.Equal(t, want.String(), strings.ToLower(gotText))

		var raw any
		err = conn.QueryRow(ctx, "select u from "+tableName+" where id = 2").Scan(&raw)
		require.NoError(t, err)
		require.Equal(t, want.Bytes[:], raw)

		_, err = conn.Exec(ctx, "insert into "+tableName+" values(3, ?)", fbtype.UUID{})
		require.NoError(t, err)

		var nullUUID fbtype.UUID
		err = conn.QueryRow(ctx, "select u from "+tableName+" where id = 3").Scan(&nullUUID)
		require.NoError(t, err)
		require.Equal(t, fbtype.UUID{}, nullUUID)
	})
}
