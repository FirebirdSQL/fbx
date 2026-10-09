package fbconn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"rdb.red-soft.ru/fbx/fbproto"
)

// RepairOperation selects a gfix maintenance action. Zero is invalid.
type RepairOperation uint8

const (
	RepairValidate    RepairOperation = iota + 1 // Validate database pages.
	RepairSweep                                  // Remove obsolete record versions.
	RepairMend                                   // Mark corrupt records unavailable, potentially losing data.
	RepairKillShadows                            // Remove references to unavailable shadow files.
	RepairUpgradeODS                             // Upgrade ODS in place when supported by the server.
	RepairFixICU                                 // Update ICU collations and rebuild their indexes.
)

// RepairOptions controls [ServiceConn.Repair]. Compatibility between options
// is checked by fbservice; this layer checks SPB representation and argument safety.
type RepairOptions struct {
	Operation RepairOperation
	// Output receives the report. Nil discards it. Writes are synchronous.
	Output          io.Writer
	Full            bool // Validate records as well as pages.
	ReadOnly        bool // Check without repairing the database.
	IgnoreChecksums bool
	// ParallelWorkers sets the requested worker count; zero omits the parameter.
	// Sweep uses a legacy parameter on server versions before 5
	// and the standard parameter on 5+. ICU repair requires Firebird 5+ or compatible.
	// Servers without parallel sweep support return an error.
	ParallelWorkers int
	// SweepTimeout limits sweep on the server in whole seconds (0..2147483).
	// Zero omits the limit. This optional extension requires server support and is
	// sent only on server versions 6+; on older versions it is omitted.
	// It is independent of ctx.
	SweepTimeout int
}

// Repair runs a gfix repair action on a server-side path or alias. It waits for
// service completion and returns server errors without retrying or removing
// unsupported options, except SweepTimeout on server versions before 6.
// Output is not closed. Cancellation does not undo server
// changes and cannot interrupt an Output.Write already in progress.
func (c *ServiceConn) Repair(ctx context.Context, database string, options RepairOptions) error {
	// Validate before the optional server-version query.
	spb, err := buildRepairActionSPB(database, options, ServerVersion{})
	if err != nil {
		return err
	}
	if options.Operation == RepairSweep && (options.ParallelWorkers > 0 || options.SweepTimeout > 0) {
		version, err := c.serviceServerVersion(ctx)
		if err != nil {
			return fmt.Errorf("repair database: %w", err)
		}
		spb, err = buildRepairActionSPB(database, options, version)
		if err != nil {
			return err
		}
	}
	if err := c.runServiceAction(ctx, spb, options.Output); err != nil {
		return fmt.Errorf("repair database: %w", err)
	}
	return nil
}

func newMaintenanceSPB(action byte, database string) (*fbproto.XPBWriter, error) {
	if database == "" || strings.HasPrefix(database, "-") {
		return nil, errors.New("maintenance database must not be empty or start with '-'")
	}
	if strings.IndexByte(database, 0) >= 0 {
		return nil, errors.New("maintenance database must not contain NUL")
	}
	spb := fbproto.NewXPBWriterFromTag(action)
	if err := putServiceActionString(spb, fbproto.Isc_spb_dbname, database); err != nil {
		return nil, fmt.Errorf("maintenance database: %w", err)
	}
	return spb, nil
}

