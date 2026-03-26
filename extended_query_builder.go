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

// ExtendedQueryBuilder encodes parameters.
type ExtendedQueryBuilder struct {
	ParamValues     [][]byte
	paramValueBytes []byte
}

// Build sets ParamValues for use with *FbConn.ExecParams or *FbConn.ExecPrepared. If
// sd is nil then QueryExecModeExec behavior will be used.
func (pb *ExtendedQueryBuilder) Build(ctx context.Context, conn *Conn, tx *dbTx, m *fbtype.Map, sd *fbconn.StatementDescription, args []any) error {
	pb.reset()

	if sd == nil {
		for i := range args {
			typeMetadata := fbtype.NewTypeMetadata(0, 0, 0, 0)
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
		arr := sd.Parameters[i].ArrayData
		typeMetadata := fbtype.NewTypeMetadata(
			sd.Parameters[i].Type,
			sd.Parameters[i].Subtype,
			sd.Parameters[i].Length,
			sd.Parameters[i].Scale,
		)
		if arr != nil {
			arrayData, err := fbtype.NewArrayData(arr.TypeId, arr.BlrTypeId, arr.SubType, arr.Length, arr.Scale,
				arr.Dimensions, arr.TableName, arr.FieldName)
			if err != nil {
				return err
			}
			typeMetadata.SetArrayMetadata(arrayData)
		}
		err := pb.appendParam(ctx, conn, tx, m, typeMetadata, args[i])
		if err != nil {
			err = fmt.Errorf("failed to encode args[%d]: %w", i, err)
			return err
		}
	}

	return nil
}

// appendParam appends a parameter to the query.
func (pb *ExtendedQueryBuilder) appendParam(ctx context.Context, conn *Conn, tx *dbTx, m *fbtype.Map, typeMetadata fbtype.TypeMetadata, arg any) error {
	v, err := pb.encodeParamValue(ctx, conn, tx, m, typeMetadata, arg)
	if err != nil {
		return err
	}
	pb.ParamValues = append(pb.ParamValues, v)

	return nil
}

// reset readies pb to build another query.
func (pb *ExtendedQueryBuilder) reset() {
	pb.ParamValues = pb.ParamValues[0:0]
	pb.paramValueBytes = pb.paramValueBytes[0:0]

	if cap(pb.ParamValues) > 64 {
		pb.ParamValues = make([][]byte, 0, 64)
	}

	if cap(pb.paramValueBytes) > 256 {
		pb.paramValueBytes = make([]byte, 0, 256)
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

	if fbconn.IsArrayType(typeMetadata.TypeID) && buf != nil {
		var lowers []int
		for _, d := range typeMetadata.ArrayData.Dimensions {
			lowers = append(lowers, d.Lower)
		}
		dimensions, err := fbtype.Dimensions(arg, lowers)
		if err != nil {
			return nil, err
		}
		if dimensions != nil {
			arrayData := fbconn.NewArray(
				typeMetadata.ArrayData.Table,
				typeMetadata.ArrayData.Field,
				dimensions,
				typeMetadata.ArrayData.Length,
				typeMetadata.ArrayData.Scale,
				typeMetadata.ArrayData.TypeId,
				typeMetadata.ArrayData.BlrTypeId,
				typeMetadata.ArrayData.SubType,
			)
			arrayId, err := fbconn.PutArray(conn.fbConn, ctx, arrayData, buf[pos:])

			if err != nil {
				return nil, err
			}
			buf = append(pb.paramValueBytes, arrayId...)
		}
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
