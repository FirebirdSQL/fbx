package fbconn

import (
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const defaultPortValue uint16 = 3050

func TestParseConfigURIHostPorts(t *testing.T) {
	t.Setenv("FBPORT", "3060")
	tests := []struct {
		list  string
		query string
		hosts []string
		ports []uint16
	}{
		{
			list:  "a,b,c",
			query: "",
			hosts: []string{"a", "b", "c"},
			ports: []uint16{defaultPortValue, defaultPortValue, defaultPortValue},
		},
		{
			list:  "a:3051,b:3052,c:3053",
			query: "",
			hosts: []string{"a", "b", "c"},
			ports: []uint16{3051, 3052, 3053},
		},
		{
			list:  "a,b:3051,c",
			query: "",
			hosts: []string{"a", "b", "c"},
			ports: []uint16{defaultPortValue, 3051, defaultPortValue},
		},
		{
			list:  "a:,b:3051,c:",
			query: "",
			hosts: []string{"a", "b", "c"},
			ports: []uint16{defaultPortValue, 3051, defaultPortValue},
		},
		{
			list:  "127.0.0.1,[::1]:3051,[2001:db8::1]",
			query: "",
			hosts: []string{"127.0.0.1", "::1", "2001:db8::1"},
			ports: []uint16{defaultPortValue, 3051, defaultPortValue},
		},
		{
			list:  "a:1,b,c:3",
			query: "?port=3051",
			hosts: []string{"a", "b", "c"},
			ports: []uint16{3051, 3051, 3051},
		},
		{
			list:  "a:1,b,c:3",
			query: "?port=,3051,",
			hosts: []string{"a", "b", "c"},
			ports: []uint16{defaultPortValue, 3051, defaultPortValue},
		},
		{
			list:  "a,b,c",
			query: "?port=",
			hosts: []string{"a", "b", "c"},
			ports: []uint16{defaultPortValue, defaultPortValue, defaultPortValue},
		},
		{
			list:  "a,,b",
			hosts: []string{"a", "localhost", "b"},
			ports: []uint16{defaultPortValue, defaultPortValue, defaultPortValue},
		},
		{
			list:  ":3051,b",
			hosts: []string{"localhost", "b"},
			ports: []uint16{3051, defaultPortValue},
		},
		{
			list:  "localhost",
			hosts: []string{"localhost"},
			ports: []uint16{3060},
		},
	}

	for _, scheme := range []string{"firebird", "firebirdsql"} {
		for _, tt := range tests {
			t.Run(scheme+"/"+tt.list+tt.query, func(t *testing.T) {
				config, err := ParseConfig(scheme + "://user:secret@" + tt.list + `/C:\db\mydb.fdb` + tt.query)

				require.NoError(t, err)
				require.Equal(t, "user", config.User)
				require.Equal(t, "secret", config.Password)
				require.Equal(t, `C:\db\mydb.fdb`, config.Database)
				require.Empty(t, config.RuntimeParams)
				hosts := []string{config.Host}
				ports := []uint16{config.Port}
				for _, fallback := range config.Fallbacks {
					hosts = append(hosts, fallback.Host)
					ports = append(ports, fallback.Port)
				}
				require.Equal(t, tt.hosts, hosts)
				require.Equal(t, tt.ports, ports)
			})
		}
	}

}

func TestParseConfigKeywordPortLists(t *testing.T) {
	tests := []struct {
		name       string
		connString string
		ports      []uint16
	}{
		{
			name:       "single port applies to all hosts",
			connString: "host=a,b,c port=3051",
			ports:      []uint16{3051, 3051, 3051},
		},
		{
			name:       "empty entries use default port",
			connString: "host=a,b,c port=,3051,",
			ports:      []uint16{defaultPortValue, 3051, defaultPortValue},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseConfig(tt.connString)
			require.NoError(t, err)
			ports := []uint16{config.Port}
			hosts := []string{config.Host}
			for _, fallback := range config.Fallbacks {
				ports = append(ports, fallback.Port)
				hosts = append(hosts, fallback.Host)
			}
			require.Equal(t, []string{"a", "b", "c"}, hosts)
			require.Equal(t, tt.ports, ports)
		})
	}
}

