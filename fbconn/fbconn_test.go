package fbconn_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbconn/ctxwatch"
	"rdb.red-soft.ru/fbx/fbproto"
	"rdb.red-soft.ru/fbx/internal/fbio"
)

func TestConnect(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{"TCP", "FBX_TEST_DATABASE"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			connString := os.Getenv(tt.env)
			if connString == "" {
				t.Skipf("Skipping due to missing environment variable %v", tt.env)
			}

			conn, err := fbconn.Connect(ctx, connString)
			require.NoError(t, err)

			closeConn(t, conn)
		})
	}
}

func TestAttachmentID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("Skipping due to missing environment variable FBX_TEST_DATABASE")
	}

	conn, err := fbconn.Connect(ctx, connString)
	require.NoError(t, err)
	defer closeConn(t, conn)

	expected := subjectAttachmentID(t, conn)
	require.Positive(t, conn.AttachmentID())
	require.Equal(t, uint64(expected), conn.AttachmentID())
}

func TestServerVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("Skipping due to missing environment variable FBX_TEST_DATABASE")
	}

	conn, err := fbconn.Connect(ctx, connString)
	require.NoError(t, err)
	defer closeConn(t, conn)

	version := conn.ServerVersion()
	require.Positive(t, version.Major)
	require.NotEmpty(t, version.String())
}

func TestConnectWithOptions(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{"TCP", "FBX_TEST_DATABASE"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			connString := os.Getenv(tt.env)
			if connString == "" {
				t.Skipf("Skipping due to missing environment variable %v", tt.env)
			}
			var sslOptions fbconn.ParseConfigOptions
			sslOptions.GetSSLPassword = GetSSLPassword
			conn, err := fbconn.ConnectWithOptions(ctx, connString, sslOptions)
			require.NoError(t, err)

			closeConn(t, conn)
		})
	}
}

func TestConnectInvalidUser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	config.User = "fbxinvalidusertest"

	_, err = fbconn.ConnectConfig(ctx, config)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "invalid SRP payload")
	var fbErr *fbconn.FbError
	require.ErrorAs(t, err, &fbErr)
	if fbErr.Code != "28000" && fbErr.Code != "28P01" {
		t.Fatalf("Expected to receive a FbError with code 28000 or 28P01, instead received: %v", fbErr)
	}
}

func TestConnectWithConnectionRefused(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Presumably nothing is listening on 127.0.0.1:1
	conn, err := fbconn.Connect(ctx, "host=127.0.0.1 port=1")
	if err == nil {
		_ = conn.Close(ctx)
		t.Fatal("Expected error establishing connection to bad port")
	}
}

func TestConnectCustomDialer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	dialed := false
	config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		dialed = true
		return net.Dial(network, address)
	}

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	require.True(t, dialed)
	closeConn(t, conn)
}

func TestLocalBLRErrorDoesNotCloseConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("Skipping due to missing environment variable FBX_TEST_DATABASE")
	}

	conn, err := fbconn.Connect(ctx, connString)
	require.NoError(t, err)
	defer closeConn(t, conn)

	statement, err := conn.Prepare(ctx, "select cast(? as integer) from rdb$database")
	require.NoError(t, err)
	require.Len(t, statement.Description.StatementInfo.ParametersDescriptors, 1)

	// Force GetParametersBlr to fail before an execute message reaches the server.
	statement.Description.StatementInfo.ParametersDescriptors[0].Type = 65000
	result := statement.Exec(ctx, [][]byte{{0, 0, 0, 7}}).Read()
	require.Error(t, result.Err)
	require.False(t, conn.IsClosed())
	require.NoError(t, conn.Ping(ctx))
	require.NoError(t, statement.Deallocate(ctx))
}

type countCloseConn struct {
	net.Conn
	closeCalls atomic.Int64
}

func (c *countCloseConn) Close() error {
	c.closeCalls.Add(1)
	return c.Conn.Close()
}

func TestCloseClosesUnderlyingConnectionOnce(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("Skipping due to missing environment variable FBX_TEST_DATABASE")
	}
	config, err := fbconn.ParseConfig(connString)
	require.NoError(t, err)

	var wireConn *countCloseConn
	config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		raw, err := (&net.Dialer{}).DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}
		wireConn = &countCloseConn{Conn: raw}
		return wireConn, nil
	}

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, wireConn)
	require.NoError(t, conn.Close(ctx))
	require.Equal(t, int64(1), wireConn.closeCalls.Load())
}

func TestWireCompression(t *testing.T) {
	t.Parallel()

	for _, wireCrypt := range []bool{false, true} {
		t.Run(fmt.Sprintf("wire_crypt=%t", wireCrypt), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
			require.NoError(t, err)
			config.WireCompression = true
			config.WireCrypt = wireCrypt

			conn, err := fbconn.ConnectConfig(ctx, config)
			require.NoError(t, err)
			defer closeConn(t, conn)
			require.True(t, conn.WireCompressionEnabled())
			ensureConnValid(t, conn)
		})
	}
}

func TestConnectCustomLookup(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	looked := false
	config.LookupFunc = func(ctx context.Context, host string) (addrs []string, err error) {
		looked = true
		return net.LookupHost(host)
	}

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	require.True(t, looked)
	closeConn(t, conn)
}

func TestConnectCustomLookupWithPort(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	origPort := config.Port
	// Change the config an invalid port so it will fail if used
	config.Port = 0

	looked := false
	config.LookupFunc = func(ctx context.Context, host string) ([]string, error) {
		looked = true
		addrs, err := net.LookupHost(host)
		if err != nil {
			return nil, err
		}
		for i := range addrs {
			addrs[i] = net.JoinHostPort(addrs[i], strconv.FormatUint(uint64(origPort), 10))
		}
		return addrs, nil
	}

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	require.True(t, looked)
	closeConn(t, conn)
}

func TestConnectWithFallback(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	// Prepend current primary config to fallbacks
	config.Fallbacks = append([]*fbconn.FallbackConfig{
		{
			Host: config.Host,
			Port: config.Port,
		},
	}, config.Fallbacks...)

	// Make primary config bad
	config.Host = "localhost"
	config.Port = 1 // presumably nothing listening here

	// Prepend bad first fallback
	config.Fallbacks = append([]*fbconn.FallbackConfig{
		{
			Host: "localhost",
			Port: 1,
		},
	}, config.Fallbacks...)

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	closeConn(t, conn)
}

func TestConnectFailsWithResolveFailureAndFailedConnectionAttempts(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := fbconn.Connect(ctx, "host=localhost,127.0.0.1,foo.invalid port=1,2,3")
	require.Error(t, err)
	require.Nil(t, conn)

	require.ErrorContains(t, err, "lookup foo.invalid")
	// Not testing the entire string as depending on IPv4 or IPv6 support localhost may resolve to 127.0.0.1 or ::1.
	require.ErrorContains(t, err, ":1 (localhost): dial error:")
	require.ErrorContains(t, err, ":2 (127.0.0.1): dial error:")
}

func TestConnectWithValidateConnect(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	dialCount := 0
	config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		dialCount++
		return net.Dial(network, address)
	}

	acceptConnCount := 0
	var validatedVersions []fbconn.ServerVersion
	config.ValidateConnect = func(ctx context.Context, conn *fbconn.FbConn) error {
		acceptConnCount++
		validatedVersions = append(validatedVersions, conn.ServerVersion())
		if acceptConnCount < 2 {
			return errors.New("reject first conn")
		}
		return nil
	}

	// Append current primary config to fallbacks
	config.Fallbacks = append(config.Fallbacks, &fbconn.FallbackConfig{
		Host: config.Host,
		Port: config.Port,
	})

	// Repeat fallbacks
	config.Fallbacks = append(config.Fallbacks, config.Fallbacks...)

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	closeConn(t, conn)

	assert.True(t, dialCount > 1)
	assert.True(t, acceptConnCount > 1)
	require.Len(t, validatedVersions, acceptConnCount)
	for _, version := range validatedVersions {
		assert.Positive(t, version.Major)
	}
}

func TestConnectWithValidateConnectTargetSessionAttrsReadWrite(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	config.ValidateConnect = fbconn.ValidateConnectTargetSessionAttrsReadWrite
	config.DefaultTransactionReadOnly = true

	conn, err := fbconn.ConnectConfig(ctx, config)
	if !assert.NotNil(t, err) {
		_ = conn.Close(ctx)
	}
}

func TestConnectWithAfterConnect(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	//"set search_path to foobar;"
	config.AfterConnect = func(ctx context.Context, conn *fbconn.FbConn) error {
		res := conn.Exec(ctx, "select RDB$SET_CONTEXT('USER_SESSION', 'search_path', 'foobar') from rdb$database").Read()
		return res.Err
	}

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, conn)

	res := conn.Exec(ctx, "select RDB$GET_CONTEXT('USER_SESSION', 'search_path') from rdb$database search_path").Read()
	require.NoError(t, res.Err)

	assert.Equal(t, "foobar", string(res.Rows[0][0]))
}

