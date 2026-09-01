# Identifier conformance matrix

The root module implements the decision-bounded profiles declared for
[RFC 9562 UUIDs](https://www.rfc-editor.org/rfc/rfc9562), the pinned
[ULID](https://raw.githubusercontent.com/ulid/spec/d0c7170df4517939e70129b4d6462cc162f2d5bf/README.md),
[TypeID 0.3.0](https://raw.githubusercontent.com/jetify-com/typeid/be8ff0daf5dc1f6d40c62a03cfc89945263a69af/spec/README.md),
[Segment KSUID](https://raw.githubusercontent.com/segmentio/ksuid/d33724947fcfba7949906c2b1821e96a1c8d06e7/README.md),
and [Nano ID](https://raw.githubusercontent.com/ai/nanoid/07a39d62d84c21af5046fe6b2ef7b3e36ee557db/README.md), plus the frozen
[Laravel English slug profile](https://raw.githubusercontent.com/laravel/framework/138e5806ed7e4e21591948b661119d3810f052cf/src/Illuminate/Support/Str.php).
The [specification decision register](../docs/specification-decisions.md)
defines the supported boundary; no broader compliance claim is implied.

[`manifest.tsv`](manifest.tsv) pins sources, fixtures, algorithms, and peer
revisions. [`monitoring.json`](monitoring.json) requires source and upstream
update review every 30 days. [`maintained-peers.json`](maintained-peers.json) records
observed peer behavior without treating popularity as normative authority.

## Decision conformance

| Decision | Authority | Executable evidence | Differential result |
| --- | --- | --- | --- |
| IDENTIFIER-UUID-DEC-001 | `rfc9562-source` | UUID official-vector, rejection, interoperability, and fuzz evidence | Deliberate strict-text policy difference from google/uuid v1.6.0 |
| IDENTIFIER-UUID-DEC-002 | `rfc9562-source` | UUIDv4 bit, UUIDv7 monotonic, rollback, overflow, and shared-state evidence | Deliberate per-generator policy difference from google/uuid v1.6.0 |
| IDENTIFIER-ULID-DEC-001 | `ulid-spec-source` | Canonical-case, overflow, maintained-parser, and fuzz evidence | Deliberate consistent-case policy difference |
| IDENTIFIER-ULID-DEC-002 | `ulid-spec-source` | Monotonic, rollback, entropy, carry, overflow, and shared-state evidence | Deliberate owned-clock policy difference from oklog/ulid v2.1.1 |
| IDENTIFIER-TYPEID-DEC-001 | `typeid-spec-source` | Complete 9 valid and 21 invalid official cases, zero semantics, and fuzz evidence | Official corpus and maintained peer agree; no separate differential artifact is claimed |
| IDENTIFIER-TYPEID-DEC-002 | `typeid-spec-source` | UUIDv7 generation, official vectors, and shared-state evidence | Maintained peer agrees; no policy divergence is claimed |
| IDENTIFIER-KSUID-DEC-001 | `segment-ksuid-source` | Segment codec differential, hostile parsing, monotonic, rollback, overflow, and fuzz evidence | Deliberate monotonic-generation difference from segmentio/ksuid v1.0.4 |
| IDENTIFIER-NANOID-DEC-001 | `nanoid-reference-source` | Reference default, configuration, uniformity, peer, and fuzz evidence | Deliberate entropy-floor and bounded-profile difference from matoous/go-nanoid v2.1.0 |
| IDENTIFIER-SLUG-DEC-001 | `laravel-slug-source`, `spatie-sluggable-source` | Frozen 1,000-plus-vector differential corpus, boundary, concurrency, and fuzz evidence | Deliberate pure normalization boundary without persistence uniqueness or suffixing |

## Update and conformance process

1. A changed authority digest, release surface, official corpus, or maintained
   peer blocks unattended adoption.
2. Maintainers review the affected decision, source requirement strength,
   public APIs, compatibility and wire effects, executable evidence, and
   changelog record before selecting behavior.
3. `golib specification check` validates the complete offline register,
   history, evidence, provenance, and change-control contract.
4. `golib specification check --online` additionally verifies every current
   authority digest over bounded public HTTPS retrieval.
5. The repository conformance gate runs focused family and cross-language
   behavior in local and hosted CI. An unresolved decision blocks release.

No unresolved decision is currently recorded. New ambiguities or peer
divergences remain unresolved until a sourced decision and executable evidence
are reviewed.