func TestParseConfigURIPortListOverridesEnvironment(t *testing.T) {
	t.Setenv("FBPORT", "1,2")
	config, err := ParseConfig("firebird://a,b,c/db")
	require.NoError(t, err)
	require.EqualValues(t, defaultPortValue, config.Port)
	require.Len(t, config.Fallbacks, 2)
	for _, fallback := range config.Fallbacks {
		require.EqualValues(t, defaultPortValue, fallback.Port)
	}
}

func TestParseConfigInvalidPortLists(t *testing.T) {
	for _, connString := range []string{
		"host=a,b,c port=1,2",
		"host=a port=1,2",
		"host=a,b port=1,2,3",
		"host=a,b port=1,0",
		"host=a,b port=1,65536",
		"firebird://a,b/db?port=1,2,3",
		"firebirdsql://a,b,c/db?port=1,2",
		"firebird://a,b:65536/db",
	} {
		t.Run(connString, func(t *testing.T) {
			config, err := ParseConfig(connString)

			require.Error(t, err)
			require.Nil(t, config)
			var parseConfigErr *ParseConfigError
			require.ErrorAs(t, err, &parseConfigErr)
			if strings.Contains(connString, "port=1,2") {
				require.ErrorContains(t, err, "could not match")
			}
		})
	}
}

func TestParseConfigURIQueryValidation(t *testing.T) {
	for _, query := range []string{
		"broken",
		"user=ok&broken",
		"broken&user=ok",
		"x=a=b",
		"=value&broken",
		"&x=ok",
		"x=ok&&y=ok",
		"x=%",
		"x=%1",
		"x=%GG",
		"%GG=x",
		"x=%00",
		"%00=x",
		"dbname=%GG&database=good",
		"database=good&dbname=%GG",
		"password=%GG&password=good",
		"user=alice&application_name=worker%2&x=ok",
		"application%1_name=worker",
	} {
		for _, scheme := range []string{"firebird", "firebirdsql"} {
			t.Run(scheme+"/"+query, func(t *testing.T) {
				config, err := ParseConfig(scheme + "://localhost/db?" + query)

				require.Error(t, err)
				require.Nil(t, config)
				var parseConfigErr *ParseConfigError
				require.ErrorAs(t, err, &parseConfigErr)
				require.ErrorContains(t, err, "failed to parse as URL")
			})
		}
	}
}

func TestParseConfigURIQueryValues(t *testing.T) {
	for _, scheme := range []string{"firebird", "firebirdsql"} {
		tests := []struct {
			query    string
			expected string
		}{
			{
				query:    "database=first&dbname=last",
				expected: "last",
			},
			{
				query:    "dbname=first&database=last",
				expected: "last",
			},
			{
				query:    "database=first&db%6Eame=last",
				expected: "last",
			},
			{
				query:    "dbname=first&data%62ase=last",
				expected: "last",
			},
			{
				query:    "dbname=first&database=",
				expected: "",
			},
			{
				query:    "database=&dbname=last",
				expected: "last",
			},
			{
				query:    "dbname=a%26b%3Dc%3Fd",
				expected: "a&b=c?d",
			},
			{
				query:    "dbname=%2520",
				expected: "%20",
			},
			{
				query:    "database=first&dbname=second&database=third",
				expected: "third",
			},
			{
				query:    "dbname=first&database=second&dbname=third",
				expected: "third",
			},
		}

		for _, tt := range tests {
			t.Run(scheme+"/"+tt.query, func(t *testing.T) {
				for i := 0; i < 32; i++ {
					config, err := ParseConfig(scheme + "://user:secret@localhost/path?" + tt.query)

					require.NoError(t, err)
					require.Equal(t, tt.expected, config.Database)
					require.NotContains(t, config.RuntimeParams, "dbname")
					require.NotContains(t, config.RuntimeParams, "database")
				}
			})
		}
		t.Run(scheme+"/overrides", func(t *testing.T) {
			config, err := ParseConfig(scheme + "://old:old@original/path?host=final&port=3100&port=3200&user=first&user=new&password=old&pass%77ord=new")

			require.NoError(t, err)
			require.Equal(t, "final", config.Host)
			require.Equal(t, uint16(3200), config.Port)
			require.Equal(t, "new", config.User)
			require.Equal(t, "new", config.Password)
			require.Empty(t, config.RuntimeParams)
		})
	}
}