func TestConnectConfigRequiresConfigFromParseConfig(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config := &fbconn.Config{}

	require.PanicsWithValue(t, "config must be created by ParseConfig", func() { _, _ = fbconn.ConnectConfig(ctx, config) })
}

func TestConnPrepareSyntaxError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	psd, err := fbConn.Prepare(ctx, "SYNTAX ERROR")
	require.Nil(t, psd)
	require.NotNil(t, err)

	ensureConnValid(t, fbConn)
}

func TestConnPrepareContextPrecanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	cancel()

	psd, err := fbConn.Prepare(ctx, "select 1")
	assert.Nil(t, psd)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.True(t, fbconn.SafeToRetry(err))

	ensureConnValid(t, fbConn)
}

func TestPrepareTxForConnectionUsesCurrentInternalTransaction(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	preparingTx, err := fbConn.StartTransaction(ctx)
	require.NoError(t, err)
	defer func() {
		if !preparingTx.Closed {
			assert.NoError(t, preparingTx.Rollback(context.Background()))
		}
	}()

	result := fbConn.ExecTx(ctx, preparingTx, "select RDB$SET_CONTEXT('USER_TRANSACTION', 'rs_328477_scope', 'preparing') from rdb$database").Read()
	require.NoError(t, result.Err)

	currentTx, err := fbConn.StartTransaction(ctx)
	require.NoError(t, err)
	defer func() {
		if !currentTx.Closed {
			assert.NoError(t, currentTx.Rollback(context.Background()))
		}
	}()

	result = fbConn.ExecTx(ctx, currentTx, "select RDB$SET_CONTEXT('USER_TRANSACTION', 'rs_328477_scope', 'current') from rdb$database").Read()
	require.NoError(t, result.Err)

	st, err := fbConn.PrepareTxForConnection(ctx, preparingTx, "select RDB$GET_CONTEXT('USER_TRANSACTION', 'rs_328477_scope') from rdb$database")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, st.Deallocate(context.Background()))
	}()

	result = st.ExecTx(ctx, preparingTx, nil).Read()
	require.NoError(t, result.Err)
	require.Equal(t, "preparing", string(result.Rows[0][0]))
	require.NoError(t, preparingTx.Rollback(ctx))

	result = st.Exec(ctx, nil).Read()
	require.NoError(t, result.Err)
	require.Equal(t, "current", string(result.Rows[0][0]))
	require.NoError(t, st.Deallocate(ctx))
	require.NoError(t, currentTx.Rollback(ctx))
	ensureConnValid(t, fbConn)
}

func TestPrepareTxRemainsBoundToPreparingTransaction(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	preparingTx, err := fbConn.StartTransaction(ctx)
	require.NoError(t, err)
	defer func() {
		if !preparingTx.Closed {
			assert.NoError(t, preparingTx.Rollback(context.Background()))
		}
	}()

	currentTx, err := fbConn.StartTransaction(ctx)
	require.NoError(t, err)
	defer func() {
		if !currentTx.Closed {
			assert.NoError(t, currentTx.Rollback(context.Background()))
		}
	}()

	st, err := fbConn.PrepareTx(ctx, preparingTx, "select 1 from rdb$database")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, st.Deallocate(context.Background()))
	}()

	require.NoError(t, preparingTx.Rollback(ctx))
	require.Error(t, st.Exec(ctx, nil).Read().Err)
	require.NoError(t, st.Deallocate(ctx))
	require.NoError(t, currentTx.Rollback(ctx))
	ensureConnValid(t, fbConn)
}

func TestConnDeallocate(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	//select 1
	st, err := fbConn.Prepare(ctx, "select 1 from rdb$database")
	require.NoError(t, err)

	_, err = st.Exec(ctx, nil).Close()
	require.NoError(t, err)

	err = st.Deallocate(ctx)
	require.NoError(t, err)

	_, err = st.Exec(ctx, nil).Close()
	assert.EqualError(t, err, "statement is closed")

	ensureConnValid(t, fbConn)
}

func TestConnDeallocateSucceedsInAbortedTransaction(t *testing.T) {
	//t.Parallel()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	//defer cancel()
	//
	//fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	//require.NoError(t, err)
	//defer closeConn(t, fbConn)
	//
	//err = fbConn.Exec(ctx, "begin").Close()
	//require.NoError(t, err)
	//
	//_, err = fbConn.PrepareTx(ctx, "ps1", "select 1", nil)
	//require.NoError(t, err)
	//
	//_, err = fbConn.ExecPrepared(ctx, "ps1", nil, nil, nil).Close()
	//require.NoError(t, err)
	//
	//err = fbConn.Exec(ctx, "select 1/0").Close() // break transaction with divide by 0 error
	//require.Error(t, err)
	//var fbErr *fbconn.FbError
	//require.ErrorAs(t, err, &fbErr)
	//require.Equal(t, "22012", fbErr.Code)
	//
	//err = fbConn.Deallocate(ctx, "ps1")
	//require.NoError(t, err)
	//
	//err = fbConn.Exec(ctx, "rollback").Close()
	//require.NoError(t, err)
	//
	//_, err = fbConn.ExecPrepared(ctx, "ps1", nil, nil, nil).Close()
	//require.Error(t, err)
	//require.ErrorAs(t, err, &fbErr)
	//require.Equal(t, "26000", fbErr.Code)
	//
	//ensureConnValid(t, fbConn)
}

func TestConnDeallocateNonExistantStatementSucceeds(t *testing.T) {
	//t.Parallel()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	//defer cancel()
	//
	//fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	//require.NoError(t, err)
	//defer closeConn(t, fbConn)
	//
	//err = fbConn.Deallocate(ctx, "ps1")
	//require.NoError(t, err)
	//
	//ensureConnValid(t, fbConn)
}

func TestConnExec(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	res := fbConn.Exec(ctx, "select cast('Hello, world' as varchar(255)) from rdb$database").Read()
	assert.NoError(t, res.Err)

	assert.Len(t, res.Rows, 1)
	assert.Equal(t, "Hello, world", string(res.Rows[0][0]))

	results, err := fbConn.Exec(ctx, "select cast('Hello, world' as varchar(255)) from rdb$database").ReadAll()
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Len(t, results[0].Rows, 1)
	assert.Equal(t, "Hello, world", string(results[0].Rows[0][0]))

	ensureConnValid(t, fbConn)
}

func TestConnExecDeferredError(t *testing.T) {
	//t.Parallel()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	//defer cancel()
	//
	//fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	//require.NoError(t, err)
	//defer closeConn(t, fbConn)
	//
	//if fbConn.ParameterStatus("crdb_version") != "" {
	//	t.Skip("Server does not support deferred constraint")
	//}
	//
	//setupSQL := `create temporary table t (
	//	id text primary key,
	//	n int not null,
	//	unique (n) deferrable initially deferred
	//);
	//
	//insert into t (id, n) values ('a', 1), ('b', 2), ('c', 3);`
	//
	//_, err = fbConn.Exec(ctx, setupSQL).ReadAll()
	//assert.NoError(t, err)
	//
	//_, err = fbConn.Exec(ctx, `update t set n=n+1 where id='b' returning *`).ReadAll()
	//require.NotNil(t, err)
	//
	//var fbErr *fbconn.FbError
	//require.True(t, errors.As(err, &fbErr))
	//require.Equal(t, "23505", fbErr.Code)
	//
	//ensureConnValid(t, fbConn)
}

func TestConnExecContextCanceled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	_, err = fbConn.Exec(ctx, "recreate function fbsleep (delay float not null) returns float external name 'fbxci!fbsleep' engine udr;").Close()
	require.NoError(t, err)

	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = fbConn.Exec(ctx, "select 'Hello, world'x, fbsleep(1) from rdb$database").ReadAll()

	assert.Error(t, err)
	assert.True(t, fbconn.Timeout(err))
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.True(t, fbConn.IsClosed())

	select {
	case <-fbConn.CleanupDone():
	case <-time.After(5 * time.Second):
		t.Fatal("Connection cleanup exceeded maximum time")
	}
}

func TestConnExecContextPrecanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	cancel()
	res := fbConn.Exec(ctx, "select cast('Hello, world' as varchar(12)) from rdb$database").Read()
	assert.Error(t, res.Err)
	assert.True(t, errors.Is(res.Err, context.Canceled))
	assert.True(t, fbconn.SafeToRetry(res.Err))

	ensureConnValid(t, fbConn)
}

