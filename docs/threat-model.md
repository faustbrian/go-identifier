# Threat model

**Model version:** 1.0
**Applies to:** `github.com/faustbrian/go-identifier/v2` source
**Reviewed:** 2026-09-13
**Owner:** `go-identifier` maintainers

This model covers identifier parsing, generation, serialization, inspection,
database scanning, structured logging, and the `idtest` support package. The
[security guidance](security.md) translates these boundaries into adoption
rules; the repository [security policy](../SECURITY.md) defines private
reporting.

Released v1.0.0 retains its published diagnostic behavior and is not described
as hardened by this model. Direct owned consumers migrate only after the
signed v2 release is public; their migration is a separate delivery boundary.

## Assets and security properties

- Canonical identifier bytes and text must not gain aliases that bypass
  equality, allowlists, signatures, caches, or audit comparison.
- Generated random fields must use the configured entropy source without bias,
  partial-success output, silent fallback, or unbounded retry.
- Per-generator monotonic state must remain race-free and must fail atomically
  on rollback or exhaustion.
- Identifiers, caller values, entropy errors, runtime types, and clock values
  must not enter implicit logs or package-generated diagnostics.
- CPU, memory, retries, and input-dependent allocations must remain bounded.

Identifiers are pseudonymous application data, not secrets, credentials,
authorization facts, idempotency proofs, or tracing contexts. Applications own
the confidentiality, uniqueness, retention, and access-control policy of each
stored identifier.

## Trust boundaries and attacker-controlled inputs

| Boundary | Untrusted or caller-controlled material | Package control |
| --- | --- | --- |
| Text, JSON, binary, SQL, and PostgreSQL decoding | Identifier bytes, lengths, alphabets, prefixes, JSON tokens, and SQL source types | Strict canonical parsing, fixed representation limits, copied values, and bounded diagnostics |
| Typed identifiers | Canonical text and zero-value `Validator` implementations, including returned errors | Reject empty, invalid, or larger-than-1,024-byte text before validation; cap encoded JSON at 6,146 bytes; retain only `ErrInvalid` and a fixed diagnostic |
| Generation | `Clock` results and `io.Reader` bytes, latency, partial reads, and errors | Fixed-size reads, bounded NanoID rejection, serialized state, atomic failure, fixed diagnostics, and no fallback |
| Serialization and inspection | Assigned identifier values | Explicit methods only; no background export, logging, metrics, network, filesystem, environment, or process access |
| Logging and test assertions | Identifier values and parser errors | `slog.LogValuer` and `idtest` diagnostics emit `[REDACTED]`; revealing text requires an explicit `String` call |

The module has no service boundary and starts no goroutines. It performs no
network, filesystem, environment, subprocess, or credential-store access.

## Controls

- Generic typed identifiers reject text above 1,024 bytes before invoking a
  validator and reject JSON above 6,146 bytes before decoding. UUID, ULID,
  TypeID, KSUID, and NanoID parsers enforce family-specific
  canonical forms and fixed maximum lengths. Their text unmarshallers and SQL
  scanners reject invalid lengths before conversion, and their JSON decoders
  cap the encoded input at six times the maximum text length plus two quotes.
  Slug work is capped at 250 input runes. NanoID configuration caps output at
  1,024 bytes and generation at 128 fixed-size rejection rounds.
- Default generation uses `crypto/rand.Reader`. Entropy, rollback, overflow,
  JSON, and invalid-input failures retain stable package sentinel
  classification but discard caller and dependency error text.
- Unsupported SQL source diagnostics do not format runtime types. Clock
  rollback diagnostics do not include observed timestamps.
- Each mutable generator owns a mutex and advances state only after successful
  entropy acquisition and validation.
- Structured logging is redacted by default. Explicit text, JSON, binary,
  database, and inspection methods are deliberate data-release boundaries.
- Hostile diagnostic tests enforce sentinel preservation, redaction, and a
  256-byte maximum across typed validation, entropy failures, malformed JSON,
  clock rollback, and SQL scanner rejection.

## Accepted risks

| ID | Risk | Owner | Rationale | Mitigation | Review condition |
| --- | --- | --- | --- | --- | --- |
| IDENTIFIER-RISK-001 | UUIDv7, ULID, generated TypeID, and KSUID reveal creation time and local issuance order; TypeID reveals its prefix and imported UUIDv1/v6 values may contain a node field. | `go-identifier` maintainers | These fields are required by the selected public wire formats. | Use UUIDv4 or NanoID when this metadata is unacceptable; never use an identifier as a secret; apply application retention and access controls. | Reassess when adding a family, changing a wire profile, or receiving evidence that metadata disclosure violates an owned use case. |
| IDENTIFIER-RISK-002 | Caller-supplied `Validator`, `Clock`, and `io.Reader` implementations are synchronous and cannot be cancelled by the v2 API if they block; these collaborators can also panic. | `go-identifier` maintainers | V2 retains the standard synchronous interfaces from released v1; a goroutine wrapper would leak work and would not cancel the collaborator, while recovering a collaborator panic would hide caller corruption. | Typed input is capped before validation; defaults use local `time.Now` and `crypto/rand.Reader`; callers must inject prompt, bounded, panic-free collaborators and apply cancellation before entering validation or generation. | Reassess after a report of blocked or panicking production validation or generation, or when adopting a context-aware collaborator contract. |
| IDENTIFIER-RISK-003 | Explicit `String`, marshal, database, and inspection calls release identifier or timestamp data. | Application owners | These methods are the package's required persistence and interoperability surface. | Classify data before use, avoid raw metric labels, prefer structured logging redaction, and restrict storage and telemetry access. | Reassess when adding an automatic exporter, logger, metric, tracing adapter, or new serialization surface. |

No accepted risk permits credentials, live customer data, entropy-source errors,
runtime type descriptions, or attacker-supplied values in implicit diagnostics.

## Change and release review

Review this model whenever a public parser, generator, serializer, scanner,
logger, dependency, identifier family, or input bound changes, and at each
major-version design review. A release is blocked by an unowned High or
Critical finding, an unbounded attacker-controlled path, loss of classified
errors, or a diagnostic that can expose caller-controlled values. Accepted
risks require the owner, rationale, mitigation, and review condition recorded
above.
