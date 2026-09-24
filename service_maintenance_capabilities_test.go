package fbx_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
)

type maintenanceIntegrationCapabilities struct {
	icu           bool
	parallelSweep bool
	parallelICU   bool
	sweepTimeout  bool
	upgradeODS    bool
}

// Firebird: gfix documentation and Firebird 5 release notes.
// RDB: gfix switches in 3.0, 5.0 and 6.0
// Sweep timeout (tag 101) is an RDB extension, not a generic Firebird 6 feature.
func (server serviceIntegrationServer) maintenanceCapabilities() maintenanceIntegrationCapabilities {
	var caps maintenanceIntegrationCapabilities
	switch server.kind {
	case "firebird":
		caps.icu = server.version.AtLeast(3, 0, 0)
		caps.parallelSweep = server.version.AtLeast(5, 0, 0)
		caps.parallelICU = server.version.AtLeast(5, 0, 0)
		caps.upgradeODS = server.version.AtLeast(5, 0, 0)
	case "rdb":
		caps.icu = server.version.AtLeast(3, 0, 0)
		caps.parallelSweep = server.version.AtLeast(3, 0, 9)
		caps.parallelICU = server.version.AtLeast(5, 0, 0)
		caps.upgradeODS = server.version.AtLeast(5, 0, 0)
		caps.sweepTimeout = server.version.AtLeast(6, 0, 0)
	}
	return caps
}

func requireMaintenanceFeature(t *testing.T, server serviceIntegrationServer, supported bool, feature string) {
	t.Helper()
	if server.kind == "" {
		t.Skipf("%s requires DB_KIND/DB_VERSION, RDB_VERSION or FB_VERSION", feature)
	}
	if !supported {
		t.Skipf("%s is not supported by %s %s", feature, server.kind, server.version)
	}
}

func TestServiceMaintenanceCapabilities(t *testing.T) {
	for _, tt := range []struct {
		name    string
		kind    string
		version fbx.ServerVersion
		want    maintenanceIntegrationCapabilities
	}{
		{"unknown family", "", fbx.ServerVersion{Major: 6}, maintenanceIntegrationCapabilities{}},
		{"Firebird 2.5", "firebird", fbx.ServerVersion{Major: 2, Minor: 5}, maintenanceIntegrationCapabilities{}},
		{"Firebird 3", "firebird", fbx.ServerVersion{Major: 3}, maintenanceIntegrationCapabilities{icu: true}},
		{"Firebird 4", "firebird", fbx.ServerVersion{Major: 4}, maintenanceIntegrationCapabilities{icu: true}},
		{"Firebird 5", "firebird", fbx.ServerVersion{Major: 5}, maintenanceIntegrationCapabilities{icu: true, parallelSweep: true, parallelICU: true, upgradeODS: true}},
		{"Firebird 6 has no RDB timeout", "firebird", fbx.ServerVersion{Major: 6}, maintenanceIntegrationCapabilities{icu: true, parallelSweep: true, parallelICU: true, upgradeODS: true}},
		{"RDB 3.0.8", "rdb", fbx.ServerVersion{Major: 3, Patch: 8}, maintenanceIntegrationCapabilities{icu: true}},
		{"RDB 3.0.9", "rdb", fbx.ServerVersion{Major: 3, Patch: 9}, maintenanceIntegrationCapabilities{icu: true, parallelSweep: true}},
		{"RDB 3.0.23", "rdb", fbx.ServerVersion{Major: 3, Patch: 23}, maintenanceIntegrationCapabilities{icu: true, parallelSweep: true}},
		{"RDB 5.0", "rdb", fbx.ServerVersion{Major: 5}, maintenanceIntegrationCapabilities{icu: true, parallelSweep: true, parallelICU: true, upgradeODS: true}},
		{"RDB 5.1", "rdb", fbx.ServerVersion{Major: 5, Minor: 1}, maintenanceIntegrationCapabilities{icu: true, parallelSweep: true, parallelICU: true, upgradeODS: true}},
		{"RDB 5.2", "rdb", fbx.ServerVersion{Major: 5, Minor: 2}, maintenanceIntegrationCapabilities{icu: true, parallelSweep: true, parallelICU: true, upgradeODS: true}},
		{"RDB 5.3", "rdb", fbx.ServerVersion{Major: 5, Minor: 3}, maintenanceIntegrationCapabilities{icu: true, parallelSweep: true, parallelICU: true, upgradeODS: true}},
		{"RDB 6", "rdb", fbx.ServerVersion{Major: 6}, maintenanceIntegrationCapabilities{icu: true, parallelSweep: true, parallelICU: true, sweepTimeout: true, upgradeODS: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := serviceIntegrationServer{kind: tt.kind, version: tt.version}
			require.Equal(t, tt.want, server.maintenanceCapabilities())
		})
	}
}