func TestConnExecParams(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	result := fbConn.ExecParams(ctx, "select cast(? as varchar(255)) as msg from rdb$database", [][]byte{[]byte("Hello, world")})
	require.Len(t, result.FieldDescriptions(), 1)
	assert.Equal(t, "MSG", result.FieldDescriptions()[0].Name)

	rowCount := 0
	for result.NextRow() {
		rowCount += 1
		assert.Equal(t, "Hello, world", string(result.Values()[0]))
	}
	assert.Equal(t, 1, rowCount)
	_, err = result.Close()
	assert.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestConnExecParamsExtraValuesError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	sql := "select cast(? as varchar(255)), cast(? as varchar(255)), cast(? as varchar(255)), cast(? as varchar(255)) from rdb$database"

	args := [][]byte{
		[]byte("0"),
		[]byte("1"),
		[]byte("2"),
		[]byte("3"),
		[]byte("4"), // one extra param
	}

	res := fbConn.ExecParams(ctx, sql, args).Read()
	require.NotNil(t, res.Err)

	var fbErr *fbconn.FbError
	require.True(t, errors.As(res.Err, &fbErr))
	assert.Equal(t, "07008", fbErr.Code)

	ensureConnValid(t, fbConn)
}

func TestConnExecParamsCanceled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	_, err = fbConn.Exec(ctx, "recreate function fbsleep (delay float not null) returns float external name 'fbxci!fbsleep' engine udr;").Close()
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	result := fbConn.ExecParams(ctx, "select mon$database_name, fbsleep(1) from mon$database", nil)
	rowCount := 0
	for result.NextRow() {
		rowCount += 1
	}
	assert.Equal(t, 0, rowCount)
	commandTag, err := result.Close()
	assert.Equal(t, fbconn.QueryStatus{}, commandTag)
	assert.True(t, fbconn.Timeout(err))
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	assert.True(t, fbConn.IsClosed())
	select {
	case <-fbConn.CleanupDone():
	case <-time.After(5 * time.Second):
		t.Fatal("Connection cleanup exceeded maximum time")
	}
}

func TestConnExecParamsPrecanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	cancel()
	result := fbConn.ExecParams(ctx, "select cast(? as varchar(255)) from rdb$database", [][]byte{[]byte("Hello, world")}).Read()
	require.Error(t, result.Err)
	assert.True(t, errors.Is(result.Err, context.Canceled))
	assert.True(t, fbconn.SafeToRetry(result.Err))

	ensureConnValid(t, fbConn)
}

func TestConnExecParamsEmptySQL(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	result := fbConn.ExecParams(ctx, "", nil).Read()
	assert.Equal(t, fbconn.QueryStatus{}, result.QueryStatus)
	assert.Len(t, result.Rows, 0)
	require.Error(t, result.Err)
	var fbErr *fbconn.FbError
	require.ErrorAs(t, result.Err, &fbErr)
	assert.Equal(t, "42000", fbErr.Code)

	ensureConnValid(t, fbConn)
}

func TestResultReaderValuesHaveSameCapacityAsLength(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	result := fbConn.ExecParams(ctx, "select cast(? as varchar(32)) as msg from rdb$database", [][]byte{[]byte("Hello, world")})
	require.Len(t, result.FieldDescriptions(), 1)
	assert.Equal(t, "MSG", result.FieldDescriptions()[0].Name)

	rowCount := 0
	for result.NextRow() {
		rowCount += 1
		assert.Equal(t, "Hello, world", string(result.Values()[0]))
		assert.Equal(t, len(result.Values()[0]), cap(result.Values()[0]))
	}
	assert.Equal(t, 1, rowCount)
	commandTag, err := result.Close()
	assert.Equal(t, "SELECT", commandTag.String())
	assert.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestResultReaderReadNil(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	result := fbConn.ExecParams(ctx, "select cast(null as varchar(255)) from rdb$database", nil).Read()
	require.NoError(t, result.Err)
	require.Nil(t, result.Rows[0][0])

	ensureConnValid(t, fbConn)
}

func TestConnExecPreparedMaxNumberOfParams(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	const chunkSize = 500

	buildSQL := func(n int) string {
		var sb strings.Builder
		sb.WriteString("select 0")

		for i := 0; i < n; i++ {
			if i%chunkSize == 0 {
				sb.WriteString("+(")
				sb.WriteString("cast(? as int)")
			} else {
				sb.WriteString("+cast(? as int)")
			}

			if i%chunkSize == chunkSize-1 || i == n-1 {
				sb.WriteString(")")
			}
		}

		sb.WriteString(" from rdb$database")
		return sb.String()
	}

	paramCount := int(math.MaxInt16)
	sql := buildSQL(paramCount)

	args := make([][]byte, paramCount)
	zero := fbio.AppendInt32LE(nil, 0)
	for i := range args {
		args[i] = zero
	}

	ps, err := fbConn.Prepare(ctx, sql)
	require.NoError(t, err)
	require.NotNil(t, ps)
	assert.Len(t, ps.Description.Parameters, paramCount)
	assert.Len(t, ps.Description.Fields, 1)

	result := ps.Exec(ctx, args).Read()
	require.NoError(t, result.Err)
	require.Len(t, result.Rows, 1)

	ensureConnValid(t, fbConn)
}

func TestConnExecParamsTooManyParams(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	const chunkSize = 500

	buildSQL := func(n int) string {
		var sb strings.Builder
		sb.WriteString("select 0")

		for i := 0; i < n; i++ {
			if i%chunkSize == 0 {
				sb.WriteString("+(")
				sb.WriteString("cast(? as int)")
			} else {
				sb.WriteString("+cast(? as int)")
			}

			if i%chunkSize == chunkSize-1 || i == n-1 {
				sb.WriteString(")")
			}
		}

		sb.WriteString(" from rdb$database")
		return sb.String()
	}

	paramCount := int(math.MaxInt16) + 1
	sql := buildSQL(paramCount)

	args := make([][]byte, paramCount)
	zero := fbio.AppendInt32LE(nil, 0)
	for i := range args {
		args[i] = zero
	}

	result := fbConn.ExecParams(ctx, sql, args).Read()
	require.Error(t, result.Err)
	var fbErr *fbconn.FbError
	require.ErrorAs(t, result.Err, &fbErr)
	require.True(t, fbErr.Code == "54023" || fbErr.Code == "54000", "unexpected SQLSTATE: %s (%s)", fbErr.Code, fbErr.Message)

	ensureConnValid(t, fbConn)
}

func TestConnExecPrepared(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	ps, err := fbConn.Prepare(ctx, "select cast(? as varchar(32)) as msg from rdb$database")
	require.NoError(t, err)
	require.NotNil(t, ps)
	assert.Len(t, ps.Description.Fields, 1)

	result := ps.Exec(ctx, [][]byte{[]byte("Hello, world")})
	require.Len(t, result.FieldDescriptions(), 1)
	assert.Equal(t, "MSG", result.FieldDescriptions()[0].Name)

	rowCount := 0
	for result.NextRow() {
		rowCount += 1
		assert.Equal(t, "Hello, world", string(result.Values()[0]))
	}
	assert.Equal(t, 1, rowCount)
	st, err := result.Close()
	//SELECT 1
	assert.Equal(t, "SELECT", st.String())
	assert.NoError(t, err)

	err = ps.Deallocate(ctx)
	assert.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestConnFetchAfterConnectionBreakReturnsFbError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	connString := os.Getenv("FBX_TEST_DATABASE")
	if connString == "" {
		t.Skip("Skipping due to missing environment variable FBX_TEST_DATABASE")
	}

	fbConn, err := fbconn.Connect(ctx, connString)
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	result := fbConn.Exec(ctx, "select cast(rdb$relation_id as integer) from rdb$relations")
	require.NotEmpty(t, result.FieldDescriptions())

	err = fbConn.Conn().Close()
	require.NoError(t, err)

	readResult := result.Read()
	require.Error(t, readResult.Err)

	var fbErr *fbconn.FbError
	require.ErrorAs(t, readResult.Err, &fbErr)
	require.Equal(t, "08006", fbErr.Code)
	require.True(t, fbConn.IsClosed())
}

func TestConnExecPreparedCanceled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	_, err = fbConn.Exec(ctx, "recreate function fbsleep (delay float not null) returns float external name 'fbxci!fbsleep' engine udr;").Close()
	require.NoError(t, err)

	ps, err := fbConn.Prepare(ctx, "select mon$database_name, fbsleep(1) from mon$database")
	require.NotNil(t, ps)
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	result := ps.Exec(ctx, nil)
	rowCount := 0
	for result.NextRow() {
		rowCount += 1
	}
	assert.Equal(t, 0, rowCount)
	commandTag, err := result.Close()
	assert.Equal(t, fbconn.QueryStatus{}, commandTag)
	assert.True(t, fbconn.Timeout(err))
	assert.True(t, fbConn.IsClosed())
	select {
	case <-fbConn.CleanupDone():
	case <-time.After(5 * time.Second):
		t.Fatal("Connection cleanup exceeded maximum time")
	}
}

