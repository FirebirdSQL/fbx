package fbproto_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/firebirdsql/fbx/fbconn"
	"github.com/firebirdsql/fbx/fbproto"
	"github.com/stretchr/testify/require"
)

const recursiveSeriesSQL = "WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series"

var expectedTraceSnapshotV5 = strings.ReplaceAll(`C\tCancel kind=2(enable)
C\tBeginTx
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tAllocateStatement
C\tPrepare stmt=65535 tx=1 dialect=3 sql="WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series" lenItems=0 bufSize=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=0 gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=4 blobId=0 lenData=1 gdsCodes=0
C\tInfoSQL stmt=2 maxBuf=524288 lenItems=29
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=151 gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
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
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tFreeStatement
C\tCancel kind=1(disable)
C\tCommit
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=-1 blobId=0 lenData=0 gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=0 blobId=0 lenData=0 gdsCodes=0
`,
	`\t`, "\t")

var expectedTraceSnapshotV50 = strings.ReplaceAll(`C\tCancel kind=2(enable)
C\tBeginTx
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tAllocateStatement
C\tPrepare stmt=65535 tx=1 dialect=3 sql="WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series" lenItems=0 bufSize=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=0 gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=4 blobId=0 lenData=1 gdsCodes=0
C\tInfoSQL stmt=2 maxBuf=524288 lenItems=29
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=151 gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tExecute kind=63 stmt=2 tx=1 inBLR=12 outBLR=0 lenParameters=1 lenParamDes=1 timeout=0 cursorFlags=0
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
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tFreeStatement
C\tCancel kind=1(disable)
C\tCommit
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=-1 blobId=0 lenData=0 gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=0 blobId=0 lenData=0 gdsCodes=0
`, `\t`, "\t")

var expectedFullLogTraceSnapshotV5 = strings.ReplaceAll(`C\tCancel kind=2(enable)
C\tBeginTx tpb=''
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 data='' gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tAllocateStatement
C\tPrepare stmt=65535 tx=1 dialect=3 sql="WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series" lenItems=0 bufSize=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=0 data='' gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=4 blobId=0 lenData=1 data='\x01' gdsCodes=0
C\tInfoSQL stmt=2 maxBuf=524288 lenItems=29 items='\x14\x02\x00\x00\x15\x04\x07\x09\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x08\x05\x07\x09\x0b\x0c\x0d\x0e\x0f\x10\x11\x08'
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=151 data='\x15\x04\x00\x01\x00\x00\x00\x04\x07\x04\x00\x01\x00\x00\x00\x09\x04\x00\x01\x00\x00\x00\x0b\x04\x00\xf0\x01\x00\x00\x0c\x04\x00\x00\x00\x00\x00\x0d\x04\x00\x00\x00\x00\x00\x0e\x04\x00\x04\x00\x00\x00\x0f\x04\x00\x00\x00\x00\x00\x10\x08\x00CONSTANT\x11\x00\x00\x12\x00\x00\x13\x0f\x00GENERATE_SERIES\x08\x05\x07\x04\x00\x01\x00\x00\x00\x09\x04\x00\x01\x00\x00\x00\x0b\x04\x00\xf0\x01\x00\x00\x0c\x04\x00\x00\x00\x00\x00\x0d\x04\x00\x00\x00\x00\x00\x0e\x04\x00\x04\x00\x00\x00\x0f\x04\x00\x00\x00\x00\x00\x10\x00\x00\x11\x00\x00\x08\x01' gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tExecute kind=63 stmt=2 tx=1 inBLR=12 outBLR=0 lenParameters=1 sqldata:4 '\x00\x00\x00\x03' lenParamDes=1 paramsdesc:1 496 0 0 4 false "" "" "" "" "" "" false timeout=0 cursorFlags=0 inlineBlobSize=65536
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
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tFreeStatement
C\tCancel kind=1(disable)
C\tCommit
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=-1 blobId=0 lenData=0 data='' gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=0 blobId=0 lenData=0 data='' gdsCodes=0
`, `\t`, "\t")

