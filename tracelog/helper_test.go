package tracelog_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
)

type serverEngineInfo struct {
	version string
	major   int
	minor   int
	patch   int
}

var (
	serverEngineInfoOnce  sync.Once
	serverEngineInfoValue serverEngineInfo
)

func getServerEngineInfo(t testing.TB) serverEngineInfo {
	t.Helper()
	serverEngineInfoOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)

		conn, err := fbx.ConnectConfig(ctx, config)
		require.NoError(t, err)
		defer func() { require.NoError(t, conn.Close(ctx)) }()

		version := conn.ServerVersion()
		serverEngineInfoValue = serverEngineInfo{
			version: version.String(),
			major:   version.Major,
			minor:   version.Minor,
			patch:   version.Patch,
		}
	})

	return serverEngineInfoValue
}

func isRedDatabase50(t testing.TB) bool {
	t.Helper()
	info := getServerEngineInfo(t)
	return info.major == 5 && info.minor == 0 && info.patch >= 8
}
