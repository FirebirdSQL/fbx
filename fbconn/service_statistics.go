package fbconn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/firebirdsql/fbx/fbproto"
)

// StatisticsOptions controls database statistics collected by [ServiceConn.Statistics].
// The zero value selects the server defaults (data and index pages) and
// discards service output. Option combinations are passed to the server for
// validation; this layer only checks SPB limits and argument safety.
type StatisticsOptions struct {
	// Output receives the textual gstat report. A nil value discards it. Writes
	// are synchronous and must not block indefinitely.
	Output io.Writer
	// DataPages analyzes data pages (-data).
	DataPages bool
	// HeaderPages reports only the database header (-header). It cannot be
	// combined with data, index, system, record, table, schema or encryption options.
	HeaderPages bool
	// IndexPages analyzes index leaf pages (-index).
	IndexPages bool
	// SystemRelations includes system tables (-system).
	SystemRelations bool
	// RecordVersions includes record and version lengths (-record). Combine it
	// with DataPages to exclude index statistics.
	RecordVersions bool
	// Tables restricts analysis to these case-sensitive table names (-table).
	// Names are passed as separate SPB strings, without SQL quoting. An empty
	// slice selects all tables. Requires Firebird 3 or later, or RedDatabase 3 or later.
	Tables []string
	// Schemas restricts analysis to these case-sensitive schema names (-schema).
	// An empty slice selects all schemas. Requires a server with schema support
	// (Firebird 6 or RedDatabase 6).
	Schemas []string
	// NoCreation suppresses the database creation timestamp (-nocreation).
	NoCreation bool
	// Encryption reports encrypted and unencrypted page counts (-encryption).
	// It cannot be combined with data, header, index, system, record, table or
	// schema options. Requires Firebird 3 or later, or RedDatabase 3 or later.
	Encryption bool
	// ParallelWorkers sets the server worker count (-parallel). Zero omits the
	// parameter. Requires a server supporting parallel gstat (RedDatabase 5+).
	ParallelWorkers int
}

// Statistics collects a textual gstat report for a server-side database path
// or alias. It waits for completion, context cancellation of a network operation,
// or an error. Cancellation cannot interrupt an Output.Write already in progress.
// Unsupported options produce a server error; they are never silently dropped.
// The database must not start with '-'; use a path such as ./-name.fdb for a
// relative filename beginning with a hyphen.
// The caller must not modify the options' slices until Statistics returns.
func (c *ServiceConn) Statistics(ctx context.Context, database string, options StatisticsOptions) error {
	actionSPB, err := buildStatisticsActionSPB(database, options)
	if err != nil {
		return err
	}
	if err := c.runServiceAction(ctx, actionSPB, options.Output); err != nil {
		return fmt.Errorf("database statistics: %w", err)
	}
	return nil
}

func buildStatisticsActionSPB(database string, options StatisticsOptions) ([]byte, error) {
	if database == "" {
		return nil, errors.New("statistics database must not be empty")
	}
	if strings.IndexByte(database, 0) >= 0 {
		return nil, errors.New("statistics database must not contain NUL")
	}
	if strings.HasPrefix(database, "-") {
		return nil, errors.New("statistics database must not start with '-'")
	}
	if options.ParallelWorkers < 0 || options.ParallelWorkers > maxServiceActionInt32 {
		return nil, fmt.Errorf("statistics parallel workers must be between 0 and %d", maxServiceActionInt32)
	}

	spb := fbproto.NewXPBWriterFromTag(fbproto.Isc_action_svc_db_stats)
	if err := putServiceActionString(spb, fbproto.Isc_spb_dbname, database); err != nil {
		return nil, fmt.Errorf("statistics database: %w", err)
	}
	for _, group := range []struct {
		name   string
		tag    byte
		values []string
	}{
		{"schema", fbproto.Isc_spb_sts_schema, options.Schemas},
		{"table", fbproto.Isc_spb_sts_table, options.Tables},
	} {
		for _, value := range group.values {
			if value == "" || strings.IndexByte(value, 0) >= 0 || strings.HasPrefix(value, "-") {
				return nil, fmt.Errorf("statistics %s name must not be empty, contain NUL or start with '-'", group.name)
			}
			if err := putServiceActionString(spb, group.tag, value); err != nil {
				return nil, fmt.Errorf("statistics %s: %w", group.name, err)
			}
		}
	}
	if options.ParallelWorkers > 0 {
		spb.AppendByte(fbproto.Isc_spb_sts_parallel_workers)
		spb.AppendInt32(int32(options.ParallelWorkers))
	}
	spb.AppendByte(fbproto.Isc_spb_options)
	spb.AppendInt32(statisticsOptionMask(options))
	return spb.Bytes(), nil
}

func statisticsOptionMask(options StatisticsOptions) int32 {
	var mask int32
	if options.DataPages {
		mask |= fbproto.Isc_spb_sts_data_pages
	}
	if options.HeaderPages {
		mask |= fbproto.Isc_spb_sts_hdr_pages
	}
	if options.IndexPages {
		mask |= fbproto.Isc_spb_sts_idx_pages
	}
	if options.SystemRelations {
		mask |= fbproto.Isc_spb_sts_sys_relations
	}
	if options.RecordVersions {
		mask |= fbproto.Isc_spb_sts_record_versions
	}
	if options.NoCreation {
		mask |= fbproto.Isc_spb_sts_nocreation
	}
	if options.Encryption {
		mask |= fbproto.Isc_spb_sts_encryption
	}
	return mask
}
