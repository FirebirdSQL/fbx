package fbproto

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

type tracer struct {
	TracerOptions

	mx  sync.Mutex
	w   io.Writer
	buf *bytes.Buffer
}

type TracerOptions struct {
	SuppressTimestamps bool
	FullLogMode        bool
}

func (t *tracer) traceRequest(sender byte, msg ClientMessage) {
	switch msg := msg.(type) {
	case *OpConnect:
		t.traceConnectMessage(sender, msg)
	case *OpAttach:
		t.traceAttachMessage(sender, msg)
	case *OpAllocateStatement:
		t.traceAllocateMessage(sender, msg)
	case *OpPrepareStatement:
		t.tracePrepareMessage(sender, msg)
	case *OpTransaction:
		t.traceTransactionMessage(sender, msg)
	case *OpCommit:
		t.traceCommitMessage(sender, msg)
	case *OpCommitRetaining:
		t.traceCommitRetainingMessage(sender, msg)
	case *OpRollback:
		t.traceRollbackMessage(sender, msg)
	case *OpRollbackRetaining:
		t.traceRollbackRetainingMessage(sender, msg)
	case *OpFreeStatement:
		t.traceFreeStatementMessage(sender, msg)
	case *OpExecute:
		t.traceExecuteMessage(sender, msg)
	case *OpFetch:
		t.traceFetchMessage(sender, msg)
	case *OpInfoSQL:
		t.traceInfoSQLMessage(sender, msg)
	case *OpDisconnect:
		t.writeTrace(sender, "Disconnect", nil)
	case *OpDetach:
		t.writeTrace(sender, "Detach", nil)
	case *OpContAuth:
		t.traceContAuthMessage(sender, msg)
	case *OpCrypt:
		t.traceCryptMessage(sender, msg)
	default:
		t.writeTrace(sender, fmt.Sprintf("Unknown message type: %T", msg), nil)
	}
}

func (t *tracer) traceResponse(sender byte, msg ServerMessage) {
	switch msg := msg.(type) {
	case *OpResponse:
		t.traceResponseMessage(sender, msg)
	case *OpSqlResponse:
		t.traceSqlResponseMessage(sender, msg)
	case *OpFetchResponse:
		t.traceFetchResponseMessage(sender, msg)
	case *OpReject:
		t.writeTrace(sender, "Reject", nil)
	case *OpContAuth:
		t.traceContAuthMessage(sender, msg)
	case *OpCondAccept:
		t.traceCondAcceptMessage(sender, msg)
	case *OpDataAccept:
		t.traceDataAcceptMessage(sender, msg)
	case *OpDummy:
		t.writeTrace(sender, "Dummy", nil)
	default:
		t.writeTrace(sender, fmt.Sprintf("Unknown message type: %T", msg), nil)
	}
}

func (t *tracer) tracePrepareMessage(sender byte, msg *OpPrepareStatement) {
	t.writeTrace(sender, "Prepare", func() {
		fmt.Fprintf(t.buf, " stmt=%d tx=%d dialect=%d", msg.StatementHandle, msg.TransactionHandle, msg.Dialect)
		fmt.Fprintf(t.buf, " sql=%s", traceDoubleQuotedString([]byte(msg.Query)))
		fmt.Fprintf(t.buf, " lenItems=%d bufSize=%d", len(msg.StatementInformationItems), msg.StatementInformationBufferSize)
		if t.FullLogMode && len(msg.StatementInformationItems) > 0 {
			fmt.Fprintf(t.buf, " items=%s", traceSingleQuotedString(msg.StatementInformationItems))
		}
	})
}

func (t *tracer) traceFreeStatementMessage(sender byte, msg *OpFreeStatement) {
	t.writeTrace(sender, "FreeStatement", nil)
}

func (t *tracer) traceFetchMessage(sender byte, msg *OpFetch) {
	t.writeTrace(sender, "Fetch", func() {
		fmt.Fprintf(t.buf, " stmt=%d msgs=%d", msg.StatementHandle, msg.Messages)
		fmt.Fprintf(t.buf, " lenRowBLR=%d", len(msg.RowBlr))
		if t.FullLogMode && len(msg.RowBlr) > 0 {
			fmt.Fprintf(t.buf, " rowBLR=%s", traceSingleQuotedString(msg.RowBlr))
		}
	})
}

