# Security

The repository-specific, versioned [threat model](threat-model.md) inventories
assets, trust boundaries, attacker-controlled inputs, controls, and accepted
risks for the v2 source. Released v1 retains its published
diagnostic behavior. This page provides the corresponding adoption guidance.

Default randomness comes from `crypto/rand.Reader`. Supplying another reader
transfers responsibility for independence, unpredictability, concurrency, and
failure behavior to the caller. `idtest.Reader` is deterministic and forbidden
for production.

Parsers reject unsupported or mixed case, ambiguous Crockford symbols, malformed
prefixes, invalid UUID versions or variants where the UUID contract requires
them, Base32/Base62 overflow, duplicate NanoID alphabet bytes, non-ASCII NanoID
alphabets, and configurations below 120 entropy bits.

UUIDv7, ULID, TypeID, and KSUID leak creation time. Monotonic generators leak
relative issuance order within their time bucket. TypeID prefixes disclose
entity categories. KSUID payload increments and UUIDv7/ULID increments can
make neighboring values guessable after one value is observed. These are
database identifiers, not secrets.

The exposed timestamp is 48 bits at millisecond resolution for UUIDv7, ULID,
and generated TypeIDs, and 32 bits at second resolution after the KSUID epoch
for KSUID. A TypeID additionally exposes up to 63 ASCII prefix bytes. Generated
families contain no node or topology field. Parse-only UUIDv1 and UUIDv6 values
can retain an externally supplied 48-bit node field. UUIDv4 and NanoID expose
no timestamp or topology field.

Every identifier implements `slog.LogValuer` and emits `[REDACTED]` when passed
directly to Go's structured logger. Revealing one requires an explicit call to
`String`. Other logging systems and metric labels are caller-owned boundaries:
apply an explicit data-classification decision, retention policy, and
cardinality budget, and never put raw identifiers in metric labels by default.
The package emits no logs or metrics of its own and provides no automatic raw
identifier labels.

Generator rollback and overflow errors are operational signals. Do not retry in
a tight loop or silently switch algorithms. Repair the clock, rotate to a new
owned generator only when duplicate domains are understood, or apply bounded
backpressure according to the application's availability contract.

Package-generated diagnostics retain stable identifier sentinel categories but
do not include caller-supplied validator or entropy errors, SQL runtime types,
raw identifiers, or observed rollback timestamps. Record the sentinel and
operation; attach sensitive source details only through a separately approved,
access-controlled channel.

Generic typed identifiers accept at most 1,024 bytes of canonical text and
6,146 bytes of encoded JSON. These limits are exposed as `MaxTypedIDBytes` and
`MaxTypedIDJSONBytes`; oversized input is rejected before a caller validator or
JSON decoder runs. Validators remain synchronous caller code and must be
bounded and panic-free.

Concrete-family text, binary, SQL, and JSON decoders apply their fixed wire
length before converting or decoding input. Encoded JSON is bounded to six
times the family text length plus the two surrounding quotes, preserving every
valid ASCII escape form without accepting proportional work from larger input.

`uuid.ID` is a public array copied by assignment, not an immutable wrapper.
Callers can construct or mutate their own value outside parsing; outbound
encoding assumes that value is assigned and valid. Validate untrusted bytes
through `uuid.FromBytes` or a decoding method before using the value.
