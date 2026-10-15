# Versioned identity normalization

Identity lookup policy is separate from UUID, ULID, TypeID, KSUID, NanoID and
slug generation. These existing families are unchanged. The root package adds
immutable canonical values bound to a caller-owned profile ID and version.
Go 1.27 is the minimum; newer toolchains do not select newer Unicode tables.

## Quick start and API

```go
profile, err := identifier.NewNormalizationProfile(identifier.NormalizationProfileSpec{
    ID: "account.username", Version: 1,
    Unicode: identifier.UnicodeNormalizationNFKC,
    CaseFold: true, TrimSpace: true, MaximumBytes: 128,
})
if err != nil { return err }
input, err := identifier.NewInput(text)
if err != nil { return err }
canonical, err := identifier.Normalize(input, profile)
if err != nil { return err }
record := canonical.Record() // sensitive persistence projection
restored, err := identifier.RestoreCanonical(record, profile)
if err != nil { return err }
_ = restored.Equal(canonical)
```

The complete runnable example is `ExampleNormalize` in
`normalization_example_test.go`. `NewInput` accepts 1–4096 valid UTF-8 bytes;
`Normalize` revalidates directly constructed inputs. A profile explicitly
selects None, NFC or NFKC, optional default full case folding, edge whitespace
trimming, whole-domain IDNA, and a final 1–4096 byte limit. ID and version are
required: IDs match `[a-z0-9][a-z0-9._-]*` in at most 128 bytes; version is
positive. The constructor copies the specification. A zero profile is invalid.

Processing order is the selected Unicode form, optional full non-Turkic fold,
the form again, optional Unicode White_Space edge trimming, then optional
non-transitional UTS #46 domain processing. Tables are fixed to Unicode 15.1.0.
IDNA applies mandatory NFC, STD3, hyphen, ContextJ, Bidi and DNS length checks.
One trailing DNS root dot is preserved; it does not count toward the 253-byte
domain limit. IDNA is not an email parser: pass only a separately parsed domain.
Long combining runs are normalized without inserting CGJ characters.

`ErrInvalid` reports invalid inputs, policies, references or restored records;
`ErrUnsupported` reports unknown forms and forbidden implicit serialization;
`ErrNormalization` reports prohibited, empty or oversized transformed values.
Errors contain no input. Failures return zero values. `NewReference` adds a
validated 1–64 byte kind using the same ASCII grammar and optional UTF-8 display
text of at most 4096 bytes. Exported Reference fields allow caller construction;
that construction is not validation or authorization.

## Persistence, adoption and migration

Store the complete `CanonicalRecord`: Value, ProfileID and ProfileVersion.
Scope uniqueness by tenant and identifier kind as appropriate. Restoration
requires the exact selected policy binding and rechecks that Value is its fixed
point. Never reinterpret a stored value under another version. Equality compares
all three fields; invalid zero values are never equal, even to one another.

Profile identity is application-owned, not a global registry: using the same
ID/version for different specifications violates the caller contract. Persist
or deploy the specification with your migration configuration. Updating a
policy requires a new version or ID, collision analysis on existing records,
explicit conflict resolution, then a transactional uniqueness/index migration.
For example, an exact-text policy distinguishes `Straße` and `STRASSE`; enabling
full folding maps both to `strasse`. Do not overwrite one account or choose the
first match. Keep old and new versions distinct during migration; mixed-version
equality deliberately fails. The package does not own repositories, migration
transactions, collision decisions or account linking.

## Security, operations and tradeoffs

Input, Canonical, CanonicalRecord and Reference redact ordinary fmt and slog
diagnostics. Input, Canonical and Reference refuse JSON/text serialization.
Explicit string conversion of Input and record fields/JSON remain sensitive:
redaction cannot protect separately projected bytes. Keep them out of telemetry.
Canonicalization proves neither ownership nor authentication, uniqueness,
confusable safety, permitted scripts, reserved names or account availability.
Passwords must not use identifier normalization.

Profiles are immutable and concurrently reusable; there is no registry,
background work, callback, IO or mutable per-request shared state. Inputs,
intermediate expansion and domain processing are bounded. NFKC and full folding
deliberately lose distinctions; None preserves them. Unicode 15.1 stability
means later Unicode characters do not acquire later normalization/folding or
IDNA validity until an explicit data/profile migration.

## FAQ and data updates

- **Email/phone support?** Higher-level packages own local-part semantics and
  E.164 parsing; this package accepts text or one domain, not a whole address.
- **Why not host Unicode?** Toolchain upgrades must not change stored identities.
- **Can a record authenticate a user?** No; it is a caller-created persistence value
  restored only for canonical lookup, never an ownership proof.
- **How are tables updated?** See `testdata/unicode151/provenance.json` and
  `scripts/generate-unicode.rb`. Official sources, checksums and licenses are
  retained; regeneration is offline and deterministic. A new Unicode version
  requires new provenance, corpus verification and an explicit profile migration.

The official normalization, full folding and IDNA corpora are retained as
tests. They do not claim downstream repository migrations or live provider
interoperability. See [verification](verification.md) for release requirements
and [the changelog](../CHANGELOG.md) for adoption changes.
