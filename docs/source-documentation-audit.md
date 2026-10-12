# Source documentation audit

## Scope and acceptance

Documentation-only Tier A pass owned by the source-documentation worker.
Base: `e0ae299e373f224d1f02ce9377f82e492e9065eb` (`origin/main`).
The module is `github.com/faustbrian/go-identifier/v2`; its eight packages are
`.`, `idtest`, `ksuid`, `nanoid`, `slug`, `typeid`, `ulid`, and `uuid`.
Consumers use typed-domain validation, family codecs, explicit generators,
and the frozen slug profile. No signatures or executable behavior may change.
Named risks are misleading nil/zero or decoder ownership contracts, overstated
concurrency/randomness guarantees, and drift from pinned format decisions.
Smallest checks: formatting, parsed/rendered package/API documentation,
comment-only token comparison, local links/markers/generated provenance,
changed-content secret inspection, and maintained API-documentation snapshots.
The two existing examples are inspected, not rerun: their executable contents
are unchanged.

## Reproduce the inventory

Run from the repository root:

```sh
git ls-files '*go.mod' '*.go'
cat modules.json packages.json
rg -n 'TODO|FIXME|HACK|NOTE|WARNING|Deprecated:|go:generate|Code generated|go:build' --glob '*.go'
rg '^func (Test|Fuzz|Benchmark|Example)' --glob '*_test.go'
```

One root module, eight packages, 42 tracked Go files. No vendored Go source,
platform-specific files, build tags, cgo, nested modules, or deprecated APIs.
Package comments exist but need responsibility/non-goal/entry-point detail.
All exported top-level declarations already have comments; `Clock.Now`,
`Generator.New`, `Validator.Validate`, `Inspection` fields, and `nanoid.Config`
fields lack their own semantic documentation. Codec comments often name only
the implemented interface; these need concrete error, zero and receiver rules.
Generators require blocking, shared-source and non-copying caveats. These are
missing documentation, not requests to alter implementation.

## Tracked Go file coverage

Every hand-maintained file is read in full against its assertions or callers;
the generated table is audited through its header, generator, source pin and
unchanged data instead of manually rewriting external replacement entries.

