package fbx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"

	"rdb.red-soft.ru/fbx/fbconn"
	"rdb.red-soft.ru/fbx/fbtype"
)

// ExtendedQueryBuilder chooses parameter formats, encodes parameters, and tracks result formats for statement execution.
type ExtendedQueryBuilder struct {
	ParamValues     [][]byte
	paramValueBytes []byte
	ParamFormats    []int16
	ResultFormats   []int16
}

// Build sets ParamValues, ParamFormats, and ResultFormats for use with *FbConn.ExecParams or *FbConn.ExecPrepared. If
// sd is nil then QueryExecModeExec behavior will be used.
func (pb *ExtendedQueryBuilder) Build(ctx context.Context, conn *Conn, tx *dbTx, m *fbtype.Map, sd *fbconn.StatementDescription, args []any) error {
	pb.reset()

	if sd == nil {
		for i := range args {
			typeMetadata := fbtype.NewTypeMetadata(0, 0, 0, 0, fbtype.TextFormatCode)
			err := pb.appendParam(ctx, conn, tx, m, typeMetadata, args[i])
			if err != nil {
				err = fmt.Errorf("failed to encode args[%d]: %w", i, err)
				return err
			}
		}
		return nil
	}

	if len(sd.Parameters) != len(args) {
		return fmt.Errorf("mismatched param and argument count")
	}

	for i := range args {
		typeMetadata := fbtype.NewTypeMetadata(sd.Parameters[i].Type, sd.Parameters[i].Subtype, sd.Parameters[i].Length, sd.Parameters[i].Scale, -1)
		err := pb.appendParam(ctx, conn, tx, m, typeMetadata, args[i])
		if err != nil {
			err = fmt.Errorf("failed to encode args[%d]: %w", i, err)
			return err
		}
	}

	return nil
}

// appendParam appends a parameter to the query. format may be -1 to automatically choose the format. If arg is nil it
// must be an untyped nil.
func (pb *ExtendedQueryBuilder) appendParam(ctx context.Context, conn *Conn, tx *dbTx, m *fbtype.Map, typeMetadata fbtype.TypeMetadata, arg any) error {
	typeMetadata.FormatCode = 1
	if typeMetadata.FormatCode == -1 {
		preferredFormat := pb.chooseParameterFormatCode(m, typeMetadata.TypeID, arg)
		preferredErr := pb.appendParam(ctx, conn, tx, m, typeMetadata, arg)
		if preferredErr == nil {
			return nil
		}

		if preferredFormat == TextFormatCode {
			typeMetadata.FormatCode = BinaryFormatCode
		} else {
			typeMetadata.FormatCode = TextFormatCode
		}

		otherErr := pb.appendParam(ctx, conn, tx, m, typeMetadata, arg)
		if otherErr == nil {
			return nil
		}

		return preferredErr // return the error from the preferred format
	}

	v, err := pb.encodeParamValue(ctx, conn, tx, m, typeMetadata, arg)
	if err != nil {
		return err
	}

	pb.ParamFormats = append(pb.ParamFormats, typeMetadata.FormatCode)
	pb.ParamValues = append(pb.ParamValues, v)

	return nil
}

// appendResultFormat appends a result format to the query.
func (pb *ExtendedQueryBuilder) appendResultFormat(format int16) {
	pb.ResultFormats = append(pb.ResultFormats, format)
}

// reset readies pb to build another query.
func (pb *ExtendedQueryBuilder) reset() {
	pb.ParamValues = pb.ParamValues[0:0]
	pb.paramValueBytes = pb.paramValueBytes[0:0]
	pb.ParamFormats = pb.ParamFormats[0:0]
	pb.ResultFormats = pb.ResultFormats[0:0]

	if cap(pb.ParamValues) > 64 {
		pb.ParamValues = make([][]byte, 0, 64)
	}

	if cap(pb.paramValueBytes) > 256 {
		pb.paramValueBytes = make([]byte, 0, 256)
	}

	if cap(pb.ParamFormats) > 64 {
		pb.ParamFormats = make([]int16, 0, 64)
	}
	if cap(pb.ResultFormats) > 64 {
		pb.ResultFormats = make([]int16, 0, 64)
	}
}

func (pb *ExtendedQueryBuilder) encodeParamValue(ctx context.Context, conn *Conn, tx *dbTx, m *fbtype.Map, typeMetadata fbtype.TypeMetadata, arg any) ([]byte, error) {
	if pb.paramValueBytes == nil {
		pb.paramValueBytes = make([]byte, 0, 128)
	}

	pos := len(pb.paramValueBytes)

	if fbconn.IsBlobType(typeMetadata.TypeID) || isBlobArgument(arg) {
		buf, err := pb.encodeBlobParam(ctx, conn, tx, typeMetadata, arg)
		if err != nil {
			return nil, err
		}
		if buf == nil {
			return nil, nil
		}
		pb.paramValueBytes = append(pb.paramValueBytes, buf...)
		return pb.paramValueBytes[pos:], nil
	}

	buf, err := m.Encode(typeMetadata, arg, pb.paramValueBytes)
	if err != nil {
		return nil, err
	}
	if buf == nil {
		return nil, nil
	}
	pb.paramValueBytes = buf
	return pb.paramValueBytes[pos:], nil
}

