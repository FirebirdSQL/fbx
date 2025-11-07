package fbproto

import (
	"fmt"
	"strconv"
	"strings"
)

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-responses-generic

type OpResponse struct {
	Encoding     *Encoding
	BlobId       int64
	Object       int
	SqlErr       int
	Data         []byte
	GdsCodes     []int
	WarningCodes []int
	SqlState     string
	Message      string
	WarnMessage  string
	Failure      bool
	Warnings     bool
	Line         int
	Position     int
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

	readLineColState := 0
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
					*currentMsgPtr += msg.message
				} else {
					*currentMsgPtr += fmt.Sprintf("unknown gds_code: %d", gdsCode)
				}
				// next two arguments will be line and column
				if gdsCode == Isc_dsql_line_col_error {
					readLineColState = 1
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
					*currentMsgPtr += msg.message
				} else {
					*currentMsgPtr += fmt.Sprintf("unknown warning code %d", wCode)
				}
			}
		case Isc_arg_number:
			if num, err := x.ReadInt32(); err == nil {
				if currentIsError && gdsCode == Isc_sqlerr {
					op.SqlErr = num
				}
				numArg++
				*currentMsgPtr = strings.Replace(*currentMsgPtr, "@"+strconv.Itoa(numArg), strconv.Itoa(num), 1)
				if readLineColState == 1 {
					op.Line = num
					readLineColState = 2
				} else if readLineColState == 2 {
					op.Position = num
					readLineColState = 0
				}
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

	// Try to find sqlstate from collected gds codes, simulate fb_sqlstate from gds.cpp behavior
	// For errors HY000 is fallback value.
	// Skip any informational 00000 values.
	// Accept 22000, 42000 values, but keep search more for exact sqlstate if possible
	if op.SqlState == "" {
		if op.Failure {
			op.SqlState = "HY000"
			for _, gdsCode := range op.GdsCodes {
				if msg, ok := errmsgs[gdsCode]; ok {
					if msg.sqlstate == "00000" {
						continue
					}
					op.SqlState = msg.sqlstate
					if op.SqlState != "22000" && op.SqlState != "42000" && op.SqlState != "HY000" {
						break
					}
				}
			}
		} else {
			op.SqlState = "00000"
		}
	}

	return nil
}

type ErrorResponse struct {
	ServerMessage
	Object   int
	BlobId   int64
	GdsCodes []int
	SqlErr   int
	SqlState string
	Message  string
	Line     int
	Position int
}