var expectedFullLogTraceSnapshotV50 = strings.ReplaceAll(`C\tCancel kind=2(enable)
C\tBeginTx tpb=''
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 data='' gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tAllocateStatement
C\tPrepare stmt=65535 tx=1 dialect=3 sql="WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series" lenItems=0 bufSize=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=0 data='' gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=4 blobId=0 lenData=1 data='\x01' gdsCodes=0
C\tInfoSQL stmt=2 maxBuf=524288 lenItems=29 items='\x14\x02\x00\x00\x15\x04\x07\x09\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x08\x05\x07\x09\x0b\x0c\x0d\x0e\x0f\x10\x11\x08'
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=151 data='\x15\x04\x00\x01\x00\x00\x00\x04\x07\x04\x00\x01\x00\x00\x00\x09\x04\x00\x01\x00\x00\x00\x0b\x04\x00\xf0\x01\x00\x00\x0c\x04\x00\x00\x00\x00\x00\x0d\x04\x00\x00\x00\x00\x00\x0e\x04\x00\x04\x00\x00\x00\x0f\x04\x00\x00\x00\x00\x00\x10\x08\x00CONSTANT\x11\x00\x00\x12\x00\x00\x13\x0f\x00GENERATE_SERIES\x08\x05\x07\x04\x00\x01\x00\x00\x00\x09\x04\x00\x01\x00\x00\x00\x0b\x04\x00\xf0\x01\x00\x00\x0c\x04\x00\x00\x00\x00\x00\x0d\x04\x00\x00\x00\x00\x00\x0e\x04\x00\x04\x00\x00\x00\x0f\x04\x00\x00\x00\x00\x00\x10\x00\x00\x11\x00\x00\x08\x01' gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tExecute kind=63 stmt=2 tx=1 inBLR=12 outBLR=0 lenParameters=1 sqldata:4 '\x00\x00\x00\x03' lenParamDes=1 paramsdesc:1 496 0 0 4 false "" "" "" "" "" "" false timeout=0 cursorFlags=0
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
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tFreeStatement
C\tCancel kind=1(disable)
C\tCommit
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=-1 blobId=0 lenData=0 data='' gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=0 blobId=0 lenData=0 data='' gdsCodes=0
`, `\t`, "\t")

var expectedTraceSnapshotV4 = strings.ReplaceAll(`C\tCancel kind=2(enable)
C\tBeginTx
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tAllocateStatement
C\tPrepare stmt=65535 tx=1 dialect=3 sql="WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series" lenItems=0 bufSize=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=0 gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=4 blobId=0 lenData=1 gdsCodes=0
C\tInfoSQL stmt=2 maxBuf=524288 lenItems=29
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=151 gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tExecute kind=63 stmt=2 tx=1 inBLR=12 outBLR=0 lenParameters=1 lenParamDes=1 timeout=0
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
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tFreeStatement
C\tCancel kind=1(disable)
C\tCommit
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=-1 blobId=0 lenData=0 gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=0 blobId=0 lenData=0 gdsCodes=0
`, `\t`, "\t")

var expectedFullLogTraceSnapshotV4 = strings.ReplaceAll(`C\tCancel kind=2(enable)
C\tBeginTx tpb=''
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 data='' gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tAllocateStatement
C\tPrepare stmt=65535 tx=1 dialect=3 sql="WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series" lenItems=0 bufSize=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=0 data='' gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=4 blobId=0 lenData=1 data='\x01' gdsCodes=0
C\tInfoSQL stmt=2 maxBuf=524288 lenItems=29 items='\x14\x02\x00\x00\x15\x04\x07\x09\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x08\x05\x07\x09\x0b\x0c\x0d\x0e\x0f\x10\x11\x08'
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=151 data='\x15\x04\x00\x01\x00\x00\x00\x04\x07\x04\x00\x01\x00\x00\x00\x09\x04\x00\x01\x00\x00\x00\x0b\x04\x00\xf0\x01\x00\x00\x0c\x04\x00\x00\x00\x00\x00\x0d\x04\x00\x00\x00\x00\x00\x0e\x04\x00\x04\x00\x00\x00\x0f\x04\x00\x00\x00\x00\x00\x10\x08\x00CONSTANT\x11\x00\x00\x12\x00\x00\x13\x0f\x00GENERATE_SERIES\x08\x05\x07\x04\x00\x01\x00\x00\x00\x09\x04\x00\x01\x00\x00\x00\x0b\x04\x00\xf0\x01\x00\x00\x0c\x04\x00\x00\x00\x00\x00\x0d\x04\x00\x00\x00\x00\x00\x0e\x04\x00\x04\x00\x00\x00\x0f\x04\x00\x00\x00\x00\x00\x10\x00\x00\x11\x00\x00\x08\x01' gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tExecute kind=63 stmt=2 tx=1 inBLR=12 outBLR=0 lenParameters=1 sqldata:4 '\x00\x00\x00\x03' lenParamDes=1 paramsdesc:1 496 0 0 4 false "" "" "" "" "" "" false timeout=0
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
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tFreeStatement
C\tCancel kind=1(disable)
C\tCommit
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=-1 blobId=0 lenData=0 data='' gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=0 blobId=0 lenData=0 data='' gdsCodes=0
`, `\t`, "\t")

