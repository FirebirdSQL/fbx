package fbproto

import (
	"encoding/binary"
	"fmt"
)

// https://firebirdsql.org/file/documentation/html/en/firebirddocs/wireprotocol/firebird-wire-protocol.html#wireprotocol-responses-slice

type OpSliceResponse struct {
	DataType            uint32
	SliceLengthElements int
	SliceLengthBytes    int
	SliceData           []byte
}

func (op *OpSliceResponse) Read(x *XDRReader) error {
	var err error
	var sliceLength int

	// p_slr_length
	if op.SliceLengthBytes, err = x.ReadInt32(); err != nil {
		return err
	}

	if sliceLength, err = x.ReadInt32(); err != nil {
		return err
	}

	if op.SliceLengthBytes != sliceLength {
		return fmt.Errorf("p_slr_length and Slice length are not equal")
	}

	isVariying := false
	elements := 0

	switch op.DataType {
	case Isc_blr_text, Isc_blr_text2, Isc_blr_cstring, Isc_blr_cstring2:
		elements = op.SliceLengthBytes / op.SliceLengthElements
		op.SliceLengthBytes += elements * ((4 - op.SliceLengthElements) & 3)
	case Isc_blr_varaying, Isc_blr_varaying2:
		elements = op.SliceLengthBytes / (op.SliceLengthElements + 2)
		isVariying = true
	case Isc_blr_short:
		op.SliceLengthBytes = op.SliceLengthBytes * op.SliceLengthElements
	case Isc_blr_bool:
		op.SliceLengthBytes *= 4 // bool type elements consist of 1 byte of data and 3 bytes of padding
	}

	var arrayElement []byte
	op.SliceData = make([]byte, 0)
	if isVariying {
		for i := 0; i < elements; i++ {
			if arrayElement, err = x.ReadBuffer(); err != nil {
				return err
			}

			elementLengthInBinary := make([]byte, 4)
			binary.BigEndian.PutUint32(elementLengthInBinary, uint32(len(arrayElement)))

			op.SliceData = append(op.SliceData, elementLengthInBinary...) // element length
			op.SliceData = append(op.SliceData, arrayElement...)          // element
		}
	} else {
		if op.SliceData, err = x.ReadBufferLen(int64(op.SliceLengthBytes)); err != nil {
			return err
		}
	}

	return nil
}
