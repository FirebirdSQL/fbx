package fbproto_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbproto"
)

const recursiveSeriesSQL = "WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series"

var expectedTraceSnapshot = strings.ReplaceAll(`C\tBeginTx
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 gdsCodes=0
C\tAllocateStatement
C\tPrepare stmt=65535 tx=1 dialect=3 sql="WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series" lenItems=0 bufSize=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=0 gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=4 blobId=0 lenData=1 gdsCodes=0
C\tInfoSQL stmt=2 maxBuf=524288 lenItems=26
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=138 gdsCodes=0
C\tExecute kind=63 stmt=2 tx=1 inBLR=12 outBLR=0 lenParameters=1 lenParamDes=1 timeout=0 cursorFlags=0 inlineBlobSize=65536
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 gdsCodes=0
C\tFetch stmt=2 msgs=400 lenRowBLR=12
S\tFetchResponse status=0 msgs=1
S\tRowData cols=1 nulls=0 [4]
S\tFetchResponse status=0 msgs=1
S\tRowData cols=1 nulls=0 [4]
S\tFetchResponse status=0 msgs=1
S\tRowData cols=1 nulls=0 [4]
S\tFetchResponse status=100 msgs=0
C\tInfoSQL stmt=2 maxBuf=64 lenItems=1
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=33 gdsCodes=0
C\tCommit
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=0 blobId=0 lenData=0 gdsCodes=0
`,
	`\t`, "\t")

var expectedFullLogTraceSnapshot = strings.ReplaceAll(`C\tBeginTx tpb=''
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 data='' gdsCodes=0
C\tAllocateStatement
C\tPrepare stmt=65535 tx=1 dialect=3 sql="WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series" lenItems=0 bufSize=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=0 data='' gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=4 blobId=0 lenData=1 data='\x01' gdsCodes=0
C\tInfoSQL stmt=2 maxBuf=524288 lenItems=26 items='\x14\x02\x00\x00\x15\x04\x07\x09\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x08\x05\x07\x09\x0b\x0c\x0d\x0e\x08'
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=138 data='\x15\x04\x00\x01\x00\x00\x00\x04\x07\x04\x00\x01\x00\x00\x00\x09\x04\x00\x01\x00\x00\x00\x0b\x04\x00\xf0\x01\x00\x00\x0c\x04\x00\x00\x00\x00\x00\x0d\x04\x00\x00\x00\x00\x00\x0e\x04\x00\x04\x00\x00\x00\x0f\x04\x00\x00\x00\x00\x00\x10\x08\x00CONSTANT\x11\x00\x00\x12\x00\x00\x13\x0f\x00GENERATE_SERIES\x08\x05\x07\x04\x00\x01\x00\x00\x00\x09\x04\x00\x01\x00\x00\x00\x0b\x04\x00\xf0\x01\x00\x00\x0c\x04\x00\x00\x00\x00\x00\x0d\x04\x00\x00\x00\x00\x00\x0e\x04\x00\x04\x00\x00\x00\x08\x01' gdsCodes=0
C\tExecute kind=63 stmt=2 tx=1 inBLR=12 outBLR=0 lenParameters=1 sqldata:4 '\x00\x00\x00\x03' lenParamDes=1 paramsdesc:1 496 1 0 0 4 false "" "" "" "" "" "" false timeout=0 cursorFlags=0 inlineBlobSize=65536
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 data='' gdsCodes=0
C\tFetch stmt=2 msgs=400 lenRowBLR=12 rowBLR='\x05\x02\x04\x00\x02\x00\x08\x00\x07\x00\xffL'
S\tFetchResponse status=0 msgs=1
S\tRowData cols=1 nulls=0 [col=0 name="GENERATE_SERIES" type=496 len=4 bytes=4 val='\x00\x00\x00\x01']
S\tFetchResponse status=0 msgs=1
S\tRowData cols=1 nulls=0 [col=0 name="GENERATE_SERIES" type=496 len=4 bytes=4 val='\x00\x00\x00\x02']
S\tFetchResponse status=0 msgs=1
S\tRowData cols=1 nulls=0 [col=0 name="GENERATE_SERIES" type=496 len=4 bytes=4 val='\x00\x00\x00\x03']
S\tFetchResponse status=100 msgs=0
C\tInfoSQL stmt=2 maxBuf=64 lenItems=1 items='\x17'
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=33 data='\x17\x1d\x00\x0f\x04\x00\x00\x00\x00\x00\x10\x04\x00\x00\x00\x00\x00\x0d\x04\x00\x03\x00\x00\x00\x0e\x04\x00\x00\x00\x00\x00\x01\x01' gdsCodes=0
C\tCommit
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=0 blobId=0 lenData=0 data='' gdsCodes=0
`, `\t`, "\t")

