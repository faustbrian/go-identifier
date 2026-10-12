// Package ulid implements the pinned ULID specification at revision
// d0c7170df4517939e70129b4d6462cc162f2d5bf, with consistent-case parsing,
// uppercase canonical output, and per-generator millisecond monotonicity.
// Start with Parse or NewGenerator; StringLower supports existing lowercase
// storage. The package requires no service or cgo and starts no background work.
//
// ULIDs expose time and local issuance order, not distributed uniqueness or
// authorization. Case and rollback policies are IDENTIFIER-ULID-DEC-001/002 in
// docs/specification-decisions.md.
package ulid

import (
	"bytes"
	cryptorand "crypto/rand"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	identifier "github.com/faustbrian/go-identifier/v2"
	oklogulid "github.com/oklog/ulid/v2"
)

const (
	alphabet          = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	maximumUnixMillis = int64(281_474_976_710_655)
	textLength        = 26
	maximumJSONBytes  = textLength*6 + 2
)

// ID is an immutable ULID. The validity bit distinguishes an unassigned Go
// zero value from the valid all-zero ULID.
// Read-only methods may run concurrently. Decode methods require a non-nil
// receiver and exclusive access; they leave it unchanged on invalid input.
type ID struct {
	value [16]byte
	valid bool
}

// Parse accepts a consistently uppercase or lowercase Crockford Base32 ULID.
// Mixed-case text and ambiguous Crockford characters are rejected. Output is
// uppercase. Failure returns the zero ID with identifier.ErrInvalid.
// The pinned ULID specification's overflow/canonical-string rules and the
// Laravel case policy are IDENTIFIER-ULID-DEC-001; protected by
// TestParserAcceptsLowercaseLaravelStorageAndRejectsMixedCase.
func Parse(text string) (ID, error) {
	if len(text) != textLength {
		return ID{}, fmt.Errorf("%w: ULID text length is invalid", identifier.ErrInvalid)
	}
	if text[0] > '7' {
		return ID{}, fmt.Errorf("%w: ULID exceeds 128 bits", identifier.ErrInvalid)
	}
	uppercase := strings.ToUpper(text)
	if text != uppercase && text != strings.ToLower(text) {
		return ID{}, fmt.Errorf("%w: ULID contains mixed-case Base32", identifier.ErrInvalid)
	}
	for index := range len(uppercase) {
		if !stringsContainsByte(alphabet, uppercase[index]) {
			return ID{}, fmt.Errorf("%w: ULID contains non-canonical Base32", identifier.ErrInvalid)
		}
	}

	parsed, _ := oklogulid.ParseStrict(uppercase)

	return ID{value: [16]byte(parsed), valid: true}, nil
}

func stringsContainsByte(text string, target byte) bool {
	for index := range len(text) {
		if text[index] == target {
			return true
		}
	}

	return false
}

// FromBytes copies any 128-bit representation as an assigned ULID, including
// all-zero bytes, which are distinct from the unassigned Go zero value.
func FromBytes(value [16]byte) ID { return ID{value: value, valid: true} }

// Bytes returns a copy of the binary representation.
func (id ID) Bytes() [16]byte { return id.value }

// IsZero reports whether no ULID has been assigned.
func (id ID) IsZero() bool { return !id.valid }

// Compare returns -1, 0, or 1 according to ULID binary ordering, ignoring the
// assignment flag. An unassigned value compares equal to the assigned all-zero ULID.
func (id ID) Compare(other ID) int { return bytes.Compare(id.value[:], other.value[:]) }

// String returns canonical uppercase text, or empty text when unassigned.
func (id ID) String() string {
	if !id.valid {
		return ""
	}

	return oklogulid.ULID(id.value).String()
}

// StringLower returns lowercase text for schemas whose established ULID
// contract is lowercase, or an empty string for an unassigned value.
func (id ID) StringLower() string {
	return strings.ToLower(id.String())
}

// LogValue redacts the ULID from structured logs.
func (id ID) LogValue() slog.Value { return slog.StringValue("[REDACTED]") }

// Inspect reports ULID sortability and, for assigned values, the embedded UTC
// millisecond timestamp. An unassigned value has HasTime false.
func (id ID) Inspect() identifier.Inspection {
	inspection := identifier.Inspection{Family: identifier.FamilyULID, Sortable: true}
	if id.valid {
		milliseconds := id.value[0:6]
		value := int64(milliseconds[0])<<40 | int64(milliseconds[1])<<32 |
			int64(milliseconds[2])<<24 | int64(milliseconds[3])<<16 |
			int64(milliseconds[4])<<8 | int64(milliseconds[5])
		inspection.Timestamp = time.UnixMilli(value).UTC()
		inspection.HasTime = true
	}

	return inspection
}

// MarshalText returns newly allocated canonical text, or identifier.ErrInvalid
// when unassigned.
func (id ID) MarshalText() ([]byte, error) {
	if !id.valid {
		return nil, fmt.Errorf("%w: unassigned ULID", identifier.ErrInvalid)
	}

	return []byte(id.String()), nil
}

// UnmarshalText parses text and replaces id only on success. Invalid input
// wraps identifier.ErrInvalid; input bytes are not retained.
func (id *ID) UnmarshalText(text []byte) error {
	if len(text) != textLength {
		return fmt.Errorf("%w: ULID text length is invalid", identifier.ErrInvalid)
	}

	parsed, err := Parse(string(text))
	if err != nil {
		return err
	}
	*id = parsed

	return nil
}