func TestConnExecPreparedPrecanceled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	_, err = fbConn.Exec(ctx, "recreate function fbsleep (delay float not null) returns float external name 'fbxci!fbsleep' engine udr;").Close()
	require.NoError(t, err)

	ps, err := fbConn.Prepare(ctx, "select mon$database_name, fbsleep(1) from mon$database")
	require.NotNil(t, ps)
	require.NoError(t, err)

	cancel()
	result := ps.Exec(ctx, nil).Read()
	require.Error(t, result.Err)
	assert.True(t, errors.Is(result.Err, context.Canceled))
	assert.True(t, fbconn.SafeToRetry(result.Err))

	ensureConnValid(t, fbConn)
}

func TestConnExecPreparedEmptySQL(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	ps, err := fbConn.Prepare(ctx, "")
	require.Nil(t, ps)
	require.Error(t, err)
	var fbErr *fbconn.FbError
	require.ErrorAs(t, err, &fbErr)
	assert.Equal(t, "42000", fbErr.Code)

	ensureConnValid(t, fbConn)
}

func TestConnExecBatch(t *testing.T) {
	//t.Parallel()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	//defer cancel()
	//
	//fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	//require.NoError(t, err)
	//defer closeConn(t, fbConn)
	//
	//_, err = fbConn.Prepare(ctx, "ps1", "select $1::text", nil)
	//require.NoError(t, err)
	//
	//batch := &fbconn.Batch{}
	//
	//batch.ExecParams("select $1::text", [][]byte{[]byte("ExecParams 1")}, nil, nil, nil)
	//batch.ExecPrepared("ps1", [][]byte{[]byte("ExecPrepared 1")}, nil, nil)
	//batch.ExecParams("select $1::text", [][]byte{[]byte("ExecParams 2")}, nil, nil, nil)
	//results, err := fbConn.ExecBatch(ctx, batch).ReadAll()
	//require.NoError(t, err)
	//require.Len(t, results, 3)
	//
	//require.Len(t, results[0].Rows, 1)
	//require.Equal(t, "ExecParams 1", string(results[0].Rows[0][0]))
	//assert.Equal(t, "SELECT 1", results[0].QueryStatus.String())
	//
	//require.Len(t, results[1].Rows, 1)
	//require.Equal(t, "ExecPrepared 1", string(results[1].Rows[0][0]))
	//assert.Equal(t, "SELECT 1", results[1].QueryStatus.String())
	//
	//require.Len(t, results[2].Rows, 1)
	//require.Equal(t, "ExecParams 2", string(results[2].Rows[0][0]))
	//assert.Equal(t, "SELECT 1", results[2].QueryStatus.String())
}

//func TestConnExecBatchWriteError(t *testing.T) {
//	t.Skip("need implementation")
//	t.Parallel()
//
//	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
//	defer cancel()
//
//	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
//	require.NoError(t, err)
//
//	var mockConn mockConnection
//	writeLatency := 0 * time.Second
//	config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
//		conn, err := net.Dial(network, address)
//		mockConn = mockConnection{conn, &writeLatency}
//		return mockConn, err
//	}
//
//	fbConn, err := fbconn.ConnectConfig(ctx, config)
//	require.NoError(t, err)
//	defer closeConn(t, fbConn)
//
//	batch := &fbconn.Batch{}
//	fbConn.Conn()
//
//	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
//	defer cancel2()
//
//	batch.ExecParams("select $1::text", [][]byte{[]byte("ExecParams 1")}, nil, nil, nil)
//	writeLatency = 2 * time.Second
//	mrr := fbConn.ExecBatch(ctx2, batch)
//	err = mrr.Close()
//	require.Error(t, err)
//	assert.ErrorIs(t, err, context.DeadlineExceeded)
//	require.True(t, fbConn.IsClosed())
//}

//func TestConnExecBatchDeferredError(t *testing.T) {
//	t.Skip("need batch implementation")
//	t.Parallel()
//
//	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
//	defer cancel()
//
//	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
//	require.NoError(t, err)
//	defer closeConn(t, fbConn)
//
//	if fbConn.ParameterStatus("crdb_version") != "" {
//		t.Skip("Server does not support deferred constraint")
//	}
//
//	setupSQL := `create temporary table t (
//		id text primary key,
//		n int not null,
//		unique (n) deferrable initially deferred
//	);
//
//	insert into t (id, n) values ('a', 1), ('b', 2), ('c', 3);`
//
//	res := fbConn.Exec(ctx, setupSQL).Read()
//	require.NoError(t, res.Err)
//
//	batch := &fbconn.Batch{}
//
//	batch.ExecParams(`update t set n=n+1 where id='b' returning *`, nil, nil, nil, nil)
//	_, err = fbConn.ExecBatch(ctx, batch).ReadAll()
//	require.NotNil(t, err)
//	var fbErr *fbconn.FbError
//	require.True(t, errors.As(err, &fbErr))
//	require.Equal(t, "23505", fbErr.Code)
//
//	ensureConnValid(t, fbConn)
//}

//func TestConnExecBatchPrecanceled(t *testing.T) {
//t.Parallel()
//
//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
//defer cancel()
//
//fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
//require.NoError(t, err)
//defer closeConn(t, fbConn)
//
//_, err = fbConn.PrepareTx(ctx, "ps1", "select $1::text", nil)
//require.NoError(t, err)
//
//batch := &fbconn.Batch{}
//
//batch.ExecParams("select $1::text", [][]byte{[]byte("ExecParams 1")}, nil, nil, nil)
//batch.ExecPrepared("ps1", [][]byte{[]byte("ExecPrepared 1")}, nil, nil)
//batch.ExecParams("select $1::text", [][]byte{[]byte("ExecParams 2")}, nil, nil, nil)
//
//cancel()
//_, err = fbConn.ExecBatch(ctx, batch).ReadAll()
//require.Error(t, err)
//assert.True(t, errors.Is(err, context.Canceled))
//assert.True(t, fbconn.SafeToRetry(err))
//
//ensureConnValid(t, fbConn)
//}

// Without concurrent reading and writing large batches can deadlock.
//
//func TestConnExecBatchHuge(t *testing.T) {
//	if testing.Short() {
//		t.Skip("skipping test in short mode.")
//	}
//
//	t.Parallel()
//
//	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
//	defer cancel()
//
//	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
//	require.NoError(t, err)
//	defer closeConn(t, fbConn)
//
//	batch := &fbconn.Batch{}
//
//	queryCount := 100000
//	args := make([]string, queryCount)
//
//	for i := range args {
//		args[i] = strconv.Itoa(i)
//		batch.ExecParams("select $1::text", [][]byte{[]byte(args[i])}, nil, nil, nil)
//	}
//
//	results, err := fbConn.ExecBatch(ctx, batch).ReadAll()
//	require.NoError(t, err)
//	require.Len(t, results, queryCount)
//
//	for i := range args {
//		require.Len(t, results[i].Rows, 1)
//		require.Equal(t, args[i], string(results[i].Rows[0][0]))
//		assert.Equal(t, "SELECT 1", results[i].QueryStatus.String())
//	}
//}

//func TestConnExecBatchImplicitTransaction(t *testing.T) {
//t.Parallel()
//
//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
//defer cancel()
//
//fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
//require.NoError(t, err)
//defer closeConn(t, fbConn)
//
//if fbConn.ParameterStatus("crdb_version") != "" {
//	t.Skip()
//}
//
//_, err = fbConn.Exec(ctx, "create temporary table t(id int)").ReadAll()
//require.NoError(t, err)
//
//batch := &fbconn.Batch{}
//
//batch.ExecParams("insert into t(id) values(1)", nil, nil, nil, nil)
//batch.ExecParams("insert into t(id) values(2)", nil, nil, nil, nil)
//batch.ExecParams("insert into t(id) values(3)", nil, nil, nil, nil)
//batch.ExecParams("select 1/0", nil, nil, nil, nil)
//_, err = fbConn.ExecBatch(ctx, batch).ReadAll()
//require.Error(t, err)
//
//result := fbConn.ExecParams(ctx, "select count(*) from t", nil, nil, nil, nil).Read()
//require.Equal(t, "0", string(result.Rows[0][0]))
//}

func TestConnLocking(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	rr := fbConn.Exec(ctx, "select cast('Hello, world' as varchar(32)) from rdb$database")
	res := fbConn.Exec(ctx, "select cast('Hello, world' as varchar(32)) from rdb$database").Read()
	assert.Error(t, res.Err)
	assert.Equal(t, "conn busy", res.Err.Error())
	assert.True(t, fbconn.SafeToRetry(res.Err))

	result := rr.Read()
	assert.NoError(t, err)
	require.Nil(t, result.Err)
	assert.Len(t, result.Rows, 1)
	assert.Equal(t, "Hello, world", strings.TrimSpace(string(result.Rows[0][0])))

	ensureConnValid(t, fbConn)
}

