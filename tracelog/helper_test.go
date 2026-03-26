package tracelog_test

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
)

type serverEngineInfo struct {
	version       string
	major         int
	minor         int
	patch         int
	isRedDatabase bool
}

var (
	serverEngineInfoOnce  sync.Once
	serverEngineInfoValue serverEngineInfo
	engineVersionRegexp   = regexp.MustCompile(`^\s*(\d+)\.(\d+)\.(\d+)`)
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
		defer conn.Close(ctx)

		var version string
		err = conn.QueryRow(ctx, "select rdb$get_context('SYSTEM','ENGINE_VERSION') from rdb$database").Scan(&version)
		require.NoError(t, err)

		serverEngineInfoValue = parseServerEngineInfo(t, version)
	})

	return serverEngineInfoValue
}

func parseServerEngineInfo(t testing.TB, version string) serverEngineInfo {
	t.Helper()

	matches := engineVersionRegexp.FindStringSubmatch(version)
	require.Lenf(t, matches, 4, "failed to parse server engine version %q: expected semantic version major.minor.patch", version)

	major := mustAtoi(t, "major", matches[1], version)
	minor := mustAtoi(t, "minor", matches[2], version)
	patch := mustAtoi(t, "patch", matches[3], version)

	return serverEngineInfo{
		version:       version,
		major:         major,
		minor:         minor,
		patch:         patch,
	}
}

func mustAtoi(t testing.TB, partName, value, version string) int {
	t.Helper()
	n, err := strconv.Atoi(value)
	require.NoErrorf(t, err, "failed to parse %s version part %q from engine version %q", partName, value, version)
	return n
}

func isRedDatabase50(t testing.TB) bool {
	t.Helper()
	info := getServerEngineInfo(t)
	return info.major == 5 && info.minor == 0 && info.patch >= 8
}
