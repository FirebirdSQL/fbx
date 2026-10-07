package fbproto

import (
	"fmt"
	"strconv"
	"strings"
)

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-responses-generic

const opResponseDataCapMax = 64 * 1024

type OpResponse struct {
	Encoding    *Encoding
	BlobId      int64
	Object      int
	SqlErr      int
	Data        []byte
	GdsCodes    []int
	SqlState    string
	Message     string
	WarnMessage string
	Failure     bool
	Warnings    bool
	Line        int
	Position    int
}

type statusVectorResult struct {
	GdsCodes    []int
	SqlErr      int
	SqlState    string
	Message     string
	WarnMessage string
	Failure     bool
	Warnings    bool
	Line        int
	Position    int
}

func (op *OpResponse) resetForRead() {
	op.Object = 0
	op.BlobId = 0
	op.SqlErr = 0
	// Avoid retaining large buffers indefinitely allow
	// growth when needed, but shrink back to a reasonable baseline.
	// Note: this limits retained capacity, not the segment size that can be read.
	if cap(op.Data) <= 0 {
		op.Data = make([]byte, 0, 512)
	} else if cap(op.Data) > opResponseDataCapMax {
		// Cap retained buffer size to a maximum 64KB to avoid keeping large allocations.
		op.Data = make([]byte, 0, opResponseDataCapMax)
	} else {
		op.Data = op.Data[:0]
	}
	op.GdsCodes = op.GdsCodes[:0]
	op.SqlState = ""
	op.Message = ""
	op.WarnMessage = ""
	op.Failure = false
	op.Warnings = false
	op.Line = 0
	op.Position = 0
}

func (sv statusVectorResult) toErrorResponse() ErrorResponse {
	return ErrorResponse{
		GdsCodes: append([]int(nil), sv.GdsCodes...),
		SqlErr:   sv.SqlErr,
		SqlState: sv.SqlState,
		Message:  sv.Message,
		Line:     sv.Line,
		Position: sv.Position,
	}
}

