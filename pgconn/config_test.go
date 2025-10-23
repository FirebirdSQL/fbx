package pgconn_test

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/red-soft/fbx/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FIXME: skipping on Windows, investigate why this test fails in CI environment")
	}
}

func getDefaultPort(t *testing.T) uint16 {
	if envPGPORT := os.Getenv("PGPORT"); envPGPORT != "" {
		p, err := strconv.ParseUint(envPGPORT, 10, 16)
		require.NoError(t, err)
		return uint16(p)
	}
	return 3050
}

func getDefaultUser(t *testing.T) string {
	if pguser := os.Getenv("PGUSER"); pguser != "" {
		return pguser
	}

	var osUserName string
	osUser, err := user.Current()
	if err == nil {
		// Windows gives us the username here as `DOMAIN\user` or `LOCALPCNAME\user`,
		// but the libpq default is just the `user` portion, so we strip off the first part.
		if runtime.GOOS == "windows" && strings.Contains(osUser.Username, "\\") {
			osUserName = osUser.Username[strings.LastIndex(osUser.Username, "\\")+1:]
		} else {
			osUserName = osUser.Username
		}
	}

	return osUserName
}

func TestParseConfig(t *testing.T) {
	skipOnWindows(t)
	t.Parallel()

	config, err := pgconn.ParseConfig("")
	require.NoError(t, err)
	defaultHost := config.Host

	defaultUser := getDefaultUser(t)
	defaultPort := getDefaultPort(t)

	tests := []struct {
		name       string
		connString string
		config     *pgconn.Config
	}{
		// Test all sslmodes
		{
			name:       "sslmode not set (prefer)",
			connString: "firebird://jack:secret@localhost:5432/mydb",
			config: &pgconn.Config{
				User:          "jack",
				Password:      "secret",
				Host:          "localhost",
				Port:          5432,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
				Fallbacks: []*pgconn.FallbackConfig{
					{
						Host: "localhost",
						Port: 5432,
					},
				},
			},
		},
		{
			name:       "database url everything",
			connString: "firebird://jack:secret@localhost:5432/mydb?application_name=pgxtest&search_path=myschema&connect_timeout=5",
			config: &pgconn.Config{
				User:           "jack",
				Password:       "secret",
				Host:           "localhost",
				Port:           5432,
				Database:       "mydb",
				ConnectTimeout: 5 * time.Second,
				RuntimeParams: map[string]string{
					"application_name": "pgxtest",
					"search_path":      "myschema",
				},
			},
		},
		{
			name:       "database url missing password",
			connString: "firebird://jack@localhost:5432/mydb",
			config: &pgconn.Config{
				User:          "jack",
				Host:          "localhost",
				Port:          5432,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url missing user and password",
			connString: "firebird://localhost:5432/mydb",
			config: &pgconn.Config{
				User:          defaultUser,
				Host:          "localhost",
				Port:          5432,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url missing port",
			connString: "firebird://jack:secret@localhost:5432/mydb",
			config: &pgconn.Config{
				User:          "jack",
				Password:      "secret",
				Host:          "localhost",
				Port:          5432,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url unix domain socket host",
			connString: "firebird:///foo?host=/tmp",
			config: &pgconn.Config{
				User:          defaultUser,
				Host:          "/tmp",
				Port:          defaultPort,
				Database:      "foo",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url unix domain socket host on windows",
			connString: "firebird:///foo?host=C:\\tmp",
			config: &pgconn.Config{
				User:          defaultUser,
				Host:          "C:\\tmp",
				Port:          defaultPort,
				Database:      "foo",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url dbname",
			connString: "firebird://localhost/?dbname=foo",
			config: &pgconn.Config{
				User:          defaultUser,
				Host:          "localhost",
				Port:          defaultPort,
				Database:      "foo",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url postgresql protocol",
			connString: "firebird://jack@localhost:5432/mydb",
			config: &pgconn.Config{
				User:          "jack",
				Host:          "localhost",
				Port:          5432,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url IPv4 with port",
			connString: "firebird://jack@127.0.0.1:5433/mydb",
			config: &pgconn.Config{
				User:          "jack",
				Host:          "127.0.0.1",
				Port:          5433,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url IPv6 with port",
			connString: "firebird://jack@[2001:db8::1]:5433/mydb",
			config: &pgconn.Config{
				User:          "jack",
				Host:          "2001:db8::1",
				Port:          5433,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url IPv6 no port",
			connString: "firebird://jack@[2001:db8::1]/mydb",
			config: &pgconn.Config{
				User:          "jack",
				Host:          "2001:db8::1",
				Port:          defaultPort,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "Key/value everything",
			connString: "user=jack password=secret host=localhost port=5432 dbname=mydb application_name=pgxtest search_path=myschema connect_timeout=5",
			config: &pgconn.Config{
				User:           "jack",
				Password:       "secret",
				Host:           "localhost",
				Port:           5432,
				Database:       "mydb",
				ConnectTimeout: 5 * time.Second,
				RuntimeParams: map[string]string{
					"application_name": "pgxtest",
					"search_path":      "myschema",
				},
			},
		},
		{
			name:       "Key/value with escaped single quote",
			connString: "user=jack\\'s password=secret host=localhost port=5432 dbname=mydb",
			config: &pgconn.Config{
				User:          "jack's",
				Password:      "secret",
				Host:          "localhost",
				Port:          5432,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "Key/value with escaped backslash",
			connString: "user=jack password=sooper\\\\secret host=localhost port=5432 dbname=mydb",
			config: &pgconn.Config{
				User:          "jack",
				Password:      "sooper\\secret",
				Host:          "localhost",
				Port:          5432,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "Key/value with single quoted values",
			connString: "user='jack' host='localhost' dbname='mydb'",
			config: &pgconn.Config{
				User:          "jack",
				Host:          "localhost",
				Port:          defaultPort,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "Key/value with single quoted value with escaped single quote",
			connString: "user='jack\\'s' host='localhost' dbname='mydb'",
			config: &pgconn.Config{
				User:          "jack's",
				Host:          "localhost",
				Port:          defaultPort,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "Key/value with empty single quoted value",
			connString: "user='jack' password='' host='localhost' dbname='mydb'",
			config: &pgconn.Config{
				User:          "jack",
				Host:          "localhost",
				Port:          defaultPort,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "Key/value with space between key and value",
			connString: "user = 'jack' password = '' host = 'localhost' dbname = 'mydb'",
			config: &pgconn.Config{
				User:          "jack",
				Host:          "localhost",
				Port:          defaultPort,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		// https://github.com/jackc/pgconn/issues/72
		{
			name:       "URL without host but with port still uses default host",
			connString: "firebird://jack:secret@:1/mydb",
			config: &pgconn.Config{
				User:          "jack",
				Password:      "secret",
				Host:          defaultHost,
				Port:          1,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url with unix path to database",
			connString: "firebird://user:secret@localhost:3050//db/mydb.fdb",
			config: &pgconn.Config{
				User:           "user",
				Password:       "secret",
				Host:           "localhost",
				Port:           3050,
				Database:       "/db/mydb.fdb",
				ConnectTimeout: 5 * time.Second,
				RuntimeParams: map[string]string{
					"application_name": "pgxtest",
				},
			},
		},
		{
			name:       "database url with windows path to database",
			connString: "firebird://user:secret@localhost:3050/c:\\db\\mydb.fdb",
			config: &pgconn.Config{
				User:           "user",
				Password:       "secret",
				Host:           "localhost",
				Port:           3050,
				Database:       "c:\\db\\mydb.fdb",
				ConnectTimeout: 5 * time.Second,
				RuntimeParams: map[string]string{
					"application_name": "pgxtest",
				},
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := pgconn.ParseConfig(tt.connString)
			assert.Nilf(t, err, "Test %d (%s)", i, tt.name)

			assertConfigsEqual(t, tt.config, config, fmt.Sprintf("Test %d (%s)", i, tt.name))
		})
	}
}

// https://github.com/jackc/pgconn/issues/47
func TestParseConfigKVWithTrailingEmptyEqualDoesNotPanic(t *testing.T) {
	_, err := pgconn.ParseConfig("host= user= password= port= database=")
	require.NoError(t, err)
}

func TestParseConfigKVLeadingEqual(t *testing.T) {
	_, err := pgconn.ParseConfig("= user=jack")
	require.Error(t, err)
}

// https://github.com/jackc/pgconn/issues/49
func TestParseConfigKVTrailingBackslash(t *testing.T) {
	_, err := pgconn.ParseConfig(`x=x\`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid backslash")
}

func TestConfigCopyReturnsEqualConfig(t *testing.T) {
	connString := "postgres://jack:secret@localhost:5432/mydb?application_name=pgxtest&search_path=myschema&connect_timeout=5"
	original, err := pgconn.ParseConfig(connString)
	require.NoError(t, err)

	copied := original.Copy()
	assertConfigsEqual(t, original, copied, "Test Config.Copy() returns equal config")
}

func TestConfigCopyOriginalConfigDidNotChange(t *testing.T) {
	connString := "postgres://jack:secret@localhost:5432/mydb?application_name=pgxtest&search_path=myschema&connect_timeout=5&sslmode=prefer"
	original, err := pgconn.ParseConfig(connString)
	require.NoError(t, err)

	copied := original.Copy()
	assertConfigsEqual(t, original, copied, "Test Config.Copy() returns equal config")

	copied.Port = uint16(5433)
	copied.RuntimeParams["foo"] = "bar"

	assert.Equal(t, uint16(3050), original.Port)
	assert.Equal(t, "", original.RuntimeParams["foo"])
}

func TestConfigCopyCanBeUsedToConnect(t *testing.T) {
	connString := os.Getenv("PGX_TEST_DATABASE")
	original, err := pgconn.ParseConfig(connString)
	require.NoError(t, err)

	copied := original.Copy()
	assert.NotPanics(t, func() {
		_, err = pgconn.ConnectConfig(context.Background(), copied)
	})
	assert.NoError(t, err)
}

func TestNetworkAddress(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantNet string
	}{
		{
			name:    "Assume TCP for unknown formats",
			host:    "a/tmp",
			wantNet: "tcp",
		},
		{
			name:    "loopback interface",
			host:    "localhost",
			wantNet: "tcp",
		},
		{
			name:    "IP address",
			host:    "127.0.0.1",
			wantNet: "tcp",
		},
	}
	for i, tt := range tests {
		gotNet, _ := pgconn.NetworkAddress(tt.host, 5432)

		assert.Equalf(t, tt.wantNet, gotNet, "Test %d (%s)", i, tt.name)
	}
}

func assertConfigsEqual(t *testing.T, expected, actual *pgconn.Config, testName string) {
	if !assert.NotNil(t, expected) {
		return
	}
	if !assert.NotNil(t, actual) {
		return
	}

	assert.Equalf(t, expected.Host, actual.Host, "%s - Host", testName)
	assert.Equalf(t, expected.Database, actual.Database, "%s - Database", testName)
	assert.Equalf(t, expected.Port, actual.Port, "%s - Port", testName)
	assert.Equalf(t, expected.User, actual.User, "%s - User", testName)
	assert.Equalf(t, expected.Password, actual.Password, "%s - Password", testName)
	assert.Equalf(t, expected.ConnectTimeout, actual.ConnectTimeout, "%s - ConnectTimeout", testName)
	assert.Equalf(t, expected.RuntimeParams, actual.RuntimeParams, "%s - RuntimeParams", testName)

	// Can't test function equality, so just test that they are set or not.
	assert.Equalf(t, expected.ValidateConnect == nil, actual.ValidateConnect == nil, "%s - ValidateConnect", testName)
	assert.Equalf(t, expected.AfterConnect == nil, actual.AfterConnect == nil, "%s - AfterConnect", testName)
}

func TestParseConfigEnvLibpq(t *testing.T) {
	var osUserName string
	osUser, err := user.Current()
	if err == nil {
		// Windows gives us the username here as `DOMAIN\user` or `LOCALPCNAME\user`,
		// but the libpq default is just the `user` portion, so we strip off the first part.
		if runtime.GOOS == "windows" && strings.Contains(osUser.Username, "\\") {
			osUserName = osUser.Username[strings.LastIndex(osUser.Username, "\\")+1:]
		} else {
			osUserName = osUser.Username
		}
	}

	pgEnvvars := []string{"PGHOST", "PGPORT", "PGDATABASE", "PGUSER", "PGPASSWORD", "PGAPPNAME", "PGSSLMODE", "PGCONNECT_TIMEOUT", "PGSSLSNI", "PGTZ", "PGOPTIONS"}

	tests := []struct {
		name    string
		envvars map[string]string
		config  *pgconn.Config
	}{
		{
			// not testing no environment at all as that would use default host and that can vary.
			name:    "PGHOST only",
			envvars: map[string]string{"PGHOST": "123.123.123.123"},
			config: &pgconn.Config{
				User: osUserName,
				Host: "123.123.123.123",
				Port: 3050,

				RuntimeParams: map[string]string{},
				Fallbacks: []*pgconn.FallbackConfig{
					{
						Host: "123.123.123.123",
						Port: 3050,
					},
				},
			},
		},
		{
			name: "All non-TLS environment",
			envvars: map[string]string{
				"PGHOST":            "123.123.123.123",
				"PGPORT":            "7777",
				"PGDATABASE":        "foo",
				"PGUSER":            "bar",
				"PGPASSWORD":        "baz",
				"PGCONNECT_TIMEOUT": "10",
				"PGSSLMODE":         "disable",
				"PGAPPNAME":         "pgxtest",
				"PGTZ":              "America/New_York",
				"PGOPTIONS":         "-c search_path=myschema",
			},
			config: &pgconn.Config{
				Host:           "123.123.123.123",
				Port:           7777,
				Database:       "foo",
				User:           "bar",
				Password:       "baz",
				ConnectTimeout: 10 * time.Second,
				RuntimeParams:  map[string]string{"application_name": "pgxtest", "timezone": "America/New_York", "options": "-c search_path=myschema"},
			},
		},
		{
			name: "SNI can be disabled via environment variable",
			envvars: map[string]string{
				"PGHOST":    "test.foo",
				"PGSSLMODE": "require",
				"PGSSLSNI":  "0",
			},
			config: &pgconn.Config{
				User:          osUserName,
				Host:          "test.foo",
				Port:          3050,
				RuntimeParams: map[string]string{},
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, env := range pgEnvvars {
				t.Setenv(env, tt.envvars[env])
			}

			config, err := pgconn.ParseConfig("")
			assert.Nilf(t, err, "Test %d (%s)", i, tt.name)

			assertConfigsEqual(t, tt.config, config, fmt.Sprintf("Test %d (%s)", i, tt.name))
		})
	}
}
