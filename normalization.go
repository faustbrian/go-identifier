package identifier

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"unicode/utf8"
)

// ErrNormalization reports an unusable transformed value, including a prohibited
// domain, empty result or exceeded output bound. It never includes input text.
var ErrNormalization = errors.New("identifier normalization failed")

// Input is enumeration-sensitive, unnormalized UTF-8 text. NewInput validates
// 1..4096 bytes. Direct conversions are revalidated by Normalize. Ordinary
// formatting/logging is redacted; explicit string conversion exposes the input.
type Input string

// NormalizationProfileID identifies a migration-owned policy: 1..128 ASCII
// bytes matching [a-z0-9][a-z0-9._-]*. An ID does not authenticate its policy.
type NormalizationProfileID string

// UnicodeNormalization explicitly selects a fixed Unicode 15.1.0 form.
type UnicodeNormalization uint8

const (
	// UnicodeNormalizationUnspecified is invalid; there is no implicit form.
	UnicodeNormalizationUnspecified UnicodeNormalization = iota
	// UnicodeNormalizationNone preserves Unicode distinctions.
	UnicodeNormalizationNone
	// UnicodeNormalizationNFC selects canonical decomposition and composition.
	UnicodeNormalizationNFC
	// UnicodeNormalizationNFKC additionally removes compatibility distinctions.
	UnicodeNormalizationNFKC
)

// NormalizationProfileSpec is copied on construction. Version must be positive;
// MaximumBytes bounds the final result to 1..4096 bytes. CaseFold means default
// non-Turkic full Unicode 15.1 folding, not locale-sensitive lowercasing.
// TrimSpace removes only leading/trailing Unicode 15.1 White_Space. IDNA treats
// the whole value as a domain, using nontransitional UTS #46, STD3, ContextJ,
// Bidi, DNS length and hyphen checks; it is not an email-address parser.
type NormalizationProfileSpec struct {
	// ID names the caller-owned migration policy, not an authentication fact.
	ID NormalizationProfileID
	// Version is the positive policy revision persisted with canonical bytes.
	Version uint32
	// Unicode explicitly selects None, NFC or NFKC using fixed 15.1.0 tables.
	Unicode UnicodeNormalization
	// CaseFold applies default full non-Turkic folding between normalization passes.
	CaseFold bool
	// TrimSpace removes only leading and trailing Unicode 15.1 White_Space.
	TrimSpace bool
	// IDNA selects whole-domain nontransitional UTS #46, not email parsing.
	IDNA bool
	// MaximumBytes bounds the final nonempty UTF-8 result to 1..4096 bytes.
	MaximumBytes uint32
}

// NormalizationProfile is an immutable, concurrently reusable policy. Its zero
// value is invalid. Callers must change ID/version and migrate collision-aware
// when changing policy; there is no global registry or implicit reinterpretation.
type NormalizationProfile struct{ spec NormalizationProfileSpec }

// Canonical is immutable lookup identity, bound to exact profile ID/version.
// It is not a display value or authentication evidence. Zero is invalid.
type Canonical struct {
	value          string
	profileID      NormalizationProfileID
	profileVersion uint32
}

// CanonicalRecord is the deliberately sensitive, complete persistence/wire
// projection. Preserve every field; restoration requires the selected policy.
// Do not log its fields or serialized bytes.
type CanonicalRecord struct {
	// Value contains sensitive canonical lookup bytes, not display text.
	Value string
	// ProfileID retains the exact caller-owned policy identity.
	ProfileID NormalizationProfileID
	// ProfileVersion retains the policy revision used to produce Value.
	ProfileVersion uint32
}

// Reference carries identifier identity and optional display text, never
// authentication/authorization evidence. NewReference validates it. Exported
// fields permit caller construction; such values are not validated by the type.
type Reference struct {
	// Kind identifies the caller's identifier category using bounded ASCII.
	Kind string
	// Canonical binds lookup identity to its normalization profile and version.
	Canonical Canonical
	// Display is optional sensitive UTF-8 presentation text, at most 4096 bytes.
	Display string
}

// NewInput validates byte length/encoding without trimming or normalizing.
func NewInput(text string) (Input, error) {
	if !validIdentityText(text, 4096) {
		return "", ErrInvalid
	}
	return Input(strings.Clone(text)), nil
}

// NewNormalizationProfile copies a bounded explicit policy. Invalid bounds,
// identity/version or unspecified form return ErrInvalid; unknown forms return
// ErrUnsupported. No registry, toolchain table selection or hidden defaults apply.
func NewNormalizationProfile(spec NormalizationProfileSpec) (NormalizationProfile, error) {
	if !identityASCII(string(spec.ID), 128) || spec.Version == 0 || spec.MaximumBytes < 1 || spec.MaximumBytes > 4096 || spec.Unicode == UnicodeNormalizationUnspecified {
		return NormalizationProfile{}, ErrInvalid
	}
	if spec.Unicode > UnicodeNormalizationNFKC {
		return NormalizationProfile{}, ErrUnsupported
	}
	spec.ID = NormalizationProfileID(strings.Clone(string(spec.ID)))
	return NormalizationProfile{spec: spec}, nil
}

