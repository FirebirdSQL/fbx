// Package fbtype converts between Go and Firebird values.
/*
The primary type is [Map]. It maps Firebird types identified by type id and metadata to a [Codec]. A [Codec] is
responsible for converting between Go and Firebird values. [NewMap] creates a [Map] with all supported standard
Firebird types already registered. Additional types can be registered with [Map.RegisterType].

Use [Map.Scan] and [Map.Encode] to decode Firebird values to Go and encode Go values to Firebird respectively.

Base Type Mapping

fbtype maps between all common base types directly between Go and Firebird. In particular:

    Go           Firebird
    -----------------------
    string        char
                  varchar

    // Integers are automatically converted to any other integer type if
    // it can be done without overflow or underflow.
    int8
    int16         smallint
    int32         integer
    int64         bigint
    int
    uint8
    uint16
    uint32
    uint64
    uint

    // Floats are strict and do not automatically convert like integers.
    float32       float
    float64       double precision

    time.Time     date
                  timestamp
                  timestamp with time zone

    []byte        char/varchar character set octets

Null Values

fbtype can map NULLs in two ways. The first is types that can directly represent NULL such as [Int4]. They work in a
similar fashion to database/sql. The second is to use a pointer to a pointer.

    var foo fbtype.Text
    var bar *string
    err := conn.QueryRow("select foo, bar from widgets where id=?", 42).Scan(&foo, &bar)
    if err != nil {
        return err
    }

When using nullable fbtype types as parameters for queries, one has to remember to explicitly set their Valid field to
true, otherwise the parameter's value will be NULL.

Extending Existing Firebird Type Support

Generally, [Codec] implementations support interfaces that can be implemented to enable scanning and encoding. So rather
than use a fbtype-provided type an application can directly use its own type with fbtype as long as it implements the
interfaces that the relevant [Codec] supports.

Sometimes fbx supports a Firebird type such as numeric but the Go type is in an external package that does not have
fbx support such as github.com/shopspring/decimal. These types can be registered with fbtype with custom conversion
logic.

New Firebird Type Support

fbtype uses the Firebird type id to determine how to encode or decode a value. The type id / [Codec] mapping can be
registered with [Map.RegisterType].

A type cannot be registered unless all types it depends on are already registered. e.g. An array type cannot be
registered until its element type is registered.

[ArrayCodec] implements support for arrays. If fbtype supports type T then it can easily support []T by registering an
[ArrayCodec] for the appropriate Firebird type id. In addition, [Array] type can support multi-dimensional arrays.

While fbtype will often still work with unregistered types it is highly recommended that all types be registered due to
an improvement in performance and the elimination of certain edge cases.

If an entirely new Firebird type is used then the application or a library can create a new [Codec]. Then the type id /
[Codec] mapping can be registered with [Map.RegisterType]. There is no difference between a [Codec] defined and
registered by the application and a [Codec] built in to fbtype. See any of the codecs in fbtype for examples of type
registration.

Encoding Unknown Types

fbtype works best when the type id of the Firebird type is known. But in some cases the type id is unknown. In this
case [Map.RegisterDefaultFbType] can be used to register an assumed Firebird type for a particular Go type.

Renamed Types

If fbtype does not recognize a type and that type is a renamed simple type (e.g. type MyInt32 int32) fbtype acts
as if it is the underlying type. It currently cannot automatically detect the underlying type of renamed structs (e.g.
type MyTime time.Time).

Compatibility with database/sql

fbtype also includes support for custom types implementing the database/sql.Scanner and database/sql/driver.Valuer
interfaces.

Encoding Typed Nils

fbtype encodes untyped and typed nils (e.g. nil and []byte(nil)) to the SQL NULL value without going through the [Codec]
system. This means that Codecs and other encoding logic do not have to handle nil or *T(nil).

However, database/sql compatibility requires Value to be called on T(nil) when T implements driver.Valuer. Therefore,
driver.Valuer values are only considered NULL when *T(nil) where driver.Valuer is implemented on T not on *T. See
https://github.com/golang/go/issues/8415 and
https://github.com/golang/go/commit/0ce1d79a6a771f7449ec493b993ed2a720917870.

Overview of Scanning Implementation

The first step is to use the type id to lookup the correct [Codec]. The [Map] will call [Codec.PlanScan] to get a
plan for scanning into the Go value. A [Codec] will support scanning into one or more Go types. Often these Go types
are interfaces rather than explicit types.

If a Go value is not supported directly by a [Codec] then [Map] will try wrapping it with additional logic and try again.
For example, [Int8Codec] does not support scanning into a renamed type (e.g. type myInt64 int64). But [Map] will detect that
myInt64 is a renamed type and create a plan that converts the value to the underlying int64 type and then passes that to
the [Codec] (see [TryFindUnderlyingTypeScanPlan]).

These plan wrappers are contained in [Map.TryWrapScanPlanFuncs]. By default these contain shared logic to handle renamed
types, pointers to pointers, slices, structs, etc. Additional plan wrappers can be added to seamlessly integrate
types that do not support fbx directly.

[Map.Scan] and [Map.Encode] are convenience methods that wrap [Map.PlanScan] and [Map.PlanEncode]. Determining how to
scan or encode a particular type may be a time consuming operation. Hence the planning and execution steps of a
conversion are internally separated.

*/
package fbtype