func TestParseConfigURIRepeatedParameters(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected map[string]string
	}{
		{
			name:     "three occurrences",
			query:    "x=first&x=second&x=third",
			expected: map[string]string{"x": "third"},
		},
		{
			name:     "interleaved parameters",
			query:    "x=first&y=other&x=last",
			expected: map[string]string{"x": "last", "y": "other"},
		},
		{
			name:     "last value empty",
			query:    "x=first&x=",
			expected: map[string]string{"x": ""},
		},
		{
			name:     "first value empty",
			query:    "x=&x=last",
			expected: map[string]string{"x": "last"},
		},
		{
			name:     "encoded last value",
			query:    "x=first&x=last%2Bvalue",
			expected: map[string]string{"x": "last+value"},
		},
		{
			name:     "encoded first key",
			query:    "%78=first&x=last",
			expected: map[string]string{"x": "last"},
		},
		{
			name:     "encoded last key",
			query:    "x=first&%78=last",
			expected: map[string]string{"x": "last"},
		},
	}
	for _, scheme := range []string{"firebird", "firebirdsql"} {
		for _, tt := range tests {
			t.Run(scheme+"/"+tt.name, func(t *testing.T) {
				config, err := ParseConfig(scheme + "://localhost/db?" + tt.query)
				require.NoError(t, err)
				require.Equal(t, tt.expected, config.RuntimeParams)
			})
		}
	}
}

func TestParseConfigKeywordValueCompatibility(t *testing.T) {
	tests := []struct {
		connString string
		expected   string
	}{
		{
			connString: `database=C:\data\db.fdb`,
			expected:   `C:\data\db.fdb`,
		},
		{
			connString: `database='C:\data\db.fdb'`,
			expected:   `C:\data\db.fdb`,
		},
		{
			connString: `database=a\nb`,
			expected:   `a\nb`,
		},
		{
			connString: `database='a\nb'`,
			expected:   `a\nb`,
		},
		{
			connString: `dbname=first database=last`,
			expected:   `last`,
		},
		{
			connString: `database=first dbname=last`,
			expected:   `last`,
		},
		{
			connString: `database=first dbname=''`,
			expected:   ``,
		},
		{
			connString: `database=a+b`,
			expected:   `a+b`,
		},
		{
			connString: `database='a\\b'`,
			expected:   `a\b`,
		},
		{
			connString: `database='a+b'`,
			expected:   "a+b",
		},
		{
			connString: `database=a\+b`,
			expected:   `a\+b`,
		},
		{
			connString: `database='a\+b'`,
			expected:   `a\+b`,
		},
		{
			connString: `database=a\ b`,
			expected:   `a\ b`,
		},
		{
			connString: `database='a\ b'`,
			expected:   `a\ b`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.connString, func(t *testing.T) {
			config, err := ParseConfig(tt.connString)

			require.NoError(t, err)
			require.Equal(t, tt.expected, config.Database)
		})
	}
}

func TestParseConfigKeywordWhitespace(t *testing.T) {
	tests := []struct {
		connString string
		key        string
		expected   string
	}{
		{
			connString: "us er=user",
			key:        "us er",
			expected:   "user",
		},
		{
			connString: "x = localhost",
			key:        "x",
			expected:   "localhost",
		},
		{
			connString: "x =localhost",
			key:        "x",
			expected:   "localhost",
		},
		{
			connString: "x= localhost",
			key:        "x",
			expected:   "localhost",
		},
	}
	for _, tt := range tests {
		t.Run(tt.connString, func(t *testing.T) {
			config, err := ParseConfig(tt.connString)
			require.NoError(t, err)
			require.Equal(t, tt.expected, config.RuntimeParams[tt.key])
		})
	}
}