var expectedTraceSnapshotV3 = strings.ReplaceAll(`C\tCancel kind=2(enable)
C\tBeginTx
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tAllocateStatement
C\tPrepare stmt=65535 tx=1 dialect=3 sql="WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series" lenItems=0 bufSize=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=0 gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=4 blobId=0 lenData=1 gdsCodes=0
C\tInfoSQL stmt=2 maxBuf=524288 lenItems=29
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=143 gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tExecute kind=63 stmt=2 tx=1 inBLR=12 outBLR=0 lenParameters=1 lenParamDes=1
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
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tFreeStatement
C\tCancel kind=1(disable)
C\tCommit
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=-1 blobId=0 lenData=0 gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=0 blobId=0 lenData=0 gdsCodes=0
`,
	`\t`, "\t")

var expectedFullLogTraceSnapshotV3 = strings.ReplaceAll(`C\tCancel kind=2(enable)
C\tBeginTx tpb=''
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 data='' gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tAllocateStatement
C\tPrepare stmt=65535 tx=1 dialect=3 sql="WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series" lenItems=0 bufSize=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=0 data='' gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=4 blobId=0 lenData=1 data='\x01' gdsCodes=0
C\tInfoSQL stmt=2 maxBuf=524288 lenItems=29 items='\x14\x02\x00\x00\x15\x04\x07\x09\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x08\x05\x07\x09\x0b\x0c\x0d\x0e\x0f\x10\x11\x08'
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=143 data='\x15\x04\x00\x01\x00\x00\x00\x04\x07\x04\x00\x01\x00\x00\x00\x09\x04\x00\x01\x00\x00\x00\x0b\x04\x00\xf0\x01\x00\x00\x0c\x04\x00\x00\x00\x00\x00\x0d\x04\x00\x00\x00\x00\x00\x0e\x04\x00\x04\x00\x00\x00\x0f\x04\x00\x00\x00\x00\x00\x10\x00\x00\x11\x00\x00\x12\x00\x00\x13\x0f\x00GENERATE_SERIES\x08\x05\x07\x04\x00\x01\x00\x00\x00\x09\x04\x00\x01\x00\x00\x00\x0b\x04\x00\xf0\x01\x00\x00\x0c\x04\x00\x00\x00\x00\x00\x0d\x04\x00\x00\x00\x00\x00\x0e\x04\x00\x04\x00\x00\x00\x0f\x04\x00\x00\x00\x00\x00\x10\x00\x00\x11\x00\x00\x08\x01' gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tExecute kind=63 stmt=2 tx=1 inBLR=12 outBLR=0 lenParameters=1 sqldata:4 '\x00\x00\x00\x03' lenParamDes=1 paramsdesc:1 496 0 0 4 false "" "" "" "" "" "" false
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
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tFreeStatement
C\tCancel kind=1(disable)
C\tCommit
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=-1 blobId=0 lenData=0 data='' gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=0 blobId=0 lenData=0 data='' gdsCodes=0
`, `\t`, "\t")

