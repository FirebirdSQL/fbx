package fbproto

import (
	"container/list"
	"fmt"
	"strconv"
	"strings"
)

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-responses-generic

type OpResponse struct {
	Encoding     *Encoding
	BlobId       int64
	Object       int
	SqlCode      int
	Data         []byte
	GdsCodes     []int
	WarningCodes []int
	SqlState     string
	Message      string
	WarnMessage  string
	Failure      bool
	Warnings     bool
}

func (op *OpResponse) Read(x *XDRReader) error {
	var err error
	// p_resp_object
	if op.Object, err = x.ReadInt32(); err != nil {
		return err
	}

	// p_resp_blob_id
	if op.BlobId, err = x.ReadInt64(); err != nil {
		return err
	}

	// p_resp_data
	if op.Data, err = x.ReadBuffer(); err != nil {
		return err
	}

	// p_resp_status_vector
	var (
		currentMsgPtr  *string = &op.Message
		numArg         int     = 0
		gdsCode        int     = 0
		firstSeen              = true
		currentIsError         = false
	)

	for {
		var tag int
		if tag, err = x.ReadInt32(); err != nil {
			return err
		}
		if tag == Isc_arg_end {
			break
		}

		switch tag {
		case Isc_arg_gds:
			if gdsCode, err = x.ReadInt32(); err != nil {
				return err
			}
			if firstSeen {
				firstSeen = false
				if gdsCode > 0 {
					op.Failure = true
				}
			}
			if gdsCode != 0 {
				currentMsgPtr = &op.Message
				currentIsError = true
				numArg = 0
				if gdsCode != Isc_random && gdsCode != Isc_sqlerr {
					op.GdsCodes = append(op.GdsCodes, gdsCode)
				}
				if msg, ok := errmsgs[gdsCode]; ok {
					*currentMsgPtr += msg
				} else {
					*currentMsgPtr += fmt.Sprintf("unknown gds_code: %d", gdsCode)
				}
			}
		case Isc_arg_warning:
			wCode, err := x.ReadInt32()
			if err != nil {
				return err
			}
			if firstSeen {
				firstSeen = false
				if wCode > 0 {
					op.Warnings = true
				}
			}
			if wCode != 0 {
				op.Warnings = true
				currentMsgPtr = &op.WarnMessage
				currentIsError = false
				numArg = 0

				if wCode != Isc_random {
					op.WarningCodes = append(op.WarningCodes, wCode)
				}
				if msg, ok := errmsgs[wCode]; ok {
					*currentMsgPtr += msg
				} else {
					*currentMsgPtr += fmt.Sprintf("unknown warning code %d", wCode)
				}
			}
		case Isc_arg_number:
			if num, err := x.ReadInt32(); err == nil {
				if currentIsError && gdsCode == Isc_sqlerr {
					op.SqlCode = num
				}
				numArg++
				*currentMsgPtr = strings.Replace(*currentMsgPtr, "@"+strconv.Itoa(numArg), strconv.Itoa(num), 1)
			} else {
				return err
			}
		case Isc_arg_string:
			if str, err := x.ReadString(op.Encoding); err == nil {
				numArg++
				*currentMsgPtr = strings.Replace(*currentMsgPtr, "@"+strconv.Itoa(numArg), str, 1)
			} else {
				return err
			}
		case Isc_arg_interpreted:
			if str, err := x.ReadString(op.Encoding); err == nil {
				*currentMsgPtr += str
			} else {
				return err
			}
		case Isc_arg_sql_state:
			if str, err := x.ReadString(op.Encoding); err == nil {
				op.SqlState = str
			} else {
				return err
			}
		default:
			panic(fmt.Sprintf("unknown isc tag: %d", tag))
		}
	}

	return nil
}

type ErrorResponse struct {
	ServerMessage
	Object   int
	BlobId   int64
	GdsCodes []int
	SqlCode  int
	SqlState string
	Message  string
}