func TestParseConfigURIPasswordDiagnostics(t *testing.T) {
	tests := []struct {
		name    string
		tail    string
		secrets []string
	}{
		{
			name:    "userinfo bad port",
			tail:    "u:uniqueSecret@host:invalid/db",
			secrets: []string{"uniqueSecret"},
		},
		{
			name:    "userinfo bad escape",
			tail:    "u:uniqueSecret%GG@host/db",
			secrets: []string{"uniqueSecret"},
		},
		{
			name:    "query malformed escape",
			tail:    "host/db?password=uniqueSecret%GG",
			secrets: []string{"uniqueSecret"},
		},
		{
			name:    "query nul",
			tail:    "host/db?pass%77ord=uniqueSecret%00",
			secrets: []string{"uniqueSecret"},
		},
		{
			name:    "all occurrences",
			tail:    "u:userSecret@host/db?password=firstSecret&pass%77ord=lastSecret&broken",
			secrets: []string{"userSecret", "firstSecret", "lastSecret"},
		},
		{
			name:    "encoded userinfo password and invalid port",
			tail:    "u:unique%53ecret@host:invalid/db",
			secrets: []string{"unique%53ecret", "uniqueSecret"},
		},
		{
			name:    "password before invalid query value",
			tail:    "host/db?password=uniqueSecret&bad=%ZZ",
			secrets: []string{"uniqueSecret"},
		},
		{
			name:    "encoded password key before invalid query value",
			tail:    "host/db?pass%77ord=uniqueSecret&bad=%ZZ",
			secrets: []string{"uniqueSecret"},
		},
		{
			name:    "stranded password in invalid port",
			tail:    "a@u:uniqueSecret@h/db",
			secrets: []string{"uniqueSecret"},
		},
		{
			name:    "query password after at sign and invalid port",
			tail:    "host:bad/db?tag=x@y&password=uriReviewSecret",
			secrets: []string{"uriReviewSecret"},
		},
		{
			name:    "query password after at sign and malformed percent encoding",
			tail:    "host:3050/db?tag=x@y&password=uriReviewSecret&bad=%GG",
			secrets: []string{"uriReviewSecret"},
		},
		{
			name:    "query password after at sign and missing query separator",
			tail:    "host:3050/db?tag=x@y&password=uriReviewSecret&broken",
			secrets: []string{"uriReviewSecret"},
		},
		{
			name:    "stranded password in port decoding error",
			tail:    "a@u:uriReviewSecret%GG@h/db",
			secrets: []string{"uriReviewSecret", "uriReviewSecret%GG@h"},
		},
		{
			name:    "stranded password without database path",
			tail:    "a@u:uriReviewSecret@h",
			secrets: []string{"uriReviewSecret"},
		},
	}

	for _, scheme := range []string{"firebird", "firebirdsql"} {
		for _, tt := range tests {
			t.Run(scheme+"/"+tt.name, func(t *testing.T) {
				connString := scheme + "://" + tt.tail
				config, err := ParseConfig(connString)

				require.Error(t, err)
				require.Nil(t, config)
				var parseConfigErr *ParseConfigError
				require.ErrorAs(t, err, &parseConfigErr)
				require.Equal(t, connString, parseConfigErr.ConnString)
				for diagnostic := err; diagnostic != nil; diagnostic = errors.Unwrap(diagnostic) {
					for _, secret := range tt.secrets {
						assert.NotContains(t, diagnostic.Error(), secret, "diagnostic type %T", diagnostic)
					}
				}
			})
		}
	}
}

