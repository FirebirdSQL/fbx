package fbconn_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/red-soft/fbx/fbconn"
	"github.com/red-soft/fbx/fbconn/ctxwatch"
	"github.com/red-soft/fbx/fbtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pgbouncerConnStringEnvVar = "PGX_TEST_PGBOUNCER_CONN_STRING"

func TestConnect(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{"TCP", "PGX_TEST_TCP_CONN_STRING"},
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

func TestConnectWithOptions(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{"TCP", "PGX_TEST_TCP_CONN_STRING"},
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

type pgmockWaitStep time.Duration

//func (s pgmockWaitStep) Step(*pgproto3.Backend) error {
//	time.Sleep(time.Duration(s))
//	return nil
//}

//FIXME: fbx
//func TestConnectTimeout(t *testing.T) {
//	t.Parallel()
//	tests := []struct {
//		name    string
//		connect func(connStr string) error
//	}{
//		{
//			name: "via context that times out",
//			connect: func(connStr string) error {
//				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
//				defer cancel()
//				_, err := fbconn.Connect(ctx, connStr)
//				return err
//			},
//		},
//		{
//			name: "via config ConnectTimeout",
//			connect: func(connStr string) error {
//				conf, err := fbconn.ParseConfig(connStr)
//				require.NoError(t, err)
//				conf.ConnectTimeout = time.Microsecond * 50
//				_, err = fbconn.ConnectConfig(context.Background(), conf)
//				return err
//			},
//		},
//	}
//	for _, tt := range tests {
//		tt := tt
//		t.Run(tt.name, func(t *testing.T) {
//			t.Parallel()
//			script := &pgmock.Script{
//				Steps: []pgmock.Step{
//					pgmock.ExpectAnyMessage(&pgproto3.StartupMessage{ProtocolVersion: pgproto3.ProtocolVersionNumber, Parameters: map[string]string{}}),
//					pgmock.SendMessage(&pgproto3.AuthenticationOk{}),
//					pgmockWaitStep(time.Millisecond * 500),
//					pgmock.SendMessage(&pgproto3.BackendKeyData{ProcessID: 0, SecretKey: 0}),
//					pgmock.SendMessage(&pgproto3.ReadyForQuery{TxStatus: 'I'}),
//				},
//			}
//
//			ln, err := net.Listen("tcp", "127.0.0.1:")
//			require.NoError(t, err)
//			defer ln.Close()
//
//			serverErrChan := make(chan error, 1)
//			go func() {
//				defer close(serverErrChan)
//
//				conn, err := ln.Accept()
//				if err != nil {
//					serverErrChan <- err
//					return
//				}
//				defer conn.Close()
//
//				err = conn.SetDeadline(time.Now().Add(time.Millisecond * 450))
//				if err != nil {
//					serverErrChan <- err
//					return
//				}
//
//				err = script.Run(pgproto3.NewBackend(conn, conn))
//				if err != nil {
//					serverErrChan <- err
//					return
//				}
//			}()
//
//			host, port, _ := strings.Cut(ln.Addr().String(), ":")
//			connStr := fmt.Sprintf("sslmode=disable host=%s port=%s", host, port)
//			tooLate := time.Now().Add(time.Millisecond * 500)
//
//			err = tt.connect(connStr)
//			require.True(t, fbconn.Timeout(err), err)
//			require.True(t, time.Now().Before(tooLate))
//		})
//	}
//}

func TestConnectTimeoutStuckOnTLSHandshake(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		connect func(connStr string) error
	}{
		{
			name: "via context that times out",
			connect: func(connStr string) error {
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
				defer cancel()
				_, err := fbconn.Connect(ctx, connStr)
				return err
			},
		},
		{
			name: "via config ConnectTimeout",
			connect: func(connStr string) error {
				conf, err := fbconn.ParseConfig(connStr)
				require.NoError(t, err)
				conf.ConnectTimeout = time.Millisecond * 10
				_, err = fbconn.ConnectConfig(context.Background(), conf)
				return err
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ln, err := net.Listen("tcp", "127.0.0.1:")
			require.NoError(t, err)
			defer ln.Close()

			serverErrChan := make(chan error, 1)
			go func() {
				conn, err := ln.Accept()
				if err != nil {
					serverErrChan <- err
					return
				}
				defer conn.Close()

				var buf []byte
				_, err = conn.Read(buf)
				if err != nil {
					serverErrChan <- err
					return
				}

				// Sleeping to hang the TLS handshake.
				time.Sleep(time.Minute)
			}()

			host, port, _ := strings.Cut(ln.Addr().String(), ":")
			connStr := fmt.Sprintf("host=%s port=%s", host, port)

			errChan := make(chan error)
			go func() {
				err := tt.connect(connStr)
				errChan <- err
			}()

			select {
			case err = <-errChan:
				require.True(t, fbconn.Timeout(err), err)
			case err = <-serverErrChan:
				t.Fatalf("server failed with error: %s", err)
			case <-time.After(time.Millisecond * 500):
				t.Fatal("exceeded connection timeout without erroring out")
			}
		})
	}
}

func TestConnectInvalidUser(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	connString := os.Getenv("PGX_TEST_TCP_CONN_STRING")
	if connString == "" {
		t.Skipf("Skipping due to missing environment variable %v", "PGX_TEST_TCP_CONN_STRING")
	}

	config, err := fbconn.ParseConfig(connString)
	require.NoError(t, err)

	config.User = "pgxinvalidusertest"

	_, err = fbconn.ConnectConfig(ctx, config)
	require.Error(t, err)
	var pgErr *fbconn.PgError
	require.ErrorAs(t, err, &pgErr)
	if pgErr.Code != "28000" && pgErr.Code != "28P01" {
		t.Fatalf("Expected to receive a PgError with code 28000 or 28P01, instead received: %v", pgErr)
	}
}

func TestConnectWithConnectionRefused(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Presumably nothing is listening on 127.0.0.1:1
	conn, err := fbconn.Connect(ctx, "host=127.0.0.1 port=1")
	if err == nil {
		conn.Close(ctx)
		t.Fatal("Expected error establishing connection to bad port")
	}
}

func TestConnectCustomDialer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
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

func TestConnectCustomLookup(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	connString := os.Getenv("PGX_TEST_TCP_CONN_STRING")
	if connString == "" {
		t.Skipf("Skipping due to missing environment variable %v", "PGX_TEST_TCP_CONN_STRING")
	}

	config, err := fbconn.ParseConfig(connString)
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

	connString := os.Getenv("PGX_TEST_TCP_CONN_STRING")
	if connString == "" {
		t.Skipf("Skipping due to missing environment variable %v", "PGX_TEST_TCP_CONN_STRING")
	}

	config, err := fbconn.ParseConfig(connString)
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

	config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
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

	conn, err := fbconn.Connect(ctx, "host=localhost,127.0.0.1,foo.invalid port=1,2,3 sslmode=disable")
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

	config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)

	dialCount := 0
	config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		dialCount++
		return net.Dial(network, address)
	}

	acceptConnCount := 0
	config.ValidateConnect = func(ctx context.Context, conn *fbconn.FbConn) error {
		acceptConnCount++
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
}

func TestConnectWithValidateConnectTargetSessionAttrsReadWrite(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)

	config.ValidateConnect = fbconn.ValidateConnectTargetSessionAttrsReadWrite
	config.RuntimeParams["default_transaction_read_only"] = "on"

	conn, err := fbconn.ConnectConfig(ctx, config)
	if !assert.NotNil(t, err) {
		conn.Close(ctx)
	}
}

func TestConnectWithAfterConnect(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)

	//"set search_path to foobar;"
	config.AfterConnect = func(ctx context.Context, conn *fbconn.FbConn) error {
		res := conn.Exec(ctx, "select RDB$SET_CONTEXT('USER_TRANSACTION', 'search_path', 'foobar') from rdb$database").Read()
		return res.Err
	}

	conn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)

	res := conn.Exec(ctx, "select RDB$GET_CONTEXT('USER_TRANSACTION', 'search_path') from rdb$database search_path;").Read()
	require.NoError(t, res.Err)
	defer closeConn(t, conn)

	assert.Equal(t, []byte("foobar"), res.Rows[0][0])
}

func TestConnectConfigRequiresConfigFromParseConfig(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config := &fbconn.Config{}

	require.PanicsWithValue(t, "config must be created by ParseConfig", func() { fbconn.ConnectConfig(ctx, config) })
}

func TestConnPrepareSyntaxError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
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

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
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

func TestConnDeallocate(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
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
	require.Error(t, err)
	var pgErr *fbconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "26000", pgErr.Code)

	ensureConnValid(t, fbConn)
}

