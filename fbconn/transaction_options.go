package fbconn

import (
	"fmt"

	"github.com/firebirdsql/fbx/fbproto"
)

// Transaction isolation levels accepted by Config.DefaultTransactionIsolation.
const (
	Serializable   = "serializable"
	RepeatableRead = "repeatable_read"
	ReadCommitted  = "read_committed"
)

func defaultTransactionTPB(isolation string, readOnly bool) ([]byte, error) {
	if isolation == "" && !readOnly {
		return nil, nil
	}
	tpb := fbproto.NewTPBWriterV3()
	if readOnly {
		tpb.PutTag(fbproto.Isc_tpb_read)
	}
	switch isolation {
	case "":
	case Serializable:
		tpb.PutTag(fbproto.Isc_tpb_consistency)
	case RepeatableRead:
		tpb.PutTag(fbproto.Isc_tpb_concurrency)
	case ReadCommitted:
		tpb.PutTag(fbproto.Isc_tpb_read_committed)
		tpb.PutTag(fbproto.Isc_tpb_rec_version)
	default:
		return nil, fmt.Errorf("unknown default transaction isolation: %q", isolation)
	}
	return tpb.Bytes(), nil
}