func TestParseConfigURIAllowedPorts(t *testing.T) {
	options := ParseConfigOptions{
		ConnStringAllowedKeys: []string{"host", "database"},
	}
	tests := []struct {
		name       string
		connString string
		wantError  bool
	}{
		{
			name:       "firebird implicit ports",
			connString: "firebird://a,b/db",
		},
		{
			name:       "firebirdsql implicit ports",
			connString: "firebirdsql://a,b/db",
		},
		{
			name:       "firebird explicit default port",
			connString: "firebird://a:3050/db",
			wantError:  true,
		},
		{
			name:       "firebirdsql explicit default port",
			connString: "firebirdsql://a:3050/db",
			wantError:  true,
		},
		{
			name:       "keyword value explicit default port",
			connString: "host=a port=3050",
			wantError:  true,
		},
		{
			name:       "explicit nondefault port",
			connString: "firebird://a,b:3051/db",
			wantError:  true,
		},
		{
			name:       "explicit empty query port",
			connString: "firebird://a,b/db?port=",
			wantError:  true,
		},
		{
			name:       "explicit empty keyword port",
			connString: "host=a port=",
			wantError:  true,
		},
		{
			name:       "explicit empty keyword port list",
			connString: "host=a,b port=,,",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseConfigWithOptions(tt.connString, options)

			if tt.wantError {
				require.Error(t, err)
				require.Nil(t, config)
				require.ErrorContains(t, err, `connection string key "port" is not in ConnStringAllowedKeys`)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	}
}

func TestParseConfigErrorURIPasswordRedaction(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "slash and query inside password",
			uri:      "user:pass/word?x=y@host/db",
			expected: "user:xxxxx@host/db",
		},
		{
			name:     "stranded password containing colons",
			uri:      "a@user:sec:ret@host/db",
			expected: "a@user:xxxxx@host/db",
		},
		{
			name:     "query password",
			uri:      "host:3050/db?application_name=app&password=secret",
			expected: "host:3050/db?application_name=app&password=xxxxx",
		},
		{
			name:     "at sign in query before password",
			uri:      "host:3050/db?application_name=me@example.com&password=secret",
			expected: "host:3050/db?application_name=me@example.com&password=xxxxx",
		},
		{
			name:     "userinfo and query passwords",
			uri:      "user:pw@host/db?options=x@y&password=secret",
			expected: "user:xxxxx@host/db?options=x@y&password=xxxxx",
		},
		{
			name:     "at sign in database",
			uri:      "host/db@name?password=secret",
			expected: "host/db@name?password=xxxxx",
		},
		{
			name:     "question mark in IPv6 brackets",
			uri:      "[foo?bar]/db?password=secret",
			expected: "[foo?bar]/db?password=xxxxx",
		},
		{
			name:     "encoded password key",
			uri:      "host/db?pass%77ord=secret",
			expected: "host/db?pass%77ord=xxxxx",
		},
		{
			name:     "spaces in password",
			uri:      "host/db?password=super secret&x=value",
			expected: "host/db?password=xxxxx&x=value",
		},
		{
			name:     "password before malformed query",
			uri:      "host/db?password=secret&broken",
			expected: "host/db?password=xxxxx&broken",
		},
		{
			name:     "repeated passwords and at signs",
			uri:      "u:one@host:3050/db@name?tag=x@y&password=two&pass%77ord=three",
			expected: "u:xxxxx@host:3050/db@name?tag=x@y&password=xxxxx&pass%77ord=xxxxx",
		},
	}
	for _, scheme := range []string{"firebird", "firebirdsql"} {
		for _, tt := range tests {
			t.Run(scheme+"/"+tt.name, func(t *testing.T) {
				connString := scheme + "://" + tt.uri
				err := NewParseConfigError(connString, "diagnostic", nil)
				require.EqualError(t, err, "cannot parse `"+scheme+"://"+tt.expected+"`: diagnostic")
			})
		}
	}
}

func TestParseURLSettingsImpliedEmptyPortList(t *testing.T) {

	tests := []struct {
		connString  string
		wantImplied bool
	}{
		{
			connString:  "firebird://h1,h2/db",
			wantImplied: true,
		},
		{
			connString:  "firebird://h1:,h2/db",
			wantImplied: true,
		},
		{
			connString:  "firebird://h1,h2:3051/db",
			wantImplied: false,
		},
		{
			connString:  "firebird://h/db",
			wantImplied: false,
		},
		{
			connString:  "firebird://h1,h2/db?port=",
			wantImplied: false,
		},
		{
			connString:  "firebird://h1,h2/db?port=3051",
			wantImplied: false,
		},
		{
			connString:  "firebird://h: /db",
			wantImplied: false,
		},
		{
			connString:  "firebird://h1: ,h2/db",
			wantImplied: false,
		},
	}
	for _, tt := range tests {
		_, implied, err := parseURLSettings(tt.connString)
		require.NoError(t, err, tt.connString)
		require.Equal(t, tt.wantImplied, implied, tt.connString)
	}
}

