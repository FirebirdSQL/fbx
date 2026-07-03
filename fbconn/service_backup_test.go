package fbconn

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbproto"
)

func TestBuildBackupActionSPB(t *testing.T) {
	options := BackupOptions{
		Verbose:               true,
		IgnoreChecksums:       true,
		IgnoreLimbo:           true,
		MetadataOnly:          true,
		NoGarbageCollect:      true,
		OldDescriptions:       true,
		NonTransportable:      true,
		ConvertExternalTables: true,
		Expand:                true,
		Factor:                32,
		ParallelWorkers:       4,
		Replace:               true,
		NoDatabaseTriggers:    true,
		Zip:                   true,
		Compressor:            "zstd:3:1",
	}

	spb, err := buildBackupActionSPB("/data/employee.fdb", "/backup/employee.fbk", options)
	require.NoError(t, err)

	offset := 0
	require.Equal(t, byte(fbproto.Isc_action_svc_backup), spb[offset])
	offset++
	offset = requireServiceActionString(t, spb, offset, fbproto.Isc_spb_dbname, "/data/employee.fdb")
	offset = requireServiceActionString(t, spb, offset, fbproto.Isc_spb_bkp_file, "/backup/employee.fbk")
	require.Equal(t, byte(fbproto.Isc_spb_verbose), spb[offset])
	offset++
	require.Equal(t, byte(fbproto.Isc_spb_bkp_factor), spb[offset])
	offset++
	require.Equal(t, uint32(32), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, byte(fbproto.Isc_spb_bkp_parallel_workers), spb[offset])
	offset++
	require.Equal(t, uint32(4), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, byte(fbproto.Isc_spb_bkp_replace), spb[offset])
	offset++
	offset = requireServiceActionString(
		t, spb, offset, fbproto.Isc_spb_bkp_compression, "zstd:3:1")
	require.Equal(t, byte(fbproto.Isc_spb_options), spb[offset])
	offset++
	require.Equal(t, uint32(0x180ff), binary.LittleEndian.Uint32(spb[offset:offset+4]))
	offset += 4
	require.Equal(t, len(spb), offset)
}

func TestBuildBackupActionSPBValidation(t *testing.T) {
	tests := []struct {
		name       string
		database   string
		backupFile string
		options    BackupOptions
		wantError  string
	}{
		{
			name:       "empty database",
			backupFile: "/backup/employee.fbk",
			wantError:  "backup database must not be empty",
		},
		{
			name:      "empty backup file",
			database:  "/data/employee.fdb",
			wantError: "backup file must not be empty",
		},
		{
			name:       "database is too long",
			database:   strings.Repeat("d", maxServiceActionStringLength+1),
			backupFile: "/backup/employee.fbk",
			wantError:  "maximum is 65535",
		},
		{
			name:       "negative factor",
			database:   "/data/employee.fdb",
			backupFile: "/backup/employee.fbk",
			options:    BackupOptions{Factor: -1},
			wantError:  "backup factor must be between",
		},
		{
			name:       "negative parallel workers",
			database:   "/data/employee.fdb",
			backupFile: "/backup/employee.fbk",
			options:    BackupOptions{ParallelWorkers: -1},
			wantError:  "backup parallel workers must be between",
		},
		{
			name:       "parallel workers overflow",
			database:   "/data/employee.fdb",
			backupFile: "/backup/employee.fbk",
			options: BackupOptions{
				ParallelWorkers: int(int64(maxServiceActionInt32) + 1),
			},
			wantError: "backup parallel workers must be between",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := buildBackupActionSPB(test.database, test.backupFile, test.options)
			require.ErrorContains(t, err, test.wantError)
		})
	}
}

