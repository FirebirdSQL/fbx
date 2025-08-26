package fbproto

import (
	"bytes"
	"encoding/binary"
	"github.com/red-soft/fbx/fbproto/utils"
	"time"
)

func _int64ToBlr(i64 int64) ([]byte, []byte) {
	v := utils.Bint64_to_bytes(i64)
	blr := []byte{16, 0}

	return blr, v
}

func _int32ToBlr(i32 int32) ([]byte, []byte) {
	v := utils.Bint32_to_bytes(i32)
	blr := []byte{8, 0}

	return blr, v
}

func _float64ToBlr(v float64) ([]byte, []byte) {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, v)
	blr := []byte{27}
	return blr, buf.Bytes()
}

func _bytesToBlr(v []byte) ([]byte, []byte) {
	nbytes := len(v)
	pad_length := ((4 - nbytes) & 3)
	padding := make([]byte, pad_length)
	v = bytes.Join([][]byte{
		v,
		padding,
	}, nil)
	blr := []byte{14, byte(nbytes & 255), byte(nbytes >> 8)}
	return blr, v
}

func _convert_date(t time.Time) []byte {
	i := int(t.Month()) + 9
	jy := t.Year() + (i / 12) - 1
	jm := i % 12
	c := jy / 100
	jy -= 100 * c
	j := (146097*c)/4 + (1461*jy)/4 + (153*jm+2)/5 + t.Day() - 678882
	return utils.Bint32_to_bytes(int32(j))
}

func _convert_time(t time.Time) []byte {
	v := (t.Hour()*3600+t.Minute()*60+t.Second())*10000 + t.Nanosecond()/100000
	return utils.Bint32_to_bytes(int32(v))
}

func _dateToBlr(t time.Time) ([]byte, []byte) {
	v := _convert_date(t)
	blr := []byte{12}
	return blr, v
}

func _timeToBlr(t time.Time) ([]byte, []byte) {
	v := _convert_time(t)
	blr := []byte{13}
	return blr, v
}

func _timestampToBlr(t time.Time) ([]byte, []byte) {
	v := bytes.Join([][]byte{
		_convert_date(t),
		_convert_time(t),
	}, nil)

	blr := []byte{35}
	return blr, v
}
