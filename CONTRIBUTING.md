# Contributing

## Before Editing

1. Read [`AGENTS.md`](AGENTS.md) and the affected package's docs.
2. Run `go test ./...` or the affected package's narrower tests.
3. Identify owned dependencies and reverse dependants in `modules.json`.
4. Preserve unrelated work and generated/corpus provenance.

## Changes

Keep commits focused and conventional. Update every affected changelog with
the behavior and migration impact. Public API changes require compatibility
evidence and documentation. Specification behavior requires a decision record,
fixture coverage, and interoperability evidence.

New direct dependencies and dependency updates must follow the
[dependency governance policy](AGENTS.md#dependencies-and-supply-chain). Package-local
update bots are forbidden; the root policy owns every module and action update.

Specification-backed changes must follow the
[specification governance contract](AGENTS.md#design), update
the affected entries in the
[specification decision register](docs/specification-decisions.md), and complete
the Specification Decisions section of the pull request template. An unresolved
interpretation or stale source pin is release-blocking; peer behavior cannot
silently select policy.

Mutation is optional local or scheduled assurance, not a required PR status.
Confirmed behavioral and security defects still require a focused regression
and repair; equivalent/invalid mutants retain reviewed narrow dispositions.

Do not add package-local workflows, permanent replacements, machine-specific
paths, bypass flags, broad mutation exclusions, or aggregate quality metrics
that hide a failing package.

## Verification

Standalone `go test ./...` runs deterministic tests, fuzz seeds and saved corpus
regressions without sibling checkouts or ecosystem ledgers. The manifest declares
no required services or test tags; PostgreSQL codec tests are serverless.

With the checksum-pinned public verifier selected by `.golib.yaml`, use:

```bash
make inventory
make check
make ci
```

`check` and `ci` use bounded local checks. One named root race test observes
atomic shared-generator state and exact unique result counts across six
families, once with a two-minute timeout; existing generation bounds remain
unchanged. Ordinary CI consumes the public verifier rather than source-building
it. `ci` also validates the offline specification register/history; online
authority refresh remains explicit full assurance.

Select one target for short active fuzz exploration. Optional local mutation
selects one package; neither command is an all-change prerequisite:

```bash
GOMAXPROCS=2 go test -run '^$' -fuzz '^FuzzUUIDParse$' -fuzztime=2s -parallel=1 -timeout=2m .
golib mutation --module . --package ./uuid
```

Parser, shared-generator, API and specification changes retain their relevant
behavioral, race, compatibility and direct-consumer checks. Preserve the pinned
official corpora, cross-language vectors and transliteration provenance; peer
behavior does not silently choose specification policy.

`make check-full` retains the complete full-assurance contract, all seven native
fuzz budgets, conformance, provenance and benchmarks, and refreshes authority
digests through `golib specification check --online`. Relevant release
qualification remains `golib release dry-run --module .`. Select full/release
checks for their material risk or release boundary, not every commit. Report
unavailable, skipped and failed boundaries honestly; a local pass is not online
authority, hosted-CI or public-release qualification.

## Adding A Module

Follow [repository structure policy](AGENTS.md#repository-structure). New modules
require an explicit purpose, ownership boundary, dependency review, package
catalog entry, full quality gates, documentation, changelog, license, security
policy, compatibility plan, and release dry-run.