func TestServiceConnBackup(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	resultCh := make(chan backupProbeResult, 1)
	go func() {
		defer func() { _ = serverConn.Close() }()
		resultCh <- runBackupProbeServer(serverConn)
	}()

	config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee")
	require.NoError(t, err)
	config.AuthPluginList = []string{"Legacy_Auth"}
	config.LookupFunc = func(context.Context, string) ([]string, error) {
		return []string{"127.0.0.1"}, nil
	}
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		return clientConn, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := ConnectServiceConfig(ctx, testServiceConfig(config))
	require.NoError(t, err)

	var output bytes.Buffer
	err = conn.Backup(context.Background(), "/data/employee.fdb", "/backup/employee.fbk", BackupOptions{
		Output:      &output,
		Verbose:     true,
		IgnoreLimbo: true,
	})
	require.NoError(t, err)
	require.Equal(t, "gbak: writing data\n", output.String())
	require.False(t, conn.actionActive)
	require.NoError(t, conn.Close(context.Background()))

	select {
	case result := <-resultCh:
		require.NoError(t, result.err)
		require.Equal(t, []int{1024, 2048, 2048, 2048}, result.bufferLengths)
		require.Equal(t, []byte{fbproto.Isc_info_svc_to_eof}, result.receiveItems)
		require.Equal(t, buildServiceOutputSendItems(), result.sendItems)

		offset := 1
		require.Equal(t, byte(fbproto.Isc_action_svc_backup), result.actionSPB[0])
		offset = requireServiceActionString(
			t, result.actionSPB, offset, fbproto.Isc_spb_dbname, "/data/employee.fdb")
		offset = requireServiceActionString(
			t, result.actionSPB, offset, fbproto.Isc_spb_bkp_file, "/backup/employee.fbk")
		require.Equal(t, byte(fbproto.Isc_spb_verbose), result.actionSPB[offset])
		offset++
		require.Equal(t, byte(fbproto.Isc_spb_options), result.actionSPB[offset])
		offset++
		require.Equal(t, uint32(fbproto.Isc_spb_bkp_ignore_limbo),
			binary.LittleEndian.Uint32(result.actionSPB[offset:offset+4]))
		require.Equal(t, 42, result.detachedHandle)
		require.Equal(t, fbproto.Op_disconnect, result.finalOpcode)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestServiceConnBackupReturnsOutputError(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	serverDone := make(chan error, 1)
	go func() {
		defer func() { _ = serverConn.Close() }()
		serverDone <- runBackupOutputProbeServer(serverConn)
	}()

	config, err := ParseConfig("firebird://sysdba:masterkey@mock/employee")
	require.NoError(t, err)
	config.AuthPluginList = []string{"Legacy_Auth"}
	config.LookupFunc = func(context.Context, string) ([]string, error) {
		return []string{"127.0.0.1"}, nil
	}
	config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
		return clientConn, nil
	}

	conn, err := ConnectServiceConfig(context.Background(), testServiceConfig(config))
	require.NoError(t, err)
	err = conn.Backup(context.Background(), "/data/employee.fdb", "/backup/employee.fbk", BackupOptions{
		Output: errorWriter{},
	})
	require.ErrorContains(t, err, "write service output: output failed")
	require.True(t, conn.actionActive)
	require.NoError(t, conn.Close(context.Background()))
	require.NoError(t, <-serverDone)
}

type backupProbeResult struct {
	actionSPB      []byte
	sendItems      []byte
	receiveItems   []byte
	bufferLengths  []int
	detachedHandle int
	finalOpcode    int
	err            error
}

func runBackupProbeServer(conn net.Conn) backupProbeResult {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return backupProbeResult{err: err}
	}
	r := fbproto.NewXDRReader(conn)
	w := fbproto.NewXDRWriter(conn)
	if err := readProbeConnect(r); err != nil {
		return backupProbeResult{err: err}
	}
	if err := writeAuthenticatedProbeAccept(w, encoding); err != nil {
		return backupProbeResult{err: err}
	}
	if err := readProbeServiceAttach(r, encoding); err != nil {
		return backupProbeResult{err: err}
	}
	if err := writeProbeResponse(w, 42); err != nil {
		return backupProbeResult{err: err}
	}

	actionSPB, err := readProbeServiceStart(r, 42)
	if err != nil {
		return backupProbeResult{err: err}
	}
	if err := writeProbeResponse(w, 0); err != nil {
		return backupProbeResult{err: err}
	}

	responses := [][]byte{
		{fbproto.Isc_info_truncated},
		{fbproto.Isc_info_svc_to_eof, 0, 0, fbproto.Isc_info_svc_timeout},
		serviceOutputResponse([]byte("gbak: writing data\n")),
		{fbproto.Isc_info_svc_to_eof, 0, 0, fbproto.Isc_info_end},
	}
	result := backupProbeResult{actionSPB: actionSPB}
	for _, response := range responses {
		sendItems, receiveItems, bufferLength, err := readProbeServiceInfo(r, 42)
		if err != nil {
			return backupProbeResult{err: err}
		}
		if result.sendItems == nil {
			result.sendItems = sendItems
			result.receiveItems = receiveItems
		} else if !bytes.Equal(result.sendItems, sendItems) || !bytes.Equal(result.receiveItems, receiveItems) {
			return backupProbeResult{err: errors.New("service info items changed between requests")}
		}
		result.bufferLengths = append(result.bufferLengths, bufferLength)
		if err := writeProbeResponseData(w, 0, response); err != nil {
			return backupProbeResult{err: err}
		}
	}

	result.detachedHandle, result.finalOpcode, result.err = readProbeServiceDetach(r)
	return result
}