func requireTraceContainsInOrder(t *testing.T, trace string, seq ...string) {
	t.Helper()
	pos := 0
	for _, token := range seq {
		if pos > len(trace) {
			require.Failf(t, "trace order", "trace ended before finding %q\nfull trace:\n%s", token, trace)
		}

		idx := strings.Index(trace[pos:], token)
		require.NotEqualf(t, -1, idx, "not found fragment %q after %d\nfull trace:\n%s", token, pos, trace)
		pos += idx + len(token)
	}
}

func TestTrace(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer conn.Close(ctx)

	traceOutput := &bytes.Buffer{}

	conn.Client().Trace(traceOutput, fbproto.TracerOptions{
		SuppressTimestamps: true,
		FullLogMode:        false,
	})

	result := conn.ExecParams(ctx, recursiveSeriesSQL, [][]byte{{0, 0, 0, 3}}).Read()

	require.NoError(t, result.Err)

	out := traceOutput.String()
	fmt.Fprint(os.Stdout, out)

	require.Equal(t, expectedTraceSnapshot, out)
}

func TestFullLogTrace(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer conn.Close(ctx)

	traceOutput := &bytes.Buffer{}

	conn.Client().Trace(traceOutput, fbproto.TracerOptions{
		SuppressTimestamps: true,
		FullLogMode:        true,
	})

	result := conn.ExecParams(ctx, recursiveSeriesSQL, [][]byte{{0, 0, 0, 3}}).Read()

	require.NoError(t, result.Err)

	out := traceOutput.String()
	fmt.Fprint(os.Stdout, out)
	require.Equal(t, expectedFullLogTraceSnapshot, out)
}

func TestTrace_SuccessCTE(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer conn.Close(ctx)

	var buf bytes.Buffer
	conn.Client().Trace(&buf, fbproto.TracerOptions{
		SuppressTimestamps: true,
		FullLogMode:        true,
	})

	rr := conn.ExecParams(ctx, recursiveSeriesSQL, [][]byte{{0, 0, 0, 3}}).Read()

	require.NoError(t, rr.Err)
	require.Len(t, rr.Rows, 3)

	out := buf.String()
	requireTraceContainsInOrder(t, out,
		"C\tBeginTx",
		"C\tAllocateStatement",
		"C\tPrepare",
		"C\tExecute",
		"C\tFetch",
		"S\tRowData cols=1",
	)
}

func TestTrace_ErrorPath(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := fbconn.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	require.NoError(t, err)
	defer conn.Close(ctx)

	var buf bytes.Buffer
	conn.Client().Trace(&buf, fbproto.TracerOptions{
		SuppressTimestamps: true,
		FullLogMode:        false,
	})

	rr := conn.ExecParams(ctx, "SELECT * FROM NO_SUCH_TABLE", nil).Read()
	require.Error(t, rr.Err)

	out := buf.String()
	requireTraceContainsInOrder(t, out,
		"C\tPrepare",
		"S\tResponse",
		"failure=true",
	)
}
