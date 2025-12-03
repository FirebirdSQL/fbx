package fbproto

import (
	"fmt"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
)

type Encoding struct {
	Charset         string
	Encoder         *encoding.Encoder
	Decoder         *encoding.Decoder
	MaxBytesPerChar uint32
	CharacterSetId  int32
}

type DummyTranscoder struct {
}

func (*DummyTranscoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	return 0, 0, nil
}

func (*DummyTranscoder) Reset() {}

func (*DummyTranscoder) Bytes([]byte) ([]byte, error) {
	return nil, nil
}
func (*DummyTranscoder) String(string) (string, error) {
	return "", nil
}

func CharacterSetIdIntoName(id int32) (cs string, err error) {
	switch id {
	case 3:
		cs = "UNICODE_FSS"
	case 4:
		cs = "UTF8"
	default:
		return "", fmt.Errorf("unknown or unsupported charset id: %d", id)
	}
	return
}

func NewEncoding(charset string) (*Encoding, error) {
	e := &Encoding{Charset: charset}
	switch charset {
	case "UNICODE_FSS":
		e.Encoder = unicode.UTF8.NewEncoder()
		e.Decoder = unicode.UTF8.NewDecoder()
		e.MaxBytesPerChar = 3
		e.CharacterSetId = 3
	case "UTF8":
		e.Encoder = unicode.UTF8.NewEncoder()
		e.Decoder = unicode.UTF8.NewDecoder()
		e.MaxBytesPerChar = 4
		e.CharacterSetId = 4
	//case "SJIS_0208":
	//	encoding.Encoder = japanese.ShiftJIS.NewEncoder()
	//	encoding.Decoder = japanese.ShiftJIS.NewDecoder()
	//case "EUCJ_0208":
	//	encoding.Encoder = japanese.EUCJP.NewEncoder()
	//	encoding.Decoder = japanese.EUCJP.NewDecoder()
	//case "ISO8859_1":
	//	encoding.Encoder = charmap.ISO8859_1.NewEncoder()
	//	encoding.Decoder = charmap.ISO8859_1.NewDecoder()
	//case "ISO8859_2":
	//	dec := charmap.ISO8859_2.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "ISO8859_3":
	//	dec := charmap.ISO8859_3.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "ISO8859_4":
	//	dec := charmap.ISO8859_5.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "ISO8859_5":
	//	dec := charmap.ISO8859_5.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "ISO8859_6":
	//	dec := charmap.ISO8859_6.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "ISO8859_7":
	//	dec := charmap.ISO8859_7.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "ISO8859_8":
	//	dec := charmap.ISO8859_8.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "ISO8859_9":
	//	dec := charmap.ISO8859_9.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "ISO8859_13":
	//	dec := charmap.ISO8859_13.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "KSC_5601":
	//	dec := korean.EUCKR.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "WIN1250":
	//	dec := charmap.Windows1250.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "WIN1251":
	//	dec := charmap.Windows1251.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "WIN1252":
	//	dec := charmap.Windows1252.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "WIN1253":
	//	dec := charmap.Windows1252.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "WIN1254":
	//	dec := charmap.Windows1252.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "BIG_5":
	//	dec := traditionalchinese.Big5.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "GB_2312":
	//	dec := simplifiedchinese.HZGB2312.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "WIN1255":
	//	dec := charmap.Windows1255.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "WIN1256":
	//	dec := charmap.Windows1256.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "WIN1257":
	//	dec := charmap.Windows1257.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "KOI8R":
	//	dec := charmap.KOI8R.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "KOI8U":
	//	dec := charmap.KOI8U.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	//case "WIN1258":
	//	dec := charmap.Windows1258.NewDecoder()
	//	v, _ := dec.Bytes(raw_value)
	//	return string(v)
	default:
		return nil, fmt.Errorf("unknown or unsupported charset: %s", charset)
	}
	return e, nil
}
