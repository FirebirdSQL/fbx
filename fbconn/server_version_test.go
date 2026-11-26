package fbconn

import (
	"testing"

	"github.com/firebirdsql/fbx/fbproto"

	"github.com/stretchr/testify/require"
)

func TestParseServerVersion(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected ServerVersion
	}{
		{
			name: "Firebird",
			raw:  "WI-V5.0.3.1683 Firebird 5.0",
			expected: ServerVersion{
				Major: 5,
				Minor: 0,
				Patch: 3,
				Build: 1683,
			},
		},
		{
			name: "RedDatabase",
			raw:  "LI-V6.0.1.42 RedDatabase 6.0",
			expected: ServerVersion{
				Major: 6,
				Minor: 0,
				Patch: 1,
				Build: 42,
			},
		},
		{
			name: "Firebird 4",
			raw:  "LI-V4.0.6.3221 Firebird 4.0",
			expected: ServerVersion{
				Major: 4,
				Minor: 0,
				Patch: 6,
				Build: 3221,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := parseServerVersion(tt.raw)
			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
			require.Equal(t, tt.expected.String(), actual.String())
		})
	}
}

func TestParseServerVersionRejectsInvalidValue(t *testing.T) {
	for _, raw := range []string{"Firebird version unknown", "WI-V4 Firebird", "WI-V4.0.6 Firebird", "WI-XX4.0.6.3221 Firebird"} {
		_, err := parseServerVersion(raw)
		require.Error(t, err)
	}
}

func TestServerVersionCompare(t *testing.T) {
	version := ServerVersion{Major: 5, Minor: 0, Patch: 3}

	require.Equal(t, "5.0.3.0", version.String())
	require.Equal(t, -1, version.Compare(5, 0, 4))
	require.Equal(t, 0, version.Compare(5, 0, 3))
	require.Equal(t, 1, version.Compare(4, 99, 99))
	require.True(t, version.AtLeast(5, 0, 3))
	require.False(t, version.AtLeast(5, 1, 0))
}

func TestParseDevelopmentServerVersions(t *testing.T) {
	for _, raw := range []string{
		"LI-T6.0.0.2459 Firebird 6.0 2f725cf",
		"LI-X6.0.0.2459-dev Firebird 6.0",
		"WI-V6.0.0.2459-rc1 Firebird 6.0",
	} {
		t.Run(raw, func(t *testing.T) {
			want := ServerVersion{Major: 6, Build: 2459}
			got, err := parseServerVersion(raw)
			require.NoError(t, err)
			require.Equal(t, want, got)
			service := tagBufferValue(fbproto.Isc_info_svc_server_version, []byte(raw))
			service = append(service, fbproto.Isc_info_end)
			got, err = parseServiceServerVersion(service)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestParseDatabaseVersionEntries(t *testing.T) {
	first := "LI-T6.0.0.2459 Firebird 6.0"
	second := "LI-V5.0.3.1683 Firebird 5.0/tcp/server/P17:C"
	payload := append([]byte{2, byte(len(first))}, first...)
	payload = append(payload, byte(len(second)))
	payload = append(payload, second...)
	conn := &FbConn{}
	buf := tagBufferValue(fbproto.Isc_info_firebird_version, payload)
	buf = append(buf, fbproto.Isc_info_end)
	require.NoError(t, conn.parseDatabaseInfo(buf))
	require.Equal(t, ServerVersion{Major: 6, Build: 2459}, conn.serverVersion)
	require.Equal(t, 6, conn.serverMajorVersion)
}

func TestParseDatabaseVersionRejectsMalformedEntries(t *testing.T) {
	for _, payload := range [][]byte{
		nil, {0}, {1}, {1, 20, 'L', 'I'},
		// A valid-looking version outside the first entry must not be used.
		append([]byte{2, 3, 'b', 'a', 'd', byte(len("LI-V5.0.3.1683 Firebird 5.0"))}, []byte("LI-V5.0.3.1683 Firebird 5.0")...),
	} {
		conn := &FbConn{}
		buf := append(tagBufferValue(fbproto.Isc_info_firebird_version, payload), fbproto.Isc_info_end)
		require.Error(t, conn.parseDatabaseInfo(buf))
	}
}
