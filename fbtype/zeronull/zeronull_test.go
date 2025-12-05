package zeronull_test

import (
	"context"
	"os"
	"testing"

	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbtype/zeronull"
	"rdb.red-soft.ru/fbx/fbxtest"
	"github.com/stretchr/testify/require"
)

var defaultConnTestRunner fbxtest.ConnTestRunner

func init() {
	defaultConnTestRunner = fbxtest.DefaultConnTestRunner()
	defaultConnTestRunner.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
		config, err := fbx.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
		require.NoError(t, err)
		return config
	}
	defaultConnTestRunner.AfterConnect = func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
		zeronull.Register(conn.TypeMap())
	}
}