func TestConnDeallocateSucceedsInAbortedTransaction(t *testing.T) {
	//t.Parallel()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	//defer cancel()
	//
	//fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
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
	//var pgErr *fbconn.PgError
	//require.ErrorAs(t, err, &pgErr)
	//require.Equal(t, "22012", pgErr.Code)
	//
	//err = fbConn.Deallocate(ctx, "ps1")
	//require.NoError(t, err)
	//
	//err = fbConn.Exec(ctx, "rollback").Close()
	//require.NoError(t, err)
	//
	//_, err = fbConn.ExecPrepared(ctx, "ps1", nil, nil, nil).Close()
	//require.Error(t, err)
	//require.ErrorAs(t, err, &pgErr)
	//require.Equal(t, "26000", pgErr.Code)
	//
	//ensureConnValid(t, fbConn)
}

func TestConnDeallocateNonExistantStatementSucceeds(t *testing.T) {
	//t.Parallel()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	//defer cancel()
	//
	//fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
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

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	res := fbConn.Exec(ctx, "select 'Hello, world' from rdb$database").Read()
	assert.NoError(t, res.Err)

	assert.Len(t, res.Rows, 1)
	assert.Equal(t, "Hello, world", string(res.Rows[0][0]))

	results, err := fbConn.Exec(ctx, "select 'Hello, world' from rdb$database").ReadAll()
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
	//fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	//require.NoError(t, err)
	//defer closeConn(t, fbConn)
	//
	//if fbConn.ParameterStatus("crdb_version") != "" {
	//	t.Skip("Server does not support deferred constraint (https://github.com/cockroachdb/cockroach/issues/31632)")
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
	//var pgErr *fbconn.PgError
	//require.True(t, errors.As(err, &pgErr))
	//require.Equal(t, "23505", pgErr.Code)
	//
	//ensureConnValid(t, fbConn)
}

func TestConnExecContextCanceled(t *testing.T) {
	//t.Parallel()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	//defer cancel()
	//
	//fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	//require.NoError(t, err)
	//defer closeConn(t, fbConn)
	//cancel()
	//
	//ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	//defer cancel()
	//multiResult := fbConn.Exec(ctx, "select 'Hello, world', pg_sleep(1)")
	//
	//for multiResult.NextResult() {
	//}
	//err = multiResult.Close()
	//assert.True(t, fbconn.Timeout(err))
	//assert.ErrorIs(t, err, context.DeadlineExceeded)
	//assert.True(t, fbConn.IsClosed())
	//select {
	//case <-fbConn.CleanupDone():
	//case <-time.After(5 * time.Second):
	//	t.Fatal("Connection cleanup exceeded maximum time")
	//}
}

func TestConnExecContextPrecanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	cancel()
	res := fbConn.Exec(ctx, "select 'Hello, world' from rdb$database").Read()
	assert.Error(t, res.Err)
	assert.True(t, errors.Is(res.Err, context.Canceled))
	assert.True(t, fbconn.SafeToRetry(res.Err))

	ensureConnValid(t, fbConn)
}

func TestConnExecParams(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	result := fbConn.ExecParams(ctx, "select cast(? as varchar) as msg from rdb$database", [][]byte{[]byte("Hello, world")})
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

func TestConnExecParamsDeferredError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	if fbConn.ParameterStatus("crdb_version") != "" {
		t.Skip("Server does not support deferred constraint (https://github.com/cockroachdb/cockroach/issues/31632)")
	}

	setupSQL := `create temporary table t (
		id text primary key,
		n int not null,
		unique (n) deferrable initially deferred
	);

	insert into t (id, n) values ('a', 1), ('b', 2), ('c', 3);`

	res := fbConn.Exec(ctx, setupSQL).Read()
	assert.NoError(t, res.Err)

	result := fbConn.ExecParams(ctx, `update t set n=n+1 where id='b' returning *`, nil).Read()
	require.NotNil(t, result.Err)
	var pgErr *fbconn.PgError
	require.True(t, errors.As(result.Err, &pgErr))
	require.Equal(t, "23505", pgErr.Code)

	ensureConnValid(t, fbConn)
}

func TestConnExecParamsMaxNumberOfParams(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	paramCount := math.MaxUint16
	params := make([]string, 0, paramCount)
	args := make([][]byte, 0, paramCount)
	for i := 0; i < paramCount; i++ {
		params = append(params, fmt.Sprintf("($%d::text)", i+1))
		args = append(args, []byte(strconv.Itoa(i)))
	}
	sql := "values" + strings.Join(params, ", ")

	result := fbConn.ExecParams(ctx, sql, args).Read()
	require.NoError(t, result.Err)
	require.Len(t, result.Rows, paramCount)

	ensureConnValid(t, fbConn)
}

func TestConnExecParamsTooManyParams(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	paramCount := math.MaxUint16 + 1
	params := make([]string, 0, paramCount)
	args := make([][]byte, 0, paramCount)
	for i := 0; i < paramCount; i++ {
		params = append(params, fmt.Sprintf("($%d::text)", i+1))
		args = append(args, []byte(strconv.Itoa(i)))
	}
	sql := "values" + strings.Join(params, ", ")

	result := fbConn.ExecParams(ctx, sql, args).Read()
	require.Error(t, result.Err)
	require.Equal(t, "extended protocol limited to 65535 parameters", result.Err.Error())

	ensureConnValid(t, fbConn)
}

func TestConnExecParamsCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	result := fbConn.ExecParams(ctx, "select current_database(), pg_sleep(1)", nil)
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

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	cancel()
	result := fbConn.ExecParams(ctx, "select $1::text", [][]byte{[]byte("Hello, world")}).Read()
	require.Error(t, result.Err)
	assert.True(t, errors.Is(result.Err, context.Canceled))
	assert.True(t, fbconn.SafeToRetry(result.Err))

	ensureConnValid(t, fbConn)
}

func TestConnExecParamsEmptySQL(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	result := fbConn.ExecParams(ctx, "", nil).Read()
	assert.Equal(t, fbconn.QueryStatus{}, result.QueryStatus)
	assert.Len(t, result.Rows, 0)
	assert.NoError(t, result.Err)

	ensureConnValid(t, fbConn)
}

// https://github.com/jackc/pgx/issues/859
func TestResultReaderValuesHaveSameCapacityAsLength(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	result := fbConn.ExecParams(ctx, "select $1::text as msg", [][]byte{[]byte("Hello, world")})
	require.Len(t, result.FieldDescriptions(), 1)
	assert.Equal(t, "msg", result.FieldDescriptions()[0].Name)

	rowCount := 0
	for result.NextRow() {
		rowCount += 1
		assert.Equal(t, "Hello, world", string(result.Values()[0]))
		assert.Equal(t, len(result.Values()[0]), cap(result.Values()[0]))
	}
	assert.Equal(t, 1, rowCount)
	commandTag, err := result.Close()
	assert.Equal(t, "SELECT 1", commandTag.String())
	assert.NoError(t, err)

	ensureConnValid(t, fbConn)
}

// https://github.com/jackc/pgx/issues/1987
func TestResultReaderReadNil(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	result := fbConn.ExecParams(ctx, "select cast(null as varchar) from rdb$database", nil).Read()
	require.NoError(t, result.Err)
	require.Nil(t, result.Rows[0][0])

	ensureConnValid(t, fbConn)
}

func TestConnExecPrepared(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	// "select $1::text as msg"
	ps, err := fbConn.Prepare(ctx, "select cast(? as varchar(32)) as msg from rdb$database")
	require.NoError(t, err)
	require.NotNil(t, ps)
	//assert.Len(t, psd.ParamOIDs, 1)
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

func TestConnExecPreparedMaxNumberOfParams(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	paramCount := math.MaxUint16
	params := make([]string, 0, paramCount)
	args := make([][]byte, 0, paramCount)
	for i := 0; i < paramCount; i++ {
		params = append(params, fmt.Sprintf("($%d::text)", i+1))
		args = append(args, []byte(strconv.Itoa(i)))
	}
	sql := "values" + strings.Join(params, ", ")

	ps, err := fbConn.Prepare(ctx, sql)
	require.NoError(t, err)
	require.NotNil(t, ps)
	assert.Len(t, ps.Description.Parameters, paramCount)
	assert.Len(t, ps.Description.Fields, 1)

	result := ps.Exec(ctx, args).Read()
	require.NoError(t, result.Err)
	require.Len(t, result.Rows, paramCount)

	ensureConnValid(t, fbConn)
}

func TestConnExecPreparedTooManyParams(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	paramCount := math.MaxUint16 + 1
	params := make([]string, 0, paramCount)
	args := make([][]byte, 0, paramCount)
	for i := 0; i < paramCount; i++ {
		params = append(params, fmt.Sprintf("($%d::text)", i+1))
		args = append(args, []byte(strconv.Itoa(i)))
	}
	sql := "values" + strings.Join(params, ", ")

	ps, err := fbConn.Prepare(ctx, sql)
	if fbConn.ParameterStatus("crdb_version") != "" {
		// CockroachDB rejects preparing a statement with more than 65535 parameters.
		require.EqualError(t, err, "ERROR: more than 65535 arguments to prepared statement: 65536 (SQLSTATE 08P01)")
	} else {
		// PostgreSQL accepts preparing a statement with more than 65535 parameters and only fails when executing it through the extended protocol.
		require.NoError(t, err)
		require.NotNil(t, ps)
		assert.Len(t, ps.Description.Parameters, paramCount)
		assert.Len(t, ps.Description.Fields, 1)

		result := ps.Exec(ctx, args).Read()
		require.EqualError(t, result.Err, "extended protocol limited to 65535 parameters")
	}

	ensureConnValid(t, fbConn)
}

func TestConnExecPreparedCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	ps, err := fbConn.Prepare(ctx, "select current_database(), pg_sleep(1)")
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
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	ps, err := fbConn.Prepare(ctx, "select current_database(), pg_sleep(1)")
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

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	ps, err := fbConn.Prepare(ctx, "")
	require.NotNil(t, ps)
	require.NoError(t, err)

	result := ps.Exec(ctx, nil).Read()
	assert.Equal(t, fbconn.QueryStatus{}, result.QueryStatus)
	assert.Len(t, result.Rows, 0)
	assert.NoError(t, result.Err)

	ensureConnValid(t, fbConn)
}