| File | Inspection and action |
| --- | --- |
| `benchmark_hardening_test.go` | Reviewed five workloads and all helpers; documented setup/copy/search-proxy limits. |
| `benchmark_test.go` | Reviewed two workloads and compile-only example; clarified comparison scope and stale example prose. |
| `concurrency_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `coverage_internal_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `cross_language_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `diagnostic_security_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `entropy_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `example_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `failure_domain_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `fuzz_test.go` | Reviewed six arbitrary-input boundaries and successful-round-trip invariants; documented. |
| `hostile_input_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `identifier.go` | Reviewed typed-tag bounds, callback/error contracts, all exports and inspection fields; repaired. |
| `identifier_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `idtest/idtest.go` | Reviewed source synchronization, seed ownership and assertion diagnostics; repaired. |
| `idtest/idtest_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `idtest/log_internal_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `immutability_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `interoperability_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `ksuid/coverage_internal_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `ksuid/ksuid.go` | Reviewed canonical Base62, epoch, assigned zero, payload carry and codecs; repaired. |
| `ksuid/ksuid_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `leakage_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `nanoid/coverage_internal_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `nanoid/nanoid.go` | Reviewed alphabet/size fields, entropy floor, configuration retention and bounded rejection; repaired. |
| `nanoid/nanoid_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `postgresql_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `privacy_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `slug/aaa_mutation_contract_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `slug/doc.go` | Reviewed canonical package comment and frozen profile responsibility; repaired. |
| `slug/slug.go` | Reviewed UTF-8 truncation, normalization order and longest-match generator ordering; repaired. |
| `slug/slug_test.go` | Reviewed frozen corpus, truncation/concurrency tests and fuzz invariants; fuzz rationale added. |
| `slug/transliteration_table.go` | Generated external table; header/source hash/license and generator ordering reviewed; no data edits. |
| `typeid/coverage_internal_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `typeid/official_spec_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `typeid/typeid.go` | Reviewed 0.3.0 grammar, unrestricted suffix, official zero, source sharing and codecs; repaired. |
| `typeid/typeid_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `ulid/coverage_internal_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `ulid/ulid.go` | Reviewed case policy, assigned zero, 80-bit carry and codecs; repaired. |
| `ulid/ulid_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `uuid/coverage_internal_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |
| `uuid/uuid.go` | Reviewed strict RFC parsing, value-copy ownership, pgx/SQL codecs and 74-bit carry; repaired. |
| `uuid/uuid_test.go` | Read complete behavioral assertions/helpers; names and outcomes already communicate purpose; no comment changes needed. |

## Investigation and marker disposition

- Generated header `scripts/generate-table.php` is correctly relative to
  `slug/`; the tracked generator is `slug/scripts/generate-table.php`.
  Regenerate from the portable-ascii 2.1.1 source pinned in
  [vector provenance](../specification/vector-provenance.tsv): from `slug/`,
  run `php scripts/generate-table.php --source=<pinned-data-dir>
  --output=transliteration_table.go`, then format and inspect the output.
  The source-map SHA-256, MIT attribution, revision and ordering agree with
  the generator. Generated source is unchanged; regeneration is not needed.
- No TODO, FIXME, HACK, NOTE, WARNING or Deprecated markers remain to resolve.
  The generated header is the sole policy marker; go:embed directives refer
  to existing official fixtures. There are no go:generate directives.
- RFC 9562, pinned ULID, TypeID 0.3.0, pinned Segment and Nano ID profiles,
  and frozen Laravel/portable-ascii versions are named next to interpretation
  code. The [decision register](specification-decisions.md) and tests establish
  rationale; no new normative policy was inferred from peers.
- All seven fuzz targets now explain their hostile-input/round-trip boundary;
  all seven benchmarks explain workload and excluded setup. The locality proxy
  is explicitly not a database performance result. Both examples were read;
  executable contents were not changed. One compile-only example's misleading
  domain-validation comment was corrected.
- No persistence/crash-recovery implementation exists. SQL/pgx methods are
  serverless codecs, not database uniqueness or transaction owners. Mutable
  generator state and borrowed callbacks are documented; no goroutines or
  resources are started/closed. No performance claims were added.
- Two existing claims in [guarantees](guarantees.md) contradicted source:
  NanoID permits 94, not 255, unique printable ASCII bytes, and default UUIDv4
  reads crypto/rand directly rather than through a generator mutex. Corrected
  only those claims and the associated per-instance monotonic-domain wording.
- Unresolved source-rationale questions: none. No specification or runtime
  correction was attempted.

## Verification and disposition

- Go 1.27.2 (the repository toolchain pin) formatted every changed Go file.
  `git diff --check` passed.
- A task-owned standard-library AST/scanner check parsed all 42 tracked files,
  validated name-led comments for 175 public declarations, fields and interface
  methods plus eight canonical package comments, and compared comment-free Go
  tokens against the base: no executable or signature difference.
- `go doc -all` rendered `.`, `./idtest`, `./ksuid`, `./nanoid`, `./slug`,
  `./typeid`, `./ulid`, and `./uuid`; contracts were inspected in that output.
- Inventory/file-row equality, local Markdown links, changed-content private-key
  and credential-pattern inspection, and generated-table/generator immutability
  checks passed. `php -l slug/scripts/generate-table.php` passed. External
  specifications were resolved from the maintained pins/register, not refreshed.
- The first disposable AST checker incorrectly included exported methods on
  private receiver types; its applicability was corrected and the complete
  check passed. Automatic cache cleanup encountered read-only downloaded module
  files; exact task-owned permissions were restored and deletion verified.
- No runtime, example, race, fuzz, mutation, benchmark or broad static campaign
  was run: executable source and examples are unchanged. All task-owned
  GOCACHE/GOMODCACHE/GOTMPDIR and transient checker files were removed.

- Regenerated `api/v2/baseline.txt` using `.golib.yaml` package order:
  `.`, `./uuid`, `./ulid`, `./typeid`, `./ksuid`, `./nanoid`, `./slug`, `./idtest`.
  Concatenated native `go doc -all <package>` stdout, trimmed trailing newline
  bytes and appended one newline, matching the shared verifier algorithm.
  A second generation was byte-identical. Exported type/function signature
  lines were identical to the original snapshot; only documentation changed.
- After the last source comments, repeated formatting and the full AST/comment
  and token comparison passed. Re-inspected affected UUID/NanoID/slug rendered
  output. The complete final diff's ordinary Tier A review found no unresolved
  contract, duplication, scope or behavior findings.

This report does not establish hosted CI, remote-main delivery or release
qualification. No new recurring checker was added in this package; shared
objective enforcement belongs to the coordinated go-library-tools pass.

## Objective enforcement adoption

The local `source-docs-check` target is required by ordinary/full checks and CI.
Shared CI uses `source_documentation: check` with the immutable Tools v2.4.0
workflow and matching authenticated binary/checksum declaration. Structural
checks do not replace this maintained semantic audit or native docs/examples.