func (pb *ExtendedQueryBuilder) encodeBlobParam(ctx context.Context, conn *Conn, tx *dbTx, typeMetadata fbtype.TypeMetadata, arg any) ([]byte, error) {

	if isNilArg(arg) {
		return nil, nil
	}
	if typeMetadata.TypeID != 0 && !fbconn.IsBlobType(typeMetadata.TypeID) {
		return nil, fmt.Errorf("blob argument is not compatible with typeid %d", typeMetadata.TypeID)
	}
	switch v := arg.(type) {
	case []byte:
		if !fbconn.IsBlobType(typeMetadata.TypeID) {
			return nil, fmt.Errorf("blob parameter expected, got raw bytes")
		}
		return pb.createBlobFromBytes(ctx, conn, tx, typeMetadata.SubType, typeMetadata.Scale, v)
	case string:
		if !fbconn.IsBlobType(typeMetadata.TypeID) {
			return nil, fmt.Errorf("blob parameter expected, got string")
		}
		return pb.createBlobFromString(ctx, conn, tx, typeMetadata.SubType, typeMetadata.Scale, v)
	case io.Reader:
		if typeMetadata.TypeID != 0 && !fbconn.IsBlobType(typeMetadata.TypeID) {
			return nil, fmt.Errorf("blob parameter expected, got reader")
		}
		id, err := pb.createBlobFromReader(ctx, conn, tx, typeMetadata.SubType, typeMetadata.Scale, v)
		if err != nil {
			return nil, err
		}
		return fbconn.EncodeBlobID(int64(id)), nil
	default:
		if fbconn.IsBlobType(typeMetadata.TypeID) {
			return nil, fmt.Errorf("unsupported blob argument type %T", arg)
		}
		return nil, fmt.Errorf("blob argument type %T is not compatible with typeid %d", arg, typeMetadata.TypeID)
	}
}

func (pb *ExtendedQueryBuilder) createBlobFromBytes(ctx context.Context, conn *Conn, tx *dbTx, subType int32, scale int32, data []byte) ([]byte, error) {
	blobID, err := pb.createBlobFromReader(ctx, conn, tx, subType, scale, &bytesReader{data: data})
	if err != nil {
		return nil, err
	}
	return fbconn.EncodeBlobID(int64(blobID)), nil
}

func (pb *ExtendedQueryBuilder) createBlobFromString(ctx context.Context, conn *Conn, tx *dbTx, subType int32, scale int32, s string) ([]byte, error) {
	if conn == nil {
		return nil, errors.New("blob parameter requires connection")
	}
	cfg, err := blobConfigFromParam(conn, subType, scale)
	if err != nil {
		return nil, err
	}
	fbTx, err := resolveExecTx(ctx, conn, tx)
	if err != nil {
		return nil, err
	}
	blob, err := newOutputBlob(ctx, conn, fbTx, cfg)
	if err != nil {
		return nil, err
	}
	writer, err := blob.Writer(ctx)
	if err != nil {
		_ = blob.Cancel(ctx)
		return nil, err
	}
	if _, err := writer.WriteString(s); err != nil {
		_ = writer.Cancel()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return fbconn.EncodeBlobID(int64(blob.ID())), nil
}

func (pb *ExtendedQueryBuilder) createBlobFromReader(ctx context.Context, conn *Conn, tx *dbTx, subType int32, scale int32, r io.Reader) (blobID, error) {
	if conn == nil {
		return 0, errors.New("blob parameter requires connection")
	}
	cfg, err := blobConfigFromParam(conn, subType, scale)
	if err != nil {
		return 0, err
	}
	fbTx, err := resolveExecTx(ctx, conn, tx)
	if err != nil {
		return 0, err
	}
	blob, err := newOutputBlob(ctx, conn, fbTx, cfg)
	if err != nil {
		return 0, err
	}
	writer, err := blob.Writer(ctx)
	if err != nil {
		_ = blob.Cancel(ctx)
		return 0, err
	}
	if _, err := io.Copy(writer, r); err != nil {
		_ = writer.Cancel()
		return 0, err
	}
	if err := writer.Close(); err != nil {
		return 0, err
	}
	return blob.ID(), nil
}

func resolveExecTx(ctx context.Context, conn *Conn, tx *dbTx) (*fbconn.FBTransaction, error) {
	if conn == nil || conn.fbConn == nil {
		return nil, errors.New("blob parameter requires connection")
	}
	if tx != nil {
		return tx.tx, nil
	}
	return nil, nil
}

func isBlobArgument(arg any) bool {
	switch arg.(type) {
	case io.Reader:
		return true
	default:
		return false
	}
}

func isNilArg(arg any) bool {
	if arg == nil {
		return true
	}
	v := reflect.ValueOf(arg)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// chooseParameterFormatCode determines the correct format code for an
// argument to a prepared statement. It defaults to TextFormatCode if no
// determination can be made.
func (pb *ExtendedQueryBuilder) chooseParameterFormatCode(m *fbtype.Map, typeid uint32, arg any) int16 {
	switch arg.(type) {
	case string, *string:
		return TextFormatCode
	}

	return m.FormatCodeForTypeID(typeid)
}
