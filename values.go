package fbx

// format codes
const (
	TextFormatCode   = 0
	BinaryFormatCode = 1
)

//func convertSimpleArgument(m *fbtype.Map, arg any) (any, error) {
//	buf, err := m.Encode(0, TextFormatCode, arg, []byte{})
//	if err != nil {
//		return nil, err
//	}
//	if buf == nil {
//		return nil, nil
//	}
//	return string(buf), nil
//}
//
//func encodeCopyValue(m *fbtype.Map, buf []byte, typeid uint32, arg any) ([]byte, error) {
//	sp := len(buf)
//	buf = pgio.AppendInt32(buf, -1)
//	argBuf, err := m.Encode(typeid, BinaryFormatCode, arg, buf)
//	if err != nil {
//		if argBuf2, err2 := tryScanStringCopyValueThenEncode(m, buf, typeid, arg); err2 == nil {
//			argBuf = argBuf2
//		} else {
//			return nil, err
//		}
//	}
//
//	if argBuf != nil {
//		buf = argBuf
//		pgio.SetInt32(buf[sp:], int32(len(buf[sp:])-4))
//	}
//	return buf, nil
//}
//
//func tryScanStringCopyValueThenEncode(m *fbtype.Map, buf []byte, typeid uint32, arg any) ([]byte, error) {
//	s, ok := arg.(string)
//	if !ok {
//		textBuf, err := m.Encode(typeid, TextFormatCode, arg, nil)
//		if err != nil {
//			return nil, errors.New("not a string and cannot be encoded as text")
//		}
//		s = string(textBuf)
//	}
//
//	var v any
//	err := m.Scan(typeid, TextFormatCode, []byte(s), &v)
//	if err != nil {
//		return nil, err
//	}
//
//	return m.Encode(typeid, BinaryFormatCode, v, buf)
//}
