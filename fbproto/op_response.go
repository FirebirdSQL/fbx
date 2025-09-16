package fbproto

import (
	"container/list"
	"fmt"
	"strconv"
	"strings"
)

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-responses-generic

type OpResponse struct {
	Encoding *Encoding
	Object   int
	BlobId   int64
	Data     []byte
	GdsCodes list.List
	SqlCode  int
	SqlState string
	Message  string
	Failure  bool
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
	var numArg int
	var gdsCode int
	start := true
	for {
		var tag int
		if tag, err = x.ReadInt32(); err != nil {
			return err
		}
		if tag == isc_arg_end {
			break
		}
		switch tag {
		case isc_arg_gds:
			if gdsCode, err = x.ReadInt32(); err == nil {
				if start {
					start = false
					if gdsCode > 0 {
						op.Failure = true
					}
				}
				if gdsCode != 0 {
					op.GdsCodes.PushBack(gdsCode)
					if msg, ok := errmsgs[gdsCode]; ok {
						op.Message += msg
					} else {
						op.Message += fmt.Sprintf("unknown gds_code: %d", gdsCode)
					}
					numArg = 0
				}
			} else {
				return err
			}
		case isc_arg_number:
			if num, err := x.ReadInt32(); err == nil {
				if gdsCode == 335544436 { //isc_sqlerr
					op.SqlCode = num
				}
				numArg++
				op.Message = strings.Replace(op.Message, "@"+strconv.Itoa(numArg), strconv.Itoa(num), 1)
			} else {
				return err
			}
		case isc_arg_string:
			if str, err := x.ReadString(op.Encoding); err == nil {
				numArg++
				op.Message = strings.Replace(op.Message, "@"+strconv.Itoa(numArg), str, 1)
			} else {
				return err
			}
		case isc_arg_interpreted:
			if str, err := x.ReadString(op.Encoding); err == nil {
				op.Message += str

			} else {
				return err
			}
		case isc_arg_sql_state:
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
	GdsCodes list.List
	SqlCode  int
	SqlState string
	Message  string
}