func readStatusVector(x *XDRReader, encoding *Encoding) (statusVectorResult, error) {
	var (
		sv               statusVectorResult
		err              error
		currentMsgPtr    = &sv.Message
		currentMsgParams int
		numArg           int
		gdsCode          int
		firstSeen        = true
		currentIsError   bool
	)

	readLineColState := 0
	for {
		var tag int
		if tag, err = x.ReadInt32(); err != nil {
			return sv, err
		}
		if tag == Isc_arg_end {
			break
		}

		switch tag {
		case Isc_arg_gds:
			if gdsCode, err = x.ReadInt32(); err != nil {
				return sv, err
			}
			if firstSeen {
				firstSeen = false
				if gdsCode > 0 {
					sv.Failure = true
				}
			}
			if gdsCode != 0 {
				currentMsgPtr = &sv.Message
				currentIsError = true
				numArg = 0
				if gdsCode != Isc_random && gdsCode != Isc_sqlerr {
					sv.GdsCodes = append(sv.GdsCodes, gdsCode)
				}
				if msg, ok := errmsgs[gdsCode]; ok {
					if len(*currentMsgPtr) == 0 {
						*currentMsgPtr += msg.message
					} else {
						*currentMsgPtr += "\n" + msg.message
					}
					currentMsgParams = msg.params
				} else {
					tmpMsg := fmt.Sprintf("unknown gds_code: %d", gdsCode)
					if len(*currentMsgPtr) == 0 {
						*currentMsgPtr += tmpMsg
					} else {
						*currentMsgPtr += "\n" + tmpMsg
					}
					currentMsgParams = 0
				}
				if gdsCode == Isc_dsql_line_col_error {
					readLineColState = 1
				}
			}
		case Isc_arg_warning:
			wCode, err := x.ReadInt32()
			if err != nil {
				return sv, err
			}
			if firstSeen {
				firstSeen = false
				if wCode > 0 {
					sv.Warnings = true
				}
			}
			if wCode != 0 {
				sv.Warnings = true
				currentMsgPtr = &sv.WarnMessage
				currentIsError = false
				numArg = 0

				if wCode != Isc_random {
					sv.GdsCodes = append(sv.GdsCodes, wCode)
				}
				if msg, ok := errmsgs[wCode]; ok {
					if len(*currentMsgPtr) == 0 {
						*currentMsgPtr += msg.message
					} else {
						*currentMsgPtr += "\n" + msg.message
					}
					currentMsgParams = msg.params
				} else {
					tmpMsg := fmt.Sprintf("unknown gds_code: %d", wCode)
					if len(*currentMsgPtr) == 0 {
						*currentMsgPtr += tmpMsg
					} else {
						*currentMsgPtr += "\n" + tmpMsg
					}
					currentMsgParams = 0
				}
			}
		case Isc_arg_number:
			num, err := x.ReadInt32()
			if err != nil {
				return sv, err
			}
			if currentIsError && gdsCode == Isc_sqlerr {
				sv.SqlErr = num
			}
			numArg++
			if numArg <= currentMsgParams {
				*currentMsgPtr = strings.Replace(*currentMsgPtr, "@"+strconv.Itoa(numArg), strconv.Itoa(num), 1)
			} else {
				*currentMsgPtr = fmt.Sprintf("%s\n%d", *currentMsgPtr, num)
			}
			switch readLineColState {
			case 1:
				sv.Line = num
				readLineColState = 2
			case 2:
				sv.Position = num
				readLineColState = 0
			}
		case Isc_arg_interpreted:
			fallthrough
		case Isc_arg_string:
			str, err := x.ReadString(encoding)
			if err != nil {
				return sv, err
			}
			numArg++
			if numArg <= currentMsgParams {
				*currentMsgPtr = strings.Replace(*currentMsgPtr, "@"+strconv.Itoa(numArg), str, 1)
			} else {
				*currentMsgPtr += "\n" + str
			}
		case Isc_arg_sql_state:
			if sv.SqlState, err = x.ReadString(encoding); err != nil {
				return sv, err
			}
		default:
			panic(fmt.Sprintf("unknown isc tag: %d", tag))
		}
	}

	if sv.SqlState == "" {
		if sv.Failure || sv.Warnings {
			sv.SqlState = "HY000"
			for _, gdsCode := range sv.GdsCodes {
				if msg, ok := errmsgs[gdsCode]; ok {
					if msg.sqlstate == "" || msg.sqlstate == "00000" {
						continue
					}
					sv.SqlState = msg.sqlstate
					if sv.SqlState != "22000" && sv.SqlState != "42000" && sv.SqlState != "HY000" {
						break
					}
				}
			}
		} else {
			sv.SqlState = "00000"
		}
	}

	return sv, nil
}

func (op *OpResponse) Read(x *XDRReader) error {
	var err error
	op.resetForRead()
	// p_resp_object
	if op.Object, err = x.ReadInt32(); err != nil {
		return err
	}

	// p_resp_blob_id
	if op.BlobId, err = x.ReadInt64(); err != nil {
		return err
	}

	// p_resp_data
	dataLen, err := x.ReadInt32()
	if err != nil {
		return err
	}
	if dataLen < 0 {
		return fmt.Errorf("invalid response data length: %d", dataLen)
	}
	if dataLen == 0 {
		op.Data = op.Data[:0]
	} else {
		if cap(op.Data) < dataLen {
			op.Data = make([]byte, dataLen)
		} else {
			op.Data = op.Data[:dataLen]
		}
		if err := x.ReadBytesInto(op.Data); err != nil {
			return err
		}
		if err := x.SkipPadding(int64(dataLen)); err != nil {
			return err
		}
	}

	sv, err := readStatusVector(x, op.Encoding)
	if err != nil {
		return err
	}
	op.GdsCodes = append(op.GdsCodes[:0], sv.GdsCodes...)
	op.SqlErr = sv.SqlErr
	op.SqlState = sv.SqlState
	op.Message = sv.Message
	op.WarnMessage = sv.WarnMessage
	op.Failure = sv.Failure
	op.Warnings = sv.Warnings
	op.Line = sv.Line
	op.Position = sv.Position

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

func errorResponseFromOpResponse(op *OpResponse) *ErrorResponse {
	return &ErrorResponse{
		Object:   op.Object,
		BlobId:   op.BlobId,
		GdsCodes: append([]int(nil), op.GdsCodes...),
		SqlErr:   op.SqlErr,
		SqlState: op.SqlState,
		Message:  op.Message,
		Line:     op.Line,
		Position: op.Position,
	}
}
