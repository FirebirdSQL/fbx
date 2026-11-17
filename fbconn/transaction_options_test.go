package fbconn

import (
	"context"
	"net"
	"testing"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

func TestDefaultTransactionTPB(t *testing.T) {
	for _, tt := range []struct {
		isolation string
		tags      []byte
	}{
		{"", nil},
		{"serializable", []byte{fbproto.Isc_tpb_consistency}},
		{"repeatable read", []byte{fbproto.Isc_tpb_concurrency}},
		{"read committed", []byte{fbproto.Isc_tpb_read_committed, fbproto.Isc_tpb_rec_version}},
		{"repeatable_read", []byte{fbproto.Isc_tpb_concurrency}},
		{"read_committed", []byte{fbproto.Isc_tpb_read_committed, fbproto.Isc_tpb_rec_version}},
	} {
		for _, readOnly := range []bool{false, true} {
			tpb, err := defaultTransactionTPB(tt.isolation, readOnly)
			require.NoError(t, err)
			var expected []byte
			if readOnly || tt.isolation != "" {
				expected = []byte{fbproto.Isc_tpb_version3}
				if readOnly {
					expected = append(expected, fbproto.Isc_tpb_read)
				}
				expected = append(expected, tt.tags...)
			}
			require.Equal(t, expected, tpb, "isolation=%q readOnly=%t", tt.isolation, readOnly)
		}
	}
	_, err := defaultTransactionTPB("invalid", false)
	require.Error(t, err)
}

func TestParseConfigDefaultTransactionIsolation(t *testing.T) {
	for _, tt := range []struct {
		connString string
		isolation  string
	}{
		{"", ""},
		{"default_transaction_isolation=''", ""},
		{"default_transaction_isolation='read committed'", "read committed"},
		{"default_transaction_isolation=' REPEATABLE READ '", "repeatable read"},
		{"default_transaction_isolation=serializable", "serializable"},
		{"default_transaction_isolation=read_committed", "read_committed"},
		{"default_transaction_isolation=repeatable_read", "repeatable_read"},
		{"default_transaction_isolation=' READ_COMMITTED '", "read_committed"},
		{"firebird://localhost/test?default_transaction_isolation=read%20committed", "read committed"},
		{"firebird://localhost/test?default_transaction_isolation=read_committed", "read_committed"},
		{"firebird://localhost/test?default_transaction_isolation=repeatable_read", "repeatable_read"},
	} {
		t.Run(tt.connString, func(t *testing.T) {
			config, err := ParseConfig(tt.connString)
			require.NoError(t, err)
			require.Equal(t, tt.isolation, config.DefaultTransactionIsolation)
			require.Equal(t, tt.isolation, config.Copy().DefaultTransactionIsolation)
			require.NotContains(t, config.RuntimeParams, "default_transaction_isolation")
		})
	}
	for _, value := range []string{"read uncommitted", "read_uncommitted", "read__committed", "snapshot", "invalid"} {
		config, err := ParseConfig("default_transaction_isolation='" + value + "'")
		require.Nil(t, config)
		var parseErr *ParseConfigError
		require.ErrorAs(t, err, &parseErr)
		require.Contains(t, err.Error(), "default_transaction_isolation")
	}
}

func TestConnectRejectsInvalidDefaultTransactionIsolation(t *testing.T) {
	config, err := ParseConfig("")
	require.NoError(t, err)
	config.DefaultTransactionIsolation = "invalid"
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		t.Fatal("invalid isolation must be rejected before dialing")
		return nil, nil
	}
	conn, err := ConnectConfig(context.Background(), config)
	require.Nil(t, conn)
	var connectErr *ConnectError
	require.ErrorAs(t, err, &connectErr)
	require.ErrorContains(t, err, "unknown default transaction isolation")
}
