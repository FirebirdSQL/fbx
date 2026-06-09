package fbx

import (
	"bytes"
	"context"
	"fmt"
)

// BatchInsertRows returns a BatchInsertRowsSource interface over the provided rows slice
// making it usable by Conn.BatchInsert.
func BatchInsertRows(rows [][]any) BatchInsertRowsSource {
	return &batchInsertRows{rows: rows, idx: -1}
}

type batchInsertRows struct {
	rows [][]any
	idx  int
}

func (b *batchInsertRows) Next() bool {
	b.idx++
	return b.idx < len(b.rows)
}

func (b *batchInsertRows) Values() ([]any, error) {
	return b.rows[b.idx], nil
}

func (b *batchInsertRows) Err() error {
	return nil
}

// BatchInsertSlice returns a BatchInsertRowsSource interface over a dynamic func
// making it usable by Conn.BatchInsert.
func BatchInsertSlice(length int, next func(int) ([]any, error)) BatchInsertRowsSource {
	return &batchInsertSliceFunc{next: next, idx: -1, len: length}
}

type batchInsertSliceFunc struct {
	next func(int) ([]any, error)
	idx  int
	len  int
	err  error
}

func (b *batchInsertSliceFunc) Next() bool {
	b.idx++
	return b.idx < b.len
}

func (b *batchInsertSliceFunc) Values() ([]any, error) {
	values, err := b.next(b.idx)
	if err != nil {
		b.err = err
	}
	return values, err
}

func (b *batchInsertSliceFunc) Err() error {
	return b.err
}

// BatchInsertFunc returns a BatchInsertRowsSource interface that relies on nxtf for values.
// nxtf returns rows until it either signals an 'end of data' by returning row=nil and err=nil,
// or it returns an error. If nxtf returns an error, the batch insert is aborted.
func BatchInsertFunc(nxtf func() (row []any, err error)) BatchInsertRowsSource {
	return &batchInsertFunc{next: nxtf}
}

type batchInsertFunc struct {
	next     func() ([]any, error)
	valueRow []any
	err      error
}

func (b *batchInsertFunc) Next() bool {
	b.valueRow, b.err = b.next()
	// only return true if valueRow exists and no error
	return b.valueRow != nil && b.err == nil
}

func (b *batchInsertFunc) Values() ([]any, error) {
	return b.valueRow, b.err
}

func (b *batchInsertFunc) Err() error {
	return b.err
}

// BatchInsertRowsSource is the interface used by Conn.BatchInsert as the source for insert data.
type BatchInsertRowsSource interface {
	// Next returns true if there is another row and makes the next row data
	// available to Values(). When there are no more rows available or an error
	// has occurred it returns false.
	Next() bool

	// Values returns the values for the current row.
	Values() ([]any, error)

	// Err returns any error that has been encountered by the BatchInsertRowsSource. If
	// this is not nil Conn.BatchInsert will abort the batch insert.
	Err() error
}

type batchInsert struct {
	conn        *Conn
	tableName   Identifier
	columnNames []string
	rowSrc      BatchInsertRowsSource
}

func (b *batchInsert) run(ctx context.Context) (int64, error) {
	quotedTableName := b.tableName.Sanitize()
	cbuf := &bytes.Buffer{}
	pbuf := &bytes.Buffer{}
	for i, cn := range b.columnNames {
		if i != 0 {
			cbuf.WriteString(", ")
			pbuf.WriteString(", ")
		}
		cbuf.WriteString(quoteIdentifier(cn))
		pbuf.WriteString("?")
	}
	quotedColumnNames := cbuf.String()
	placeholders := pbuf.String()

	batch, err := b.conn.PrepareBatchWithOptions(ctx, fmt.Sprintf("insert into %s (%s) values (%s)", quotedTableName, quotedColumnNames, placeholders), BatchOptions{ContinueOnError: false})
	if err != nil {
		return 0, err
	}
	defer batch.Close(ctx)

	for b.rowSrc.Next() {
		values, err := b.rowSrc.Values()
		if err != nil {
			return 0, err
		}
		err = batch.Add(values...)
		if err != nil {
			return 0, err
		}

		if batch.PendingBytes() > b.conn.config.BatchInsertPendingSize {
			if batch.Flush(ctx) != nil {
				return 0, err
			}
		}
	}

	if b.rowSrc.Err() != nil {
		return 0, b.rowSrc.Err()
	}

	br, err := batch.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return br.Affected, nil
}

// BatchInsert uses the Firebird's batch operations to perform bulk data insertion. It returns the number of rows inserted and
// an error.
//
// BatchInsert uses delimited (quoted) table and column identifiers thus identifiers are case-sensitive. Use uppercased
// identifiers if your objects is created with regular (non-quoted) identifiers
//
// See BatchInsertRows, BatchInsertFunc, BatchInsertSlice
func (c *Conn) BatchInsert(ctx context.Context, tableName Identifier, columnNames []string, rowSrc BatchInsertRowsSource) (int64, error) {
	bi := &batchInsert{
		conn:        c,
		tableName:   tableName,
		columnNames: columnNames,
		rowSrc:      rowSrc,
	}

	return bi.run(ctx)
}