func TestConnOnNotice(t *testing.T) {
	// 	t.Skip("need events implementation")
	// 	t.Parallel()

	// 	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	// 	defer cancel()

	// 	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	// 	require.NoError(t, err)

	// 	var notice *fbconn.Notice
	// 	config.OnNotice = func(c *fbconn.FbConn, n *fbconn.Notice) {
	// 		notice = n
	// 	}
	// 	config.RuntimeParams["client_min_messages"] = "notice" // Ensure we only get the message we expect.

	// 	fbConn, err := fbconn.ConnectConfig(ctx, config)
	// 	require.NoError(t, err)
	// 	defer closeConn(t, fbConn)

	// 	if fbConn.ParameterStatus("crdb_version") != "" {
	// 		t.Skip("")
	// 	}

	// 	rr := fbConn.Exec(ctx, `do $$
	// begin
	//   raise notice 'hello, world';
	// end$$;`)
	// 	res := rr.Read()
	// 	require.NoError(t, res.Err)
	// 	//assert.Equal(t, "NOTICE", notice.SeverityUnlocalized)
	// 	assert.Equal(t, "hello, world", notice.Message)

	// ensureConnValid(t, fbConn)
}

func TestConnOnNotification(t *testing.T) {
	//t.Parallel()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	//defer cancel()
	//
	//config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	//require.NoError(t, err)
	//
	//var msg string
	//config.OnNotification = func(c *fbconn.FbConn, n *fbconn.Notification) {
	//	msg = n.Payload
	//}
	//
	//fbConn, err := fbconn.ConnectConfig(ctx, config)
	//require.NoError(t, err)
	//defer closeConn(t, fbConn)
	//
	//if fbConn.ParameterStatus("crdb_version") != "" {
	//	t.Skip("Server does not support LISTEN / NOTIFY")
	//}
	//
	//_, err = fbConn.Exec(ctx, "listen foo").ReadAll()
	//require.NoError(t, err)
	//
	//notifier, err := fbconn.ConnectConfig(ctx, config)
	//require.NoError(t, err)
	//defer closeConn(t, notifier)
	//_, err = notifier.Exec(ctx, "notify foo, 'bar'").ReadAll()
	//require.NoError(t, err)
	//
	//_, err = fbConn.Exec(ctx, "select 1").ReadAll()
	//require.NoError(t, err)
	//
	//assert.Equal(t, "bar", msg)
	//
	//ensureConnValid(t, fbConn)
}

func TestConnWaitForNotification(t *testing.T) {
	//t.Parallel()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	//defer cancel()
	//
	//config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	//require.NoError(t, err)
	//
	//var msg string
	//config.OnNotification = func(c *fbconn.FbConn, n *fbconn.Notification) {
	//	msg = n.Payload
	//}
	//
	//fbConn, err := fbconn.ConnectConfig(ctx, config)
	//require.NoError(t, err)
	//defer closeConn(t, fbConn)
	//
	//if fbConn.ParameterStatus("crdb_version") != "" {
	//	t.Skip("Server does not support LISTEN / NOTIFY")
	//}
	//
	//_, err = fbConn.Exec(ctx, "listen foo").ReadAll()
	//require.NoError(t, err)
	//
	//notifier, err := fbconn.ConnectConfig(ctx, config)
	//require.NoError(t, err)
	//defer closeConn(t, notifier)
	//_, err = notifier.Exec(ctx, "notify foo, 'bar'").ReadAll()
	//require.NoError(t, err)
	//
	//err = fbConn.WaitForNotification(ctx)
	//require.NoError(t, err)
	//
	//assert.Equal(t, "bar", msg)
	//
	//ensureConnValid(t, fbConn)
}

func TestConnWaitForNotificationPrecanceled(t *testing.T) {
	//t.Parallel()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	//defer cancel()
	//
	//config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	//require.NoError(t, err)
	//
	//fbConn, err := fbconn.ConnectConfig(ctx, config)
	//require.NoError(t, err)
	//defer closeConn(t, fbConn)
	//
	//cancel()
	//err = fbConn.WaitForNotification(ctx)
	//require.ErrorIs(t, err, context.Canceled)
	//
	//ensureConnValid(t, fbConn)
}

func TestConnWaitForNotificationTimeout(t *testing.T) {
	//t.Parallel()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	//defer cancel()
	//
	//config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	//require.NoError(t, err)
	//
	//fbConn, err := fbconn.ConnectConfig(ctx, config)
	//require.NoError(t, err)
	//defer closeConn(t, fbConn)
	//
	//ctx, cancel = context.WithTimeout(ctx, 5*time.Millisecond)
	//err = fbConn.WaitForNotification(ctx)
	//cancel()
	//assert.True(t, fbconn.Timeout(err))
	//assert.ErrorIs(t, err, context.DeadlineExceeded)
	//
	//ensureConnValid(t, fbConn)
}

func TestConnEscapeString(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	tests := []struct {
		in  string
		out string
	}{
		{in: "", out: ""},
		{in: "42", out: "42"},
		{in: "'", out: "''"},
		{in: "''", out: "''''"},
		{in: "hi'there", out: "hi''there"},
		{in: "'hi there'", out: "''hi there''"},
		{in: "Hi, world! 👋", out: "Hi, world! 👋"},
		{in: "Na'vi", out: "Na''vi"},
	}

	for i, tt := range tests {
		value, err := fbConn.EscapeString(tt.in)
		if assert.NoErrorf(t, err, "%d.", i) {
			assert.Equalf(t, tt.out, value, "%d.", i)
		}
	}

	ensureConnValid(t, fbConn)
}

func TestConnCancelRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	_, err = fbConn.Exec(ctx, "recreate function fbsleep (delay float not null) returns float external name 'fbxci!fbsleep' engine udr;").Close()
	require.NoError(t, err)

	rr := fbConn.Exec(ctx, `select 'Hello, world', fbsleep(3) from (`+generateSeriesQuery(1, 10, "")+`)`)

	errChan := make(chan error)
	go func() {
		// The query is actually sent when multiResult.NextResult() is called. So wait to ensure it is sent.
		// Once Flush is available this could use that instead.
		time.Sleep(1 * time.Second)

		err := fbConn.CancelRequest(ctx)
		errChan <- err
	}()

	for rr.NextRow() {
	}
	require.False(t, fbConn.IsClosed(), "connection closed while receiving cancellation")
	res := rr.Read()

	var fbErr *fbconn.FbError
	require.ErrorAs(t, res.Err, &fbErr)
	require.Equal(t, "HY008", fbErr.Code)
	require.False(t, fbConn.IsClosed(), "cancelled query error: %v; GDS codes: %v", res.Err, fbErr.GdsCodes)

	err = <-errChan
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestConnContextCanceledCancelsRunningQueryOnServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)
	_, err = fbConn.Exec(ctx, "recreate function fbsleep (delay float not null) returns float external name 'fbxci!fbsleep' engine udr;").Close()
	require.NoError(t, err)

	t.Run("reddatabase", func(t *testing.T) {
		t.Parallel()

		testConnContextCanceledCancelsRunningQueryOnServer(t, os.Getenv("FBX_TEST_DATABASE"))
	})
}