// Normalize revalidates Input and applies form, full fold, the selected form
// again, edge whitespace trimming, then optional domain processing. Domain
// processing always applies its mandated NFC independently of the selected form.
// The final nonempty UTF-8 result obeys MaximumBytes. Work and intermediate
// storage are bounded; long combining runs are ordered, not rewritten with CGJ.
// ErrInvalid denotes invalid input/profile; ErrNormalization denotes unusable
// transformed output. Errors and ordinary diagnostics are redacted.
func Normalize(input Input, profile NormalizationProfile) (Canonical, error) {
	if !validIdentityText(string(input), 4096) {
		return Canonical{}, ErrInvalid
	}
	if _, err := NewNormalizationProfile(profile.spec); err != nil {
		return Canonical{}, err
	}
	s, err := unicodeNormalize(string(input), profile.spec.Unicode)
	if err != nil {
		return Canonical{}, err
	}
	if profile.spec.CaseFold {
		s, err = unicodeFold(s)
		if err == nil {
			s, err = unicodeNormalize(s, profile.spec.Unicode)
		}
		if err != nil {
			return Canonical{}, err
		}
	}
	if profile.spec.TrimSpace {
		s = strings.TrimFunc(s, unicodeSpace)
	}
	if profile.spec.IDNA {
		s, err = unicodeDomain(s)
		if err != nil {
			return Canonical{}, ErrNormalization
		}
	}
	if !validIdentityText(s, int(profile.spec.MaximumBytes)) {
		return Canonical{}, ErrNormalization
	}
	return Canonical{strings.Clone(s), profile.spec.ID, profile.spec.Version}, nil
}

// RestoreCanonical requires the exact profile binding and proves Value is a
// fixed point. Unknown/missing/mismatched bindings or forged bytes return
// ErrInvalid. A record is not normalization or ownership proof by itself.
func RestoreCanonical(record CanonicalRecord, profile NormalizationProfile) (Canonical, error) {
	if record.ProfileID != profile.spec.ID || record.ProfileVersion != profile.spec.Version {
		return Canonical{}, ErrInvalid
	}
	value, err := Normalize(Input(record.Value), profile)
	if err != nil || value.value != record.Value {
		return Canonical{}, ErrInvalid
	}
	return value, nil
}

// Record returns the complete explicit sensitive projection, or zero for zero.
func (c Canonical) Record() CanonicalRecord {
	return CanonicalRecord{c.value, c.profileID, c.profileVersion}
}

// Equal compares all identity fields. Invalid zero values never compare equal,
// including to another zero; equality says nothing about account ownership.
func (c Canonical) Equal(other Canonical) bool {
	return c.valid() && c == other
}

func (c Canonical) valid() bool {
	return c.value != "" && c.profileID != "" && c.profileVersion != 0
}

// NewReference validates a nonzero canonical, a 1..64 canonical ASCII kind
// using the profile-ID grammar, and optional display of at most4096 UTF-8 bytes.
func NewReference(kind string, canonical Canonical, display string) (Reference, error) {
	if !identityASCII(kind, 64) || !canonical.valid() || len(display) > 4096 || !utf8.ValidString(display) {
		return Reference{}, ErrInvalid
	}
	return Reference{strings.Clone(kind), canonical, strings.Clone(display)}, nil
}

func validIdentityText(s string, maximumBytes int) bool {
	return len(s) > 0 && len(s) <= maximumBytes && utf8.ValidString(s)
}
func identityASCII(s string, maximumBytes int) bool {
	if len(s) == 0 || len(s) > maximumBytes {
		return false
	}
	for i, b := range []byte(s) {
		alnum := b >= 'a' && b <= 'z' || b >= '0' && b <= '9'
		if !alnum && (i == 0 || b != '.' && b != '_' && b != '-') {
			return false
		}
	}
	return true
}

func redactIdentity(s fmt.State) { _, _ = io.WriteString(s, "identifier(redacted)") }

// Format redacts Input for every formatting verb, including quoted and Go syntax.
func (Input) Format(s fmt.State, _ rune) { redactIdentity(s) }

// Format redacts Canonical; use Record only at an intentional sensitive boundary.
func (Canonical) Format(s fmt.State, _ rune) { redactIdentity(s) }

// Format redacts the record even though its exported fields remain sensitive.
func (CanonicalRecord) Format(s fmt.State, _ rune) { redactIdentity(s) }

// Format redacts both canonical identity and display text for every verb.
func (Reference) Format(s fmt.State, _ rune) { redactIdentity(s) }

// LogValue prevents ordinary structured logging from exposing Input.
func (Input) LogValue() slog.Value { return slog.StringValue("identifier(redacted)") }

// LogValue prevents ordinary structured logging from exposing Canonical.
func (Canonical) LogValue() slog.Value { return slog.StringValue("identifier(redacted)") }

// LogValue redacts the record; separately logging its fields is not protected.
func (CanonicalRecord) LogValue() slog.Value { return slog.StringValue("identifier(redacted)") }

// LogValue prevents ordinary structured logging from exposing Reference fields.
func (Reference) LogValue() slog.Value { return slog.StringValue("identifier(redacted)") }

// MarshalJSON refuses implicit Input disclosure with ErrUnsupported.
func (Input) MarshalJSON() ([]byte, error) { return nil, ErrUnsupported }

// MarshalText refuses implicit Input disclosure with ErrUnsupported.
func (Input) MarshalText() ([]byte, error) { return nil, ErrUnsupported }

// MarshalJSON refuses implicit disclosure; serialize the explicit Record instead.
func (Canonical) MarshalJSON() ([]byte, error) { return nil, ErrUnsupported }

// MarshalText refuses implicit disclosure; Record retains required policy metadata.
func (Canonical) MarshalText() ([]byte, error) { return nil, ErrUnsupported }

// MarshalJSON refuses implicit canonical and display disclosure with ErrUnsupported.
func (Reference) MarshalJSON() ([]byte, error) { return nil, ErrUnsupported }

// MarshalText refuses implicit canonical and display disclosure with ErrUnsupported.
func (Reference) MarshalText() ([]byte, error) { return nil, ErrUnsupported }
