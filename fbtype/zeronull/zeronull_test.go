package zeronull_test

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbtype/zeronull"
	"rdb.red-soft.ru/fbx/fbxtest"
)

var defaultConnTestRunner fbxtest.ConnTestRunner
var (
	serverMajorVersionCheckOnce sync.Once
	serverMajorVersionValue     int
	engineMajorRegexp           = regexp.MustCompile(`\d+`)
)

func init() {
	defaultConnTestRunner = fbxtest.DefaultConnTestRunner()
	defaultConnTestRunner.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config, err := fbx.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)
		return config
	}
	defaultConnTestRunner.AfterConnect = func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		zeronull.Register(conn.TypeMap())
	}
}

func getServerMajorVersionForConn(ctx context.Context, t testing.TB, conn interface {
	QueryRow(ctx context.Context, sql string, args ...any) fbx.Row
}) int {
	t.Helper()
	serverMajorVersionCheckOnce.Do(func() {
		var version string
		err := conn.QueryRow(ctx, "select rdb$get_context('SYSTEM','ENGINE_VERSION') from rdb$database").Scan(&version)
		require.NoError(t, err)

		major, ok := parseEngineMajorVersion(version)
		require.Truef(t, ok, "failed to parse engine version: %q", version)
		serverMajorVersionValue = major
	})
	return serverMajorVersionValue
}

func parseEngineMajorVersion(version string) (int, bool) {
	match := engineMajorRegexp.FindString(version)
	if match == "" {
		return 0, false
	}
	major, err := strconv.Atoi(match)
	if err != nil {
		return 0, false
	}
	return major, true
}
