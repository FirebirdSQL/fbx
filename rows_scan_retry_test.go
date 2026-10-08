package fbx_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbxtest"
)

type retryableRowsScanError struct{ error }

func (retryableRowsScanError) SafeToRetry() bool { return true }

type failingRowsScanner struct{ err error }

func (s failingRowsScanner) ScanRow(fbx.Rows) error { return s.err }

type failingValueScanner struct{ err error }

func (s *failingValueScanner) Scan(any) error { return s.err }

type canceledBlobRowScanner struct{ cancel context.CancelFunc }

func (s canceledBlobRowScanner) ScanRow(rows fbx.Rows) error {
	s.cancel()
	var value []byte
	return rows.Scan(&value)
}

func TestRowsScanFailureDoesNotAuthorizeRetry(t *testing.T) {
	if os.Getenv("FBX_TEST_DATABASE") == "" {
		t.Skip("FBX_TEST_DATABASE is not configured")
	}
	for _, failure := range []string{"blob", "row scanner", "value scanner"} {
		t.Run(failure, func(t *testing.T) {
			for _, queryRow := range []bool{false, true} {
				t.Run(fmt.Sprintf("query_row=%t", queryRow), func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					ctr := defaultConnTestRunner
					ctr.CreateConfig = func(ctx context.Context, t testing.TB) *fbx.ConnConfig {
						config := defaultConnTestRunner.CreateConfig(ctx, t)
						config.Autocommit = true
						config.MaxInlineBlobSize = 0
						return config
					}
					fbxtest.RunWithQueryExecModes(ctx, t, ctr, nil, func(ctx context.Context, t testing.TB, conn *fbx.Conn) {
						table := fmt.Sprintf("FBX_SCAN_%x", time.Now().UnixNano())
						_, err := conn.Exec(ctx, "create table "+table+" (id integer, payload blob sub_type 0)")
						require.NoError(t, err)
						defer func() {
							cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
							defer cancel()
							// Cached verification queries keep the table in use on some servers.
							if !assert.NoError(t, conn.DeallocateAll(cleanupCtx)) {
								return
							}
							_, err := conn.Exec(cleanupCtx, "drop table "+table, fbx.QueryExecModeExec)
							assert.NoError(t, err)
						}()

						queryCtx, stop := context.WithCancel(ctx)
						defer stop()
						cause := error(&retryableRowsScanError{error: errors.New("scanner requested retry")})
						require.True(t, fbconn.SafeToRetry(cause))
						column := "ID"
						var destination any
						switch failure {
						case "blob":
							column, cause = "PAYLOAD", context.Canceled
							if queryRow {
								destination = canceledBlobRowScanner{cancel: stop}
							} else {
								destination = new([]byte)
							}
						case "row scanner":
							destination = failingRowsScanner{err: cause}
						case "value scanner":
							destination = &failingValueScanner{err: cause}
						}
						sql := "insert into " + table + " values (1, cast('scan payload' as blob sub_type 0)) returning " + column
						if queryRow {
							err = conn.QueryRow(queryCtx, sql).Scan(destination)
						} else {
							rows, queryErr := conn.Query(queryCtx, sql)
							require.NoError(t, queryErr)
							defer rows.Close()
							require.True(t, rows.Next())
							if failure == "blob" {
								stop()
							}
							err = rows.Scan(destination)
							require.ErrorIs(t, rows.Err(), cause)
							assert.False(t, fbconn.SafeToRetry(rows.Err()))
							assert.Equal(t, rows.Err(), err, "Scan must return the error after cleanup")
							assert.False(t, rows.Next())
						}
						require.ErrorIs(t, err, cause)
						assert.False(t, fbconn.SafeToRetry(err), "a scan failure must not authorize another INSERT")
						if failure != "row scanner" {
							var scanErr fbx.ScanArgError
							require.ErrorAs(t, err, &scanErr)
							assert.Equal(t, 0, scanErr.ColumnIndex)
							assert.Equal(t, column, scanErr.FieldName)
						}
						require.False(t, conn.IsClosed())
						require.False(t, conn.FbConn().IsBusy())
						require.False(t, conn.FbConn().ActiveTx())
						_, err = conn.Exec(ctx, "insert into "+table+" (id) values (2)")
						require.NoError(t, err)
						var failedRows, nextRows int
						require.NoError(t, conn.QueryRow(ctx, "select cast(count(*) as integer) from "+table+" where id=1").Scan(&failedRows))
						require.NoError(t, conn.QueryRow(ctx, "select cast(count(*) as integer) from "+table+" where id=2").Scan(&nextRows))
						assert.Zero(t, failedRows, "the failed scan must roll back the pending INSERT")
						assert.Equal(t, 1, nextRows, "the next request must not commit changes from the failed scan")
					})
				})
			}
		})
	}
}