func TestConnExecBatch(t *testing.T) {
	//t.Parallel()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	//defer cancel()
	//
	//fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
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

type mockConnection struct {
	net.Conn
	writeLatency *time.Duration
}

func (m mockConnection) Write(b []byte) (n int, err error) {
	time.Sleep(*m.writeLatency)
	return m.Conn.Write(b)
}

//func TestConnExecBatchWriteError(t *testing.T) {
//	t.Skip("need implementation")
//	t.Parallel()
//
//	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
//	defer cancel()
//
//	config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
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
//	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
//	require.NoError(t, err)
//	defer closeConn(t, fbConn)
//
//	if fbConn.ParameterStatus("crdb_version") != "" {
//		t.Skip("Server does not support deferred constraint (https://github.com/cockroachdb/cockroach/issues/31632)")
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
//	var pgErr *fbconn.PgError
//	require.True(t, errors.As(err, &pgErr))
//	require.Equal(t, "23505", pgErr.Code)
//
//	ensureConnValid(t, fbConn)
//}

//func TestConnExecBatchPrecanceled(t *testing.T) {
//t.Parallel()
//
//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
//defer cancel()
//
//fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
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
// See https://github.com/jackc/pgx/issues/374.
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
//	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
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
//fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
//require.NoError(t, err)
//defer closeConn(t, fbConn)
//
//if fbConn.ParameterStatus("crdb_version") != "" {
//	t.Skip("Skipping due to known server issue: (https://github.com/cockroachdb/cockroach/issues/44803)")
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

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	rr := fbConn.Exec(ctx, "select 'Hello, world' from rdb$database")
	res := fbConn.Exec(ctx, "select 'Hello, world' from rdb$database").Read()
	assert.Error(t, res.Err)
	assert.Equal(t, "conn busy", res.Err.Error())
	assert.True(t, fbconn.SafeToRetry(res.Err))

	result := rr.Read()
	assert.NoError(t, err)
	require.Nil(t, result.Err)
	assert.Len(t, result.Rows, 1)
	assert.Equal(t, "Hello, world", string(result.Rows[0][0]))

	ensureConnValid(t, fbConn)
}

func TestConnOnNotice(t *testing.T) {
	t.Skip("need events implementation")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)

	var notice *fbconn.Notice
	config.OnNotice = func(c *fbconn.FbConn, n *fbconn.Notice) {
		notice = n
	}
	config.RuntimeParams["client_min_messages"] = "notice" // Ensure we only get the message we expect.

	fbConn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	if fbConn.ParameterStatus("crdb_version") != "" {
		t.Skip("Server does not support PL/PGSQL (https://github.com/cockroachdb/cockroach/issues/17511)")
	}

	rr := fbConn.Exec(ctx, `do $$
begin
  raise notice 'hello, world';
end$$;`)
	res := rr.Read()
	require.NoError(t, res.Err)
	assert.Equal(t, "NOTICE", notice.SeverityUnlocalized)
	assert.Equal(t, "hello, world", notice.Message)

	ensureConnValid(t, fbConn)
}

func TestConnOnNotification(t *testing.T) {
	//t.Parallel()
	//
	//ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	//defer cancel()
	//
	//config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
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
	//	t.Skip("Server does not support LISTEN / NOTIFY (https://github.com/cockroachdb/cockroach/issues/41522)")
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
	//config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
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
	//	t.Skip("Server does not support LISTEN / NOTIFY (https://github.com/cockroachdb/cockroach/issues/41522)")
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
	//config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
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
	//config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
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

type delayedReader struct {
	r io.Reader
}

func (d delayedReader) Read(p []byte) (int, error) {
	// W/o sleep test passes, with sleep it fails.
	time.Sleep(time.Millisecond)
	return d.r.Read(p)
}

// https://github.com/jackc/fbconn/issues/128
func TestConnCopyFromDataWriteAfterErrorAndReturn(t *testing.T) {
	t.Skip("PG only")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	connString := os.Getenv("PGX_TEST_DATABASE")
	if connString == "" {
		t.Skipf("Skipping due to missing environment variable %v", "PGX_TEST_DATABASE")
	}

	config, err := fbconn.ParseConfig(connString)
	require.NoError(t, err)

	fbConn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)

	if fbConn.ParameterStatus("crdb_version") != "" {
		t.Skip("Server does not fully support COPY FROM")
	}

	setupSQL := `create temporary table t (
		id text primary key,
		n int not null
	);`

	res := fbConn.Exec(ctx, setupSQL).Read()
	assert.NoError(t, res.Err)

	r1 := delayedReader{r: strings.NewReader(`id	0\n`)}
	// Generate an error with a bogus COPY command
	_, err = fbConn.CopyFrom(ctx, r1, "COPY nosuchtable FROM STDIN ")
	assert.Error(t, err)

	r2 := delayedReader{r: strings.NewReader(`id	0\n`)}
	_, err = fbConn.CopyFrom(ctx, r2, "COPY t FROM STDIN")
	assert.NoError(t, err)
}

func TestConnEscapeString(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	tests := []struct {
		in  string
		out string
	}{
		{in: "", out: ""},
		{in: "42", out: "42"},
		{in: "'", out: "''"},
		{in: "hi'there", out: "hi''there"},
		{in: "'hi there'", out: "''hi there''"},
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
	t.Skip("need implementation")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	if fbConn.ParameterStatus("crdb_version") != "" {
		t.Skip("Server does not support query cancellation (https://github.com/cockroachdb/cockroach/issues/41335)")
	}

	rr := fbConn.Exec(ctx, "select 'Hello, world', pg_sleep(25)")

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
	res := rr.Read()

	require.IsType(t, &fbconn.PgError{}, res.Err)
	require.Equal(t, "57014", err.(*fbconn.PgError).Code)

	err = <-errChan
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

// https://github.com/jackc/pgx/issues/659
func TestConnContextCanceledCancelsRunningQueryOnServer(t *testing.T) {
	t.Parallel()

	t.Run("postgres", func(t *testing.T) {
		t.Parallel()

		testConnContextCanceledCancelsRunningQueryOnServer(t, os.Getenv("PGX_TEST_DATABASE"), "postgres")
	})

	t.Run("pgbouncer", func(t *testing.T) {
		t.Parallel()

		connString := os.Getenv(pgbouncerConnStringEnvVar)
		if connString == "" {
			t.Skipf("Skipping due to missing environment variable %v", pgbouncerConnStringEnvVar)
		}

		testConnContextCanceledCancelsRunningQueryOnServer(t, connString, "pgbouncer")
	})
}

func testConnContextCanceledCancelsRunningQueryOnServer(t *testing.T, connString, dbType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, connString)
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	// Getting the actual PostgreSQL server process ID (PID) from a query executed through pgbouncer is not straightforward
	// because pgbouncer abstracts the underlying database connections, and it doesn't expose the PID of the PostgreSQL
	// server process to clients. However, we can check if the query is running by checking the generated query ID.
	queryID := fmt.Sprintf("%s testConnContextCanceled %d", dbType, time.Now().UnixNano())

	rr := fbConn.Exec(ctx, fmt.Sprintf(`
	-- %v
	select 'Hello, world', pg_sleep(30)
	`, queryID))

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

	ctx, cancel = context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	for {
		result := otherConn.ExecParams(ctx,
			`select 1 from pg_stat_activity where query like $1`,
			[][]byte{[]byte("%" + queryID + "%")}).Read()
		require.NoError(t, result.Err)

		if len(result.Rows) == 0 {
			break
		}
	}
}

func TestHijackAndConstruct(t *testing.T) {
	t.Skip("need implementation?")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	origConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)

	err = origConn.SyncConn(ctx)
	require.NoError(t, err)

	hc, err := origConn.Hijack()
	require.NoError(t, err)

	res := origConn.Exec(ctx, "select 'Hello, world'").Read()
	require.Error(t, res.Err)

	newConn, err := fbconn.Construct(hc)
	require.NoError(t, err)

	defer closeConn(t, newConn)

	result := newConn.Exec(ctx, "select 'Hello, world'").Read()

	assert.Nil(t, result.Err)
	assert.Len(t, result.Rows, 1)
	assert.Equal(t, "Hello, world", string(result.Rows[0][0]))

	ensureConnValid(t, newConn)
}