// MarshalBinary returns a new 16-byte representation, or identifier.ErrInvalid
// when unassigned.
func (id ID) MarshalBinary() ([]byte, error) {
	if !id.valid {
		return nil, fmt.Errorf("%w: unassigned ULID", identifier.ErrInvalid)
	}

	result := make([]byte, len(id.value))
	copy(result, id.value[:])

	return result, nil
}

// UnmarshalBinary requires exactly 16 bytes and copies them on success.
// Invalid input wraps identifier.ErrInvalid and leaves id unchanged.
// Every bit pattern, including all-zero bytes, is an assigned value.
func (id *ID) UnmarshalBinary(data []byte) error {
	if len(data) != 16 {
		return fmt.Errorf("%w: ULID binary length is invalid", identifier.ErrInvalid)
	}

	var value [16]byte
	copy(value[:], data)
	*id = FromBytes(value)

	return nil
}

// MarshalJSON encodes canonical text or null for an unassigned value.
func (id ID) MarshalJSON() ([]byte, error) {
	if !id.valid {
		return []byte("null"), nil
	}

	return json.Marshal(id.String())
}

// UnmarshalJSON parses a bounded JSON string, or clears id for the exact token
// null. Invalid input wraps identifier.ErrInvalid and leaves id unchanged.
// Encoded input is limited to 158 bytes, including quotes and escapes.
func (id *ID) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*id = ID{}

		return nil
	}
	if len(data) > maximumJSONBytes {
		return fmt.Errorf("%w: ULID JSON length is invalid", identifier.ErrInvalid)
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("%w: malformed ULID JSON", identifier.ErrInvalid)
	}

	parsed, err := Parse(text)
	if err != nil {
		return err
	}
	*id = parsed

	return nil
}

// Value returns canonical text for database/sql, or nil (SQL NULL) when unassigned.
func (id ID) Value() (driver.Value, error) {
	if !id.valid {
		return nil, nil
	}

	return id.String(), nil
}

// Scan accepts canonical strings, byte slices, or nil (SQL NULL). A 16-byte
// slice is interpreted as binary; other slices are text. Invalid input wraps
// identifier.ErrInvalid and leaves id unchanged; nil clears id. Bytes are copied.
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
		if len(value) == 16 {
			return id.UnmarshalBinary(value)
		}

		return id.UnmarshalText(value)
	default:
		return fmt.Errorf("%w: unsupported ULID scan source", identifier.ErrInvalid)
	}
}

// Generator serializes New calls and owns a millisecond monotonic sequence.
// Construct it with NewGenerator; its zero value is not ready for use and it
// must not be copied after first use. Clock and entropy are borrowed, never
// closed. They run under the lock, must not reenter the generator, and need
// their own synchronization if shared elsewhere. Ordering is per instance,
// not distributed uniqueness.
type Generator struct {
	mutex       sync.Mutex
	clock       identifier.Clock
	entropy     io.Reader
	initialized bool
	lastMillis  int64
	last        ID
}

// NewGenerator constructs a monotonic ULID generator. Nil inputs select a
// system clock and crypto/rand.Reader for this generator instance.
func NewGenerator(clock identifier.Clock, entropy io.Reader) *Generator {
	if clock == nil {
		clock = identifier.ClockFunc(time.Now)
	}
	if entropy == nil {
		entropy = cryptorand.Reader
	}

	return &Generator{clock: clock, entropy: entropy}
}

// New generates a ULID with fresh entropy on each new millisecond and an
// incremented 80-bit payload within that millisecond. Failure returns a zero ID:
// identifier.ErrInvalid for timestamps outside 48-bit Unix milliseconds,
// ErrClockRollback for rollback, ErrOverflow for payload exhaustion, and
// ErrEntropy for reader failures. Failed calls preserve the committed sequence
// but may consume entropy. Clock/reader calls can block without cancellation.
func (generator *Generator) New() (ID, error) {
	generator.mutex.Lock()
	defer generator.mutex.Unlock()

	milliseconds := generator.clock.Now().UnixMilli()
	if milliseconds < 0 || milliseconds > maximumUnixMillis {
		return ID{}, fmt.Errorf("%w: ULID timestamp is outside 48 bits", identifier.ErrInvalid)
	}
	if generator.initialized && milliseconds < generator.lastMillis {
		return ID{}, fmt.Errorf("%w: ULID clock moved backward", identifier.ErrClockRollback)
	}
	if generator.initialized && milliseconds == generator.lastMillis {
		id := generator.last
		if !increment(&id.value) {
			return ID{}, fmt.Errorf("%w: ULID entropy field", identifier.ErrOverflow)
		}
		generator.last = id

		return id, nil
	}

	var value [16]byte
	timestamp := uint64(milliseconds)
	for position := 5; position >= 0; position-- {
		// #nosec G115 -- the low byte is selected intentionally before shifting.
		value[position] = byte(timestamp)
		timestamp >>= 8
	}
	if _, err := io.ReadFull(generator.entropy, value[6:]); err != nil {
		return ID{}, fmt.Errorf("%w: ULID entropy source failed", identifier.ErrEntropy)
	}

	id := FromBytes(value)
	generator.initialized = true
	generator.lastMillis = milliseconds
	generator.last = id

	return id, nil
}

// The pinned ULID specification's Monotonicity rule increments the 80-bit
// random field without changing the timestamp (IDENTIFIER-ULID-DEC-002;
// TestGeneratorAcceptsTimestampAndCarryBoundaries). Increment a copy so failure
// leaves the last issued value intact.
func increment(value *[16]byte) bool {
	for position := 15; position >= 6; position-- {
		value[position]++
		if value[position] != 0 {
			return true
		}
	}

	return false
}
