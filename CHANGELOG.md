# Changelog

All notable changes follow Keep a Changelog and semantic versioning.

## Unreleased

### Changed

- Review Laravel 13.x Cloud queue, concurrency context, JSON:API stub, and HTTP
  client callback-lifetime updates through
  `cdbd17f7e3257e8ae4207d3c3bef6058452d5f72`; the frozen `Str.php` source and
  English slug behavior remain unchanged.
  - IDENTIFIER-SLUG-DEC-001 sha256:f5beb4c9e50201006e5df3baf217e7724764e1a9da634aafdc0048320bc56c09
- Adopt the checksum-verified `go-library-tools` v1.4.0 CLI and immutable
  reusable workflow so local and hosted cohesion and specification checks use
  the final monitored-authority contract.
- Review additional Laravel 13.x test-fixture, generic-type, style, facade,
  test-fake, and exception-handling updates through
  `402ab340f895525709c1037d5ba52d926865a7ca`; the pinned `Str.php` source and
  frozen English slug behavior remain unchanged.
  - IDENTIFIER-SLUG-DEC-001 sha256:198d1a7a4f7ab6590953ef75a10609b53b81a4eae4575e4bcf2015cc0a88cafd

- Record the reviewed Nano ID reference clarification as behavior-neutral for
  the package's stricter bounded custom-alphabet profile while retaining the
  pinned source and public behavior.
  - IDENTIFIER-NANOID-DEC-001 sha256:1f0f06dfcfbed64c8990b10494c5932db444aa5900c1427f2653dcf98e1015eb
- Review Laravel 13.x updates through `5ba4bc549544a1bbe7dfff45094a2a0befeff3fc`;
  the pinned `Str.php` source and frozen English slug behavior remain unchanged.
  - IDENTIFIER-SLUG-DEC-001 sha256:1643f5b39e02cd8bde30ee3a42533e739317ffc44ae734019bc81789186cd358

- Adopt the checksum-verified `go-library-tools` v1.3.0 CLI, schema-v2 cohesion
  metadata, and repository-local cohesion gate while retaining the module's
  public API and runtime behavior.
- Pin reusable CI to the immutable v1.3.0 cohesion workflow so hosted checks
  enforce the same metadata contract as local development.
- Adopt the checksum-verified `go-library-tools` v1.2.0 CLI and immutable
  shared workflow so local and hosted gates enforce specification governance
  while retaining package-owned API, provenance, and mutation evidence.
- Review Laravel 13.x updates through `f9c010aaca36398c9d88ffd4f2629da290bc0878`;
  the frozen `Str::slug` profile and identifier behavior remain unchanged.

### Documentation

- Document stable-v1 maturity, Go 1.26.6 support, installation, lifecycle,
  executable adoption, troubleshooting, support, and security entry points.
- Correct the security-reporting repository identity and the v1.0.0 heading to
  the signed tag and release publication date.
- Link identifier selection and adoption guidance to the immutable v1.4.0
  ecosystem index and Foundations family.
- Replace archived monorepo and hardening terminology with package-owned
  documentation and verification guidance.

- Declare and govern the externally specified UUID, ULID, TypeID, KSUID,
  Nano ID, and frozen Laravel slug compatibility surface in the
  [specification decision register](docs/specification-decisions.md):
  - IDENTIFIER-UUID-DEC-001 sha256:a6465fa9bcb66362430458b9d8ef44c644841952dc316588418cc36444641e7a
  - IDENTIFIER-UUID-DEC-002 sha256:10ff4097fed07721c5ec6dbc4d40a3e1b4d1e062d5373a7bf39c53e4f134c96e
  - IDENTIFIER-ULID-DEC-001 sha256:b3a6091389aee2883827a1d3b9184fea4e1ba0d60a8fb1335be9349a0b964eab
  - IDENTIFIER-ULID-DEC-002 sha256:9a8779c12a55915d18646c61a36fa769f024144449f72408ddcad82619ffc041
  - IDENTIFIER-TYPEID-DEC-001 sha256:d0e25b9ecb97f70dffd054cd63f8f0c423c7b52672a7898b6440348c6294907c
  - IDENTIFIER-TYPEID-DEC-002 sha256:0c7c9bfb75618c4ebceaa8facad48b9aa4d695d70270daa1dd3fe6b4d32bbd8e
  - IDENTIFIER-KSUID-DEC-001 sha256:89ad85c83ce29b92eaa2e10d5889c329190ed8ac65b7e029610a47a06af270f9
  - IDENTIFIER-NANOID-DEC-001 sha256:4262c4d507befaf8bb8ae13ed0abf153b0ff2dcc1f8fa11d0117e14929d4ac93
  - IDENTIFIER-SLUG-DEC-001 sha256:8c3431d77c68c089e940105a76af959f3e99fe8aea9536a46e608aae94b6af8b

## 1.0.0 - 2026-08-26

### Changed

- Exclude intentional nested modules from root local-proxy archives so local,
  bootstrap, CI, and public module checksums describe the same source
  boundary.

- Track the pinned documentation-tool lockfile so clean CI checkouts install
  the exact validated cspell dependency.

- Reconcile standalone dependency checksums against deterministic current
  module archives so CI, local verification, and release consumers resolve
  identical content.

- Harden standalone documentation validation with deterministic spelling and
  link checks, package-specific documentation gates, and repository-local
  contributor guidance.

### Documentation

- Link the package README to package-owned documentation.

### Changed

- Publish the module from its standalone `github.com/faustbrian/go-identifier` identity while preserving its documented API and behavior.
- Use the concurrency-safe default cryptographic entropy source directly for
  UUIDv4 generation while retaining serialization for caller-provided readers.
- Escape line breaks in generic identifier assertion diagnostics so hostile
  canonical values cannot forge test log entries.
- ULID parsing now accepts consistently lowercase Laravel storage values while
  retaining uppercase canonical output and rejecting mixed-case input.
- The module workflow gate now validates the repository's single root workflow
  instead of requiring a forbidden package-local workflow.
- Refreshed the exported API fingerprint for the canonical Go 1.26
  documentation output, including explicit lowercase ULID output and
  concurrency-safe UUIDv4 entropy ownership.

### Added

- A deterministic `slug.LaravelEnglish` compatibility profile backed by the
  attributed voku/portable-ascii 2.1.1 replacement data used by Laravel 13.
- Typed domain identifiers without reflection or runtime registration.
- UUID v4/v7, ULID, TypeID 0.3.0, KSUID, and NanoID packages.
- Text, JSON, binary, SQL, and PostgreSQL UUID serialization.
- Injected clocks and entropy, deterministic test utilities, fuzzing,
  mutation, race, benchmark, documentation, and CI gates.
- Structured-log redaction plus exhaustive hostile-input, collision,
  cross-language, concurrency, PostgreSQL-codec, and leakage evidence.