func TestConnCloseWhileCancellableQueryInProgress(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)

	fbConn.Exec(ctx, generateSeriesQuery(1, 10, "n"))

	closeCtx, closeCancel := context.WithCancel(ctx)
	defer closeCancel()
	fbConn.Close(closeCtx)
	select {
	case <-fbConn.CleanupDone():
	case <-time.After(5 * time.Second):
		t.Fatal("Connection cleanup exceeded maximum time")
	}
}

// FIXME: fbx
//// https://github.com/jackc/pgx/issues/800
//func TestFatalErrorReceivedAfterCommandComplete(t *testing.T) {
//	t.Parallel()
//
//	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
//	defer cancel()
//
//	steps := pgmock.AcceptUnauthenticatedConnRequestSteps()
//	steps = append(steps, pgmock.ExpectAnyMessage(&pgproto3.Parse{}))
//	steps = append(steps, pgmock.ExpectAnyMessage(&pgproto3.Bind{}))
//	steps = append(steps, pgmock.ExpectAnyMessage(&pgproto3.Describe{}))
//	steps = append(steps, pgmock.ExpectAnyMessage(&pgproto3.Execute{}))
//	steps = append(steps, pgmock.ExpectAnyMessage(&pgproto3.Sync{}))
//	steps = append(steps, pgmock.SendMessage(&pgproto3.RowDescription{Fields: []pgproto3.FieldDescription{
//		{Name: []byte("mock")},
//	}}))
//	steps = append(steps, pgmock.SendMessage(&pgproto3.CommandComplete{QueryStatus: []byte("SELECT 0")}))
//	steps = append(steps, pgmock.SendMessage(&pgproto3.ErrorResponse{Severity: "FATAL", Code: "57P01"}))
//
//	script := &pgmock.Script{Steps: steps}
//
//	ln, err := net.Listen("tcp", "127.0.0.1:")
//	require.NoError(t, err)
//	defer ln.Close()
//
//	serverErrChan := make(chan error, 1)
//	go func() {
//		defer close(serverErrChan)
//
//		conn, err := ln.Accept()
//		if err != nil {
//			serverErrChan <- err
//			return
//		}
//		defer conn.Close()
//
//		err = conn.SetDeadline(time.Now().Add(5 * time.Second))
//		if err != nil {
//			serverErrChan <- err
//			return
//		}
//
//		err = script.Run(pgproto3.NewBackend(conn, conn))
//		if err != nil {
//			serverErrChan <- err
//			return
//		}
//	}()
//
//	host, port, _ := strings.Cut(ln.Addr().String(), ":")
//	connStr := fmt.Sprintf("sslmode=disable host=%s port=%s", host, port)
//
//	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
//	defer cancel()
//	conn, err := fbconn.Connect(ctx, connStr)
//	require.NoError(t, err)
//
//	rr := conn.ExecParams(ctx, "mocked...", nil, nil, nil, nil)
//
//	for rr.NextRow() {
//	}
//
//	_, err = rr.Close()
//	require.Error(t, err)
//}

// https://github.com/jackc/fbconn/issues/27
func TestConnLargeResponseWhileWritingDoesNotDeadlock(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	res := fbConn.Exec(ctx, "set client_min_messages = debug5").Read()
	require.NoError(t, res.Err)

	// The actual contents of this test aren't important. What's important is a large amount of data to be written and
	// because of client_min_messages = debug5 the server will return a large amount of data.

	paramCount := math.MaxUint16
	params := make([]string, 0, paramCount)
	args := make([][]byte, 0, paramCount)
	for i := 0; i < paramCount; i++ {
		params = append(params, fmt.Sprintf("($%d::text)", i+1))
		args = append(args, []byte(strconv.Itoa(i)))
	}
	sql := "values" + strings.Join(params, ", ")

	result := fbConn.ExecParams(ctx, sql, args).Read()
	require.NoError(t, result.Err)
	require.Len(t, result.Rows, paramCount)

	ensureConnValid(t, fbConn)
}

func TestConnCheckConn(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Intentionally using TCP connection for more predictable close behavior. (Not sure if Unix domain sockets would behave subtly different.)

	connString := os.Getenv("PGX_TEST_TCP_CONN_STRING")
	if connString == "" {
		t.Skipf("Skipping due to missing environment variable %v", "PGX_TEST_TCP_CONN_STRING")
	}

	c1, err := fbconn.Connect(ctx, connString)
	require.NoError(t, err)
	defer c1.Close(ctx)

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

	// Intentionally using TCP connection for more predictable close behavior. (Not sure if Unix domain sockets would behave subtly different.)

	connString := os.Getenv("PGX_TEST_TCP_CONN_STRING")
	if connString == "" {
		t.Skipf("Skipping due to missing environment variable %v", "PGX_TEST_TCP_CONN_STRING")
	}

	c1, err := fbconn.Connect(ctx, connString)
	require.NoError(t, err)
	defer c1.Close(ctx)

	if c1.ParameterStatus("crdb_version") != "" {
		t.Skip("Server does not support pg_terminate_backend() (https://github.com/cockroachdb/cockroach/issues/35897)")
	}

	_, err = c1.Exec(ctx, "select * from rdb$database").Close()
	require.NoError(t, err)

	err = c1.Ping(ctx)
	require.NoError(t, err)

	err = c1.Disconnect(ctx)
	require.NoError(t, err)

	// Give a little time for the signal to actually kill the backend.
	time.Sleep(500 * time.Millisecond)

	err = c1.Ping(ctx)
	require.Error(t, err)
}