func (t *tracer) traceInfoSQLMessage(sender byte, msg *OpInfoSQL) {
	t.writeTrace(sender, "InfoSQL", func() {
		fmt.Fprintf(t.buf, " stmt=%d maxBuf=%d", msg.StatementHandle, msg.MaxBufferLength)
		fmt.Fprintf(t.buf, " lenItems=%d", len(msg.StatementInformationItems))
		if t.FullLogMode && len(msg.StatementInformationItems) > 0 {
			fmt.Fprintf(t.buf, " items=%s", traceSingleQuotedString(msg.StatementInformationItems))
		}
	})
}

func (t *tracer) traceAllocateMessage(sender byte, msg *OpAllocateStatement) {
	t.writeTrace(sender, "AllocateStatement", nil)
}

func (t *tracer) traceCommitRetainingMessage(sender byte, msg *OpCommitRetaining) {
	t.writeTrace(sender, "CommitRetaining", nil)
}

func (t *tracer) traceResponseMessage(sender byte, msg *OpResponse) {
	writeDetails := func() {
		fmt.Fprintf(t.buf, " failure=%t warnings=%t sqlErr=%d sqlState=%s", msg.Failure, msg.Warnings, msg.SqlErr, msg.SqlState)
		fmt.Fprintf(t.buf, " sqlErr=%d", msg.SqlErr)
		fmt.Fprintf(t.buf, " sqlState=%s", traceDoubleQuotedString([]byte(msg.SqlState)))
		fmt.Fprintf(t.buf, " message=%s", traceDoubleQuotedString([]byte(msg.Message)))
		fmt.Fprintf(t.buf, " warn=%s", traceDoubleQuotedString([]byte(msg.WarnMessage)))
		fmt.Fprintf(t.buf, " object=%d", msg.Object)
		fmt.Fprintf(t.buf, " blobId=%d", msg.BlobId)
		fmt.Fprintf(t.buf, " lenData=%d", len(msg.Data))
		if t.FullLogMode {
			fmt.Fprintf(t.buf, " data=%s", traceSingleQuotedString(msg.Data))
		}
		fmt.Fprintf(t.buf, " gdsCodes=%d", len(msg.GdsCodes))
	}

	t.writeTrace(sender, "Response", writeDetails)
}

func (t *tracer) traceExecuteMessage(sender byte, op *OpExecute) {
	writeDetails := func() {
		fmt.Fprintf(t.buf, " kind=%d", op.Kind)
		fmt.Fprintf(t.buf, " stmt=%d tx=%d", op.StatementHandle, op.TransactionHandle)
		fmt.Fprintf(t.buf, " inBLR=%d outBLR=%d", len(op.SQLDataBlr), len(op.SQLDataOutBlr))
		fmt.Fprintf(t.buf, " lenSQLData=%d", len(op.SQLData))

		if t.FullLogMode {
			for _, v := range op.SQLData {
				if v == nil {
					t.buf.WriteString(" sqldata:-1")
				} else {
					fmt.Fprintf(t.buf, " sqldata:%d %s", len(v), traceSingleQuotedString(v))
				}
			}
		}

		fmt.Fprintf(t.buf, " lenParamDes=%d", len(op.ParametersDescriptors))
		if t.FullLogMode {
			for _, v := range op.ParametersDescriptors {
				fmt.Fprintf(t.buf, " paramsdesc:%d %d %d %d %d %d %t %s %s %s %s %s %s %t", v.Index, v.Type, v.Format, v.Scale,
					v.Subtype, v.Length, v.NullOk, traceDoubleQuotedString([]byte(v.Name)), traceDoubleQuotedString([]byte(v.TableAliasName)),
					traceDoubleQuotedString([]byte(v.RelName)), traceDoubleQuotedString([]byte(v.OwnerName)), traceDoubleQuotedString([]byte(v.OriginalName)),
					traceDoubleQuotedString([]byte(v.OriginalTableName)), v.Nullable)
			}
		}
	}

	t.writeTrace(sender, "Execute", writeDetails)
}