func buildRepairActionSPB(database string, options RepairOptions, version ServerVersion) ([]byte, error) {
	spb, err := newMaintenanceSPB(fbproto.Isc_action_svc_repair, database)
	if err != nil {
		return nil, err
	}
	var mask int32
	switch options.Operation {
	case RepairValidate:
		mask = fbproto.Isc_spb_rpr_validate_db
	case RepairSweep:
		mask = fbproto.Isc_spb_rpr_sweep_db
	case RepairMend:
		mask = fbproto.Isc_spb_rpr_mend_db
	case RepairKillShadows:
		mask = fbproto.Isc_spb_rpr_kill_shadows
	case RepairUpgradeODS:
		mask = fbproto.Isc_spb_rpr_upgrade_db
	case RepairFixICU:
		mask = fbproto.Isc_spb_rpr_icu
	default:
		return nil, errors.New("invalid repair operation")
	}
	if options.Full {
		mask |= fbproto.Isc_spb_rpr_full
	}
	if options.ReadOnly {
		mask |= fbproto.Isc_spb_rpr_check_db
	}
	if options.IgnoreChecksums {
		mask |= fbproto.Isc_spb_rpr_ignore_checksum
	}
	spb.AppendByte(fbproto.Isc_spb_options).AppendInt32(mask)
	if options.ParallelWorkers < 0 || options.ParallelWorkers > maxServiceActionInt32 {
		return nil, fmt.Errorf("repair parallel workers must be between 0 and %d", maxServiceActionInt32)
	}
	if options.ParallelWorkers != 0 {
		tag := byte(fbproto.Isc_spb_rpr_par_workers)
		if options.Operation == RepairSweep && !version.AtLeast(5, 0, 0) {
			tag = fbproto.Isc_spb_rpr_par_workers_rs
		}
		spb.AppendByte(tag).AppendInt32(int32(options.ParallelWorkers))
	}
	if options.SweepTimeout < 0 || options.SweepTimeout > maxServiceActionInt32/1000 {
		return nil, errors.New("sweep timeout must be between 0 and 2147483 seconds")
	}
	if options.SweepTimeout != 0 && version.AtLeast(6, 0, 0) {
		spb.AppendByte(fbproto.Isc_spb_rpr_sweep_timeout).AppendInt32(int32(options.SweepTimeout))
	}
	return spb.Bytes(), nil
}

// DatabaseAccessMode selects whether a database accepts writes.
type DatabaseAccessMode uint8

const (
	DatabaseAccessModeUnchanged DatabaseAccessMode = iota
	DatabaseAccessModeReadOnly
	DatabaseAccessModeReadWrite
)

// DatabaseWriteMode selects asynchronous or forced writes.
type DatabaseWriteMode uint8

const (
	DatabaseWriteModeUnchanged DatabaseWriteMode = iota
	DatabaseWriteModeAsync
	DatabaseWriteModeSync
)

// DatabaseMode selects the database's shutdown/online state.
type DatabaseMode uint8

const (
	DatabaseModeUnchanged DatabaseMode = iota
	DatabaseModeNormal
	DatabaseModeMulti
	DatabaseModeSingle
	DatabaseModeFull
)

// ShutdownMethod selects which activity to wait for before shutdown.
type ShutdownMethod uint8

const (
	ShutdownForce ShutdownMethod = iota + 1
	ShutdownDenyAttachments
	ShutdownDenyTransactions
)

// ShutdownOptions controls a database shutdown. Timeout is in whole seconds;
// zero requests immediate shutdown. Mode and Method must be set explicitly.
type ShutdownOptions struct {
	Mode    DatabaseMode
	Method  ShutdownMethod
	Timeout int
}

// DatabaseProperties contains gfix property changes. Nil pointers and zero
// enums omit the corresponding setting. Values pointed to must not be changed
// during the call. Use fbservice to check mutually exclusive changes.
type DatabaseProperties struct {
	Output     io.Writer
	AccessMode DatabaseAccessMode
	WriteMode  DatabaseWriteMode
	// PageBuffers sets the database cache in pages; zero restores the server default.
	PageBuffers *int
	// SweepInterval sets the automatic sweep threshold; zero disables it.
	SweepInterval *int
	// SQLDialect is 1 or 3; zero leaves it unchanged.
	SQLDialect int
	// ReserveSpace selects whether to reserve space for record versions.
	ReserveSpace *bool
	Shutdown     *ShutdownOptions
	OnlineMode   DatabaseMode
	// ActivateShadow promotes database (the shadow file path) to a database.
	ActivateShadow bool
}

// SetDatabaseProperties runs a gfix properties action. The server validates
// combinations and version support. Output and cancellation behave as in Repair.
func (c *ServiceConn) SetDatabaseProperties(ctx context.Context, database string, options DatabaseProperties) error {
	spb, err := buildPropertiesActionSPB(database, options)
	if err != nil {
		return err
	}
	if err := c.runServiceAction(ctx, spb, options.Output); err != nil {
		return fmt.Errorf("set database properties: %w", err)
	}
	return nil
}

