# Change Log

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
