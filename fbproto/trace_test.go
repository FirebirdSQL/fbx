package fbproto_test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/red-soft/fbx/fbproto"
	"github.com/red-soft/fbx/pgconn"
	"github.com/stretchr/testify/require"
)

func TestTrace(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := pgconn.Connect(ctx, os.Getenv(""))
	require.NoError(t, err)
	defer conn.Close(ctx)

	// if conn.ParameterStatus() // TODO

	traceOutput := &bytes.Buffer{}

	conn.Client().Trace(traceOutput, fbproto.TracerOptions{
		SuppressTimestamps: true,
		FullLogMode:        true,
	})
}
