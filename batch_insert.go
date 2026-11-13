package fbx

import (
	"bytes"
	"context"
	"fmt"

	"github.com/firebirdsql/fbx/internal/operationerror"
)

const defaultBatchInsertPendingSize = 128 * 1024

// BatchInsertRows adapts rows to the [BatchInsertRowsSource] interface used by
// [Conn.BatchInsert].
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

// BatchInsertSlice adapts an indexed callback to the [BatchInsertRowsSource]
// interface used by [Conn.BatchInsert].
//
// The callback is called with indexes from 0 to length-1.
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

// BatchInsertFunc adapts next to the [BatchInsertRowsSource] interface used by
// [Conn.BatchInsert].
//
// next returns rows until it either signals the end of data by returning
// row == nil and err == nil, or it returns an error. If next returns an error,
// the batch insert is aborted.
func BatchInsertFunc(next func() (row []any, err error)) BatchInsertRowsSource {
	return &batchInsertFunc{next: next}
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

// BatchInsertRowsSource provides rows to [Conn.BatchInsert].
type BatchInsertRowsSource interface {
	// Next returns true if there is another row and makes the next row data
	// available to Values(). When there are no more rows available or an error
	// has occurred it returns false.
	Next() bool

	// Values returns the values for the current row.
	Values() ([]any, error)

	// Err returns any error that has been encountered by the [BatchInsertRowsSource].
	// If this is not nil, [Conn.BatchInsert] will abort the batch insert.
	Err() error
}

type batchInsert struct {
	conn        *Conn
	tableName   Identifier
	columnNames []string
	rowSrc      BatchInsertRowsSource
}

func (b *batchInsert) run(ctx context.Context) (affected int64, err error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if b.conn.config.BatchInsertPendingSize < 0 {
		return 0, fmt.Errorf("batch insert pending size must not be negative")
	}

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

	batch, err := b.conn.PrepareBatch(ctx, fmt.Sprintf("insert into %s (%s) values (%s)", quotedTableName, quotedColumnNames, placeholders))
	if err != nil {
		return 0, err
	}
	sourceRead := false
	defer func() {
		cleanupCtx, cancel := cleanupContext(ctx)
		defer cancel()
		cleanupErr := batch.Close(cleanupCtx)
		if cleanupErr != nil && !b.conn.IsClosed() {
			b.conn.die()
		}
		err = operationerror.Join(err, cleanupErr)
		if sourceRead {
			// Even a failed Next may advance the source. Repeating the call
			// with the same source cannot replay the original rows reliably.
			err = operationerror.WithRetry(err, false)
		}
	}()

	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		sourceRead = true
		if !b.rowSrc.Next() {
			break
		}
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		values, err := b.rowSrc.Values()
		if err := operationerror.Join(err, ctx.Err()); err != nil {
			return 0, err
		}
		err = batch.Add(values...)
		if err != nil {
			return 0, err
		}

		if batch.PendingBytes() > b.conn.config.BatchInsertPendingSize {
			if err := batch.Flush(ctx); err != nil {
				return 0, err
			}
		}
	}

	if err := operationerror.Join(b.rowSrc.Err(), ctx.Err()); err != nil {
		return 0, err
	}

	br, err := batch.Exec(ctx)
	return br.Affected, err
}

// BatchInsert uses Firebird native batch operations to perform bulk insertion.
//
// BatchInsert sends rows as the source is read and executes them once, after
// the source is exhausted. All encoded rows must fit in the server batch buffer;
// flushing only limits client-side buffering and does not execute rows.
//
// The returned count comes from the server completion, or is zero if no
// completion was received. A count of -1 means the count is unknown. On error,
// even a positive count does not mean the rows were committed: in autocommit
// mode, a row error rolls back the entire execution.
//
// Errors after reading starts are not safe to retry with the same source.
// Context cancellation is checked between source calls; blocking source methods
// must handle context cancellation themselves.
//
// Table and column identifiers are quoted and case-sensitive. Use upper-case
// identifiers for objects created with regular, unquoted SQL identifiers.
//
// See [BatchInsertRows], [BatchInsertFunc], and [BatchInsertSlice].
func (c *Conn) BatchInsert(ctx context.Context, tableName Identifier, columnNames []string, rowSrc BatchInsertRowsSource) (int64, error) {
	bi := &batchInsert{
		conn:        c,
		tableName:   tableName,
		columnNames: columnNames,
		rowSrc:      rowSrc,
	}

	return bi.run(ctx)
}