func buildPropertiesActionSPB(database string, options DatabaseProperties) ([]byte, error) {
	spb, err := newMaintenanceSPB(fbproto.Isc_action_svc_properties, database)
	if err != nil {
		return nil, err
	}
	switch options.AccessMode {
	case DatabaseAccessModeUnchanged:
	case DatabaseAccessModeReadOnly:
		spb.AppendByte(fbproto.Isc_spb_prp_access_mode).AppendByte(fbproto.Isc_spb_prp_am_readonly)
	case DatabaseAccessModeReadWrite:
		spb.AppendByte(fbproto.Isc_spb_prp_access_mode).AppendByte(fbproto.Isc_spb_prp_am_readwrite)
	default:
		return nil, errors.New("invalid database access mode")
	}
	switch options.WriteMode {
	case DatabaseWriteModeUnchanged:
	case DatabaseWriteModeAsync:
		spb.AppendByte(fbproto.Isc_spb_prp_write_mode).AppendByte(fbproto.Isc_spb_prp_wm_async)
	case DatabaseWriteModeSync:
		spb.AppendByte(fbproto.Isc_spb_prp_write_mode).AppendByte(fbproto.Isc_spb_prp_wm_sync)
	default:
		return nil, errors.New("invalid database write mode")
	}
	for _, field := range []struct {
		tag   byte
		value *int
		name  string
	}{
		{fbproto.Isc_spb_prp_page_buffers, options.PageBuffers, "page buffers"},
		{fbproto.Isc_spb_prp_sweep_interval, options.SweepInterval, "sweep interval"},
	} {
		if field.value == nil {
			continue
		}
		if *field.value < 0 || *field.value > maxServiceActionInt32 {
			return nil, fmt.Errorf("database %s must be between 0 and %d", field.name, maxServiceActionInt32)
		}
		spb.AppendByte(field.tag).AppendInt32(int32(*field.value))
	}
	if options.SQLDialect != 0 {
		if options.SQLDialect != 1 && options.SQLDialect != 3 {
			return nil, errors.New("database SQL dialect must be 1 or 3")
		}
		spb.AppendByte(fbproto.Isc_spb_prp_set_sql_dialect).AppendInt32(int32(options.SQLDialect))
	}
	if options.ReserveSpace != nil {
		value := byte(fbproto.Isc_spb_prp_res_use_full)
		if *options.ReserveSpace {
			value = fbproto.Isc_spb_prp_res
		}
		spb.AppendByte(fbproto.Isc_spb_prp_reserve_space).AppendByte(value)
	}
	if shutdown := options.Shutdown; shutdown != nil {
		if shutdown.Mode < DatabaseModeMulti || shutdown.Mode > DatabaseModeFull {
			return nil, errors.New("invalid shutdown database mode")
		}
		if shutdown.Timeout < 0 || shutdown.Timeout > maxServiceActionInt32 {
			return nil, errors.New("shutdown timeout must fit a nonnegative int32")
		}
		var tag byte
		switch shutdown.Method {
		case ShutdownForce:
			tag = fbproto.Isc_spb_prp_force_shutdown
		case ShutdownDenyAttachments:
			tag = fbproto.Isc_spb_prp_attachments_shutdown
		case ShutdownDenyTransactions:
			tag = fbproto.Isc_spb_prp_transactions_shutdown
		default:
			return nil, errors.New("invalid database shutdown method")
		}
		spb.AppendByte(fbproto.Isc_spb_prp_shutdown_mode).AppendByte(byte(shutdown.Mode - DatabaseModeNormal))
		spb.AppendByte(tag).AppendInt32(int32(shutdown.Timeout))
	}
	if options.OnlineMode != DatabaseModeUnchanged {
		if options.OnlineMode < DatabaseModeNormal || options.OnlineMode > DatabaseModeSingle {
			return nil, errors.New("invalid online database mode")
		}
		spb.AppendByte(fbproto.Isc_spb_prp_online_mode).AppendByte(byte(options.OnlineMode - DatabaseModeNormal))
	}
	if options.ActivateShadow {
		spb.AppendByte(fbproto.Isc_spb_options).AppendInt32(fbproto.Isc_spb_prp_activate)
	}
	return spb.Bytes(), nil
}
