/*******************************************************************************
The MIT License (MIT)

Copyright (c) 2013-2019 Hajime Nakagami

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*******************************************************************************/

package utils

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"math/big"
	"strconv"
)

func Str_to_bytes(s string) []byte {
	return []byte(s)
}

func Int32_to_bytes(i32 int32) []byte {
	bs := []byte{
		byte(i32 & 0xFF),
		byte(i32 >> 8 & 0xFF),
		byte(i32 >> 16 & 0xFF),
		byte(i32 >> 24 & 0xFF),
	}
	return bs
}

func Int64_to_bytes(i64 int64) []byte {
	bs := []byte{
		byte(i64 & 0xFF),
		byte(i64 >> 8 & 0xFF),
		byte(i64 >> 16 & 0xFF),
		byte(i64 >> 24 & 0xFF),
		byte(i64 >> 32 & 0xFF),
		byte(i64 >> 40 & 0xFF),
		byte(i64 >> 40 & 0xFF),
		byte(i64 >> 56 & 0xFF),
	}
	return bs
}

func Bint64_to_bytes(i64 int64) []byte {
	bs := []byte{
		byte(i64 >> 56 & 0xFF),
		byte(i64 >> 48 & 0xFF),
		byte(i64 >> 40 & 0xFF),
		byte(i64 >> 32 & 0xFF),
		byte(i64 >> 24 & 0xFF),
		byte(i64 >> 16 & 0xFF),
		byte(i64 >> 8 & 0xFF),
		byte(i64 & 0xFF),
	}
	return bs
}

func Bint32_to_bytes(i32 int32) []byte {
	bs := []byte{
		byte(i32 >> 24 & 0xFF),
		byte(i32 >> 16 & 0xFF),
		byte(i32 >> 8 & 0xFF),
		byte(i32 & 0xFF),
	}
	return bs
}

func Int16_to_bytes(i16 int16) []byte {
	bs := []byte{
		byte(i16 & 0xFF),
		byte(i16 >> 8 & 0xFF),
	}
	return bs
}
func Bytes_to_str(b []byte) string {
	return string(b)
}

func Bytes_to_bint32(b []byte) int32 {
	return int32(binary.BigEndian.Uint32(b))
}

func Bytes_to_int32(b []byte) int32 {
	return int32(binary.LittleEndian.Uint32(b))
}

func bytes_to_bint16(b []byte) int16 {
	return int16(binary.BigEndian.Uint16(b))
}

func Bytes_to_buint16(b []byte) uint16 {
	return uint16(binary.BigEndian.Uint16(b))
}

func Bytes_to_int16(b []byte) int16 {
	return int16(binary.LittleEndian.Uint16(b))
}

func Bytes_to_bint64(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

func Bytes_to_int64(b []byte) int64 {
	return int64(binary.LittleEndian.Uint64(b))
}

func BigIntFromHexString(s string) *big.Int {
	ret := new(big.Int)
	ret.SetString(s, 16)
	return ret
}

func BigIntFromString(s string) *big.Int {
	ret := new(big.Int)
	ret.SetString(s, 10)
	return ret
}

func FlattenBytes(l *list.List) []byte {
	n := 0
	for e := l.Front(); e != nil; e = e.Next() {
		n += len((e.Value).([]byte))
	}

	bs := make([]byte, n)

	n = 0
	for e := l.Front(); e != nil; e = e.Next() {
		for i, b := range (e.Value).([]byte) {
			bs[n+i] = b
		}
		n += len((e.Value).([]byte))
	}

	return bs
}

func XdrBytes(bs []byte) []byte {
	// XDRReader encoding bytes
	n := len(bs)
	padding := 0
	if n%4 != 0 {
		padding = 4 - n%4
	}
	buf := make([]byte, 4+n+padding)
	buf[0] = byte(n >> 24 & 0xFF)
	buf[1] = byte(n >> 16 & 0xFF)
	buf[2] = byte(n >> 8 & 0xFF)
	buf[3] = byte(n & 0xFF)
	for i, b := range bs {
		buf[4+i] = b
	}
	return buf
}

func xdrString(s string) []byte {
	// XDRReader encoding string
	bs := bytes.NewBufferString(s).Bytes()
	return XdrBytes(bs)
}

func convertToBool(s string, defaultValue bool) bool {
	v, err := strconv.ParseBool(s)
	if err != nil {
		v = defaultValue
	}
	return v
}

func fitsUint32(val int64) bool {
	return val >= 0 && val <= 0xffffffff
}
