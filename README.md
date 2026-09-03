# identifier

[![CI](https://github.com/faustbrian/go-identifier/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/faustbrian/go-identifier/actions/workflows/ci.yml)
[![CodeQL](https://img.shields.io/badge/CodeQL-required-blue)](https://github.com/faustbrian/go-identifier/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Mutation](https://img.shields.io/badge/mutation-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Documentation](https://img.shields.io/badge/docs-checked_in_CI-blue)](docs/)
[![Go Reference](https://pkg.go.dev/badge/github.com/faustbrian/go-identifier.svg)](https://pkg.go.dev/github.com/faustbrian/go-identifier)
[![Release](https://img.shields.io/github/v/release/faustbrian/go-identifier?sort=semver)](https://github.com/faustbrian/go-identifier/releases)
[![Go](https://img.shields.io/badge/go-1.26.6-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

`identifier` provides strict, immutable UUID, ULID, TypeID, KSUID, and
NanoID values, compile-time domain wrappers, and explicit compatibility
profiles for derived public identifiers. Each generated family keeps its own
clock, entropy, ordering, leakage, and persistence contract; an identifier is
never treated as a secret, authorization fact, idempotency proof, or tracing
context merely because it is unique.

The minimum toolchain is Go 1.26.6. Random generators use `crypto/rand` by
default and own all mutable state. Tests can inject deterministic clocks and
entropy through `idtest`.

## Choose a family

| Family | Best fit | Ordering | Exposed time | Random strength |
| --- | --- | --- | --- | --- |
| UUIDv4 | Standard opaque database/API ID | None | None | 122 random bits |
| UUIDv7 | Standard time-local database key | Millisecond, monotonic per generator | Millisecond | 74 bits initially |
| ULID | Existing 26-byte text schemas | Millisecond, monotonic per generator | Millisecond | 80 bits initially |
| TypeID | Human-visible typed UUID values | Prefix then UUIDv7 | Millisecond | UUIDv7 contract |
| KSUID | Existing Segment-compatible values | Second, monotonic per generator | Second | 128 bits initially |
| NanoID | Compact URL-safe random text | None | None | At least 120 configured bits |

Read [selection guidance](docs/selection.md) before choosing. Sortable
generators reveal creation time and may reveal local issuance order.

## Quick start

```go
clock := identifier.ClockFunc(time.Now)
generator := uuid.NewV7Generator(clock, nil)
id, err := generator.New()
if err != nil {
    return err
}
fmt.Println(id.String())
```

There is no package-global generator. Keep one generator per ownership and
failure domain, and share that instance only when its monotonic sequence should
also be shared.

The `slug` package is intentionally narrower than a general transliteration
library. `slug.LaravelEnglish` reproduces the frozen Laravel 13 and
spatie/laravel-sluggable 4.0.2 English profile needed when an existing public
slug contract must survive a service rewrite. Database uniqueness and suffix
selection remain application persistence concerns.

## Contracts

- [Selection](docs/selection.md)
- [Guarantees and leakage](docs/guarantees.md)
- [Verification](docs/verification.md)
- [Serialization](docs/serialization.md)
- [Database behavior](docs/database.md)
- [Migration](docs/migration.md)
- [Security](docs/security.md)
- [Performance](docs/performance.md)
- [Compatibility](docs/compatibility.md)
- [Specification decisions](docs/specification-decisions.md)
- [API map](docs/api.md)
- [Architecture](docs/architecture.md)
- [FAQ](docs/faq.md)

For neighboring packages and composition guidance, see the versioned
[Golib ecosystem index](https://github.com/faustbrian/go-library-tools/blob/v1.4.0/docs/ecosystem/README.md)
and its
[Foundations family](https://github.com/faustbrian/go-library-tools/blob/v1.4.0/docs/ecosystem/design-language.md#package-families-and-selection).

## Development

Install the `golib` version declared by `.golib.yaml`, then run `make ci`.
That target includes repository, cohesion, online specification, and module
checks. Fuzzing, race tests, mutation tests, API baselines, documentation
links, security scans, provenance checks, and comparative benchmarks are part
of that repository contract.

Run `make cohesion` to validate the repository's design-language metadata and
versioned ecosystem navigation.

## License

MIT. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