func (t *tracer) traceConnectMessage(sender byte, op *OpConnect) {
	writeDetails := func() {
		fmt.Fprintf(t.buf, " db=%s", traceDoubleQuotedString([]byte(op.Database)))
		fmt.Fprintf(t.buf, " user=%s", traceDoubleQuotedString([]byte(op.User)))
		fmt.Fprintf(t.buf, " systemUser=%s", traceDoubleQuotedString([]byte(op.SystemUser)))
		fmt.Fprintf(t.buf, " host=%s", traceDoubleQuotedString([]byte(op.Hostname)))
		fmt.Fprintf(t.buf, " pass=%s", traceDoubleQuotedString([]byte(op.Password)))
		fmt.Fprintf(t.buf, " plugin=%s", traceDoubleQuotedString([]byte(op.AuthPluginName)))

		n := len(op.AuthPluginsList)
		fmt.Fprintf(t.buf, " pluginList=%d", n)

		n = len(op.AuthSpecificData)
		fmt.Fprintf(t.buf, " authSpecificChunks=%d", n)
		if t.FullLogMode {
			for _, data := range op.AuthSpecificData {
				fmt.Fprintf(t.buf, " authspdata:%c %s", data.Part, traceSingleQuotedString(data.Data))
			}
		}
		fmt.Fprintf(t.buf, " charset=%s", traceDoubleQuotedString([]byte(op.Charset)))
		fmt.Fprintf(t.buf, " wireCrypt=%t", op.WireCrypt)
		fmt.Fprintf(t.buf, " compress=%t", op.Compress)

		if n := len(op.Options); n > 0 {
			fmt.Fprintf(t.buf, " options=%d", n)
		}
	}

	t.writeTrace(sender, "Connect", writeDetails)
}

func (t *tracer) traceContAuthMessage(sender byte, op *OpContAuth) {
	t.writeTrace(sender, "ContAuth", func() {
		fmt.Fprintf(t.buf, " name=%s", traceDoubleQuotedString([]byte(op.PluginName)))
		fmt.Fprintf(t.buf, " list=%s", traceDoubleQuotedString([]byte(op.PluginList)))
		fmt.Fprintf(t.buf, " lenAuthData=%d", len(op.AuthData))
		fmt.Fprintf(t.buf, " lenKeys=%d", len(op.Keys))
		if t.FullLogMode {
			fmt.Fprintf(t.buf, " authData=%s", traceSingleQuotedString(op.AuthData))
			fmt.Fprintf(t.buf, " keys=%s", traceSingleQuotedString(op.Keys))
		}
	})
}

func (t *tracer) traceSqlResponseMessage(sender byte, op *OpSqlResponse) {
	t.writeTrace(sender, "SqlResponse", func() {
		fmt.Fprintf(t.buf, " messages=%d", op.Messages)
	})
}

func (t *tracer) traceFetchResponseMessage(sender byte, op *OpFetchResponse) {
	t.writeTrace(sender, "FetchResponse", func() {
		fmt.Fprintf(t.buf, " status=%d", op.SQLDataStatus)
		fmt.Fprintf(t.buf, " msgs=%d", op.Messages)
	})
}

func (t *tracer) traceCondAcceptMessage(sender byte, op *OpCondAccept) {
	t.writeTrace(sender, "CondAccept", func() {
		fmt.Fprintf(t.buf, " ver=%d arch=%d type=0x%x", op.Version, op.Architecture, op.Type)
		fmt.Fprintf(t.buf, " authComplete=%t", op.AuthComplete)
		fmt.Fprintf(t.buf, " lenPluginName=%d", len(op.PluginName))
		fmt.Fprintf(t.buf, " plugin=%s", traceDoubleQuotedString([]byte(op.PluginName)))
		fmt.Fprintf(t.buf, " lenData=%d", len(op.Data))
		fmt.Fprintf(t.buf, " lenServKeys=%d", len(op.ServerKeys))
		if t.FullLogMode {
			fmt.Fprintf(t.buf, " data=%s", traceSingleQuotedString(op.Data))
			fmt.Fprintf(t.buf, " serverKeys=%s", traceSingleQuotedString(op.ServerKeys))
		}
		fmt.Fprintf(t.buf, " compress=%t", op.Compress)
	})
}

func (t *tracer) traceDataAcceptMessage(sender byte, op *OpDataAccept) {
	t.traceCondAcceptMessage(sender, &op.OpCondAccept)
}

func (t *tracer) traceCryptMessage(sender byte, _ *OpCrypt) {
	t.writeTrace(sender, "Crypt", nil)
}

