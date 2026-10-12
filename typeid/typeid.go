// Package typeid implements TypeID specification 0.3.0 pinned at revision
// be8ff0daf5dc1f6d40c62a03cfc89945263a69af. Start with Parse, FromUUID, or
// NewGenerator. Parsing accepts every 128-bit suffix; generation uses UUIDv7.
// The package requires no service or cgo and starts no background work.
//
// Prefixes are application labels, not authorization boundaries. Values may
// reveal creation time, and generators do not coordinate distributed uniqueness.
// Parsing/zero semantics and generation are IDENTIFIER-TYPEID-DEC-001/002 in
// docs/specification-decisions.md.
package typeid

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	identifier "github.com/faustbrian/go-identifier/v2"
	identifieruuid "github.com/faustbrian/go-identifier/v2/uuid"
	oklogulid "github.com/oklog/ulid/v2"
)

const (
	suffixAlphabet    = "0123456789abcdefghjkmnpqrstvwxyz"
	zeroSuffix        = "00000000000000000000000000"
	minimumTextLength = 26
	maximumTextLength = 90
	maximumJSONBytes  = maximumTextLength*6 + 2
)

// ID is an immutable TypeID. Its Go zero value is the official unprefixed
// all-zero TypeID; the validity bit preserves every other prefix/value pair.
// Read-only methods may run concurrently. Decode methods require a non-nil
// receiver and exclusive access; they leave it unchanged on invalid input.
type ID struct {
	value  [16]byte
	prefix string
	valid  bool
}

// ValidatePrefix accepts empty text or up to 63 lowercase ASCII letters and
// underscores, with a letter at each end. Consecutive interior underscores are
// valid. Invalid input wraps identifier.ErrInvalid.
func ValidatePrefix(prefix string) error {
	if len(prefix) > 63 {
		return fmt.Errorf("%w: TypeID prefix exceeds 63 characters", identifier.ErrInvalid)
	}
	if prefix == "" {
		return nil
	}
	if prefix[0] < 'a' || prefix[0] > 'z' || prefix[len(prefix)-1] < 'a' || prefix[len(prefix)-1] > 'z' {
		return fmt.Errorf("%w: TypeID prefix must start and end with a letter", identifier.ErrInvalid)
	}
	for index := range len(prefix) {
		character := prefix[index]
		if character != '_' && (character < 'a' || character > 'z') {
			return fmt.Errorf("%w: TypeID prefix contains invalid ASCII", identifier.ErrInvalid)
		}
	}

	return nil
}

// Parse accepts only canonical TypeID 0.3.0 text. Parsing permits every
// 128-bit suffix required by the official vectors; generation remains UUIDv7.
// Text is 26 bytes, or a nonempty prefix plus '_' plus the 26-byte suffix.
// Failure returns the zero ID with identifier.ErrInvalid.
// The pinned specification's Type Prefix/Base32 Encoding and official
// valid.json/invalid.json corpora protect IDENTIFIER-TYPEID-DEC-001;
// TestOfficialValidCorpus and TestOfficialInvalidCorpus exercise that decision.
func Parse(text string) (ID, error) {
	if len(text) < minimumTextLength || len(text) > maximumTextLength {
		return ID{}, fmt.Errorf("%w: TypeID text length is invalid", identifier.ErrInvalid)
	}

	prefix := ""
	suffix := text
	if len(text) > minimumTextLength {
		separator := len(text) - minimumTextLength - 1
		if text[separator] != '_' {
			return ID{}, fmt.Errorf("%w: TypeID separator is missing", identifier.ErrInvalid)
		}
		prefix = text[:separator]
		if prefix == "" {
			return ID{}, fmt.Errorf("%w: empty TypeID prefix must omit separator", identifier.ErrInvalid)
		}
		suffix = text[separator+1:]
	}
	if err := ValidatePrefix(prefix); err != nil {
		return ID{}, err
	}
	if suffix[0] > '7' {
		return ID{}, fmt.Errorf("%w: TypeID suffix exceeds 128 bits", identifier.ErrInvalid)
	}
	for index := range len(suffix) {
		if !strings.ContainsRune(suffixAlphabet, rune(suffix[index])) {
			return ID{}, fmt.Errorf("%w: TypeID suffix is not canonical Base32", identifier.ErrInvalid)
		}
	}

	decoded, _ := oklogulid.ParseStrict(strings.ToUpper(suffix))

	return FromBytes(prefix, [16]byte(decoded))
}