func runBackupOutputProbeServer(conn net.Conn) error {
	encoding, err := fbproto.NewEncoding("UTF8")
	if err != nil {
		return err
	}
	r := fbproto.NewXDRReader(conn)
	w := fbproto.NewXDRWriter(conn)
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
	if err := writeProbeResponse(w, 0); err != nil {
		return err
	}
	if _, _, _, err := readProbeServiceInfo(r, 42); err != nil {
		return err
	}
	if err := writeProbeResponseData(w, 0, serviceOutputResponse([]byte("output"))); err != nil {
		return err
	}
	_, _, err = readProbeServiceDetach(r)
	return err
}

func readProbeServiceStart(r *fbproto.XDRReader, expectedHandle int) ([]byte, error) {
	opcode, err := r.ReadInt32()
	if err != nil {
		return nil, err
	}
	if opcode != fbproto.Op_service_start {
		return nil, fmt.Errorf("expected op_service_start, got %d", opcode)
	}
	handle, err := r.ReadInt32()
	if err != nil {
		return nil, err
	}
	if handle != expectedHandle {
		return nil, fmt.Errorf("expected service handle %d, got %d", expectedHandle, handle)
	}
	if _, err := r.ReadInt32(); err != nil {
		return nil, err
	}
	return r.ReadBuffer()
}

func readProbeServiceInfo(
	r *fbproto.XDRReader,
	expectedHandle int,
) ([]byte, []byte, int, error) {
	opcode, err := r.ReadInt32()
	if err != nil {
		return nil, nil, 0, err
	}
	if opcode != fbproto.Op_service_info {
		return nil, nil, 0, fmt.Errorf("expected op_service_info, got %d", opcode)
	}
	handle, err := r.ReadInt32()
	if err != nil {
		return nil, nil, 0, err
	}
	if handle != expectedHandle {
		return nil, nil, 0, fmt.Errorf("expected service handle %d, got %d", expectedHandle, handle)
	}
	if _, err := r.ReadInt32(); err != nil {
		return nil, nil, 0, err
	}
	sendItems, err := r.ReadBuffer()
	if err != nil {
		return nil, nil, 0, err
	}
	receiveItems, err := r.ReadBuffer()
	if err != nil {
		return nil, nil, 0, err
	}
	bufferLength, err := r.ReadInt32()
	return sendItems, receiveItems, bufferLength, err
}

func readProbeServiceDetach(r *fbproto.XDRReader) (int, int, error) {
	opcode, err := r.ReadInt32()
	if err != nil {
		return 0, 0, err
	}
	if opcode != fbproto.Op_service_detach {
		return 0, 0, fmt.Errorf("expected op_service_detach, got %d", opcode)
	}
	handle, err := r.ReadInt32()
	if err != nil {
		return 0, 0, err
	}
	finalOpcode, err := r.ReadInt32()
	return handle, finalOpcode, err
}

func requireServiceActionString(
	t *testing.T,
	spb []byte,
	offset int,
	expectedTag byte,
	expectedValue string,
) int {
	t.Helper()
	require.Less(t, offset+3, len(spb))
	require.Equal(t, expectedTag, spb[offset])
	offset++
	length := int(binary.LittleEndian.Uint16(spb[offset : offset+2]))
	offset += 2
	require.LessOrEqual(t, offset+length, len(spb))
	require.Equal(t, expectedValue, string(spb[offset:offset+length]))
	return offset + length
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("output failed")
}

func writeProbeResponseData(w *fbproto.XDRWriter, object int, data []byte) error {
	if err := w.WriteInt32(fbproto.Op_response); err != nil {
		return err
	}
	if err := w.WriteInt32(object); err != nil {
		return err
	}
	if err := w.WriteInt64(0); err != nil {
		return err
	}
	if err := w.WriteBuffer(data); err != nil {
		return err
	}
	return w.WriteInt32(fbproto.Isc_arg_end)
}
