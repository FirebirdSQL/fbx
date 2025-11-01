package fbxpool

import (
	"github.com/red-soft/fbx"
	"github.com/red-soft/fbx/fbconn"
)

type errBatchResults struct {
	err error
}

func (br errBatchResults) Exec() (fbconn.QueryStatus, error) {
	return fbconn.QueryStatus{}, br.err
}

func (br errBatchResults) Query() (fbx.Rows, error) {
	return errRows{err: br.err}, br.err
}

func (br errBatchResults) QueryRow() fbx.Row {
	return errRow{err: br.err}
}

func (br errBatchResults) Close() error {
	return br.err
}

type poolBatchResults struct {
	br fbx.BatchResults
	c  *Conn
}

func (br *poolBatchResults) Exec() (fbconn.QueryStatus, error) {
	return br.br.Exec()
}

func (br *poolBatchResults) Query() (fbx.Rows, error) {
	return br.br.Query()
}

func (br *poolBatchResults) QueryRow() fbx.Row {
	return br.br.QueryRow()
}

func (br *poolBatchResults) Close() error {
	err := br.br.Close()
	if br.c != nil {
		br.c.Release()
		br.c = nil
	}
	return err
}