func testConnContextCanceledCancelsRunningQueryOnServer(t *testing.T, connString string) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, connString)
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	// We tag the query text with a unique marker (queryID) and then, from a separate connection,
	// poll MON$STATEMENTS to ensure that the canceled statement is no longer running on the server.
	queryID := fmt.Sprintf("testConnContextCanceled %d", time.Now().UnixNano())

	rr := fbConn.Exec(ctx, fmt.Sprintf(`
	-- %v
	select 'Hello, world', fbsleep(1) from (`+generateSeriesQuery(1, 30, "n")+`)`, queryID))

	for rr.NextRow() {
	}
	res := rr.Read()
	assert.True(t, fbconn.Timeout(res.Err))
	assert.True(t, fbConn.IsClosed())
	select {
	case <-fbConn.CleanupDone():
	case <-time.After(5 * time.Second):
		t.Fatal("Connection cleanup exceeded maximum time")
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	otherConn, err := fbconn.Connect(ctx, connString)
	require.NoError(t, err)
	defer closeConn(t, otherConn)

	pollDeadline := time.Now().Add(15 * time.Second)
	for {
		if time.Now().After(pollDeadline) {
			t.Fatalf("canceled query marker %q is still active in MON$STATEMENTS", queryID)
		}

		pollCtx, pollCancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		result := otherConn.ExecParams(pollCtx,
			`select 1, cast(mon$sql_text as varchar(255))
			   from mon$statements
			  where mon$state = 1
			    and cast(mon$sql_text as varchar(255)) like ?`,
			[][]byte{[]byte("%" + queryID + "%")}).Read()
		pollCancel()
		require.NoError(t, result.Err)

		if len(result.Rows) == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestHijackAndConstruct(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	origConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	origAttachmentID := origConn.AttachmentID()
	origServerVersion := origConn.ServerVersion()

	err = origConn.SyncConn(ctx)
	require.NoError(t, err)

	hc, err := origConn.Hijack()
	require.NoError(t, err)
	require.Equal(t, origAttachmentID, hc.AttachmentID)
	require.Equal(t, origServerVersion, hc.ServerVersion)

	res := origConn.Exec(ctx, "select cast('Hello, world' as varchar(12)) from rdb$database").Read()
	require.Error(t, res.Err)

	newConn, err := fbconn.Construct(hc)
	require.NoError(t, err)
	require.Equal(t, origAttachmentID, newConn.AttachmentID())
	require.Equal(t, origServerVersion, newConn.ServerVersion())

	defer closeConn(t, newConn)

	result := newConn.Exec(ctx, "select cast('Hello, world' as varchar(12)) from rdb$database").Read()

	assert.Nil(t, result.Err)
	assert.Len(t, result.Rows, 1)
	assert.Equal(t, "Hello, world", string(result.Rows[0][0]))

	ensureConnValid(t, newConn)
}

func TestHijackFailsWithActiveTransaction(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	tx, err := fbConn.StartTransaction(ctx)
	require.NoError(t, err)

	_, err = fbConn.Hijack()
	require.Error(t, err, "Hijack must fail while a transaction is active")

	require.NoError(t, tx.Rollback(ctx))
	ensureConnValid(t, fbConn)
}

func TestHijackFailsWithPreparedStatementAfterTransactionCommit(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer func() {
		if fbConn != nil {
			closeConn(t, fbConn)
		}
	}()

	tx, err := fbConn.StartTransaction(ctx)
	require.NoError(t, err)

	st, err := fbConn.PrepareTx(ctx, tx, "select 1 from rdb$database")
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))
	require.NoError(t, fbConn.SyncConn(ctx))

	hc, err := fbConn.Hijack()
	if hc != nil {
		fbConn = nil
		reconstructed, constructErr := fbconn.Construct(hc)
		require.NoError(t, constructErr)
		closeConn(t, reconstructed)
	}
	require.Error(t, err, "Hijack must fail while a prepared statement is active")

	require.NoError(t, st.Deallocate(ctx))
	require.NoError(t, fbConn.SyncConn(ctx))

	hc, err = fbConn.Hijack()
	require.NoError(t, err)
	fbConn = nil

	reconstructed, err := fbconn.Construct(hc)
	require.NoError(t, err)
	closeConn(t, reconstructed)
}

func TestArrayHelperStatementsDoNotLeak(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	const iterations = 10

	_ = countIdleStatements(t, fbConn, ctx)
	baseline := countIdleStatements(t, fbConn, ctx)

	for i := 0; i < iterations; i++ {
		st, err := fbConn.Prepare(ctx, "select language_req from job")
		require.NoError(t, err)
		require.NoError(t, st.Deallocate(ctx))
	}

	after := countIdleStatements(t, fbConn, ctx)
	assert.Equal(t, baseline, after,
		"array helper statements must be deallocated; pre-fix leak was 2 handles per Prepare")
}

func countIdleStatements(t *testing.T, fbConn *fbconn.FbConn, ctx context.Context) int64 {
	t.Helper()
	res := fbConn.Exec(ctx, `SELECT COUNT(*) FROM MON$STATEMENTS
WHERE MON$ATTACHMENT_ID = CURRENT_CONNECTION AND MON$STATE = 0`).Read()
	require.NoError(t, res.Err)
	require.Len(t, res.Rows, 1)
	require.Len(t, res.Rows[0], 1)
	var v int64
	for _, b := range res.Rows[0][0] {
		v = (v << 8) | int64(b)
	}
	return v
}

func TestSyncConnOnIdleConn(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	require.NoError(t, fbConn.SyncConn(ctx))
	ensureConnValid(t, fbConn)
}

func TestHijackAfterExecParamsErrorDoesNotKeepAutoPreparedStatement(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer func() {
		if fbConn != nil {
			closeConn(t, fbConn)
		}
	}()

	sql := "select cast(? as varchar(255)), cast(? as varchar(255)) from rdb$database"
	args := [][]byte{
		[]byte("0"),
		[]byte("1"),
		[]byte("2"),
	}

	res := fbConn.ExecParams(ctx, sql, args).Read()
	require.Error(t, res.Err)

	var fbErr *fbconn.FbError
	require.True(t, errors.As(res.Err, &fbErr))
	require.Equal(t, "07008", fbErr.Code)

	require.NoError(t, fbConn.SyncConn(ctx))

	hc, err := fbConn.Hijack()
	require.NoError(t, err, "ExecParams error must not leave an unreachable prepared statement")
	fbConn = nil

	reconstructed, err := fbconn.Construct(hc)
	require.NoError(t, err)
	closeConn(t, reconstructed)
}

func TestConnCloseWhileCancellableQueryInProgress(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	fbConn.Exec(ctx, generateSeriesQuery(1, 10, "n"))

	closeCtx, closeCancel := context.WithCancel(ctx)
	defer closeCancel()
	require.NoError(t, fbConn.Close(closeCtx))
	select {
	case <-fbConn.CleanupDone():
	case <-time.After(5 * time.Second):
		t.Fatal("Connection cleanup exceeded maximum time")
	}
}

func TestConnCheckConn(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Intentionally using TCP connection for more predictable close behavior. (Not sure if Unix domain sockets would behave subtly different.)

	c1, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer func() { _ = c1.Close(ctx) }()

	err = c1.CheckConn()
	require.NoError(t, err)

	err = c1.Disconnect(ctx)
	require.NoError(t, err)

	for err == nil && ctx.Err() == nil {
		time.Sleep(50 * time.Millisecond)
		err = c1.CheckConn()
	}
	require.Error(t, err)
}

func TestConnPing(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	c1, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer func() { _ = c1.Close(ctx) }()

	_, err = c1.Exec(ctx, "select * from rdb$database").Close()
	require.NoError(t, err)

	err = c1.Ping(ctx)
	require.NoError(t, err)

	err = c1.Disconnect(ctx)
	require.NoError(t, err)

	deadline := time.Now().Add(5 * time.Second)
	for {
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		err = c1.Ping(pingCtx)
		pingCancel()

		if err != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expected Ping to fail after Disconnect, but it kept succeeding")
		}
		time.Sleep(50 * time.Millisecond)
	}

	require.Error(t, err)
}

func TestConnOnFbError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	config.OnFbError = func(c *fbconn.FbConn, fbErr *fbconn.FbError) bool {
		require.NotNil(t, c)
		require.NotNil(t, fbErr)
		// close connection on undefined tables only
		if fbErr.Code == "42S02" {
			return false
		}
		return true
	}

	fbConn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	res := fbConn.Exec(ctx, "select cast('Hello, world' as varchar(12)) from rdb$database").Read()
	assert.NoError(t, res.Err)
	assert.False(t, fbConn.IsClosed())

	res = fbConn.Exec(ctx, "select 1/0 from rdb$database").Read()
	assert.Error(t, res.Err)
	assert.False(t, fbConn.IsClosed())

	res = fbConn.Exec(ctx, "select * from non_existant_table").Read()
	assert.Error(t, res.Err)
	assert.True(t, fbConn.IsClosed())
}

func TestConnDefaultServerErrorKeepsConnectionOpen(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	require.NotNil(t, config.OnFbError)

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, conn)

	res := conn.Exec(ctx, "select 1/0 from rdb$database").Read()
	require.Error(t, res.Err)
	require.False(t, conn.IsClosed())

	res = conn.Exec(ctx, "select * from fbx_missing_table_rs_317500").Read()
	require.Error(t, res.Err)
	require.False(t, conn.IsClosed())

	require.NoError(t, conn.Ping(ctx))
}

func TestConnDefaultFatalServerErrorClosesConnection(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	require.NotNil(t, config.OnFbError)

	subject, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, subject)

	killer, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, killer)

	attachmentID := subjectAttachmentID(t, subject)
	res := killer.Exec(ctx, fmt.Sprintf(
		"delete from mon$attachments where mon$attachment_id = %d",
		attachmentID,
	)).Read()
	require.NoError(t, res.Err)

	err = subject.Ping(ctx)
	require.Error(t, err)

	var fbErr *fbconn.FbError
	require.ErrorAs(t, err, &fbErr)
	require.Contains(t, fbErr.GdsCodes, fbproto.Isc_att_shutdown)
	require.True(t, fbconn.IsConnectionFatal(fbErr))
	require.True(t, subject.IsClosed())
}

