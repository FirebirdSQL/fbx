package fbconn_test

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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto/auth"
)

func getDefaultPort(t *testing.T) uint16 {
	if envFBPORT := os.Getenv("FBPORT"); envFBPORT != "" {
		p, err := strconv.ParseUint(envFBPORT, 10, 16)
		require.NoError(t, err)
		return uint16(p)
	}
	return 3050
}

func getDefaultUser(t *testing.T) string {
	if fbuser := os.Getenv("FBUSER"); fbuser != "" {
		return fbuser
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
	t.Parallel()

	config, err := fbconn.ParseConfig("")
	require.NoError(t, err)
	defaultHost := config.Host

	defaultUser := getDefaultUser(t)
	defaultPort := getDefaultPort(t)

	tests := []struct {
		name       string
		connString string
		config     *fbconn.Config
	}{
		{
			name:       "simple url",
			connString: "firebird://ivan:secret@localhost:3050/mydb",
			config: &fbconn.Config{
				User:          "ivan",
				Password:      "secret",
				Host:          "localhost",
				Port:          3050,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url everything",
			connString: "firebird://ivan:secret@localhost:3050/mydb?application_name=fbxtest&search_path=myschema&connect_timeout=5",
			config: &fbconn.Config{
				User:           "ivan",
				Password:       "secret",
				Host:           "localhost",
				Port:           3050,
				Database:       "mydb",
				ConnectTimeout: 5 * time.Second,
				RuntimeParams: map[string]string{
					"application_name": "fbxtest",
					"search_path":      "myschema",
				},
			},
		},
		{
			name:       "database url fetch batch size",
			connString: "firebird://ivan:secret@localhost:3050/mydb?fetch_batch_size=123",
			config: &fbconn.Config{
				User:           "ivan",
				Password:       "secret",
				Host:           "localhost",
				Port:           3050,
				Database:       "mydb",
				FetchBatchSize: 123,
				RuntimeParams:  map[string]string{},
			},
		},
		{
			name:       "database url create database",
			connString: "firebird://ivan:secret@localhost:3050/mydb?create_database=true",
			config: &fbconn.Config{
				User:           "ivan",
				Password:       "secret",
				Host:           "localhost",
				Port:           3050,
				Database:       "mydb",
				CreateDatabase: true,
				RuntimeParams:  map[string]string{},
			},
		},
		{
			name:       "database url missing password",
			connString: "firebird://ivan@localhost:3050/mydb",
			config: &fbconn.Config{
				User:          "ivan",
				Host:          "localhost",
				Port:          3050,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url custom auth plugin list",
			connString: "firebird://ivan:secret@localhost:3050/mydb?auth_plugin_list=Srp,Legacy_Auth",
			config: &fbconn.Config{
				User:           "ivan",
				Password:       "secret",
				Host:           "localhost",
				Port:           3050,
				Database:       "mydb",
				RuntimeParams:  map[string]string{},
				AuthPluginList: []string{"Srp", "Legacy_Auth"},
			},
		},
		{
			name:       "database url missing user and password",
			connString: "firebird://localhost:3050/mydb",
			config: &fbconn.Config{
				SystemUser:    defaultUser,
				Host:          "localhost",
				Port:          3050,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url missing port",
			connString: "firebird://ivan:secret@localhost:3050/mydb",
			config: &fbconn.Config{
				User:          "ivan",
				Password:      "secret",
				Host:          "localhost",
				Port:          3050,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url unix domain socket host",
			connString: "firebird:///foo?host=/tmp",
			config: &fbconn.Config{
				SystemUser:    defaultUser,
				Host:          "/tmp",
				Port:          defaultPort,
				Database:      "foo",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url unix domain socket host on windows",
			connString: "firebird:///foo?host=C:\\tmp",
			config: &fbconn.Config{
				SystemUser:    defaultUser,
				Host:          "C:\\tmp",
				Port:          defaultPort,
				Database:      "foo",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url dbname",
			connString: "firebird://localhost/?dbname=foo",
			config: &fbconn.Config{
				SystemUser:    defaultUser,
				Host:          "localhost",
				Port:          defaultPort,
				Database:      "foo",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url firebird protocol",
			connString: "firebird://ivan@localhost:3050/mydb",
			config: &fbconn.Config{
				User:          "ivan",
				Host:          "localhost",
				Port:          3050,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url IPv4 with port",
			connString: "firebird://ivan@127.0.0.1:3051/mydb",
			config: &fbconn.Config{
				User:          "ivan",
				Host:          "127.0.0.1",
				Port:          3051,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url IPv6 with port",
			connString: "firebird://ivan@[2001:db8::1]:3051/mydb",
			config: &fbconn.Config{
				User:          "ivan",
				Host:          "2001:db8::1",
				Port:          3051,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url IPv6 no port",
			connString: "firebird://ivan@[2001:db8::1]/mydb",
			config: &fbconn.Config{
				User:          "ivan",
				Host:          "2001:db8::1",
				Port:          defaultPort,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "Key/value everything",
			connString: "user=ivan password=secret host=localhost port=3050 dbname=mydb application_name=fbxtest search_path=myschema connect_timeout=5",
			config: &fbconn.Config{
				User:           "ivan",
				Password:       "secret",
				Host:           "localhost",
				Port:           3050,
				Database:       "mydb",
				ConnectTimeout: 5 * time.Second,
				RuntimeParams: map[string]string{
					"application_name": "fbxtest",
					"search_path":      "myschema",
				},
			},
		},
		{
			name:       "Key/value fetch batch size",
			connString: "user=ivan password=secret host=localhost port=3050 dbname=mydb fetch_batch_size=77",
			config: &fbconn.Config{
				User:           "ivan",
				Password:       "secret",
				Host:           "localhost",
				Port:           3050,
				Database:       "mydb",
				FetchBatchSize: 77,
				RuntimeParams:  map[string]string{},
			},
		},
		{
			name:       "Key/value create database",
			connString: "user=ivan password=secret host=localhost port=3050 dbname=mydb create_database=true",
			config: &fbconn.Config{
				User:           "ivan",
				Password:       "secret",
				Host:           "localhost",
				Port:           3050,
				Database:       "mydb",
				CreateDatabase: true,
				RuntimeParams:  map[string]string{},
			},
		},
		{
			name:       "Key/value with escaped single quote",
			connString: "user=ivan\\'s password=secret host=localhost port=3050 dbname=mydb",
			config: &fbconn.Config{
				User:          "ivan's",
				Password:      "secret",
				Host:          "localhost",
				Port:          3050,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "Key/value with escaped backslash",
			connString: "user=ivan password=sooper\\\\secret host=localhost port=3050 dbname=mydb",
			config: &fbconn.Config{
				User:          "ivan",
				Password:      "sooper\\secret",
				Host:          "localhost",
				Port:          3050,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "Key/value with single quoted values",
			connString: "user='ivan' host='localhost' dbname='mydb'",
			config: &fbconn.Config{
				User:          "ivan",
				Host:          "localhost",
				Port:          defaultPort,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "Key/value with single quoted value with escaped single quote",
			connString: "user='ivan\\'s' host='localhost' dbname='mydb'",
			config: &fbconn.Config{
				User:          "ivan's",
				Host:          "localhost",
				Port:          defaultPort,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "Key/value with empty single quoted value",
			connString: "user='ivan' password='' host='localhost' dbname='mydb'",
			config: &fbconn.Config{
				User:          "ivan",
				Host:          "localhost",
				Port:          defaultPort,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "Key/value with space between key and value",
			connString: "user = 'ivan' password = '' host = 'localhost' dbname = 'mydb'",
			config: &fbconn.Config{
				User:          "ivan",
				Host:          "localhost",
				Port:          defaultPort,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "Key/value multiple hosts one port",
			connString: "user=sysdba password=secret host=foo,bar,baz port=3050 dbname=mydb",
			config: &fbconn.Config{
				User:          "sysdba",
				Password:      "secret",
				Host:          "foo",
				Port:          3050,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
				Fallbacks: []*fbconn.FallbackConfig{
					{
						Host: "bar",
						Port: 3050,
					},
					{
						Host: "baz",
						Port: 3050,
					},
				},
			},
		},
		{
			name:       "Key/value multiple hosts multiple ports",
			connString: "user=sysdba password=secret host=foo,bar,baz port=3050,3051,3052 dbname=mydb",
			config: &fbconn.Config{
				User:          "sysdba",
				Password:      "secret",
				Host:          "foo",
				Port:          3050,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
				Fallbacks: []*fbconn.FallbackConfig{
					{
						Host: "bar",
						Port: 3051,
					},
					{
						Host: "baz",
						Port: 3052,
					},
				},
			},
		},
		{
			name:       "URL without host but with port still uses default host",
			connString: "firebird://ivan:secret@:1/mydb",
			config: &fbconn.Config{
				User:          "ivan",
				Password:      "secret",
				Host:          defaultHost,
				Port:          1,
				Database:      "mydb",
				RuntimeParams: map[string]string{},
			},
		},
		{
			name:       "database url with unix path to database",
			connString: "firebird://user:secret@localhost:3050//db/mydb.fdb?connect_timeout=5&application_name=fbxtest",
			config: &fbconn.Config{
				User:           "user",
				Password:       "secret",
				Host:           "localhost",
				Port:           3050,
				Database:       "/db/mydb.fdb",
				ConnectTimeout: 5 * time.Second,
				RuntimeParams: map[string]string{
					"application_name": "fbxtest",
				},
			},
		},
		{
			name:       "database url with windows path to database",
			connString: "firebird://user:secret@localhost:3050/c:\\db\\mydb.fdb?connect_timeout=5&application_name=fbxtest",
			config: &fbconn.Config{
				User:           "user",
				Password:       "secret",
				Host:           "localhost",
				Port:           3050,
				Database:       "c:\\db\\mydb.fdb",
				ConnectTimeout: 5 * time.Second,
				RuntimeParams: map[string]string{
					"application_name": "fbxtest",
				},
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := fbconn.ParseConfig(tt.connString)
			assert.Nilf(t, err, "Test %d (%s)", i, tt.name)

			assertConfigsEqual(t, tt.config, config, fmt.Sprintf("Test %d (%s)", i, tt.name))
		})
	}
}

func TestParseConfigKVWithTrailingEmptyEqualDoesNotPanic(t *testing.T) {
	_, err := fbconn.ParseConfig("host= user= password= port= database=")
	require.NoError(t, err)
}

func TestParseConfigWireCompression(t *testing.T) {
	tests := []string{
		"firebird://ivan:secret@localhost:3050/mydb?wire_compression=true",
		"firebird://ivan:secret@localhost:3050/mydb?wire_compress=true",
		"firebird://ivan:secret@localhost:3050/mydb?wireCompression=true",
		"user=ivan password=secret host=localhost port=3050 database=mydb wire_compression=true",
		"user=ivan password=secret host=localhost port=3050 database=mydb wire_compress=true",
	}

	for _, connString := range tests {
		config, err := fbconn.ParseConfig(connString)
		require.NoError(t, err)
		require.True(t, config.WireCompression)
		require.NotContains(t, config.RuntimeParams, "wire_compression")
		require.NotContains(t, config.RuntimeParams, "wire_compress")
	}
}

func TestParseConfigKVTrailingWhitespace(t *testing.T) {
	config, err := fbconn.ParseConfig("host=localhost port=3050 database=testdb   \t\n")
	require.NoError(t, err)
	require.Equal(t, "localhost", config.Host)
	require.Equal(t, uint16(3050), config.Port)
	require.Equal(t, "testdb", config.Database)
}

func TestParseConfigKVLeadingEqual(t *testing.T) {
	_, err := fbconn.ParseConfig("= user=ivan")
	require.Error(t, err)
}

func TestParseConfigKVTrailingBackslash(t *testing.T) {
	_, err := fbconn.ParseConfig(`x=x\`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid backslash")
}

func TestParseConfigInvalidFetchBatchSize(t *testing.T) {
	t.Parallel()

	_, err := fbconn.ParseConfig("fetch_batch_size=0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid fetch_batch_size")
}

func TestParseConfigConnStringAllowedKeys(t *testing.T) {
	config, err := fbconn.ParseConfigWithOptions(
		"host=localhost dbname=testdb createdb=true",
		fbconn.ParseConfigOptions{ConnStringAllowedKeys: []string{"host", "database", "create_database"}},
	)
	require.NoError(t, err)
	require.Equal(t, "localhost", config.Host)
	require.Equal(t, "testdb", config.Database)
	require.True(t, config.CreateDatabase)

	_, err = fbconn.ParseConfigWithOptions(
		"host=localhost password=secret",
		fbconn.ParseConfigOptions{ConnStringAllowedKeys: []string{"host"}},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), `connection string key "password" is not in ConnStringAllowedKeys`)
}

func TestParseConfigConnStringAllowedKeysURLAliases(t *testing.T) {
	config, err := fbconn.ParseConfigWithOptions(
		"firebird://localhost/?dbname=testdb&createdb=true",
		fbconn.ParseConfigOptions{ConnStringAllowedKeys: []string{"host", "dbname", "createdb"}},
	)
	require.NoError(t, err)
	require.Equal(t, "testdb", config.Database)
	require.True(t, config.CreateDatabase)
}

func TestConfigCopyReturnsEqualConfig(t *testing.T) {
	connString := "firebird://ivan:secret@localhost:3050/mydb?application_name=fbxtest&connect_timeout=5"
	original, err := fbconn.ParseConfig(connString)
	require.NoError(t, err)

	copied := original.Copy()
	assertConfigsEqual(t, original, copied, "Test Config.Copy() returns equal config")
}

func TestConfigCopyOriginalConfigDidNotChange(t *testing.T) {
	connString := "firebird://ivan:secret@localhost:3050/mydb?application_name=fbxtest&connect_timeout=5"
	original, err := fbconn.ParseConfig(connString)
	require.NoError(t, err)

	copied := original.Copy()
	assertConfigsEqual(t, original, copied, "Test Config.Copy() returns equal config")

	copied.Port = uint16(3051)
	copied.RuntimeParams["foo"] = "bar"

	assert.Equal(t, uint16(3050), original.Port)
	assert.Equal(t, "", original.RuntimeParams["foo"])
}

func TestConfigCopyCanBeUsedToConnect(t *testing.T) {
	connString := os.Getenv("FBX_TEST_DATABASE")
	original, err := fbconn.ParseConfig(connString)
	require.NoError(t, err)

	copied := original.Copy()
	assert.NotPanics(t, func() {
		_, err = fbconn.ConnectConfig(context.Background(), copied)
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
		gotNet, _ := fbconn.NetworkAddress(tt.host, 3050)

		assert.Equalf(t, tt.wantNet, gotNet, "Test %d (%s)", i, tt.name)
	}
}

func assertConfigsEqual(t *testing.T, expected, actual *fbconn.Config, testName string) {
	if !assert.NotNil(t, expected) {
		return
	}
	if !assert.NotNil(t, actual) {
		return
	}

	assert.Equalf(t, expected.Host, actual.Host, "%s - Host", testName)
	assert.Equalf(t, expected.Database, actual.Database, "%s - Database", testName)
	assert.Equalf(t, expected.CreateDatabase, actual.CreateDatabase, "%s - CreateDatabase", testName)
	assert.Equalf(t, expected.Port, actual.Port, "%s - Port", testName)
	assert.Equalf(t, expected.User, actual.User, "%s - User", testName)
	assert.Equalf(t, expected.Password, actual.Password, "%s - Password", testName)
	assert.Equalf(t, expected.ConnectTimeout, actual.ConnectTimeout, "%s - ConnectTimeout", testName)
	assert.Equalf(t, expected.WireCompression, actual.WireCompression, "%s - WireCompression", testName)
	assert.Equalf(t, expected.RuntimeParams, actual.RuntimeParams, "%s - RuntimeParams", testName)
	expectedFetchBatchSize := expected.FetchBatchSize
	if expectedFetchBatchSize == 0 {
		expectedFetchBatchSize = fbconn.DefaultFetchBatchSize
	}
	assert.Equalf(t, expectedFetchBatchSize, actual.FetchBatchSize, "%s - FetchBatchSize", testName)
	expectedPlugins := expected.AuthPluginList
	if len(expectedPlugins) == 0 {
		expectedPlugins = auth.DefaultPluginSequence()
	}
	assert.Equalf(t, expectedPlugins, actual.AuthPluginList, "%s - AuthPluginList", testName)

	// Can't test function equality, so just test that they are set or not.
	assert.Equalf(t, expected.ValidateConnect == nil, actual.ValidateConnect == nil, "%s - ValidateConnect", testName)
	assert.Equalf(t, expected.AfterConnect == nil, actual.AfterConnect == nil, "%s - AfterConnect", testName)

	if assert.Equalf(t, len(expected.Fallbacks), len(actual.Fallbacks), "%s - Fallbacks", testName) {
		for i := range expected.Fallbacks {
			assert.Equalf(t, expected.Fallbacks[i].Host, actual.Fallbacks[i].Host, "%s - Fallback %d - Host", testName, i)
			assert.Equalf(t, expected.Fallbacks[i].Port, actual.Fallbacks[i].Port, "%s - Fallback %d - Port", testName, i)
		}
	}
}

func TestParseConfigEnv(t *testing.T) {
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

	fbEnvvars := []string{"FBHOST", "FBPORT", "FBDATABASE", "FBUSER", "FBPASSWORD", "FBAPPNAME", "FBCONNECT_TIMEOUT", "FBTZ", "FBFETCH_BATCH_SIZE", "FBWIRECOMPRESSION"}

	tests := []struct {
		name    string
		envvars map[string]string
		config  *fbconn.Config
	}{
		{
			// not testing no environment at all as that would use default host and that can vary.
			name:    "FBHOST only",
			envvars: map[string]string{"FBHOST": "123.123.123.123"},
			config: &fbconn.Config{
				SystemUser: osUserName,
				Host:       "123.123.123.123",
				Port:       3050,

				RuntimeParams: map[string]string{},
			},
		},
		{
			name: "All environment",
			envvars: map[string]string{
				"FBHOST":             "123.123.123.123",
				"FBPORT":             "7777",
				"FBDATABASE":         "foo",
				"FBUSER":             "bar",
				"FBPASSWORD":         "baz",
				"FBCONNECT_TIMEOUT":  "10",
				"FBAPPNAME":          "fbxtest",
				"FBFETCH_BATCH_SIZE": "64",
				"FBWIRECOMPRESSION":  "true",
				"FBTZ":               "America/New_York",
			},
			config: &fbconn.Config{
				Host:            "123.123.123.123",
				Port:            7777,
				Database:        "foo",
				User:            "bar",
				Password:        "baz",
				ConnectTimeout:  10 * time.Second,
				FetchBatchSize:  64,
				WireCompression: true,
				RuntimeParams:   map[string]string{"application_name": "fbxtest", "timezone": "America/New_York"},
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, env := range fbEnvvars {
				t.Setenv(env, tt.envvars[env])
			}

			config, err := fbconn.ParseConfig("")
			assert.Nilf(t, err, "Test %d (%s)", i, tt.name)

			assertConfigsEqual(t, tt.config, config, fmt.Sprintf("Test %d (%s)", i, tt.name))
		})
	}
}