// FromBytes validates prefix and copies any 128-bit suffix. Empty prefix with
// all-zero bytes yields the Go zero ID. Invalid prefix returns the zero ID with
// identifier.ErrInvalid; UUID version and variant are deliberately unrestricted.
func FromBytes(prefix string, value [16]byte) (ID, error) {
	if err := ValidatePrefix(prefix); err != nil {
		return ID{}, err
	}
	if prefix == "" && value == ([16]byte{}) {
		return ID{}, nil
	}

	return ID{value: value, prefix: prefix, valid: true}, nil
}

// UUIDSource constrains FromUUID to an exact string or identifieruuid.ID type.
// Array values are copied as supplied; this constraint does not establish validity.
type UUIDSource interface {
	string | identifieruuid.ID
}

// FromUUID encodes any 128-bit UUID value with a validated prefix. String
// input must be lowercase hyphenated hex but permits every version and variant,
// including all-zero values. Array input is copied without UUID validation.
// Invalid prefix or text returns the zero ID with identifier.ErrInvalid.
func FromUUID[T UUIDSource](prefix string, value T) (ID, error) {
	var bytes [16]byte
	switch typed := any(value).(type) {
	case string:
		parsed, err := parseUUIDText(typed)
		if err != nil {
			return ID{}, err
		}
		bytes = parsed
	case identifieruuid.ID:
		bytes = typed.Bytes()
	}

	return FromBytes(prefix, bytes)
}

func parseUUIDText(text string) ([16]byte, error) {
	if len(text) != 36 || text[8] != '-' || text[13] != '-' || text[18] != '-' || text[23] != '-' {
		return [16]byte{}, fmt.Errorf("%w: UUID must use canonical 8-4-4-4-12 form", identifier.ErrInvalid)
	}

	var compact [32]byte
	index := 0
	for position := range len(text) {
		if position == 8 || position == 13 || position == 18 || position == 23 {
			continue
		}
		character := text[position]
		if character < '0' || character > '9' && character < 'a' || character > 'f' {
			return [16]byte{}, fmt.Errorf("%w: UUID contains non-canonical hexadecimal", identifier.ErrInvalid)
		}
		compact[index] = character
		index++
	}

	var value [16]byte
	_, _ = hex.Decode(value[:], compact[:])

	return value, nil
}

// Bytes returns a copy of the 128-bit suffix value.
func (id ID) Bytes() [16]byte { return id.value }

// Prefix returns the validated type prefix.
func (id ID) Prefix() string { return id.prefix }

// IsZero reports whether the ID is the unprefixed all-zero TypeID.
func (id ID) IsZero() bool { return !id.valid }

// Compare orders prefixes lexically and suffixes by unsigned bytes.
func (id ID) Compare(other ID) int {
	if compared := strings.Compare(id.prefix, other.prefix); compared != 0 {
		return compared
	}

	return bytes.Compare(id.value[:], other.value[:])
}

// String returns canonical lowercase text, including the canonical zero ID.
func (id ID) String() string {
	if !id.valid {
		return zeroSuffix
	}

	suffix := strings.ToLower(oklogulid.ULID(id.value).String())
	if id.prefix == "" {
		return suffix
	}

	return id.prefix + "_" + suffix
}

// LogValue redacts the TypeID from structured logs.
func (id ID) LogValue() slog.Value { return slog.StringValue("[REDACTED]") }

// Inspect reports the suffix version nibble and a UTC millisecond timestamp
// only for assigned RFC-variant UUIDv7 suffixes. Sortable is true for assigned
// IDs even when their unrestricted suffix does not expose a timestamp.
func (id ID) Inspect() identifier.Inspection {
	inspection := identifier.Inspection{
		Family:   identifier.FamilyTypeID,
		Version:  int(id.value[6] >> 4),
		Sortable: id.valid,
	}
	if id.valid && inspection.Version == 7 && id.value[8]&0xc0 == 0x80 {
		milliseconds := int64(id.value[0])<<40 | int64(id.value[1])<<32 |
			int64(id.value[2])<<24 | int64(id.value[3])<<16 |
			int64(id.value[4])<<8 | int64(id.value[5])
		inspection.Timestamp = time.UnixMilli(milliseconds).UTC()
		inspection.HasTime = true
	}

	return inspection
}

