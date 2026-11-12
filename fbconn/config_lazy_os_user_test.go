package fbconn_test

import (
	"errors"
	"os/user"
	"runtime"
	"testing"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/fbproto/auth"
	"github.com/stretchr/testify/require"
)

func TestParseConfigDoesNotLookUpOSUserWhenSystemUserExplicit(t *testing.T) {
	restore := fbconn.SetOSUserLookupForTest(func() (*user.User, error) {
		panic("OS user lookup must not run when system_user is explicit")
	})
	t.Cleanup(restore)

	config, err := fbconn.ParseConfig("host=localhost port=3050 database=mydb system_user=explicit-os-user")
	require.NoError(t, err)
	require.Equal(t, "explicit-os-user", config.SystemUser)
}

func TestParseConfigDoesNotLookUpOSUserWhenSystemUserAndPasswordExplicit(t *testing.T) {
	restore := fbconn.SetOSUserLookupForTest(func() (*user.User, error) {
		panic("OS user lookup must not run when system_user is explicit")
	})
	t.Cleanup(restore)

	config, err := fbconn.ParseConfig("firebird://ivan:secret@localhost:3050/mydb?system_user=explicit-os-user")
	require.NoError(t, err)
	require.Equal(t, "explicit-os-user", config.SystemUser)
	require.Equal(t, "ivan", config.User)
	require.Equal(t, "secret", config.Password)
}

func TestParseConfigStillLooksUpOSUserWhenNeeded(t *testing.T) {
	calls := 0
	restore := fbconn.SetOSUserLookupForTest(func() (*user.User, error) {
		calls++
		name := "mocked-os-user"
		if runtime.GOOS == "windows" {
			name = `DOMAIN\mocked-os-user`
		}
		return &user.User{Username: name}, nil
	})
	t.Cleanup(restore)

	// Explicit database credentials must not suppress the independent system_user default.
	config, err := fbconn.ParseConfig("firebird://ivan:secret@localhost:3050/mydb")
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Equal(t, "mocked-os-user", config.SystemUser)
	require.Equal(t, "ivan", config.User)
}

func TestParseConfigOSUserLookupFailureStillUsesExplicitCredentials(t *testing.T) {
	restore := fbconn.SetOSUserLookupForTest(func() (*user.User, error) {
		return nil, errors.New("OS account lookup failed")
	})
	t.Cleanup(restore)

	config, err := fbconn.ParseConfig("firebird://ivan:secret@localhost:3050/mydb")
	require.NoError(t, err)
	require.Empty(t, config.SystemUser)
	require.Equal(t, "ivan", config.User)
	require.Equal(t, "secret", config.Password)
}

// fbx uses Firebird wire encryption rather than home-directory TLS files.
// Skipping or failing the OS account lookup must preserve its security defaults.
func TestParseConfigSecurityDefaultsIndependentOfOSUserLookup(t *testing.T) {
	tests := []struct {
		name        string
		connString  string
		lookupFails bool
		skipLookup  bool
	}{
		{
			name:       "explicit system user",
			connString: "firebird://ivan:secret@localhost:3050/mydb?system_user=explicit-os-user",
			skipLookup: true,
		},
		{
			name:       "default system user",
			connString: "firebird://ivan:secret@localhost:3050/mydb",
		},
		{
			name:        "lookup failure",
			connString:  "firebird://ivan:secret@localhost:3050/mydb",
			lookupFails: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := fbconn.SetOSUserLookupForTest(func() (*user.User, error) {
				if tt.skipLookup {
					panic("OS user lookup must not run when system_user is explicit")
				}
				if tt.lookupFails {
					return nil, errors.New("OS account lookup failed")
				}
				return &user.User{Username: "mocked-os-user"}, nil
			})
			t.Cleanup(restore)

			config, err := fbconn.ParseConfig(tt.connString)
			require.NoError(t, err)
			require.True(t, config.WireCrypt)
			require.Equal(t, auth.DefaultPluginSequence(), config.AuthPluginList)
			require.Equal(t, "ivan", config.User)
			require.Equal(t, "secret", config.Password)
		})
	}
}
