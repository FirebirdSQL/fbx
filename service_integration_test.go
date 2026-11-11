package fbx_test

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/firebirdsql/fbx"
	"github.com/stretchr/testify/require"
)

type serviceIntegrationServer struct {
	profile string
	kind    string
	version fbx.ServerVersion
}

var serviceIntegrationVersionRegexp = regexp.MustCompile(`^(\d+)\.(\d+)`)

// An empty kind means that no server profile was declared. Each test decides
// whether its operation needs a profile or can use the actual server version.
func serviceIntegrationServerFromEnvironment() (serviceIntegrationServer, error) {
	server := serviceIntegrationServer{
		profile: strings.TrimSpace(os.Getenv("TARGET")),
		kind:    strings.ToLower(strings.TrimSpace(os.Getenv("DB_KIND"))),
	}
	declaredVersion := strings.TrimSpace(os.Getenv("DB_VERSION"))
	if server.kind == "" && declaredVersion == "" {
		rdbVersion := strings.TrimSpace(os.Getenv("RDB_VERSION"))
		firebirdVersion := strings.TrimSpace(os.Getenv("FB_VERSION"))
		switch {
		case rdbVersion != "" && firebirdVersion != "":
			return server, errors.New("both RDB_VERSION and FB_VERSION are set")
		case rdbVersion != "":
			server.kind, declaredVersion = "rdb", rdbVersion
		case firebirdVersion != "":
			server.kind, declaredVersion = "firebird", firebirdVersion
		default:
			return server, nil
		}
	}
	if server.kind == "" || declaredVersion == "" {
		return server, errors.New("database kind and version must be provided together")
	}
	if server.kind != "firebird" && server.kind != "rdb" {
		return server, fmt.Errorf("unsupported DB_KIND %q", server.kind)
	}

	matches := serviceIntegrationVersionRegexp.FindStringSubmatch(declaredVersion)
	if len(matches) != 3 {
		return server, fmt.Errorf("failed to parse database version %q", declaredVersion)
	}
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return server, fmt.Errorf("parse database major version: %w", err)
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return server, fmt.Errorf("parse database minor version: %w", err)
	}
	server.version = fbx.ServerVersion{Major: major, Minor: minor}
	return server, nil
}

func (server *serviceIntegrationServer) setVersion(version fbx.ServerVersion) error {
	if server.kind != "" &&
		(server.version.Major != version.Major || server.version.Minor != version.Minor) {
		return fmt.Errorf("declared database version %q does not match server version %q", server.version, version)
	}
	server.version = version
	if server.profile == "" && server.kind != "" {
		server.profile = fmt.Sprintf("%s-%s", server.kind, server.version)
	}
	return nil
}

func TestServiceIntegrationServerFromEnvironment(t *testing.T) {
	tests := map[string]struct {
		env     map[string]string
		want    serviceIntegrationServer
		wantErr string
	}{
		"no profile": {},
		"whitespace only": {
			env: map[string]string{"DB_KIND": " ", "DB_VERSION": "\t", "RDB_VERSION": " ", "FB_VERSION": "\n"},
		},
		"explicit profile": {
			env:  map[string]string{"TARGET": " custom ", "DB_KIND": " RDB ", "DB_VERSION": " 5.2.1 "},
			want: serviceIntegrationServer{profile: "custom", kind: "rdb", version: fbx.ServerVersion{Major: 5, Minor: 2}},
		},
		"RDB fallback": {
			env:  map[string]string{"RDB_VERSION": " 6.0 "},
			want: serviceIntegrationServer{kind: "rdb", version: fbx.ServerVersion{Major: 6}},
		},
		"Firebird fallback": {
			env:  map[string]string{"FB_VERSION": " 5.0.3.1683-0 "},
			want: serviceIntegrationServer{kind: "firebird", version: fbx.ServerVersion{Major: 5}},
		},
		"explicit profile takes precedence": {
			env:  map[string]string{"DB_KIND": " Firebird ", "DB_VERSION": " 4.0 ", "RDB_VERSION": "6.0", "FB_VERSION": "5.0"},
			want: serviceIntegrationServer{kind: "firebird", version: fbx.ServerVersion{Major: 4}},
		},
		"conflicting fallbacks": {
			env: map[string]string{"RDB_VERSION": "6.0", "FB_VERSION": "5.0"}, wantErr: "both RDB_VERSION and FB_VERSION are set",
		},
		"missing version": {
			env: map[string]string{"DB_KIND": "rdb"}, wantErr: "kind and version must be provided together",
		},
		"missing kind": {
			env: map[string]string{"DB_VERSION": "6.0"}, wantErr: "kind and version must be provided together",
		},
		"partial explicit profile does not use fallback": {
			env: map[string]string{"DB_KIND": "rdb", "RDB_VERSION": "6.0"}, wantErr: "kind and version must be provided together",
		},
		"unknown kind": {
			env: map[string]string{"DB_KIND": "other", "DB_VERSION": "6.0"}, wantErr: "unsupported DB_KIND",
		},
		"invalid version": {
			env: map[string]string{"RDB_VERSION": "six"}, wantErr: "failed to parse database version",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			for _, key := range []string{"TARGET", "DB_KIND", "DB_VERSION", "RDB_VERSION", "FB_VERSION"} {
				t.Setenv(key, tt.env[key])
			}
			server, err := serviceIntegrationServerFromEnvironment()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, server)
		})
	}
}

func TestServiceIntegrationServerVersion(t *testing.T) {
	actual := fbx.ServerVersion{Major: 6, Minor: 0, Patch: 1, Build: 42}
	for name, tt := range map[string]struct {
		server  serviceIntegrationServer
		profile string
	}{
		"no profile": {},
		"default profile": {
			server: serviceIntegrationServer{kind: "rdb", version: fbx.ServerVersion{Major: 6}}, profile: "rdb-6.0.1.42",
		},
		"custom profile": {
			server: serviceIntegrationServer{profile: "custom", kind: "rdb", version: fbx.ServerVersion{Major: 6}}, profile: "custom",
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, tt.server.setVersion(actual))
			require.Equal(t, actual, tt.server.version)
			require.Equal(t, tt.profile, tt.server.profile)
		})
	}
	for name, version := range map[string]fbx.ServerVersion{
		"major mismatch": {Major: 5},
		"minor mismatch": {Major: 6, Minor: 1},
	} {
		t.Run(name, func(t *testing.T) {
			server := serviceIntegrationServer{kind: "rdb", version: version}
			require.ErrorContains(t, server.setVersion(actual), "does not match server version")
			require.Equal(t, version, server.version)
		})
	}
}
