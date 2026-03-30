# Change Log

# [0.3] - 06.04.2026

### Added
- RS-279430. zeronull/int128 support.
- RS-286253. Encoding octets to string.
- RS-280247. Allow to configure fetch size with fetch_batch_size option.
- RS-278714. Add support for time, int128, dec types in stdlib.
- RS-278082. Port pgx v5.7.5..v5.7.6. pool API updates: add fbxpool hooks PrepareConn and ShouldPing.
- RS-278082. Port pgx v5.7.5..v5.7.6. Add Int64Valuer support for zeronull.Int2 / zeronull.Int4 / zeronull.Int8.
- RS-278083. Port pgx v5.7.6..v5.8.0. feat(stdlib): add OptionShouldPing to control ResetSession ping behavior.
- RS-278083. Port pgx v5.7.6..v5.8.0. feat(fbxpool): acquire ping timeout this commit adds ping timeout in acquire loop for when the connection ping results in timeout instead of an error.

### Improved

- RS-285525. Add more patterns to date conversions formats of fbtype.Date.
- RS-278082. Port pgx v5.7.5..v5.7.6. Decrease allocations in stdlib ExecContext / QueryContext argument conversion path.
- RS-278082. Port pgx v5.7.5..v5.7.6. Protect fbxpool acquire from infinite retry loops by capping attempts to MaxConns + 1.
- RS-278083. Port pgx v5.7.6..v5.8.0. Faster shutdown of fbxpool.Pool background goroutines.
- RS-278083. Port pgx v5.7.6..v5.8.0. perf(stmtcache): replace O(n) invalidation scan with O(1) map lookup.
- RS-278083. Port pgx v5.7.6..v5.8.0. perf(iobufpool): optimize pool index calculations with O(1) bit manipulation.
- RS-278083. Port pgx v5.7.6..v5.8.0. fbtype numeric optimization: replace Cmp(big0) with Sign().

### Fixed

- RS-277802. Panic when passing argument as SQL_NULL type.
- RS-282682. Wrong SRP payload format detection.
- RS-276860. Error when numeric to string scanning.
- RS-276815. Error when scanning/encoding string to FB types.
- RS-281968. Wrong octets handling on RDB 6.
- RS-278531: Fix blob fetch batching and align seek behavior tests for stream/segmented blobs.
- RS-278082. Port pgx v5.7.5..v5.7.6. Return fbconn.ParseConfigError from fbx.ParseConfig / fbxpool.ParseConfig using fbconn.NewParseConfigError.
- RS-278083. Port pgx v5.7.6..v5.8.0. fix: invalidate statement/description cache in Exec.
- RS-278083. Port pgx v5.7.6..v5.8.0. Array and Composite codecs handle typed nils.
- RS-278083. Port pgx v5.7.6..v5.8.0. Scan unknown types into *any as string or []byte based on format code.
- RS-278083. Port pgx v5.7.6..v5.8.0. Avoid overflow when MaxConns is set to MaxInt32.

### Changed
- RS-278083. Port pgx v5.7.6..v5.8.0. Remove unused stmcache.UnlimitedCache.
- RS-278083. Port pgx v5.7.6..v5.8.0. Migrate from math/rand to math/rand/v2.

# [0.2] - 09.02.2026

### Added

- RS-259205. Add BLOB support
- RS-275689. Support DECIMAL(16) и DECIMAL(34) types
- RS-270665. Support INT128 type 
- RS-276843. Add support of named arguments
- RS-261350. Add session validators ValidateConnectTargetSessionAttrsPrimary, ValidateConnectTargetSessionAttrsReplica and ValidateConnectTargetSessionAttrsPreferReplica.
             Respect target_session_attrs connection variable with values primary, replica, prefer-replica, read-only, read-write.

### Improved

- RS-275530. RS-274062. Decrease allocations for wire client messages

### Fixed

- RS-272354. RS-264224. Multiple TIMESTAMP WITH TIMEZONE type scan/encode fixes
- RS-270082. RS-270067. RS-270062. RS-270056. Multiple NUMERIC type encoding fixes
- RS-268164. Return error if count of passed parameters do not match with count of placeholders in query
- RS-264078. Fix crash on when passing go float64 into fb float
- RS-277296. Fix Rows.ColumnType* returning values for some types
- RS-275664. Respect wire protocol version for op_execute

# [0.1] - 11.12.2025

### Added

- RS-243155. Initial public release. Most of PGX driver ported to Firebird Wire protocol
- RS-252977. Implement Legacy and SRP authentication and Wire encryption
- RS-258021. Implement GSS authentication (based on gokrb5)
