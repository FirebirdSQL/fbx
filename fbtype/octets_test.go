package fbtype_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/fbtype"
)

type testBytesValuer struct {
	value []byte
}

func (v testBytesValuer) BytesValue() ([]byte, error) {
	return v.value, nil
}

func isExpectedEqBytes(a any) func(any) bool {
	return func(v any) bool {
		ab := a.([]byte)
		vb := v.([]byte)

		if (ab == nil) != (vb == nil) {
			return false
		}

		if ab == nil {
			return true
		}

		return bytes.Equal(ab, vb)
	}
}

func TestOctetsCodecPlanEncodeString(t *testing.T) {
	t.Parallel()

	plan := fbtype.OctetsCodec{}.PlanEncode(nil, fbtype.TypeMetadata{}, "random_str")
	require.NotNil(t, plan)

	buf, err := plan.Encode("random_str", nil)
	require.NoError(t, err)
	require.Equal(t, []byte("random_str"), buf)
}

func TestOctetsStringParamWithCryptHash(t *testing.T) {

	t.Parallel()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		tb.Helper()

		if getServerMajorVersionForConn(ctx, tb, conn) <= 3 {
			t.Skip("crypt_hash is not supported by server engine version <= 3")
		}

		var got []byte
		err := conn.QueryRow(ctx,
			"select crypt_hash(? using sha256) from rdb$database",
			"mypassword",
		).Scan(&got)
		require.NoError(tb, err)
		require.Len(tb, got, 32)
	})
}

func TestOctetsCodecPlanEncodeEmptyString(t *testing.T) {
	t.Parallel()

	plan := fbtype.OctetsCodec{}.PlanEncode(nil, fbtype.TypeMetadata{}, "")
	require.NotNil(t, plan)

	buf, err := plan.Encode("", nil)
	require.NoError(t, err)
	require.NotNil(t, buf)
	require.Len(t, buf, 0)
}

func TestOctetsCodecEncodePreservesEmptyValue(t *testing.T) {
	t.Parallel()

	metadata := fbtype.NewTypeMetadata(fbproto.SQL_TYPE_VARYING, fbproto.CS_BINARY, 16, 0)
	tests := []struct {
		name  string
		value any
	}{
		{name: "string", value: ""},
		{name: "bytes", value: []byte{}},
		{name: "BytesValuer", value: testBytesValuer{value: []byte{}}},
	}

	m := fbtype.NewMap()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := m.Encode(metadata, tt.value, nil)
			require.NoError(t, err)
			require.NotNil(t, buf)
			require.Empty(t, buf)
		})
	}

	buf, err := m.Encode(metadata, testBytesValuer{}, nil)
	require.NoError(t, err)
	require.Nil(t, buf)
}

func TestOctetsScanIntoString(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		tb.Helper()

		var got string
		err := conn.QueryRow(ctx,
			"select cast(? as varchar(32) character set octets) from rdb$database",
			"somestring",
		).Scan(&got)
		require.NoError(tb, err)
		require.Equal(tb, "somestring", got)
	})
}

func TestOctetsScanIntoStringPreservesBytes(t *testing.T) {
	t.Parallel()

	m := fbtype.NewMap()
	want := []byte{0xff, 0xfe, 0xfd}

	for _, typeID := range []uint32{fbproto.SQL_TYPE_TEXT, fbproto.SQL_TYPE_VARYING} {
		var got string
		typeMetadata := fbtype.NewTypeMetadata(typeID, 1, uint32(len(want)), 0)
		err := m.Scan(typeMetadata, want, &got)
		require.NoError(t, err)
		require.Equal(t, want, []byte(got))
	}
}

func TestOctetsScanPlanStringErrors(t *testing.T) {
	t.Parallel()

	plan := fbtype.OctetsCodec{}.PlanScan(nil, fbtype.TypeMetadata{}, new(string))
	require.NotNil(t, plan)

	var got string
	err := plan.Scan(nil, &got)
	require.ErrorContains(t, err, "cannot scan NULL into *string")

	err = plan.Scan([]byte("value"), new([]byte))
	require.ErrorIs(t, err, fbtype.ErrScanTargetTypeChanged)
}

func TestOctetsScanPlansRejectChangedTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target any
		other  any
	}{
		{name: "bytes", target: new([]byte), other: new(string)},
		{name: "bytes scanner", target: new(fbtype.PreallocBytes), other: new(string)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := fbtype.OctetsCodec{}.PlanScan(nil, fbtype.TypeMetadata{}, tt.target)
			require.NotNil(t, plan)
			require.ErrorIs(t, plan.Scan([]byte("value"), tt.other), fbtype.ErrScanTargetTypeChanged)
		})
	}
}

func TestOctetsRoundTripStringParamToBytes(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		tb.Helper()

		var got []byte
		err := conn.QueryRow(ctx,
			"select cast(? as varchar(32) character set octets) from rdb$database",
			"mypassword",
		).Scan(&got)
		require.NoError(tb, err)
		require.Equal(tb, []byte("mypassword"), got)
	})
}

