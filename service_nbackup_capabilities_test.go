package fbx_test

import (
	"testing"

	"github.com/firebirdsql/fbx"
	"github.com/stretchr/testify/require"
)

type nbackupIntegrationCapabilities struct {
	guid             bool
	inPlace          bool
	fixup            bool
	preserveSequence bool
	cleanHistory     bool
}

// RDB 3 supports GUID (native change 07fd8f0d56). Its console has in-place
// restore, but the service bitmask table rejects it; RDB 5 accepts that option.
func (server serviceIntegrationServer) nbackupCapabilities() nbackupIntegrationCapabilities {
	var caps nbackupIntegrationCapabilities
	switch server.kind {
	case "firebird":
		caps.guid = server.version.AtLeast(4, 0, 0)
		caps.inPlace = server.version.AtLeast(4, 0, 0)
		caps.fixup = server.version.AtLeast(4, 0, 0)
		caps.preserveSequence = server.version.AtLeast(4, 0, 0)
		caps.cleanHistory = server.version.AtLeast(4, 0, 3)
	case "rdb":
		caps.guid = server.version.AtLeast(3, 0, 0)
		caps.inPlace = server.version.AtLeast(5, 0, 0)
		caps.fixup = server.version.AtLeast(5, 0, 0)
		caps.preserveSequence = server.version.AtLeast(5, 0, 0)
		caps.cleanHistory = server.version.AtLeast(5, 0, 0)
	}
	return caps
}

func TestServiceNBackupCapabilities(t *testing.T) {
	all := nbackupIntegrationCapabilities{guid: true, inPlace: true, fixup: true, preserveSequence: true, cleanHistory: true}
	for _, tt := range []struct {
		name    string
		kind    string
		version fbx.ServerVersion
		want    nbackupIntegrationCapabilities
	}{
		{"unknown family", "", fbx.ServerVersion{Major: 5}, nbackupIntegrationCapabilities{}},
		{"Firebird 3", "firebird", fbx.ServerVersion{Major: 3}, nbackupIntegrationCapabilities{}},
		{"Firebird 4.0.2", "firebird", fbx.ServerVersion{Major: 4, Patch: 2}, nbackupIntegrationCapabilities{guid: true, inPlace: true, fixup: true, preserveSequence: true}},
		{"Firebird 4.0.3", "firebird", fbx.ServerVersion{Major: 4, Patch: 3}, all},
		{"Firebird 5", "firebird", fbx.ServerVersion{Major: 5}, all},
		{"RDB 3", "rdb", fbx.ServerVersion{Major: 3, Patch: 23}, nbackupIntegrationCapabilities{guid: true}},
		{"RDB 5", "rdb", fbx.ServerVersion{Major: 5}, all},
		{"RDB 5.1", "rdb", fbx.ServerVersion{Major: 5, Minor: 1}, all},
		{"RDB 5.2", "rdb", fbx.ServerVersion{Major: 5, Minor: 2}, all},
		{"RDB 5.3", "rdb", fbx.ServerVersion{Major: 5, Minor: 3}, all},
		{"RDB 6", "rdb", fbx.ServerVersion{Major: 6}, all},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := serviceIntegrationServer{kind: tt.kind, version: tt.version}
			require.Equal(t, tt.want, server.nbackupCapabilities())
		})
	}
}