var expectedTraceSnapshotV6 = strings.ReplaceAll(`C\tCancel kind=2(enable)
C\tBeginTx
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tAllocateStatement
C\tPrepare stmt=65535 tx=1 dialect=3 sql="WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series" lenItems=0 bufSize=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=0 gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=4 blobId=0 lenData=1 gdsCodes=0
C\tInfoSQL stmt=2 maxBuf=524288 lenItems=31
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=157 gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
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
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tFreeStatement
C\tCancel kind=1(disable)
C\tCommit
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=-1 blobId=0 lenData=0 gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=0 blobId=0 lenData=0 gdsCodes=0
`,
	`\t`, "\t")

var expectedFullLogTraceSnapshotV6 = strings.ReplaceAll(`C\tCancel kind=2(enable)
C\tBeginTx tpb=''
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=1 blobId=0 lenData=0 data='' gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tAllocateStatement
C\tPrepare stmt=65535 tx=1 dialect=3 sql="WITH RECURSIVE generate_series(generate_series) AS (SELECT 1 FROM RDB$DATABASE UNION ALL SELECT (generate_series + 1) FROM generate_series WHERE generate_series < ?) SELECT generate_series FROM generate_series" lenItems=0 bufSize=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=0 data='' gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=4 blobId=0 lenData=1 data='\x01' gdsCodes=0
C\tInfoSQL stmt=2 maxBuf=524288 lenItems=31 items='\x14\x02\x00\x00\x15\x04\x07\x09\x0b\x0c\x0d\x0e\x0f\x10\x11!\x12\x13\x08\x05\x07\x09\x0b\x0c\x0d\x0e\x0f\x10\x11!\x08'
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=2 blobId=0 lenData=157 data='\x15\x04\x00\x01\x00\x00\x00\x04\x07\x04\x00\x01\x00\x00\x00\x09\x04\x00\x01\x00\x00\x00\x0b\x04\x00\xf0\x01\x00\x00\x0c\x04\x00\x00\x00\x00\x00\x0d\x04\x00\x00\x00\x00\x00\x0e\x04\x00\x04\x00\x00\x00\x0f\x04\x00\x00\x00\x00\x00\x10\x08\x00CONSTANT\x11\x00\x00!\x00\x00\x12\x00\x00\x13\x0f\x00GENERATE_SERIES\x08\x05\x07\x04\x00\x01\x00\x00\x00\x09\x04\x00\x01\x00\x00\x00\x0b\x04\x00\xf0\x01\x00\x00\x0c\x04\x00\x00\x00\x00\x00\x0d\x04\x00\x00\x00\x00\x00\x0e\x04\x00\x04\x00\x00\x00\x0f\x04\x00\x00\x00\x00\x00\x10\x00\x00\x11\x00\x00!\x00\x00\x08\x01' gdsCodes=0
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tExecute kind=63 stmt=2 tx=1 inBLR=12 outBLR=0 lenParameters=1 sqldata:4 '\x00\x00\x00\x03' lenParamDes=1 paramsdesc:1 496 0 0 4 false "" "" "" "" "" "" false timeout=0 cursorFlags=0 inlineBlobSize=65536
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
C\tCancel kind=1(disable)
C\tCancel kind=2(enable)
C\tFreeStatement
C\tCancel kind=1(disable)
C\tCommit
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=-1 blobId=0 lenData=0 data='' gdsCodes=0
S\tResponse failure=false warnings=false sqlErr=0 sqlState=00000 sqlErr=0 sqlState="00000" message="" warn="" object=0 blobId=0 lenData=0 data='' gdsCodes=0
`, `\t`, "\t")

type traceSnapshotProfile string

const (
	traceSnapshotProfileV3  traceSnapshotProfile = "v3"
	traceSnapshotProfileV4  traceSnapshotProfile = "v4.x"
	traceSnapshotProfileV50 traceSnapshotProfile = "v5.0.x"
	traceSnapshotProfileV5  traceSnapshotProfile = "v5.x"
	traceSnapshotProfileV6  traceSnapshotProfile = "v6.x"
)

type traceSnapshotSet struct {
	trace   string
	fullLog string
}

