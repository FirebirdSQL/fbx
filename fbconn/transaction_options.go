package fbconn

import (
	"fmt"

	"github.com/firebirdsql/fbx/fbproto"
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
	case "serializable":
		tpb.PutTag(fbproto.Isc_tpb_consistency)
	case "repeatable read":
		tpb.PutTag(fbproto.Isc_tpb_concurrency)
	case "read committed":
		tpb.PutTag(fbproto.Isc_tpb_read_committed)
		tpb.PutTag(fbproto.Isc_tpb_rec_version)
	default:
		return nil, fmt.Errorf("unknown default transaction isolation: %q", isolation)
	}
	return tpb.Bytes(), nil
}