func TestURIDecode(t *testing.T) {

	tests := []struct {
		raw     string
		want    string
		wantErr string
	}{
		{
			raw:  "abc",
			want: "abc",
		},
		{
			raw:  "",
			want: "",
		},
		{
			raw:  "%41",
			want: "A",
		},
		{
			raw:  "%4a%4A",
			want: "JJ",
		},
		{
			raw:  "%c3%a9",
			want: "\xc3\xa9",
		},
		{
			raw:  "a+b",
			want: "a+b",
		},
		{
			raw:  "%2520",
			want: "%20",
		},
		{
			raw:  "  abc  ",
			want: "abc",
		},
		{
			raw:  "   ",
			want: "",
		},
		{
			raw:  "%20",
			want: " ",
		},
		{
			raw:     "a b",
			wantErr: `unexpected spaces found in "a b", use percent-encoded spaces (%20) instead`,
		},
		{
			raw:     "%",
			wantErr: `invalid percent-encoded token: "%"`,
		},
		{
			raw:     "%1",
			wantErr: `invalid percent-encoded token: "%1"`,
		},
		{
			raw:     "%zz",
			wantErr: `invalid percent-encoded token: "%zz"`,
		},
		{
			raw:     "%4 ",
			wantErr: `invalid percent-encoded token: "%4 "`,
		},
		{
			raw:     "a%00b",
			wantErr: `forbidden value %00 in percent-encoded value: "a%00b"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := uriDecode(tt.raw, "")
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseConfigURIMultiSegmentPath(t *testing.T) {
	config, err := ParseConfig("firebird://host/a/b")
	require.NoError(t, err)
	require.Equal(t, "a/b", config.Database)
}

func FuzzParseConfigURIQueryRoundTrip(f *testing.F) {
	for _, seed := range []string{"db", "a+b", `C:\data\db.fdb`, "a&b=c?d#e", "%20", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if strings.ContainsRune(value, 0) {
			t.Skip("URI query decoder rejects NUL")
		}
		encoded := strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
		for _, scheme := range []string{"firebird", "firebirdsql"} {
			config, err := ParseConfig(scheme + "://localhost/path?database=old&db%6Eame=" + encoded + "&password=" + encoded)

			require.NoError(t, err)
			require.Equal(t, value, config.Database)
			require.Equal(t, value, config.Password)
		}
	})
}

func TestParseConfigURICharacterCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		key      string
		expected string
	}{
		{
			name:     "plus in userinfo username",
			uri:      "u+v:p@host/db",
			key:      "user",
			expected: "u+v",
		},
		{
			name:     "plus in userinfo password",
			uri:      "u:p+q@host/db",
			key:      "password",
			expected: "p+q",
		},
		{
			name:     "plus in path",
			uri:      "host/a+b",
			key:      "database",
			expected: "a+b",
		},
		{
			name:     "encoded plus in userinfo",
			uri:      "u:p%2Bq@host/db",
			key:      "password",
			expected: "p+q",
		},
		{
			name:     "encoded plus in path",
			uri:      "host/a%2Bb",
			key:      "database",
			expected: "a+b",
		},
		{
			name:     "plus in query database",
			uri:      "host/db?dbname=a+b",
			key:      "database",
			expected: "a+b",
		},
		{
			name:     "plus in query username",
			uri:      "host/db?user=a+b",
			key:      "user",
			expected: "a+b",
		},
		{
			name:     "plus in query password",
			uri:      "host/db?password=a+b",
			key:      "password",
			expected: "a+b",
		},
		{
			name:     "plus in query host",
			uri:      "host/db?host=a+b",
			key:      "host",
			expected: "a+b",
		},
		{
			name:     "plus in query key and value",
			uri:      "host/db?x+y=a+b",
			key:      "x+y",
			expected: "a+b",
		},
		{
			name:     "encoded plus in query database",
			uri:      "host/db?dbname=a%2Bb",
			key:      "database",
			expected: "a+b",
		},
		{
			name:     "encoded plus in query key and value",
			uri:      "host/db?x%2By=a%2Bb",
			key:      "x+y",
			expected: "a+b",
		},
		{
			name:     "encoded space in query",
			uri:      "host/db?dbname=a%20b",
			key:      "database",
			expected: "a b",
		},
		{
			name:     "decode query only once",
			uri:      "host/db?dbname=a%252Bb",
			key:      "database",
			expected: "a%2Bb",
		},
		{
			name:     "Windows path",
			uri:      `host/C:\data\db.fdb`,
			key:      "database",
			expected: `C:\data\db.fdb`,
		},
		{
			name:     "Windows path in query",
			uri:      `host/db?dbname=C:\data\db.fdb`,
			key:      "database",
			expected: `C:\data\db.fdb`,
		},
		{
			name:     "encoded backslashes in query",
			uri:      "host/db?dbname=C:%5Cdata%5Cdb.fdb",
			key:      "database",
			expected: `C:\data\db.fdb`,
		},
		{
			name:     "encoded backslash in userinfo",
			uri:      "u:p%5Cq@host/db",
			key:      "password",
			expected: `p\q`,
		},
		{
			name:     "leading slash in database path",
			uri:      "host//a/b",
			key:      "database",
			expected: "/a/b",
		},
		{
			name:     "encoded forward slash in path",
			uri:      "host/a%2Fb",
			key:      "database",
			expected: "a/b",
		},
		{
			name:     "backslash does not escape query plus",
			uri:      `host/db?x=a\+b`,
			key:      "x",
			expected: `a\+b`,
		},
	}

	for _, scheme := range []string{"firebird", "firebirdsql"} {
		for _, tt := range tests {
			t.Run(scheme+"/"+tt.name, func(t *testing.T) {
				config, err := ParseConfig(scheme + "://" + tt.uri)

				require.NoError(t, err)
				var actual string
				switch tt.key {
				case "user":
					actual = config.User
				case "password":
					actual = config.Password
				case "database":
					actual = config.Database
				case "host":
					actual = config.Host
				default:
					require.Contains(t, config.RuntimeParams, tt.key)
					actual = config.RuntimeParams[tt.key]
				}
				require.Equal(t, tt.expected, actual)
			})
		}
	}
}

func TestParseConfigURIEmptyComponentsOverrideEnvironment(t *testing.T) {
	t.Setenv("FBUSER", "env-user")
	t.Setenv("FBPASSWORD", "env-password")
	t.Setenv("FBDATABASE", "env-database")

	tests := []struct {
		name     string
		uri      string
		user     string
		password string
		database string
	}{
		{
			name:     "explicitly empty password",
			uri:      "u:@host/db",
			user:     "u",
			password: "",
			database: "db",
		},
		{
			name:     "omitted password",
			uri:      "u@host/db",
			user:     "u",
			password: "env-password",
			database: "db",
		},
		{
			name:     "explicitly empty username",
			uri:      "@host/db",
			user:     "",
			password: "env-password",
			database: "db",
		},
		{
			name:     "omitted username",
			uri:      "host/db",
			user:     "env-user",
			password: "env-password",
			database: "db",
		},
		{
			name:     "explicitly empty database path",
			uri:      "host/",
			user:     "env-user",
			password: "env-password",
			database: "",
		},
		{
			name:     "omitted database path",
			uri:      "host",
			user:     "env-user",
			password: "env-password",
			database: "env-database",
		},
	}

	for _, scheme := range []string{"firebird", "firebirdsql"} {
		for _, tt := range tests {
			t.Run(scheme+"/"+tt.name, func(t *testing.T) {
				config, err := ParseConfig(scheme + "://" + tt.uri)

				require.NoError(t, err)
				assert.Equal(t, tt.user, config.User)
				assert.Equal(t, tt.password, config.Password)
				assert.Equal(t, tt.database, config.Database)
			})
		}
	}
}