func TestConnFatalServerErrorCanBeOverridden(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)

	handlerCalled := false
	config.OnFbError = func(c *fbconn.FbConn, err *fbconn.FbError) bool {
		handlerCalled = true
		require.NotNil(t, c)
		require.True(t, fbconn.IsConnectionFatal(err))
		return true
	}

	subject, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, subject)

	killer, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, killer)

	attachmentID := subjectAttachmentID(t, subject)
	res := killer.Exec(ctx, fmt.Sprintf(
		"delete from mon$attachments where mon$attachment_id = %d",
		attachmentID,
	)).Read()
	require.NoError(t, res.Err)

	err = subject.Ping(ctx)
	require.Error(t, err)

	var fbErr *fbconn.FbError
	require.ErrorAs(t, err, &fbErr)
	require.Contains(t, fbErr.GdsCodes, fbproto.Isc_att_shutdown)
	require.True(t, fbconn.IsConnectionFatal(fbErr))
	require.True(t, handlerCalled)
	require.False(t, subject.IsClosed())
}

func TestConnCustomData(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	fbConn.CustomData()["foo"] = "bar"
	assert.Equal(t, "bar", fbConn.CustomData()["foo"])

	ensureConnValid(t, fbConn)
}

func Example() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	if err != nil {
		log.Fatalln(err)
	}
	defer func() { _ = fbConn.Close(ctx) }()

	result := fbConn.ExecParams(ctx, generateSeriesQuery(1, 3, ""), nil).Read()
	if result.Err != nil {
		log.Fatalln(result.Err)
	}

	for _, row := range result.Rows {
		fmt.Println(row[0][3])
	}
	fmt.Println(result.QueryStatus.String())

	// Output:
	// 1
	// 2
	// 3
	// SELECT
}

func GetSSLPassword(ctx context.Context) string {
	connString := os.Getenv("FBX_SSL_PASSWORD")
	return connString
}

func TestDeadlineContextWatcherHandler(t *testing.T) {
	t.Run("DeadlineExceeded with zero DeadlineDelay", func(t *testing.T) {
		config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)
		config.BuildContextWatcherHandler = func(conn *fbconn.FbConn) ctxwatch.Handler {
			return &fbconn.DeadlineContextWatcherHandler{Conn: conn.Conn()}
		}
		config.ConnectTimeout = 5 * time.Second

		fbConn, err := fbconn.ConnectConfig(context.Background(), config)
		require.NoError(t, err)
		defer closeConn(t, fbConn)

		setupCtx, setupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer setupCancel()
		_, err = fbConn.Exec(setupCtx, "recreate function fbsleep (delay float not null) returns float external name 'fbxci!fbsleep' engine udr;").Close()
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		res := fbConn.Exec(ctx, "select 1, fbsleep(1) from rdb$database").Read()
		require.Error(t, res.Err)
		require.ErrorIs(t, res.Err, context.DeadlineExceeded)
		require.True(t, fbConn.IsClosed())
	})

	t.Run("DeadlineExceeded with DeadlineDelay", func(t *testing.T) {
		config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)
		config.BuildContextWatcherHandler = func(conn *fbconn.FbConn) ctxwatch.Handler {
			return &fbconn.DeadlineContextWatcherHandler{Conn: conn.Conn(), DeadlineDelay: 500 * time.Millisecond}
		}
		config.ConnectTimeout = 5 * time.Second

		fbConn, err := fbconn.ConnectConfig(context.Background(), config)
		require.NoError(t, err)
		defer closeConn(t, fbConn)

		setupCtx, setupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer setupCancel()

		_, err = fbConn.Exec(setupCtx, "recreate function fbsleep (delay float not null) returns float external name 'fbxci!fbsleep' engine udr;").Close()
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		res := fbConn.Exec(ctx, "select 1, fbsleep(0.250) from rdb$database").Read()
		require.NoError(t, res.Err)

		ensureConnValid(t, fbConn)
	})
}

func TestCancelRequestContextWatcherHandler(t *testing.T) {
	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fbConn, err := fbconn.ConnectConfig(context.Background(), config)
	require.NoError(t, err)
	defer closeConn(t, fbConn)
	_, err = fbConn.Exec(ctx, "recreate function fbsleep (delay float not null) returns float external name 'fbxci!fbsleep' engine udr;").Close()
	require.NoError(t, err)

	t.Run("DeadlineExceeded cancels request after CancelRequestDelay", func(t *testing.T) {
		config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)
		config.BuildContextWatcherHandler = func(conn *fbconn.FbConn) ctxwatch.Handler {
			return &fbconn.CancelRequestContextWatcherHandler{
				Conn:               conn,
				CancelRequestDelay: 250 * time.Millisecond,
				DeadlineDelay:      5000 * time.Millisecond,
			}
		}
		config.ConnectTimeout = 5 * time.Second

		fbConn, err := fbconn.ConnectConfig(context.Background(), config)
		require.NoError(t, err)
		defer closeConn(t, fbConn)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		res := fbConn.Exec(ctx, "select 1, fbsleep(3) from rdb$database").Read()
		require.Error(t, res.Err)
		var fbErr *fbconn.FbError
		require.ErrorAs(t, res.Err, &fbErr)

		ensureConnValid(t, fbConn)
	})

	t.Run("DeadlineExceeded - do not send cancel request when query finishes in grace period", func(t *testing.T) {
		config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)
		config.BuildContextWatcherHandler = func(conn *fbconn.FbConn) ctxwatch.Handler {
			return &fbconn.CancelRequestContextWatcherHandler{
				Conn:               conn,
				CancelRequestDelay: 1000 * time.Millisecond,
				DeadlineDelay:      5000 * time.Millisecond,
			}
		}
		config.ConnectTimeout = 5 * time.Second

		fbConn, err := fbconn.ConnectConfig(context.Background(), config)
		require.NoError(t, err)
		defer closeConn(t, fbConn)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		res := fbConn.Exec(ctx, "select 1, fbsleep(0.250) from rdb$database").Read()
		require.NoError(t, res.Err)

		ensureConnValid(t, fbConn)
	})

	t.Run("DeadlineExceeded sets conn deadline with DeadlineDelay", func(t *testing.T) {
		config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)
		config.BuildContextWatcherHandler = func(conn *fbconn.FbConn) ctxwatch.Handler {
			return &fbconn.CancelRequestContextWatcherHandler{
				Conn:               conn,
				CancelRequestDelay: 5000 * time.Millisecond, // purposely setting this higher than DeadlineDelay to ensure the cancel request never happens.
				DeadlineDelay:      250 * time.Millisecond,
			}
		}
		config.ConnectTimeout = 5 * time.Second

		fbConn, err := fbconn.ConnectConfig(context.Background(), config)
		require.NoError(t, err)
		defer closeConn(t, fbConn)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		res := fbConn.Exec(ctx, "select 1, fbsleep(1) from rdb$database").Read()
		require.Error(t, res.Err)
		require.ErrorIs(t, res.Err, context.DeadlineExceeded)
		require.True(t, fbConn.IsClosed())
	})

	t.Run("Late cancel does not affect next query", func(t *testing.T) {
		config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
		require.NoError(t, err)
		config.BuildContextWatcherHandler = func(conn *fbconn.FbConn) ctxwatch.Handler {
			return &fbconn.CancelRequestContextWatcherHandler{
				Conn:               conn,
				CancelRequestDelay: 5 * time.Millisecond,
				DeadlineDelay:      1000 * time.Millisecond,
			}
		}
		config.ConnectTimeout = 5 * time.Second

		fbConn, err := fbconn.ConnectConfig(context.Background(), config)
		require.NoError(t, err)
		defer closeConn(t, fbConn)

		for i := 0; i < 20; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Millisecond)
			fbConn.Exec(ctx, "select 1, fbsleep(0.010) from rdb$database").Read()
			cancel()
			ensureConnValid(t, fbConn)
		}
	})

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("Stress %d", i), func(t *testing.T) {
			t.Parallel()

			config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
			require.NoError(t, err)
			config.BuildContextWatcherHandler = func(conn *fbconn.FbConn) ctxwatch.Handler {
				return &fbconn.CancelRequestContextWatcherHandler{
					Conn:               conn,
					CancelRequestDelay: 5 * time.Millisecond,
					DeadlineDelay:      1000 * time.Millisecond,
				}
			}
			config.ConnectTimeout = 5 * time.Second

			fbConn, err := fbconn.ConnectConfig(context.Background(), config)
			require.NoError(t, err)
			defer closeConn(t, fbConn)

			for i := 0; i < 20; i++ {
				func() {
					ctx, cancel := context.WithTimeout(context.Background(), 4*time.Millisecond)
					defer cancel()
					fbConn.Exec(ctx, "select 1, fbsleep(0.010) from rdb$database").Read()
					ensureConnValid(t, fbConn)
				}()
			}
		})
	}
}

