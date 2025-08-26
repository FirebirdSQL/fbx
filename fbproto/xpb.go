package fbproto

import "github.com/red-soft/fbx/fbproto/utils"

const xpbPreallocBufSize = 16

type XPBReader struct {
	buf []byte
	pos int
}

type XPBWriter struct {
	buf []byte
}

func NewXPBReader(buf []byte) *XPBReader {
	return &XPBReader{buf, 0}
}

func (pb *XPBReader) Next() (have bool, value byte) {
	if pb.End() {
		return false, 0
	}
	b := pb.buf[pb.pos]
	pb.pos++
	return true, b
}

func (pb *XPBReader) Skip(count int) {
	pb.pos += count
}

func (pb *XPBReader) End() bool {
	return pb.pos >= len(pb.buf)
}

func (pb *XPBReader) Get() byte {
	b := pb.buf[pb.pos]
	return b
}

func (pb *XPBReader) GetString() string {
	l := int(pb.GetInt16())
	s := utils.Bytes_to_str(pb.buf[pb.pos : pb.pos+l])
	pb.pos += l
	return s
}

func (pb *XPBReader) GetInt16() int16 {
	r := utils.Bytes_to_int16(pb.buf[pb.pos : pb.pos+2])
	pb.pos += 2
	return r
}

func (pb *XPBReader) GetInt32() int32 {
	r := utils.Bytes_to_int32(pb.buf[pb.pos : pb.pos+4])
	pb.pos += 4
	return r
}

func (pb *XPBReader) GetInt64() int64 {
	r := utils.Bytes_to_int64(pb.buf[pb.pos : pb.pos+8])
	pb.pos += 8
	return r
}

func (pb *XPBReader) Reset() {
	pb.pos = 0
}

func NewXPBWriter() *XPBWriter {
	return &XPBWriter{
		buf: make([]byte, 0, xpbPreallocBufSize),
	}
}

func NewXPBWriterFromTag(tag byte) *XPBWriter {
	return NewXPBWriter().AppendTag(tag)
}

func NewXPBWriterFromBytes(bytes []byte) *XPBWriter {
	return NewXPBWriter().AppendBytes(bytes)
}

func (pb *XPBWriter) AppendTag(tag byte) *XPBWriter {
	pb.buf = append(pb.buf, []byte{tag}...)
	return pb
}

func (pb *XPBWriter) AppendByte(val byte) *XPBWriter {
	pb.buf = append(pb.buf, val)
	return pb
}

func (pb *XPBWriter) AppendInt16(val int16) *XPBWriter {
	pb.buf = append(pb.buf, utils.Int16_to_bytes(val)...)
	return pb
}

func (pb *XPBWriter) AppendInt32(val int32) *XPBWriter {
	pb.buf = append(pb.buf, utils.Int32_to_bytes(val)...)
	return pb
}

func (pb *XPBWriter) PutInt64(val int64) *XPBWriter {
	pb.buf = append(pb.buf, utils.Int64_to_bytes(val)...)
	return pb
}

func (pb *XPBWriter) AppendBytes(bytes []byte) *XPBWriter {
	pb.buf = append(pb.buf, bytes...)
	return pb
}

func (pb *XPBWriter) Bytes() []byte {
	return pb.buf
}

func (pb *XPBWriter) Reset() *XPBWriter {
	pb.buf = make([]byte, 0, xpbPreallocBufSize)
	return pb
}

type DPBWriterV2 struct {
	w *XPBWriter
}

func NewDPBWriterV2() *DPBWriterV2 {
	return &DPBWriterV2{w: NewXPBWriterFromTag(Isc_dpb_version2)}
}

func (pb *DPBWriterV2) PutTag(tag byte) {
	pb.w.AppendByte(tag)
}

func (pb *DPBWriterV2) PutByte(tag byte, val byte) {
	pb.w.AppendByte(tag)
	pb.w.AppendInt32(1)
	pb.w.AppendByte(val)
}

func (pb *DPBWriterV2) PutInt16(tag byte, val int16) {
	pb.w.AppendByte(tag)
	pb.w.AppendInt32(2)
	pb.w.AppendInt16(val)
}

func (pb *DPBWriterV2) PutInt32(tag byte, val int32) {
	pb.w.AppendByte(tag)
	pb.w.AppendInt32(4)
	pb.w.AppendInt32(val)
}

func (pb *DPBWriterV2) PutInt64(tag byte, val int64) {
	pb.w.AppendByte(tag)
	pb.w.AppendInt32(8)
	pb.w.PutInt64(val)
}

func (pb *DPBWriterV2) PutBuffer(bytes []byte) {
	pb.w.AppendInt32(int32(len(bytes)))
	pb.w.AppendBytes(bytes)
}

func (pb *DPBWriterV2) PutString(tag byte, val string) {
	strBytes := utils.Str_to_bytes(val)
	pb.w.AppendByte(tag)
	pb.w.AppendInt32(int32(len(strBytes)))
	pb.w.AppendBytes(strBytes)
}

func (pb *DPBWriterV2) Bytes() []byte {
	return pb.w.buf
}

type TPBWriterV3 struct {
	w *XPBWriter
}

func NewTPBWriterV3() *TPBWriterV3 {
	return &TPBWriterV3{w: NewXPBWriterFromTag(isc_tpb_version3)}
}

func (pb *TPBWriterV3) PutTag(tag byte) {
	pb.w.AppendByte(tag)
}

func (pb *TPBWriterV3) Bytes() []byte {
	return pb.w.buf
}