func TestOctetsRoundTripBytes(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		tb.Helper()

		tests := []struct {
			name  string
			input []byte
		}{
			{name: "non-empty", input: []byte{1, 2, 3, 4}},
			{name: "empty", input: []byte{}},
			{name: "nil", input: nil},
		}

		for _, tt := range tests {
			var got []byte
			err := conn.QueryRow(ctx,
				"select cast(? as varchar(16) character set octets) from rdb$database",
				tt.input,
			).Scan(&got)
			require.NoError(tb, err, "case=%s", tt.name)

			if tt.input == nil {
				require.Nil(tb, got, "case=%s", tt.name)
			} else {
				require.Equal(tb, tt.input, got, "case=%s", tt.name)
			}
		}
	})
}

func TestPreallocBytes(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		tb.Helper()

		origBuf := []byte{5, 6, 7, 8}
		buf := fbtype.PreallocBytes(origBuf[:0])

		err := conn.QueryRow(ctx,
			"select cast(? as varchar(2) character set octets) from rdb$database",
			[]byte{1, 2},
		).Scan(&buf)
		require.NoError(tb, err)

		require.Len(tb, buf, 2)
		require.Equal(tb, 4, cap(buf))
		require.Equal(tb, []byte{1, 2}, []byte(buf))
		require.Equal(tb, []byte{1, 2, 7, 8}, origBuf)

		err = conn.QueryRow(ctx,
			"select cast(? as varchar(4) character set octets) from rdb$database",
			[]byte{9, 10, 11, 12},
		).Scan(&buf)
		require.NoError(tb, err)

		require.Len(tb, buf, 4)
		require.Equal(tb, 4, cap(buf))
		require.Equal(tb, []byte{9, 10, 11, 12}, []byte(buf))
		require.Equal(tb, []byte{9, 10, 11, 12}, origBuf)

		err = conn.QueryRow(ctx,
			"select cast(? as varchar(5) character set octets) from rdb$database",
			[]byte{3, 4, 5, 6, 7},
		).Scan(&buf)
		require.NoError(tb, err)

		require.Len(tb, buf, 5)
		require.Equal(tb, 5, cap(buf))
		require.Equal(tb, []byte{3, 4, 5, 6, 7}, []byte(buf))
		require.Equal(tb, []byte{9, 10, 11, 12}, origBuf)

		err = conn.QueryRow(ctx,
			"select cast(null as varchar(1) character set octets) from rdb$database",
		).Scan(&buf)
		require.NoError(tb, err)
		require.Nil(tb, buf)

		err = conn.QueryRow(ctx,
			"select cast('' as varchar(1) character set octets) from rdb$database",
		).Scan(&buf)
		require.NoError(tb, err)
		require.NotNil(tb, buf)
		require.Empty(tb, buf)
	})
}

func TestPreallocBytesScanBytes(t *testing.T) {
	t.Parallel()

	storage := []byte{5, 6, 7, 8}
	buf := fbtype.PreallocBytes(storage[:0])

	require.NoError(t, buf.ScanBytes([]byte{1, 2}))
	require.Equal(t, []byte{1, 2}, []byte(buf))
	require.Equal(t, 4, cap(buf))
	require.Equal(t, []byte{1, 2, 7, 8}, storage)

	require.NoError(t, buf.ScanBytes([]byte{9, 10, 11, 12}))
	require.Equal(t, []byte{9, 10, 11, 12}, []byte(buf))
	require.Equal(t, 4, cap(buf))
	require.Equal(t, []byte{9, 10, 11, 12}, storage)

	require.NoError(t, buf.ScanBytes([]byte{13, 14, 15, 16, 17}))
	require.Equal(t, []byte{13, 14, 15, 16, 17}, []byte(buf))
	require.Equal(t, 5, cap(buf))
	require.Equal(t, []byte{9, 10, 11, 12}, storage)

	require.NoError(t, buf.ScanBytes(nil))
	require.Nil(t, buf)

	require.NoError(t, buf.ScanBytes([]byte{}))
	require.NotNil(t, buf)
	require.Empty(t, buf)
}

func TestUndecodedBytes(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		tb.Helper()

		var buf []byte
		err := conn.QueryRow(ctx, "select cast(1 as integer) from rdb$database").Scan((*fbtype.UndecodedBytes)(&buf))
		require.NoError(tb, err)

		require.Len(tb, buf, 4)
		require.Equal(tb, []byte{0, 0, 0, 1}, buf)
	})
}

func TestOctetsStringParamInPasswordComparison(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, tb testing.TB, conn *fbx.Conn) {
		tb.Helper()

		if getServerMajorVersionForConn(ctx, tb, conn) <= 3 {
			t.Skip("crypt_hash is not supported by server engine version <= 3")
		}

		_, _ = conn.Exec(ctx, "drop table test_user_list")

		_, err := conn.Exec(ctx, `
			create table test_user_list (
				id integer not null primary key,
				password blob sub_type binary
			)
		`)
		require.NoError(tb, err)

		defer func() {
			_, _ = conn.Exec(ctx, "drop table test_user_list")
		}()

		_, err = conn.Exec(ctx, `
			insert into test_user_list (id, password)
			values (1, crypt_hash('mypassword' using sha256))
		`)
		require.NoError(tb, err)

		var one int32
		err = conn.QueryRow(ctx, `
			select 1
			from test_user_list l
			where l.id = ?
			  and l.password = crypt_hash(? using sha256)
		`, int32(1), "mypassword").Scan(&one)
		require.NoError(tb, err)
		require.Equal(tb, int32(1), one)
	})
}