func TestReadSingletonResult(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	_, err = fbConn.Exec(ctx, `
recreate procedure `+t.Name()+`(a int, b int) returns (c int)
as
begin
    c = a + b;
    suspend;
end
`).Close()
	require.NoError(t, err)

	res := fbConn.Exec(ctx, `execute procedure `+t.Name()+` 1, 2`).Read()
	require.NoError(t, res.Err)
	assert.Equal(t, 3, int(res.Rows[0][0][3]))

	_, err = fbConn.Exec(ctx, `
recreate procedure `+t.Name()+`(a int, b int)
as
begin
end
`).Close()
	require.NoError(t, err)

	res = fbConn.Exec(ctx, `execute procedure `+t.Name()+` 1, 2`).Read()
	require.NoError(t, res.Err)
	assert.Equal(t, 0, len(res.Rows))
}

func TestConnExecImmediate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	err = fbConn.ExecImmediate(ctx, "recreate table a(id int)")
	require.NoError(t, err)

	tx, err := fbConn.StartTransaction(ctx)
	require.NoError(t, err)

	err = fbConn.ExecImmediateTx(ctx, tx, "drop table a")
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestConnExecImmediateNoTx(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	err = fbConn.ExecImmediateTx(ctx, nil, "set transaction")
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestConnWithWarning(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var warnings string
	// Assume default database dialect is 3, but connect with dialect 1 to get warning
	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	dpb := fbproto.NewDPBWriterV2()
	dpb.PutInt32(fbproto.Isc_dpb_set_db_sql_dialect, 1)
	config.DPB = dpb.Bytes()
	config.OnFbWarning = func(c *fbconn.FbConn, warn *fbconn.FbWarning) {
		warnings += warn.Message + "SQLSTATE: " + warn.Code
	}

	fbConn, err := fbconn.ConnectConfig(ctx, config)
	assert.NoError(t, err)
	closeConn(t, fbConn)

	dpb = fbproto.NewDPBWriterV2()
	dpb.PutInt32(fbproto.Isc_dpb_set_db_sql_dialect, 3)
	config.DPB = dpb.Bytes()

	fbConn, err = fbconn.ConnectConfig(ctx, config)
	assert.NoError(t, err)
	closeConn(t, fbConn)

	assert.Equal(t, "Database dialect being changed from 3 to 1\nSQLSTATE: 01000", warnings)
}

func TestPassProcessName(t *testing.T) {
	exec, err := os.Executable()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	res := fbConn.Exec(ctx, "select mon$remote_process from mon$attachments where mon$attachment_id = current_connection").Read()
	assert.NoError(t, res.Err)

	require.Equal(t, 1, len(res.Rows))
	assert.Equal(t, exec, string(res.Rows[0][0]))
}

func TestRollbackInternalTxOnExecImmediateErrorAutocommit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	require.True(t, config.Autocommit, "default Autocommit must be true for this test")

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, conn)

	subjectID := subjectAttachmentID(t, conn)

	err = conn.ExecImmediate(ctx, "select * from fbx_table_does_not_exist_for_rollback_test")
	require.Error(t, err)

	assert.False(t, conn.ActiveTx(), "internal tx must be rolled back on error in autocommit mode")
	assert.Equal(t, 0, activeTxCountOnServer(t, subjectID),
		"server-side: no leaked tx after ExecImmediate failure")
}

func TestPreserveInternalTxOnExecImmediateErrorNoAutocommit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	config.Autocommit = false

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, conn)

	err = conn.ExecImmediate(ctx, "select * from fbx_table_does_not_exist_for_rollback_test")
	require.Error(t, err)

	assert.True(t, conn.ActiveTx(), "internal tx must persist on error when Autocommit is disabled")
}

func TestPrepareErrorReleasesInternalTxOnServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	require.True(t, config.Autocommit)

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, conn)

	subjectID := subjectAttachmentID(t, conn)

	_, err = conn.Prepare(ctx, "select * from fbx_no_such_table_prepare_err")
	require.Error(t, err)

	assert.False(t, conn.ActiveTx(), "Go-side: internal tx must be rolled back")
	assert.Equal(t, 0, activeTxCountOnServer(t, subjectID),
		"server-side: no leaked tx on subject after Prepare failure")
}

func TestExecParamsPrepareErrorReleasesInternalTxOnServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	require.True(t, config.Autocommit)

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, conn)

	subjectID := subjectAttachmentID(t, conn)

	res := conn.ExecParams(ctx, "select * from fbx_no_such_table_execparams_err", nil).Read()
	require.Error(t, res.Err)

	assert.False(t, conn.ActiveTx())
	assert.Equal(t, 0, activeTxCountOnServer(t, subjectID))
}

func TestExecParamsExecErrorReleasesInternalTxOnServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	require.True(t, config.Autocommit)

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, conn)

	const table = "fbx_dup_key_test"
	require.NoError(t, conn.ExecImmediate(ctx, "recreate table "+table+" (id integer not null primary key)"))
	defer func() { _ = conn.ExecImmediate(context.Background(), "drop table "+table) }()
	require.NoError(t, conn.ExecImmediate(ctx, "insert into "+table+" (id) values (1)"))

	subjectID := subjectAttachmentID(t, conn)

	res := conn.ExecParams(ctx, "insert into "+table+" (id) values (?)", [][]byte{{0, 0, 0, 1}}).Read()
	require.Error(t, res.Err)

	assert.False(t, conn.ActiveTx())
	assert.Equal(t, 0, activeTxCountOnServer(t, subjectID))
}

func TestOpenBlobErrorReleasesInternalTxOnServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	require.True(t, config.Autocommit)

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, conn)

	subjectID := subjectAttachmentID(t, conn)

	_, err = conn.OpenBlob(ctx, nil, 1234567890123456, nil, false)
	require.Error(t, err)

	assert.False(t, conn.ActiveTx())
	assert.Equal(t, 0, activeTxCountOnServer(t, subjectID))
}

func TestSuccessThenFailReleasesInternalTxOnServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	require.True(t, config.Autocommit)

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, conn)

	const table = "fbx_success_then_fail_test"
	require.NoError(t, conn.ExecImmediate(ctx, "recreate table "+table+" (id integer)"))
	defer func() { _ = conn.ExecImmediate(context.Background(), "drop table "+table) }()

	subjectID := subjectAttachmentID(t, conn)

	require.NoError(t, conn.ExecImmediate(ctx, "insert into "+table+" (id) values (42)"))
	require.False(t, conn.ActiveTx(), "internal tx must be committed after successful autocommit Exec")

	err = conn.ExecImmediate(ctx, "select * from fbx_no_such_table_combined_scenario")
	require.Error(t, err)

	assert.False(t, conn.ActiveTx(), "Go-side: internal tx of the failed call must be rolled back")
	assert.Equal(t, 0, activeTxCountOnServer(t, subjectID),
		"server-side: no leaked tx from either prior success or current failure")

	res := conn.Exec(ctx, "select cast(id as varchar(20)) from "+table).Read()
	require.NoError(t, res.Err)
	require.Len(t, res.Rows, 1)
	assert.Equal(t, "42", strings.TrimSpace(string(res.Rows[0][0])))
}

func TestRollbackInternalTxOnResultReaderCloseErrorAutocommit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	require.True(t, config.Autocommit, "default Autocommit must be true for this test")

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, conn)

	const table = "fbx_rr_rollback_test"
	require.NoError(t, conn.ExecImmediate(ctx, "recreate table "+table+"(id integer)"))
	defer func() {
		_ = conn.ExecImmediate(context.Background(), "drop table "+table)
	}()

	subjectID := subjectAttachmentID(t, conn)

	config.Autocommit = false
	require.NoError(t, conn.ExecImmediate(ctx, "insert into "+table+"(id) values (1)"))
	require.True(t, conn.ActiveTx(), "setup insert must remain in the internal tx")

	config.Autocommit = true
	rr := conn.ExecParams(ctx, "select 1 / cast(? as integer) from rdb$database", [][]byte{{0, 0, 0, 0}})
	for rr.NextRow() {
	}
	_, err = rr.Close()
	require.Error(t, err)

	res := conn.Exec(ctx, "select id from "+table).Read()
	require.NoError(t, res.Err)
	assert.Empty(t, res.Rows, "fetch-time error must roll back the implicit internal tx")

	assert.Equal(t, 0, activeTxCountOnServer(t, subjectID),
		"server-side: no leaked tx after fetch-time error and Close")
}