func (t *tracer) traceAttachMessage(sender byte, msg *OpAttach) {
	t.writeTrace(sender, "Attach", func() {
		fmt.Fprintf(t.buf, " file=%s", traceDoubleQuotedString([]byte(msg.File)))
		fmt.Fprintf(t.buf, " dpbLen=%d", len(msg.DPB))
		if t.FullLogMode {
			fmt.Fprintf(t.buf, " DPB=%s", traceSingleQuotedString(msg.DPB))
		}
	})
}

func (t *tracer) traceTransactionMessage(sender byte, op *OpTransaction) {
	if t.FullLogMode {
		t.writeTrace(sender, "BeginTx", func() {
			fmt.Fprintf(t.buf, " tpb=%s", traceSingleQuotedString(op.TPB))
		})
	} else {
		t.writeTrace(sender, "BeginTx", nil)
	}

}

func (t *tracer) traceCommitMessage(sender byte, op *OpCommit) {
	t.writeTrace(sender, "Commit", nil)
}

func (t *tracer) traceRollbackMessage(sender byte, op *OpRollback) {
	t.writeTrace(sender, "Rollback", nil)
}

func (t *tracer) traceRollbackRetainingMessage(sender byte, op *OpRollbackRetaining) {
	t.writeTrace(sender, "RollbackRetaining", nil)
}

func (t *tracer) traceRowData(sender byte, sd *SQLData, cols int) {
	if sd == nil {
		t.writeTrace(sender, "RowData", func() {
			fmt.Fprint(t.buf, " <nil>")
		})
		return
	}

	if cols <= 0 || cols > len(sd.RowData) {
		cols = len(sd.RowData)
	}

	nulls := 0
	if sd.Nulls != nil {
		for i := 0; i < cols; i++ {
			if sd.Nulls.Get(i) {
				nulls++
			}
		}
	}

	writeDetails := func() {
		fmt.Fprintf(t.buf, " cols=%d nulls=%d", cols, nulls)

		if !t.FullLogMode {
			for i := 0; i < cols; i++ {
				if sd.RowData[i] == nil {
					t.buf.WriteString(" [NULL]")
				} else {
					fmt.Fprintf(t.buf, " [%d]", len(sd.RowData[i]))
				}
			}
			return
		}

		for i := 0; i < cols; i++ {
			var fd FieldDescriptor
			if i < len(sd.RowDescriptor) {
				fd = sd.RowDescriptor[i]
			}

			fmt.Fprintf(t.buf, " [col=%d", i)
			if fd.Name != "" {
				fmt.Fprintf(t.buf, " name=%s", traceDoubleQuotedString([]byte(fd.Name)))
			}
			if fd.Type != 0 {
				fmt.Fprintf(t.buf, " type=%d", fd.Type)
			}
			if fd.Length != 0 {
				fmt.Fprintf(t.buf, " len=%d", fd.Length)
			}

			val := sd.RowData[i]
			if val == nil {
				t.buf.WriteString(" NULL]")
			} else {
				fmt.Fprintf(t.buf, " bytes=%d val=%s]", len(val), traceSingleQuotedString(val))
			}
		}
	}

	t.writeTrace(sender, "RowData", writeDetails)
}

func (t *tracer) writeTrace(sender byte, msgType string, writeDetails func()) {
	t.mx.Lock()
	defer t.mx.Unlock()
	defer func() {
		if t.buf.Cap() > 1024 {
			t.buf = &bytes.Buffer{}
		} else {
			t.buf.Reset()
		}
	}()

	if !t.SuppressTimestamps {
		now := time.Now()
		t.buf.WriteString(now.Format("2006-01-02 15:04:05.000000"))
		t.buf.WriteByte('\t')
	}

	t.buf.WriteByte(sender)
	t.buf.WriteByte('\t')
	t.buf.WriteString(msgType)
	// t.buf.WriteByte('\t')
	//t.buf.WriteString(strconv.FormatInt(int64(encodedLen), 10))

	if writeDetails != nil {
		writeDetails()
	}

	t.buf.WriteByte('\n')
	t.buf.WriteTo(t.w)
}

func traceSingleQuotedString(buf []byte) string {
	sb := &strings.Builder{}

	sb.WriteByte('\'')
	for _, b := range buf {
		if b < 32 || b > 126 {
			fmt.Fprintf(sb, `\x%02x`, b)
		} else {
			sb.WriteByte(b)
		}
	}
	sb.WriteByte('\'')

	return sb.String()
}

func traceDoubleQuotedString(buf []byte) string {
	return `"` + string(buf) + `"`
}
