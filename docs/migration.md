# Migration

## V2 diagnostic redaction

Version 2's error and test-diagnostic surfaces redact
caller-supplied validator and entropy errors, malformed JSON tokens, SQL source
types, and rollback clock values. The package sentinel remains available to
`errors.Is`, but v2 intentionally no longer exposes an injected validator or
entropy error through the error chain.

Version 2 also rejects generic typed-identifier text above 1,024 bytes before
calling its validator and encoded JSON above 6,146 bytes before decoding. Audit
typed identifiers longer than `MaxTypedIDBytes` before migration.

The direct owned consumers below remain on released v1 until a signed v2 tag,
clean public module-consumer proof, and separate migration review. They block
declaring the ecosystem migration complete, not publishing the v2 module:

- `go-correlation` uses Identifier v1.0.0 UUID generation;
- `go-queue-control-plane` uses Identifier v1.0.0 ULID generation; and
- `go-library-tools/release/compatibility-consumer` pins Identifier v1.0.0 as
  its released compatibility boundary.

Those consumers must migrate only after a public v2 release and a separate
review of each affected runtime or compatibility boundary. Do not add a local
`replace` directive to bridge an unpublished source. V2 imports add the `/v2`
path segment; released-v1 imports remain unchanged.

## Laravel and Postal ULIDs

Laravel ULIDs and Postal persistence can use a lowercase 26-character ULID
representation. Migrate values as text without changing case, decoding,
re-encoding, or generating replacements. Before cutover, stream every stored
value through `ulid.Parse`; compare it with `parsed.StringLower()` for a
lowercase schema or `parsed.String()` for an uppercase schema; and compare the
old and new bytewise sort order. Reject mixed-case rows and rows that depend on
a case-insensitive or locale-specific collation.

For a zero-downtime migration, dual-read the old column, validate in shadow,
then dual-write the identical schema-compatible string. Backfill with an equality
check, add the new unique constraint, switch reads, and retain rollback until
counts, extrema, and ordered samples match. The official ULID vector
`01ARZ3NDEKTSV4RRFFQ69G5FAV` is covered by the compatibility suite.

## Cline Mint and strongly typed IDs

Treat a Mint or `cline/strongly-typed-id` value as an existing wire contract,
not as permission to infer a new family. Export the exact canonical string,
identify whether its payload is UUID, ULID, TypeID, KSUID, or a project-specific
format, and validate it with that parser. Use `identifier.ID[DomainTag]` to add
compile-time domain separation without changing stored text.

Do not convert ULID to UUIDv7 or add a TypeID prefix in place: those create new
identifiers. If a new representation is required, add a new column and retain
an explicit old-to-new mapping. Compare null behavior, JSON shape, SQL type,
case, collation, timestamp extraction, and ordering before switching readers.

## Rollback

Rollback restores readers to the old column; it never attempts to reconstruct
old values from newly generated IDs. Keep the old unique index until every
writer and asynchronous consumer has moved and reconciliation is complete.
