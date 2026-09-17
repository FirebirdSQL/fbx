package fbservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"rdb.red-soft.ru/fbx/fbconn"
)

// StatisticsOptions controls the gstat report. Output is required. Leaving all
// other fields at their zero values selects the server's default data and index analysis.
type StatisticsOptions struct {
	// Output receives the textual report and must not be nil. Use io.Discard to
	// explicitly discard the report. Writes are synchronous and must not block
	// indefinitely. Shared writers must be concurrency safe.
	// Statistics does not close the writer. On error, it may contain a partial report.
	Output io.Writer
	// DataPages analyzes data pages (-data).
	DataPages bool
	// HeaderPages reports only the database header (-header), excluding other
	// analysis options. NoCreation and ParallelWorkers may still be specified.
	HeaderPages bool
	// IndexPages analyzes index leaf pages (-index).
	IndexPages bool
	// SystemRelations includes system tables (-system).
	SystemRelations bool
	// RecordVersions includes record and version lengths (-record).
	RecordVersions bool
	// Tables restricts analysis to case-sensitive table names (-table). Use raw
	// names without SQL quoting. Empty selects all tables. Requires Firebird 3+
	// or RedDatabase 3+.
	Tables []string
	// Schemas restricts analysis to case-sensitive schema names (-schema).
	// Empty selects all schemas. Requires Firebird 6 or RedDatabase 6.
	Schemas []string
	// NoCreation suppresses the database creation timestamp (-nocreation).
	NoCreation bool
	// Encryption reports encrypted and unencrypted page counts (-encryption),
	// excluding other analysis options. NoCreation and ParallelWorkers may
	// still be specified. Requires Firebird 3+ or RedDatabase 3+.
	Encryption bool
	// ParallelWorkers sets the server worker count (-parallel). Zero uses the
	// server default. Requires a server supporting parallel gstat (RedDatabase 5+).
	ParallelWorkers int
}

// Statistics writes a gstat report for database, a server-side path or alias.
// It validates the request before connecting and closes its service connection
// before returning. Exactly one options value with a non-nil Output is required.
// Use io.Discard to explicitly discard the report. Unsupported options
// return a server error and are never silently ignored. Context cancellation
// cannot interrupt an Output.Write call already in progress. The caller must
// not modify the options' slices until Statistics returns.
// For a valid request, an already canceled ctx returns its error without connecting.
// The database must not start with '-'; use a path such as ./-name.fdb for a
// relative filename beginning with a hyphen.
func (m *Manager) Statistics(ctx context.Context, database string, options ...StatisticsOptions) (err error) {
	if len(options) > 1 {
		return errors.New("statistics accepts at most one StatisticsOptions value")
	}
	var option StatisticsOptions
	if len(options) == 1 {
		option = options[0]
	}
	if err := validateStatisticsRequest(database, option); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := m.open(ctx, database)
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer func() { err = errors.Join(err, closeService(conn)) }()
	return conn.Statistics(ctx, database, fbconn.StatisticsOptions{
		Output:          option.Output,
		DataPages:       option.DataPages,
		HeaderPages:     option.HeaderPages,
		IndexPages:      option.IndexPages,
		SystemRelations: option.SystemRelations,
		RecordVersions:  option.RecordVersions,
		Tables:          append([]string(nil), option.Tables...),
		Schemas:         append([]string(nil), option.Schemas...),
		NoCreation:      option.NoCreation,
		Encryption:      option.Encryption,
		ParallelWorkers: option.ParallelWorkers,
	})
}

func validateStatisticsRequest(database string, options StatisticsOptions) error {
	if options.Output == nil {
		return errors.New("statistics output must not be nil; use io.Discard to discard the report")
	}
	if database == "" {
		return errors.New("statistics database must not be empty")
	}
	if strings.HasPrefix(database, "-") {
		return errors.New("statistics database must not start with '-'")
	}
	if err := validateStatisticsString("database", database); err != nil {
		return err
	}
	for _, group := range []struct {
		name   string
		values []string
	}{
		{"table", options.Tables}, {"schema", options.Schemas},
	} {
		for _, value := range group.values {
			if value == "" || strings.HasPrefix(value, "-") {
				return fmt.Errorf("statistics %s name must not be empty or start with '-'", group.name)
			}
			if err := validateStatisticsString(group.name, value); err != nil {
				return err
			}
		}
	}
	if options.ParallelWorkers < 0 || options.ParallelWorkers > maxServiceActionInt32 {
		return fmt.Errorf("statistics parallel workers must be between 0 and %d", maxServiceActionInt32)
	}
	analysis := options.DataPages || options.IndexPages || options.SystemRelations ||
		options.RecordVersions || len(options.Tables) > 0 || len(options.Schemas) > 0
	if options.HeaderPages && (analysis || options.Encryption) {
		return errors.New("statistics header pages cannot be combined with other analysis options")
	}
	if options.Encryption && analysis {
		return errors.New("statistics encryption cannot be combined with other analysis options")
	}
	return nil
}

func validateStatisticsString(name, value string) error {
	if strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("statistics %s must not contain NUL", name)
	}
	if len(value) > maxServiceActionStringLength {
		return fmt.Errorf("statistics %s is %d bytes; maximum is %d", name, len(value), maxServiceActionStringLength)
	}
	return nil
}