// MarshalText returns newly allocated canonical text, including the zero TypeID.
func (id ID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

// UnmarshalText parses text and replaces id only on success. Invalid input
// wraps identifier.ErrInvalid; input bytes are not retained.
func (id *ID) UnmarshalText(text []byte) error {
	if len(text) < minimumTextLength || len(text) > maximumTextLength {
		return fmt.Errorf("%w: TypeID text length is invalid", identifier.ErrInvalid)
	}

	parsed, err := Parse(string(text))
	if err != nil {
		return err
	}
	*id = parsed

	return nil
}

// MarshalBinary encodes canonical text because the prefix is part of a TypeID.
func (id ID) MarshalBinary() ([]byte, error) { return id.MarshalText() }

// UnmarshalBinary uses UnmarshalText; raw UUID bytes alone cannot preserve a prefix.
func (id *ID) UnmarshalBinary(data []byte) error { return id.UnmarshalText(data) }

// MarshalJSON encodes canonical text, including the canonical zero TypeID.
func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

// UnmarshalJSON parses a bounded JSON string or maps the exact null token to
// the zero TypeID. Invalid input wraps identifier.ErrInvalid and leaves id unchanged.
// Encoded input is limited to 542 bytes, including quotes and escapes.
func (id *ID) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*id = ID{}

		return nil
	}
	if len(data) > maximumJSONBytes {
		return fmt.Errorf("%w: TypeID JSON length is invalid", identifier.ErrInvalid)
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("%w: malformed TypeID JSON", identifier.ErrInvalid)
	}

	parsed, err := Parse(text)
	if err != nil {
		return err
	}
	*id = parsed

	return nil
}

// Value returns canonical text for database/sql, including the valid zero TypeID.
// Unlike an unassigned UUID/ULID/KSUID/NanoID, it never returns SQL NULL.
func (id ID) Value() (driver.Value, error) {
	return id.String(), nil
}

// Scan accepts strings and text byte slices through Parse; nil maps to the zero
// TypeID. Invalid sources wrap identifier.ErrInvalid and leave id unchanged.
// Accepted bytes are not retained.
func (id *ID) Scan(src any) error {
	switch value := src.(type) {
	case nil:
		*id = ID{}

		return nil
	case string:
		parsed, err := Parse(value)
		if err != nil {
			return err
		}
		*id = parsed

		return nil
	case []byte:
		return id.UnmarshalText(value)
	default:
		return fmt.Errorf("%w: unsupported TypeID scan source", identifier.ErrInvalid)
	}
}

// Generator binds a prefix to a UUIDv7 generator. Concurrent New calls share
// the source's serialized sequence. Its zero value is not ready for use;
// construct it with NewGenerator. Ordering is per prefix and source, not global.
type Generator struct {
	prefix string
	uuid   *identifieruuid.V7Generator
}

// NewGenerator validates prefix and retains generator without copying it.
// Nil selects a fresh default UUIDv7 generator. A source shared with other
// callers remains one sequence; invalid prefix returns nil with identifier.ErrInvalid.
func NewGenerator(prefix string, generator *identifieruuid.V7Generator) (*Generator, error) {
	if err := ValidatePrefix(prefix); err != nil {
		return nil, err
	}
	if generator == nil {
		generator = identifieruuid.NewV7Generator(nil, nil)
	}

	return &Generator{prefix: prefix, uuid: generator}, nil
}

// New generates a UUIDv7-backed TypeID, inheriting source blocking, entropy,
// rollback, overflow and time-range behavior. Failure returns the zero ID and
// wraps the source error, preserving its identifier sentinel classification.
func (generator *Generator) New() (ID, error) {
	value, err := generator.uuid.New()
	if err != nil {
		return ID{}, fmt.Errorf("generate TypeID: %w", err)
	}

	return FromUUID(generator.prefix, value)
}
