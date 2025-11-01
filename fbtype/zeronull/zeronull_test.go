package zeronull_test

import (
	"context"
	"os"
	"testing"

	"github.com/red-soft/fbx"
	"github.com/red-soft/fbx/fbtype/zeronull"
	"github.com/red-soft/fbx/fbxtest"
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