func TestPipelinePrepare(t *testing.T) {
	t.Skip("PG only")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	result := fbConn.ExecParams(ctx, `create temporary table t (id text primary key)`, nil).Read()
	require.NoError(t, result.Err)

	pipeline := fbConn.StartPipeline(ctx)
	pipeline.SendPrepare("selectInt", "select $1::bigint as a", nil)
	pipeline.SendPrepare("selectText", "select $1::text as b", nil)
	pipeline.SendPrepare("selectNoParams", "select 42 as c", nil)
	pipeline.SendPrepare("insertNoResults", "insert into t (id) values ($1)", nil)
	pipeline.SendPrepare("insertNoParamsOrResults", "insert into t (id) values ('foo')", nil)
	err = pipeline.Sync()
	require.NoError(t, err)

	results, err := pipeline.GetResults()
	require.NoError(t, err)
	sd, ok := results.(*fbconn.StatementDescription)
	require.Truef(t, ok, "expected StatementDescription, got: %#v", results)
	require.Len(t, sd.Fields, 1)
	require.Equal(t, "a", string(sd.Fields[0].Name))
	require.Equal(t, []uint32{fbtype.Int8OID}, sd.Parameters[0].Type)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	sd, ok = results.(*fbconn.StatementDescription)
	require.Truef(t, ok, "expected StatementDescription, got: %#v", results)
	require.Len(t, sd.Fields, 1)
	require.Equal(t, "b", string(sd.Fields[0].Name))
	require.Equal(t, []uint32{fbtype.TextOID}, sd.Parameters[0].Type)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	sd, ok = results.(*fbconn.StatementDescription)
	require.Truef(t, ok, "expected StatementDescription, got: %#v", results)
	require.Len(t, sd.Fields, 1)
	require.Equal(t, "c", string(sd.Fields[0].Name))
	require.Len(t, sd.Parameters, 0)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	sd, ok = results.(*fbconn.StatementDescription)
	require.Truef(t, ok, "expected StatementDescription, got: %#v", results)
	require.Len(t, sd.Fields, 0)
	require.Equal(t, []uint32{fbtype.TextOID}, sd.Parameters[0].Type)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	sd, ok = results.(*fbconn.StatementDescription)
	require.Truef(t, ok, "expected StatementDescription, got: %#v", results)
	require.Len(t, sd.Fields, 0)
	require.Len(t, sd.Parameters, 0)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	err = pipeline.Close()
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestPipelinePrepareError(t *testing.T) {
	t.Skip("PG only")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	pipeline := fbConn.StartPipeline(ctx)
	pipeline.SendPrepare("selectInt", "select $1::bigint as a", nil)
	pipeline.SendPrepare("selectError", "bad", nil)
	pipeline.SendPrepare("selectText", "select $1::text as b", nil)
	err = pipeline.Sync()
	require.NoError(t, err)

	results, err := pipeline.GetResults()
	require.NoError(t, err)
	sd, ok := results.(*fbconn.StatementDescription)
	require.Truef(t, ok, "expected StatementDescription, got: %#v", results)
	require.Len(t, sd.Fields, 1)
	require.Equal(t, "a", string(sd.Fields[0].Name))
	require.Equal(t, []uint32{fbtype.Int8OID}, sd.Parameters[0].Type)

	results, err = pipeline.GetResults()
	var pgErr *fbconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Nil(t, results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	err = pipeline.Close()
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestPipelinePrepareAndDeallocate(t *testing.T) {
	t.Skip("PG only")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	pipeline := fbConn.StartPipeline(ctx)
	pipeline.SendPrepare("selectInt", "select $1::bigint as a", nil)
	pipeline.SendDeallocate("selectInt")
	err = pipeline.Sync()
	require.NoError(t, err)

	results, err := pipeline.GetResults()
	require.NoError(t, err)
	sd, ok := results.(*fbconn.StatementDescription)
	require.Truef(t, ok, "expected StatementDescription, got: %#v", results)
	require.Len(t, sd.Fields, 1)
	require.Equal(t, "a", string(sd.Fields[0].Name))
	require.Equal(t, []uint32{fbtype.Int8OID}, sd.Parameters[0].Type)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.CloseComplete)
	require.Truef(t, ok, "expected CloseComplete, got: %#v", results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	err = pipeline.Close()
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestPipelineQuery(t *testing.T) {
	t.Skip("PG only")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	pipeline := fbConn.StartPipeline(ctx)
	pipeline.SendQueryParams(`select 1`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 2`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 3`, nil, nil, nil, nil)
	err = pipeline.Sync()
	require.NoError(t, err)

	pipeline.SendQueryParams(`select 4`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 5`, nil, nil, nil, nil)
	err = pipeline.Sync()
	require.NoError(t, err)

	results, err := pipeline.GetResults()
	require.NoError(t, err)
	rr, ok := results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult := rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "1", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "2", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "3", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "4", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "5", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	err = pipeline.Close()
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestPipelinePrepareQuery(t *testing.T) {
	t.Skip("PG only")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	pipeline := fbConn.StartPipeline(ctx)
	pipeline.SendPrepare("ps", "select $1::text as msg", nil)
	pipeline.SendQueryPrepared(`ps`, [][]byte{[]byte("hello")}, nil, nil)
	pipeline.SendQueryPrepared(`ps`, [][]byte{[]byte("goodbye")}, nil, nil)
	err = pipeline.Sync()
	require.NoError(t, err)

	results, err := pipeline.GetResults()
	require.NoError(t, err)
	sd, ok := results.(*fbconn.StatementDescription)
	require.Truef(t, ok, "expected StatementDescription, got: %#v", results)
	require.Len(t, sd.Fields, 1)
	require.Equal(t, "msg", string(sd.Fields[0].Name))
	require.Equal(t, []uint32{fbtype.TextOID}, sd.Parameters[0].Type)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok := results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult := rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "hello", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "goodbye", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	err = pipeline.Close()
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestPipelineQueryErrorBetweenSyncs(t *testing.T) {
	t.Skip("PG only")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	pipeline := fbConn.StartPipeline(ctx)
	pipeline.SendQueryParams(`select 1`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 2`, nil, nil, nil, nil)
	err = pipeline.Sync()
	require.NoError(t, err)

	pipeline.SendQueryParams(`select 3`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 1/(3-n) from generate_series(1,10) n`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 4`, nil, nil, nil, nil)
	err = pipeline.Sync()
	require.NoError(t, err)

	pipeline.SendQueryParams(`select 5`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 6`, nil, nil, nil, nil)
	err = pipeline.Sync()
	require.NoError(t, err)

	results, err := pipeline.GetResults()
	require.NoError(t, err)
	rr, ok := results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult := rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "1", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "2", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "3", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	var pgErr *fbconn.PgError
	require.ErrorAs(t, readResult.Err, &pgErr)
	require.Equal(t, "22012", pgErr.Code)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "5", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "6", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	err = pipeline.Close()
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestPipelineFlushForSingleRequests(t *testing.T) {
	t.Skip("PG only")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	pipeline := fbConn.StartPipeline(ctx)

	pipeline.SendPrepare("ps", "select $1::text as msg", nil)
	pipeline.SendFlushRequest()
	err = pipeline.Flush()
	require.NoError(t, err)

	results, err := pipeline.GetResults()
	require.NoError(t, err)
	sd, ok := results.(*fbconn.StatementDescription)
	require.Truef(t, ok, "expected StatementDescription, got: %#v", results)
	require.Len(t, sd.Fields, 1)
	require.Equal(t, "msg", string(sd.Fields[0].Name))
	require.Equal(t, []uint32{fbtype.TextOID}, sd.Parameters[0].Type)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	pipeline.SendQueryPrepared(`ps`, [][]byte{[]byte("hello")}, nil, nil)
	pipeline.SendFlushRequest()
	err = pipeline.Flush()
	require.NoError(t, err)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok := results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult := rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "hello", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	pipeline.SendDeallocate("ps")
	pipeline.SendFlushRequest()
	err = pipeline.Flush()
	require.NoError(t, err)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.CloseComplete)
	require.Truef(t, ok, "expected CloseComplete, got: %#v", results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	pipeline.SendQueryParams(`select 1`, nil, nil, nil, nil)
	pipeline.SendFlushRequest()
	err = pipeline.Flush()
	require.NoError(t, err)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "1", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	err = pipeline.Sync()
	require.NoError(t, err)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	err = pipeline.Close()
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestPipelineFlushForRequestSeries(t *testing.T) {
	t.Skip("PG only")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	pipeline := fbConn.StartPipeline(ctx)
	pipeline.SendPrepare("ps", "select $1::bigint as num", nil)
	err = pipeline.Sync()
	require.NoError(t, err)

	results, err := pipeline.GetResults()
	require.NoError(t, err)
	sd, ok := results.(*fbconn.StatementDescription)
	require.Truef(t, ok, "expected StatementDescription, got: %#v", results)
	require.Len(t, sd.Fields, 1)
	require.Equal(t, "num", string(sd.Fields[0].Name))
	require.Equal(t, []uint32{fbtype.Int8OID}, sd.Parameters[0].Type)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	pipeline.SendQueryPrepared(`ps`, [][]byte{[]byte("1")}, nil, nil)
	pipeline.SendQueryPrepared(`ps`, [][]byte{[]byte("2")}, nil, nil)
	pipeline.SendFlushRequest()
	err = pipeline.Flush()
	require.NoError(t, err)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok := results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult := rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "1", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "2", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	pipeline.SendQueryPrepared(`ps`, [][]byte{[]byte("3")}, nil, nil)
	err = pipeline.Flush()
	require.NoError(t, err)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	pipeline.SendQueryPrepared(`ps`, [][]byte{[]byte("4")}, nil, nil)
	pipeline.SendFlushRequest()
	err = pipeline.Flush()
	require.NoError(t, err)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "3", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "4", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	pipeline.SendQueryPrepared(`ps`, [][]byte{[]byte("5")}, nil, nil)
	pipeline.SendFlushRequest()

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	pipeline.SendQueryPrepared(`ps`, [][]byte{[]byte("6")}, nil, nil)
	pipeline.SendFlushRequest()
	err = pipeline.Flush()
	require.NoError(t, err)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "5", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "6", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	err = pipeline.Sync()
	require.NoError(t, err)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	err = pipeline.Close()
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestPipelineFlushWithError(t *testing.T) {
	t.Skip("PG only")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	pipeline := fbConn.StartPipeline(ctx)
	pipeline.SendQueryParams(`select 1`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 1/(3-n) from generate_series(1,10) n`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 2`, nil, nil, nil, nil)
	pipeline.SendFlushRequest()
	err = pipeline.Flush()
	require.NoError(t, err)

	results, err := pipeline.GetResults()
	require.NoError(t, err)
	rr, ok := results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult := rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "1", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	var pgErr *fbconn.PgError
	require.ErrorAs(t, readResult.Err, &pgErr)
	require.Equal(t, "22012", pgErr.Code)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	pipeline.SendQueryParams(`select 3`, nil, nil, nil, nil)
	pipeline.SendFlushRequest()
	err = pipeline.Flush()
	require.NoError(t, err)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	pipeline.SendQueryParams(`select 4`, nil, nil, nil, nil)
	pipeline.SendPipelineSync()
	pipeline.SendQueryParams(`select 5`, nil, nil, nil, nil)
	pipeline.SendFlushRequest()
	err = pipeline.Flush()
	require.NoError(t, err)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	rr, ok = results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult = rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "5", string(readResult.Rows[0][0]))

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	require.Nil(t, results)

	err = pipeline.Sync()
	require.NoError(t, err)

	results, err = pipeline.GetResults()
	require.NoError(t, err)
	_, ok = results.(*fbconn.PipelineSync)
	require.Truef(t, ok, "expected PipelineSync, got: %#v", results)

	err = pipeline.Close()
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestPipelineCloseReadsUnreadResults(t *testing.T) {
	t.Skip("PG only")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	pipeline := fbConn.StartPipeline(ctx)
	pipeline.SendQueryParams(`select 1`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 2`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 3`, nil, nil, nil, nil)
	err = pipeline.Sync()
	require.NoError(t, err)

	pipeline.SendQueryParams(`select 4`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 5`, nil, nil, nil, nil)
	err = pipeline.Sync()
	require.NoError(t, err)

	results, err := pipeline.GetResults()
	require.NoError(t, err)
	rr, ok := results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult := rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "1", string(readResult.Rows[0][0]))

	err = pipeline.Close()
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}

func TestPipelineCloseDetectsUnsyncedRequests(t *testing.T) {
	t.Skip("PG only")
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	pipeline := fbConn.StartPipeline(ctx)
	pipeline.SendQueryParams(`select 1`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 2`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 3`, nil, nil, nil, nil)
	err = pipeline.Sync()
	require.NoError(t, err)

	pipeline.SendQueryParams(`select 4`, nil, nil, nil, nil)
	pipeline.SendQueryParams(`select 5`, nil, nil, nil, nil)

	results, err := pipeline.GetResults()
	require.NoError(t, err)
	rr, ok := results.(*fbconn.ResultReader)
	require.Truef(t, ok, "expected ResultReader, got: %#v", results)
	readResult := rr.Read()
	require.NoError(t, readResult.Err)
	require.Len(t, readResult.Rows, 1)
	require.Len(t, readResult.Rows[0], 1)
	require.Equal(t, "1", string(readResult.Rows[0][0]))

	err = pipeline.Close()
	require.EqualError(t, err, "pipeline has unsynced requests")
}

func TestConnOnPgError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	config.OnPgError = func(c *fbconn.FbConn, pgErr *fbconn.FbError) bool {
		require.NotNil(t, c)
		require.NotNil(t, pgErr)
		// close connection on undefined tables only
		//FIXME: fbx set proper code
		if pgErr.SqlCode == 12345 {
			return false
		}
		return true
	}

	fbConn, err := fbconn.ConnectConfig(ctx, config)
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	res := fbConn.Exec(ctx, "select 'Hello, world' from rdb$database").Read()
	assert.NoError(t, res.Err)
	assert.False(t, fbConn.IsClosed())

	res = fbConn.Exec(ctx, "select 1/0 from rdb$database").Read()
	assert.Error(t, res.Err)
	assert.False(t, fbConn.IsClosed())

	res = fbConn.Exec(ctx, "select * from non_existant_table").Read()
	assert.Error(t, res.Err)
	assert.True(t, fbConn.IsClosed())
}

func TestConnCustomData(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	fbConn.CustomData()["foo"] = "bar"
	assert.Equal(t, "bar", fbConn.CustomData()["foo"])

	ensureConnValid(t, fbConn)
}

func Example() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	if err != nil {
		log.Fatalln(err)
	}
	defer fbConn.Close(ctx)

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
	connString := os.Getenv("PGX_SSL_PASSWORD")
	return connString
}

var rsaCertPEM = `-----BEGIN CERTIFICATE-----
MIIDCTCCAfGgAwIBAgIUQDlN1g1bzxIJ8KWkayNcQY5gzMEwDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTIyMDgxNTIxNDgyNloXDTIzMDgx
NTIxNDgyNlowFDESMBAGA1UEAwwJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEA0vOppiT8zE+076acRORzD5JVbRYKMK3XlWLVrHua4+ct
Rm54WyP+3XsYU4JGGGKgb8E+u2UosGJYcSM+b+U1/5XPTcpuumS+pCiD9WP++A39
tsukYwR7m65cgpiI4dlLEZI3EWpAW+Bb3230KiYW4sAmQ0Ih4PrN+oPvzcs86F4d
9Y03CqVUxRKLBLaClZQAg8qz2Pawwj1FKKjDX7u2fRVR0wgOugpCMOBJMcCgz9pp
0HSa4x3KZDHEZY7Pah5XwWrCfAEfRWsSTGcNaoN8gSxGFM1JOEJa8SAuPGjFcYIv
MmVWdw0FXCgYlSDL02fzLE0uyvXBDibzSqOk770JhQIDAQABo1MwUTAdBgNVHQ4E
FgQUiJ8JLENJ+2k1Xl4o6y2Lc/qHTh0wHwYDVR0jBBgwFoAUiJ8JLENJ+2k1Xl4o
6y2Lc/qHTh0wDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAwjn2
gnNAhFvh58VqLIjU6ftvn6rhz5B9dg2+XyY8sskLhhkO1nL9339BVZsRt+eI3a7I
81GNIm9qHVM3MUAcQv3SZy+0UPVUT8DNH2LwHT3CHnYTBP8U+8n8TDNGSTMUhIBB
Rx+6KwODpwLdI79VGT3IkbU9bZwuepB9I9nM5t/tt5kS4gHmJFlO0aLJFCTO4Scf
hp/WLPv4XQUH+I3cPfaJRxz2j0Kc8iOzMhFmvl1XOGByjX6X33LnOzY/LVeTSGyS
VgC32BGtnMwuy5XZYgFAeUx9HKy4tG4OH2Ux6uPF/WAhsug6PXSjV7BK6wYT5i27
MlascjupnaptKX/wMA==
-----END CERTIFICATE-----
`

var rsaKeyPEM = testingKey(`-----BEGIN TESTING KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDS86mmJPzMT7Tv
ppxE5HMPklVtFgowrdeVYtWse5rj5y1GbnhbI/7dexhTgkYYYqBvwT67ZSiwYlhx
Iz5v5TX/lc9Nym66ZL6kKIP1Y/74Df22y6RjBHubrlyCmIjh2UsRkjcRakBb4Fvf
bfQqJhbiwCZDQiHg+s36g+/NyzzoXh31jTcKpVTFEosEtoKVlACDyrPY9rDCPUUo
qMNfu7Z9FVHTCA66CkIw4EkxwKDP2mnQdJrjHcpkMcRljs9qHlfBasJ8AR9FaxJM
Zw1qg3yBLEYUzUk4QlrxIC48aMVxgi8yZVZ3DQVcKBiVIMvTZ/MsTS7K9cEOJvNK
o6TvvQmFAgMBAAECggEAKzTK54Ol33bn2TnnwdiElIjlRE2CUswYXrl6iDRc2hbs
WAOiVRB/T/+5UMla7/2rXJhY7+rdNZs/ABU24ZYxxCJ77jPrD/Q4c8j0lhsgCtBa
ycjV543wf0dsHTd+ubtWu8eVzdRUUD0YtB+CJevdPh4a+CWgaMMV0xyYzi61T+Yv
Z7Uc3awIAiT4Kw9JRmJiTnyMJg5vZqW3BBAX4ZIvS/54ipwEU+9sWLcuH7WmCR0B
QCTqS6hfJDLm//dGC89Iyno57zfYuiT3PYCWH5crr/DH3LqnwlNaOGSBkhkXuIL+
QvOaUMe2i0pjqxDrkBx05V554vyy9jEvK7i330HL4QKBgQDUJmouEr0+o7EMBApC
CPPu58K04qY5t9aGciG/pOurN42PF99yNZ1CnynH6DbcnzSl8rjc6Y65tzTlWods
bjwVfcmcokG7sPcivJvVjrjKpSQhL8xdZwSAjcqjN4yoJ/+ghm9w+SRmZr6oCQZ3
1jREfJKT+PGiWTEjYcExPWUD2QKBgQD+jdgq4c3tFavU8Hjnlf75xbStr5qu+fp2
SGLRRbX+msQwVbl2ZM9AJLoX9MTCl7D9zaI3ONhheMmfJ77lDTa3VMFtr3NevGA6
MxbiCEfRtQpNkJnsqCixLckx3bskj5+IF9BWzw7y7nOzdhoWVFv/+TltTm3RB51G
McdlmmVjjQKBgQDSFAw2/YV6vtu2O1XxGC591/Bd8MaMBziev+wde3GHhaZfGVPC
I8dLTpMwCwowpFKdNeLLl1gnHX161I+f1vUWjw4TVjVjaBUBx+VEr2Tb/nXtiwiD
QV0a883CnGJjreAblKRMKdpasMmBWhaWmn39h6Iad3zHuCzJjaaiXNpn2QKBgQCf
k1Q8LanmQnuh1c41f7aD5gjKCRezMUpt9BrejhD1NxheJJ9LNQ8nat6uPedLBcUS
lmJms+AR2qKqf0QQWyQ98YgAtshgTz8TvQtPT1mWgSOgVFHqJdC8obNK63FyDgc4
TZVxlgQNDqbBjfv0m5XA9f+mIlB9hYR2iKYzb4K30QKBgQC+LEJYZh00zsXttGHr
5wU1RzbgDIEsNuu+nZ4MxsaCik8ILNRHNXdeQbnADKuo6ATfhdmDIQMVZLG8Mivi
UwnwLd1GhizvqvLHa3ULnFphRyMGFxaLGV48axTT2ADoMX67ILrIY/yjycLqRZ3T
z3w+CgS20UrbLIR1YXfqUXge1g==
-----END TESTING KEY-----
`)

func testingKey(s string) string { return strings.ReplaceAll(s, "TESTING KEY", "PRIVATE KEY") }

//FIXME: fbx
//func TestSNISupport(t *testing.T) {
//	t.Parallel()
//	tests := []struct {
//		name      string
//		sni_param string
//		sni_set   bool
//	}{
//		{
//			name:      "SNI is passed by default",
//			sni_param: "",
//			sni_set:   true,
//		},
//		{
//			name:      "SNI is passed when asked for",
//			sni_param: "sslsni=1",
//			sni_set:   true,
//		},
//		{
//			name:      "SNI is not passed when disabled",
//			sni_param: "sslsni=0",
//			sni_set:   false,
//		},
//	}
//	for _, tt := range tests {
//		tt := tt
//		t.Run(tt.name, func(t *testing.T) {
//			t.Parallel()
//
//			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
//			defer cancel()
//
//			ln, err := net.Listen("tcp", "127.0.0.1:")
//			require.NoError(t, err)
//			defer ln.Close()
//
//			serverErrChan := make(chan error, 1)
//			serverSNINameChan := make(chan string, 1)
//			defer close(serverErrChan)
//			defer close(serverSNINameChan)
//
//			go func() {
//				var sniHost string
//
//				conn, err := ln.Accept()
//				if err != nil {
//					serverErrChan <- err
//					return
//				}
//				defer conn.Close()
//
//				err = conn.SetDeadline(time.Now().Add(5 * time.Second))
//				if err != nil {
//					serverErrChan <- err
//					return
//				}
//
//				backend := pgproto3.NewBackend(conn, conn)
//				startupMessage, err := backend.ReceiveStartupMessage()
//				if err != nil {
//					serverErrChan <- err
//					return
//				}
//
//				switch startupMessage.(type) {
//				case *pgproto3.SSLRequest:
//					_, err = conn.Write([]byte("S"))
//					if err != nil {
//						serverErrChan <- err
//						return
//					}
//				default:
//					serverErrChan <- fmt.Errorf("unexpected startup message: %#v", startupMessage)
//					return
//				}
//
//				cert, err := tls.X509KeyPair([]byte(rsaCertPEM), []byte(rsaKeyPEM))
//				if err != nil {
//					serverErrChan <- err
//					return
//				}
//
//				srv := tls.Server(conn, &tls.Config{
//					Certificates: []tls.Certificate{cert},
//					GetConfigForClient: func(argHello *tls.ClientHelloInfo) (*tls.Config, error) {
//						sniHost = argHello.ServerName
//						return nil, nil
//					},
//				})
//				defer srv.Close()
//
//				if err := srv.Handshake(); err != nil {
//					serverErrChan <- fmt.Errorf("handshake: %w", err)
//					return
//				}
//
//				srv.Write(mustEncode((&pgproto3.AuthenticationOk{}).Encode(nil)))
//				srv.Write(mustEncode((&pgproto3.BackendKeyData{ProcessID: 0, SecretKey: 0}).Encode(nil)))
//				srv.Write(mustEncode((&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(nil)))
//
//				serverSNINameChan <- sniHost
//			}()
//
//			_, port, _ := strings.Cut(ln.Addr().String(), ":")
//			connStr := fmt.Sprintf("sslmode=require host=localhost port=%s %s", port, tt.sni_param)
//			_, err = fbconn.Connect(ctx, connStr)
//
//			select {
//			case sniHost := <-serverSNINameChan:
//				if tt.sni_set {
//					require.Equal(t, "localhost", sniHost)
//				} else {
//					require.Equal(t, "", sniHost)
//				}
//			case err = <-serverErrChan:
//				t.Fatalf("server failed with error: %+v", err)
//			case <-time.After(time.Millisecond * 100):
//				t.Fatal("exceeded connection timeout without erroring out")
//			}
//		})
//	}
//}

//FIXME: fbx
//func TestConnectWithDirectSSLNegotiation(t *testing.T) {
//	t.Parallel()
//
//	tests := []struct {
//		name             string
//		connString       string
//		expectDirectNego bool
//	}{
//		{
//			name:             "Default negotiation (postgres)",
//			connString:       "sslmode=require",
//			expectDirectNego: false,
//		},
//		{
//			name:             "Direct negotiation",
//			connString:       "sslmode=require sslnegotiation=direct",
//			expectDirectNego: true,
//		},
//		{
//			name:             "Explicit postgres negotiation",
//			connString:       "sslmode=require sslnegotiation=postgres",
//			expectDirectNego: false,
//		},
//	}
//
//	for _, tt := range tests {
//		tt := tt
//		t.Run(tt.name, func(t *testing.T) {
//			t.Parallel()
//
//			script := &pgmock.Script{
//				Steps: pgmock.AcceptUnauthenticatedConnRequestSteps(),
//			}
//
//			ln, err := net.Listen("tcp", "127.0.0.1:")
//			require.NoError(t, err)
//			defer ln.Close()
//
//			_, port, err := net.SplitHostPort(ln.Addr().String())
//			require.NoError(t, err)
//
//			var directNegoObserved atomic.Bool
//
//			serverErrCh := make(chan error, 1)
//			go func() {
//				defer close(serverErrCh)
//
//				conn, err := ln.Accept()
//				if err != nil {
//					serverErrCh <- fmt.Errorf("accept error: %w", err)
//					return
//				}
//				defer conn.Close()
//
//				conn.SetDeadline(time.Now().Add(5 * time.Second))
//
//				firstByte := make([]byte, 1)
//				_, err = conn.Read(firstByte)
//				if err != nil {
//					serverErrCh <- fmt.Errorf("read first byte error: %w", err)
//					return
//				}
//
//				// Check if TLS Client Hello (direct) or PostgreSQL SSLRequest
//				isDirect := firstByte[0] >= 20 && firstByte[0] <= 23
//				directNegoObserved.Store(isDirect)
//
//				var tlsConn *tls.Conn
//
//				if !isDirect {
//					// Handle standard PostgreSQL SSL negotiation
//					// Read the rest of the SSL request message
//					sslRequestRemainder := make([]byte, 7)
//					_, err = io.ReadFull(conn, sslRequestRemainder)
//					if err != nil {
//						serverErrCh <- fmt.Errorf("read ssl request remainder error: %w", err)
//						return
//					}
//
//					// WriteMessage SSL acceptance response
//					_, err = conn.Write([]byte("S"))
//					if err != nil {
//						serverErrCh <- fmt.Errorf("write ssl acceptance error: %w", err)
//						return
//					}
//
//					// Setup TLS server without needing to reuse the first byte
//					cert, err := tls.X509KeyPair([]byte(rsaCertPEM), []byte(rsaKeyPEM))
//					if err != nil {
//						serverErrCh <- fmt.Errorf("cert error: %w", err)
//						return
//					}
//
//					tlsConn = tls.Server(conn, &tls.Config{
//						Certificates: []tls.Certificate{cert},
//					})
//				} else {
//					// Handle direct TLS negotiation
//					// Setup TLS server with the first byte already read
//					cert, err := tls.X509KeyPair([]byte(rsaCertPEM), []byte(rsaKeyPEM))
//					if err != nil {
//						serverErrCh <- fmt.Errorf("cert error: %w", err)
//						return
//					}
//
//					// Use a wrapper to inject the first byte back into the TLS handshake
//					bufConn := &prefixConn{
//						Conn:       conn,
//						prefixData: firstByte,
//					}
//
//					tlsConn = tls.Server(bufConn, &tls.Config{
//						Certificates: []tls.Certificate{cert},
//					})
//				}
//
//				// Complete TLS handshake
//				if err := tlsConn.Handshake(); err != nil {
//					serverErrCh <- fmt.Errorf("TLS handshake error: %w", err)
//					return
//				}
//				defer tlsConn.Close()
//
//				err = script.Run(pgproto3.NewBackend(tlsConn, tlsConn))
//				if err != nil {
//					serverErrCh <- fmt.Errorf("pgmock run error: %w", err)
//					return
//				}
//			}()
//
//			connStr := fmt.Sprintf("%s host=localhost port=%s sslmode=require sslinsecure=1",
//				tt.connString, port)
//
//			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
//			defer cancel()
//
//			conn, err := fbconn.Connect(ctx, connStr)
//
//			require.NoError(t, err)
//
//			defer conn.Close(ctx)
//
//			err = <-serverErrCh
//			require.NoError(t, err)
//
//			require.Equal(t, tt.expectDirectNego, directNegoObserved.Load())
//		})
//	}
//}

// prefixConn implements a net.Conn that prepends some data to the first Read
type prefixConn struct {
	net.Conn
	prefixData     []byte
	prefixConsumed bool
}

func (c *prefixConn) Read(b []byte) (n int, err error) {
	if !c.prefixConsumed && len(c.prefixData) > 0 {
		n = copy(b, c.prefixData)
		c.prefixData = c.prefixData[n:]
		c.prefixConsumed = len(c.prefixData) == 0
		return n, nil
	}
	return c.Conn.Read(b)
}

//FIXME: fbx
//// https://github.com/jackc/pgx/issues/1920
//func TestFatalErrorReceivedInPipelineMode(t *testing.T) {
//	t.Parallel()
//
//	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
//	defer cancel()
//
//	steps := pgmock.AcceptUnauthenticatedConnRequestSteps()
//	steps = append(steps, pgmock.ExpectAnyMessage(&pgproto3.Parse{}))
//	steps = append(steps, pgmock.ExpectAnyMessage(&pgproto3.Describe{}))
//	steps = append(steps, pgmock.ExpectAnyMessage(&pgproto3.Parse{}))
//	steps = append(steps, pgmock.ExpectAnyMessage(&pgproto3.Describe{}))
//	steps = append(steps, pgmock.ExpectAnyMessage(&pgproto3.Parse{}))
//	steps = append(steps, pgmock.ExpectAnyMessage(&pgproto3.Describe{}))
//	steps = append(steps, pgmock.SendMessage(&pgproto3.RowDescription{Fields: []pgproto3.FieldDescription{
//		{Name: []byte("mock")},
//	}}))
//	steps = append(steps, pgmock.SendMessage(&pgproto3.ErrorResponse{Severity: "FATAL", Code: "57P01"}))
//	// We shouldn't get anything after the first fatal error. But the reported issue was with PgBouncer so maybe that
//	// causes the issue. Anyway, a FATAL error after the connection had already been killed could cause a panic.
//	steps = append(steps, pgmock.SendMessage(&pgproto3.ErrorResponse{Severity: "FATAL", Code: "57P01"}))
//
//	script := &pgmock.Script{Steps: steps}
//
//	ln, err := net.Listen("tcp", "127.0.0.1:")
//	require.NoError(t, err)
//	defer ln.Close()
//
//	serverKeepAlive := make(chan struct{})
//	defer close(serverKeepAlive)
//
//	serverErrChan := make(chan error, 1)
//	go func() {
//		defer close(serverErrChan)
//
//		conn, err := ln.Accept()
//		if err != nil {
//			serverErrChan <- err
//			return
//		}
//		defer conn.Close()
//
//		err = conn.SetDeadline(time.Now().Add(59 * time.Second))
//		if err != nil {
//			serverErrChan <- err
//			return
//		}
//
//		err = script.Run(pgproto3.NewBackend(conn, conn))
//		if err != nil {
//			serverErrChan <- err
//			return
//		}
//
//		<-serverKeepAlive
//	}()
//
//	parts := strings.Split(ln.Addr().String(), ":")
//	host := parts[0]
//	port := parts[1]
//	connStr := fmt.Sprintf("sslmode=disable host=%s port=%s", host, port)
//
//	ctx, cancel = context.WithTimeout(ctx, 59*time.Second)
//	defer cancel()
//	conn, err := fbconn.Connect(ctx, connStr)
//	require.NoError(t, err)
//
//	pipeline := conn.StartPipeline(ctx)
//	pipeline.SendPrepare("s1", "select 1", nil)
//	pipeline.SendPrepare("s2", "select 2", nil)
//	pipeline.SendPrepare("s3", "select 3", nil)
//	err = pipeline.Sync()
//	require.NoError(t, err)
//
//	_, err = pipeline.GetResults()
//	require.NoError(t, err)
//	_, err = pipeline.GetResults()
//	require.Error(t, err)
//
//	err = pipeline.Close()
//	require.Error(t, err)
//}

func mustEncode(buf []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return buf
}

func TestDeadlineContextWatcherHandler(t *testing.T) {
	t.Parallel()

	t.Run("DeadlineExceeded with zero DeadlineDelay", func(t *testing.T) {
		config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
		require.NoError(t, err)
		config.BuildContextWatcherHandler = func(conn *fbconn.FbConn) ctxwatch.Handler {
			return &fbconn.DeadlineContextWatcherHandler{Conn: conn.Conn()}
		}
		config.ConnectTimeout = 5 * time.Second

		fbConn, err := fbconn.ConnectConfig(context.Background(), config)
		require.NoError(t, err)
		defer closeConn(t, fbConn)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		res := fbConn.Exec(ctx, "select 1, pg_sleep(1) ").Read()
		require.Error(t, res.Err)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.True(t, fbConn.IsClosed())
	})

	t.Run("DeadlineExceeded with DeadlineDelay", func(t *testing.T) {
		config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
		require.NoError(t, err)
		config.BuildContextWatcherHandler = func(conn *fbconn.FbConn) ctxwatch.Handler {
			return &fbconn.DeadlineContextWatcherHandler{Conn: conn.Conn(), DeadlineDelay: 500 * time.Millisecond}
		}
		config.ConnectTimeout = 5 * time.Second

		fbConn, err := fbconn.ConnectConfig(context.Background(), config)
		require.NoError(t, err)
		defer closeConn(t, fbConn)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		res := fbConn.Exec(ctx, "select 1, pg_sleep(0.250)").Read()
		require.NoError(t, res.Err)

		ensureConnValid(t, fbConn)
	})
}

func TestCancelRequestContextWatcherHandler(t *testing.T) {
	t.Parallel()

	t.Run("DeadlineExceeded cancels request after CancelRequestDelay", func(t *testing.T) {
		config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
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

		res := fbConn.Exec(ctx, "select 1, pg_sleep(3)").Read()
		require.Error(t, res.Err)
		var pgErr *fbconn.PgError
		require.ErrorAs(t, err, &pgErr)

		ensureConnValid(t, fbConn)
	})

	t.Run("DeadlineExceeded - do not send cancel request when query finishes in grace period", func(t *testing.T) {
		config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
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

		res := fbConn.Exec(ctx, "select 1, pg_sleep(0.250)").Read()
		require.NoError(t, res.Err)

		ensureConnValid(t, fbConn)
	})

	t.Run("DeadlineExceeded sets conn deadline with DeadlineDelay", func(t *testing.T) {
		config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
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

		res := fbConn.Exec(ctx, "select 1, pg_sleep(1)").Read()
		require.Error(t, res.Err)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.True(t, fbConn.IsClosed())
	})

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("Stress %d", i), func(t *testing.T) {
			t.Parallel()

			config, err := fbconn.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
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
					fbConn.Exec(ctx, "select 1, pg_sleep(0.010)").Read()
					time.Sleep(100 * time.Millisecond) // ensure a cancel request that was a little late doesn't interrupt ensureConnValid.
					ensureConnValid(t, fbConn)
				}()
			}
		})
	}
}

func TestReadSingletonResult(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
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
	require.NoError(t, fbConn.Commit(ctx))

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
	require.NoError(t, fbConn.Commit(ctx))

	res = fbConn.Exec(ctx, `execute procedure `+t.Name()+` 1, 2`).Read()
	require.NoError(t, res.Err)
	assert.Equal(t, 0, len(res.Rows))
}

func TestConnExecImmediate(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	err = fbConn.ExecImmediate(ctx, "recreate table a(id int)")
	require.NoError(t, err)

	err = fbConn.Commit(ctx)
	require.NoError(t, err)

	tx, err := fbConn.StartTransaction(ctx)
	require.NoError(t, err)

	err = fbConn.ExecImmediateTx(ctx, tx, "drop table a")
	require.NoError(t, err)
	ensureConnValid(t, fbConn)
}

func TestConnExecImmediateNoTx(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fbConn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer closeConn(t, fbConn)

	err = fbConn.ExecImmediateTx(ctx, nil, "set transaction")
	require.NoError(t, err)

	ensureConnValid(t, fbConn)
}