var traceSnapshotsByProfile = map[traceSnapshotProfile]traceSnapshotSet{
	traceSnapshotProfileV3: {
		trace:   expectedTraceSnapshotV3,
		fullLog: expectedFullLogTraceSnapshotV3,
	},
	traceSnapshotProfileV4: {
		trace:   expectedTraceSnapshotV4,
		fullLog: expectedFullLogTraceSnapshotV4,
	},
	traceSnapshotProfileV50: {
		trace:   expectedTraceSnapshotV50,
		fullLog: expectedFullLogTraceSnapshotV50,
	},
	traceSnapshotProfileV5: {
		trace:   expectedTraceSnapshotV5,
		fullLog: expectedFullLogTraceSnapshotV5,
	},
	traceSnapshotProfileV6: {
		trace:   expectedTraceSnapshotV6,
		fullLog: expectedFullLogTraceSnapshotV6,
	},
}

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

func getTraceSnapshotProfileForServerEngineVersion(versionNumber int) (traceSnapshotProfile, bool) {
	major := versionNumber / 10000
	minor := (versionNumber % 10000) / 100
	patch := versionNumber % 10000

	switch major {
	case 3:
		return traceSnapshotProfileV3, true
	case 4:
		return traceSnapshotProfileV4, true
	case 5:
		if minor == 0 && patch >= 8 {
			return traceSnapshotProfileV50, true
		}
		return traceSnapshotProfileV5, true
	case 6:
		return traceSnapshotProfileV6, true
	}
	return "", false
}

func getExpectedTraceSnapshotsForServerEngineVersion(t testing.TB, versionNumber int) traceSnapshotSet {
	t.Helper()

	profile, ok := getTraceSnapshotProfileForServerEngineVersion(versionNumber)
	if !ok {
		t.Fatalf("unsupported server engine version number %d", versionNumber)
	}

	snapshots, ok := traceSnapshotsByProfile[profile]
	require.Truef(t, ok, "no trace snapshots defined for server engine version %d (profile %q)", versionNumber, profile)

	return snapshots
}

func getServerEngineVersionNumber(t testing.TB, conn *fbconn.FbConn) int {
	t.Helper()
	version := conn.ServerVersion()
	return version.Major*10000 + version.Minor*100 + version.Patch
}

func TestTrace(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer func() { _ = conn.Close(ctx) }()

	serverEngineVersionNumber := getServerEngineVersionNumber(t, conn)
	expectedSnapshots := getExpectedTraceSnapshotsForServerEngineVersion(t, serverEngineVersionNumber)

	traceOutput := &bytes.Buffer{}

	conn.Client().Trace(traceOutput, fbproto.TracerOptions{
		SuppressTimestamps: true,
		FullLogMode:        false,
	})

	result := conn.ExecParams(ctx, recursiveSeriesSQL, [][]byte{{0, 0, 0, 3}}).Read()

	require.NoError(t, result.Err)

	out := traceOutput.String()
	_, err = fmt.Fprint(os.Stdout, out)
	require.NoError(t, err)

	require.Equal(t, expectedSnapshots.trace, out)
}

func TestFullLogTrace(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	conn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer func() { _ = conn.Close(ctx) }()

	serverEngineVersionNumber := getServerEngineVersionNumber(t, conn)
	expectedSnapshots := getExpectedTraceSnapshotsForServerEngineVersion(t, serverEngineVersionNumber)

	traceOutput := &bytes.Buffer{}

	conn.Client().Trace(traceOutput, fbproto.TracerOptions{
		SuppressTimestamps: true,
		FullLogMode:        true,
	})

	result := conn.ExecParams(ctx, recursiveSeriesSQL, [][]byte{{0, 0, 0, 3}}).Read()

	require.NoError(t, result.Err)

	out := traceOutput.String()
	_, err = fmt.Fprint(os.Stdout, out)
	require.NoError(t, err)
	require.Equal(t, expectedSnapshots.fullLog, out)
}

func TestTrace_SuccessCTE(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer func() { _ = conn.Close(ctx) }()

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

	conn, err := fbconn.Connect(ctx, os.Getenv("FBX_TEST_DATABASE"))
	require.NoError(t, err)
	defer func() { _ = conn.Close(ctx) }()

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
