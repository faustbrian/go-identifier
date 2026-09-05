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

## Upstream review history

### 2026-09-05

- `laravel-framework-updates` advanced from SHA-256
  `0786f3a3a049ea729e48d87c01ac54c48fde29a49f09d4478b2f951e5c027fb5`
  to
  `9ec8486ac832ed7cd2ef2f767bac5119bd7d2d681baa721a3c4229790e03cc2b`.
  The exact Laravel 13.x range
  `402ab340f895525709c1037d5ba52d926865a7ca...cdbd17f7e3257e8ae4207d3c3bef6058452d5f72`
  contains four commits limited to Cloud queue configuration,
  concurrent-process context propagation, JSON:API resource stub cleanup,
  HTTP client callback lifetime, and their tests. The
  `src/Illuminate/Support/Str.php` content is byte-identical at both range
  endpoints (SHA-256
  `14a4be5b420ed2fed14855cc622067ecfe9d73b038d2cdb683e47c5d38196183`).
  The frozen English slug source pin, selected normalization, public API,
  compatibility, and wire contract therefore remain unchanged.

### 2026-09-03

- `nanoid-reference-updates` advanced from SHA-256
  `c4100862408ac8b00b1da16eb92b33ab40a0deddcff4af380a459a3b99c429d3`
  to
  `04181577ec5e0ac803ea289c7a750017da1fcabbd4af77ed2a1cd88f4a760808`.
  The reviewed upstream range
  `07a39d62d84c21af5046fe6b2ef7b3e36ee557db...8588e3a0db022b3876a4724e25cce4961e2a47a4`
  only clarifies that custom alphabets contain 1 through 256 symbols and that
  values outside those bounds can loop forever. This intersects only the Nano
  ID decision mapped below and is behavior-neutral: the package already
  enforces a stricter 2 through 94 unique printable-ASCII alphabet, its 120-bit
  entropy floor, and bounded rejection sampling. The pinned source, selected
  behavior, public API, compatibility, and wire contract remain unchanged.
- `laravel-framework-updates` advanced from SHA-256
  `021b48f3ea3d844cf7eb2131dc602aa0559be94d1e6b4158328471053e9d24fe`
  to
  `73a48d124a01570fc61e8bb936563c8b9793ee0ca9c5bb2c8ea0800edfe67317`.
  The reviewed Laravel 13.x range
  `f9c010aaca36398c9d88ffd4f2629da290bc0878...5ba4bc549544a1bbe7dfff45094a2a0befeff3fc`
  changes 47 files outside `src/Illuminate/Support/Str.php`. The frozen English
  slug decision mapped below is therefore behavior-neutral; its source pin,
  selected normalization, public API, compatibility, and wire contract remain
  unchanged.
- `laravel-framework-updates` advanced again from SHA-256
  `73a48d124a01570fc61e8bb936563c8b9793ee0ca9c5bb2c8ea0800edfe67317`
  to
  `0786f3a3a049ea729e48d87c01ac54c48fde29a49f09d4478b2f951e5c027fb5`.
  The additional reviewed Laravel 13.x range
  `5ba4bc549544a1bbe7dfff45094a2a0befeff3fc...402ab340f895525709c1037d5ba52d926865a7ca`
  contains six commits affecting test fixtures, generic types, style, facade
  docblocks, mail and notification test fakes, and exception handling. The
  production changes are limited to collection, exception-bootstrap, facade,
  test-fake, tap, helper, and type-annotation files; none modifies
  `src/Illuminate/Support/Str.php`. The frozen English slug source pin,
  selected normalization, public API, compatibility, and wire contract
  therefore remain unchanged.

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
