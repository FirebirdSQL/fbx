package fbconn

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

type maintenanceTestWriter struct {
	bytes.Buffer
	err    error
	short  bool
	closed bool
}

func (w *maintenanceTestWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if w.short {
		return len(p) - 1, nil
	}
	return w.Buffer.Write(p)
}

func (w *maintenanceTestWriter) Close() error {
	w.closed = true
	return nil
}

func TestServiceConnMaintenanceOutputFailures(t *testing.T) {
	writeErr := errors.New("report writer failed")
	for name, call := range maintenanceOutputCalls() {
		for _, short := range []bool{false, true} {
			suffix := "/writer error"
			if short {
				suffix = "/short write"
			}
			t.Run(name+suffix, func(t *testing.T) {
				client, server := net.Pipe()
				t.Cleanup(func() { _ = server.Close() })
				done := make(chan error, 1)
				go func() {
					defer func() { _ = server.Close() }()
					done <- runBackupOutputProbeServer(server)
				}()
				conn, ctx := newStatisticsProbeConnection(t, client)
				out := &maintenanceTestWriter{err: writeErr}
				want := writeErr
				if short {
					out.err, out.short = nil, true
					want = io.ErrShortWrite
				}
				require.ErrorIs(t, call(ctx, conn, out), want)
				require.False(t, out.closed)
				require.NoError(t, conn.Close(ctx))
				select {
				case err := <-done:
					require.NoError(t, err)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			})
		}
	}
}

func TestServiceConnMaintenanceCancellation(t *testing.T) {
	for name, call := range maintenanceOutputCalls() {
		t.Run(name, func(t *testing.T) {
			client, server := net.Pipe()
			t.Cleanup(func() { _ = server.Close() })
			started := make(chan struct{})
			done := make(chan error, 1)
			action := byte(fbproto.Isc_action_svc_repair)
			if name == "properties" {
				action = fbproto.Isc_action_svc_properties
			}
			go func() {
				defer func() { _ = server.Close() }()
				done <- runCanceledServiceActionProbeServer(server, started, action)
			}()
			conn, deadline := newStatisticsProbeConnection(t, client)
			ctx, cancel := context.WithCancel(deadline)
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- call(ctx, conn, nil) }()
			select {
			case <-started:
				cancel()
			case <-deadline.Done():
				t.Fatal(deadline.Err())
			}
			select {
			case err := <-result:
				require.ErrorIs(t, err, context.Canceled)
			case <-deadline.Done():
				t.Fatal(deadline.Err())
			}
			require.True(t, conn.conn.IsClosed())
			select {
			case <-conn.conn.CleanupDone():
			case <-deadline.Done():
				t.Fatal("connection cleanup did not finish")
			}
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-deadline.Done():
				t.Fatal(deadline.Err())
			}
			require.NoError(t, conn.Close(deadline))
		})
	}
}

func TestServiceConnMaintenanceServerErrors(t *testing.T) {
	calls := maintenanceOutputCalls()
	calls["limbo"] = func(ctx context.Context, c *ServiceConn, _ io.Writer) error {
		ids, err := c.LimboTransactions(ctx, "db")
		require.Nil(t, ids)
		return err
	}
	for name, call := range calls {
		for _, duringOutput := range []bool{false, true} {
			suffix := "/start"
			if duringOutput {
				suffix = "/output"
			}
			t.Run(name+suffix, func(t *testing.T) {
				client, server := net.Pipe()
				t.Cleanup(func() { _ = server.Close() })
				done := make(chan error, 1)
				go func() {
					defer func() { _ = server.Close() }()
					done <- runMaintenanceServerErrorProbe(server, duringOutput, name == "limbo",
						[]int{fbproto.Isc_arg_gds, fbproto.Isc_bad_spb_form, fbproto.Isc_arg_end})
				}()
				conn, ctx := newStatisticsProbeConnection(t, client)
				err := call(ctx, conn, nil)
				var serverErr *FbError
				require.ErrorAs(t, err, &serverErr)
				require.Contains(t, serverErr.GdsCodes, fbproto.Isc_bad_spb_form)
				require.NoError(t, conn.Close(ctx))
				select {
				case err := <-done:
					require.NoError(t, err)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			})
		}
	}
}

func TestServiceConnRepairValidationMessages(t *testing.T) {
	for _, duringOutput := range []bool{false, true} {
		name := "start"
		if duringOutput {
			name = "output"
		}
		t.Run(name, func(t *testing.T) {
			client, server := net.Pipe()
			t.Cleanup(func() { _ = server.Close() })
			done := make(chan error, 1)
			go func() {
				defer func() { _ = server.Close() }()
				done <- runMaintenanceServerErrorProbe(server, duringOutput, false, []int{
					fbproto.Isc_arg_gds, 335740952, // gfix_summary
					fbproto.Isc_arg_gds, 335740984, // gfix_blob_err
					fbproto.Isc_arg_number, 3,
					fbproto.Isc_arg_gds, 335741050, // gfix_pip_err
					fbproto.Isc_arg_number, 7,
					fbproto.Isc_arg_end,
				})
			}()
			conn, ctx := newStatisticsProbeConnection(t, client)
			err := conn.Repair(ctx, "db", RepairOptions{Operation: RepairValidate, ReadOnly: true})
			var serverErr *FbError
			require.ErrorAs(t, err, &serverErr)
			require.Equal(t, []int{335740952, 335740984, 335741050}, serverErr.GdsCodes)
			require.Equal(t, "Summary of validation errors\n\tNumber of Blob page errors\t: 3\n\tNumber of inventory page errors\t: 7", serverErr.Message)
			require.Equal(t, "HY000", serverErr.Code)
			require.False(t, SafeToRetry(err))
			require.NoError(t, conn.Close(ctx))
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
	}
}

func runMaintenanceServerErrorProbe(conn net.Conn, duringOutput, limbo bool, status []int) error {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return err
	}
	r, w := fbproto.NewXDRReader(conn), fbproto.NewXDRWriter(conn)
	if err := readProbeConnect(r); err != nil {
		return err
	}
	if err := writeAuthenticatedProbeAccept(w, encoding); err != nil {
		return err
	}
	if err := readProbeServiceAttach(r, encoding); err != nil {
		return err
	}
	if err := writeProbeResponse(w, 42); err != nil {
		return err
	}
	if _, err := readProbeServiceStart(r, 42); err != nil {
		return err
	}
	if duringOutput {
		if err := writeProbeResponse(w, 0); err != nil {
			return err
		}
		if _, _, _, err := readProbeServiceInfo(r, 42); err != nil {
			return err
		}
	}
	if err := w.WriteInt32(fbproto.Op_response); err != nil {
		return err
	}
	if err := w.WriteInt32(0); err != nil {
		return err
	}
	if err := w.WriteInt64(0); err != nil {
		return err
	}
	if err := w.WriteBuffer(nil); err != nil {
		return err
	}
	for _, v := range status {
		if err := w.WriteInt32(v); err != nil {
			return err
		}
	}
	if duringOutput && limbo {
		_, err := r.ReadInt32()
		if !errors.Is(err, io.EOF) {
			return errors.New("limbo connection was not closed after a query failure")
		}
		return nil
	}
	_, _, err = readProbeServiceDetach(r)
	return err
}
