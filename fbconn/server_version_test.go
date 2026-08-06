package fbconn

import (
	"testing"

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
			name: "database info with multiple version strings",
			raw:  "\x02\x1bLI-V4.0.6.3221 Firebird 4.04LI-V4.0.6.3221 Firebird 4.0/tcp/server/P17:C",
			expected: ServerVersion{
				Major: 4,
				Minor: 0,
				Patch: 6,
				Build: 3221,
			},
		},
		{
			name: "missing optional components",
			raw:  "WI-V4 Firebird",
			expected: ServerVersion{
				Major: 4,
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
	_, err := parseServerVersion("Firebird version unknown")
	require.EqualError(t, err, `server version string parsing error: "Firebird version unknown"`)
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
