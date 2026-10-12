// Package ksuid implements the pinned Segment KSUID format at revision
// d33724947fcfba7949906c2b1821e96a1c8d06e7: 20-byte values and 27-byte Base62
// text. Start with Parse or NewGenerator. The package requires no service or
// cgo and starts no background work.
//
// Generation is monotonic within one instance, unlike independent-random
// Segment generation. It reveals seconds and local issuance order, not global
// ordering or authorization. IDENTIFIER-KSUID-DEC-001 in
// docs/specification-decisions.md records this wire-compatible policy.
package ksuid

import (
	"bytes"
	cryptorand "crypto/rand"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"sync"
	"time"

	identifier "github.com/faustbrian/go-identifier/v2"
	segmentksuid "github.com/segmentio/ksuid"
)

const (
	epoch            = int64(1_400_000_000)
	textLength       = 27
	maximumJSONBytes = textLength*6 + 2
)

// ID is an immutable KSUID. A validity bit distinguishes an unassigned value
// from the valid all-zero KSUID.
// Read-only methods may run concurrently. Decode methods require a non-nil
// receiver and exclusive access; they leave it unchanged on invalid input.
type ID struct {
	value [20]byte
	valid bool
}

// Parse accepts exactly the canonical 27-character Base62 representation.
// Failure returns the zero ID with identifier.ErrInvalid. Re-encoding checks
// the pinned Segment format's canonical representation (IDENTIFIER-KSUID-DEC-001;
// TestReferenceVectorAndDifferentialParsing).
func Parse(text string) (ID, error) {
	if len(text) != textLength {
		return ID{}, fmt.Errorf("%w: KSUID text length is invalid", identifier.ErrInvalid)
	}

	parsed, err := segmentksuid.Parse(text)
	if err != nil {
		return ID{}, fmt.Errorf("%w: KSUID decoding failed", identifier.ErrInvalid)
	}
	if parsed.String() != text {
		return ID{}, fmt.Errorf("%w: non-canonical KSUID", identifier.ErrInvalid)
	}

	return ID{value: [20]byte(parsed), valid: true}, nil
}

// FromBytes copies any 20-byte representation as assigned, including all-zero
// bytes, which are distinct from the unassigned Go zero value.
func FromBytes(value [20]byte) ID { return ID{value: value, valid: true} }

// Bytes returns a copy of the KSUID representation.
func (id ID) Bytes() [20]byte { return id.value }

// IsZero reports whether no KSUID has been assigned.
func (id ID) IsZero() bool { return !id.valid }

// Compare returns -1, 0, or 1 according to binary ordering, ignoring assignment.
// An unassigned value compares equal to the assigned all-zero KSUID.
func (id ID) Compare(other ID) int { return bytes.Compare(id.value[:], other.value[:]) }

// String returns canonical Base62 text, or empty text when unassigned.
func (id ID) String() string {
	if !id.valid {
		return ""
	}

	return segmentksuid.KSUID(id.value).String()
}

// LogValue redacts the KSUID from structured logs.
func (id ID) LogValue() slog.Value { return slog.StringValue("[REDACTED]") }

// Inspect returns the second-resolution timestamp exposed by a KSUID.
func (id ID) Inspect() identifier.Inspection {
	inspection := identifier.Inspection{Family: identifier.FamilyKSUID, Sortable: id.valid}
	if id.valid {
		seconds := int64(binary.BigEndian.Uint32(id.value[0:4])) + epoch
		inspection.Timestamp = time.Unix(seconds, 0).UTC()
		inspection.HasTime = true
	}

	return inspection
}

// MarshalText returns newly allocated canonical text, or identifier.ErrInvalid
// when unassigned.
func (id ID) MarshalText() ([]byte, error) {
	if !id.valid {
		return nil, fmt.Errorf("%w: unassigned KSUID", identifier.ErrInvalid)
	}

	return []byte(id.String()), nil
}

// UnmarshalText parses text and replaces id only on success. Invalid input
// wraps identifier.ErrInvalid; input bytes are not retained.
func (id *ID) UnmarshalText(text []byte) error {
	if len(text) != textLength {
		return fmt.Errorf("%w: KSUID text length is invalid", identifier.ErrInvalid)
	}

	parsed, err := Parse(string(text))
	if err != nil {
		return err
	}
	*id = parsed

	return nil
}

// MarshalBinary returns a new 20-byte representation, or identifier.ErrInvalid
// when unassigned.
func (id ID) MarshalBinary() ([]byte, error) {
	if !id.valid {
		return nil, fmt.Errorf("%w: unassigned KSUID", identifier.ErrInvalid)
	}

	result := make([]byte, len(id.value))
	copy(result, id.value[:])

	return result, nil
}

// UnmarshalBinary requires exactly 20 bytes and copies them on success.
// Invalid input wraps identifier.ErrInvalid and leaves id unchanged.
// Every bit pattern, including all-zero bytes, is an assigned value.
func (id *ID) UnmarshalBinary(data []byte) error {
	if len(data) != 20 {
		return fmt.Errorf("%w: KSUID binary length is invalid", identifier.ErrInvalid)
	}

	var value [20]byte
	copy(value[:], data)
	*id = FromBytes(value)

	return nil
}

// MarshalJSON encodes canonical text or null when unassigned.
func (id ID) MarshalJSON() ([]byte, error) {
	if !id.valid {
		return []byte("null"), nil
	}

	return json.Marshal(id.String())
}

// UnmarshalJSON parses a bounded JSON string, or clears id for the exact token
// null. Invalid input wraps identifier.ErrInvalid and leaves id unchanged.
// Encoded input is limited to 164 bytes, including quotes and escapes.
func (id *ID) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*id = ID{}

		return nil
	}
	if len(data) > maximumJSONBytes {
		return fmt.Errorf("%w: KSUID JSON length is invalid", identifier.ErrInvalid)
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("%w: malformed KSUID JSON", identifier.ErrInvalid)
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

// Scan accepts canonical strings, byte slices, or nil (SQL NULL). A 20-byte
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
		if len(value) == 20 {
			return id.UnmarshalBinary(value)
		}

		return id.UnmarshalText(value)
	default:
		return fmt.Errorf("%w: unsupported KSUID scan source", identifier.ErrInvalid)
	}
}

// Generator serializes New calls and owns a monotonic sequence. Within a
// second, it increments the initial random payload; this leaks local issuance
// order and differs from independent-random KSUID generation while preserving
// wire interoperability. Construct it with NewGenerator; its zero value is not
// ready for use and it must not be copied after first use. Clock/entropy are
// borrowed, never closed, and called under the lock. They must not reenter the
// generator and need their own synchronization if shared elsewhere.
type Generator struct {
	mutex       sync.Mutex
	clock       identifier.Clock
	entropy     io.Reader
	initialized bool
	lastSecond  int64
	last        ID
}

// NewGenerator creates a monotonic KSUID generator. Nil inputs select a system
// clock and crypto/rand.Reader for this generator instance.
func NewGenerator(clock identifier.Clock, entropy io.Reader) *Generator {
	if clock == nil {
		clock = identifier.ClockFunc(time.Now)
	}
	if entropy == nil {
		entropy = cryptorand.Reader
	}

	return &Generator{clock: clock, entropy: entropy}
}

// New draws fresh 128-bit payload entropy on a new second and increments it
// within that second. Failure returns a zero ID: identifier.ErrInvalid for time
// outside Unix seconds [1400000000, 1400000000+2^32-1], ErrClockRollback for
// rollback, ErrOverflow for payload exhaustion, and ErrEntropy for reader errors.
// Failed calls preserve committed state but may consume entropy. Clock/reader
// calls can block without cancellation; ordering and uniqueness are not global.
func (generator *Generator) New() (ID, error) {
	generator.mutex.Lock()
	defer generator.mutex.Unlock()

	seconds := generator.clock.Now().Unix()
	if seconds < epoch || seconds > epoch+int64(math.MaxUint32) {
		return ID{}, fmt.Errorf("%w: KSUID timestamp is outside its epoch", identifier.ErrInvalid)
	}
	if generator.initialized && seconds < generator.lastSecond {
		return ID{}, fmt.Errorf("%w: KSUID clock moved backward", identifier.ErrClockRollback)
	}
	if generator.initialized && seconds == generator.lastSecond {
		id := generator.last
		if !increment(&id.value) {
			return ID{}, fmt.Errorf("%w: KSUID payload", identifier.ErrOverflow)
		}
		generator.last = id

		return id, nil
	}

	var value [20]byte
	binary.BigEndian.PutUint32(value[0:4], uint32(seconds-epoch))
	if _, err := io.ReadFull(generator.entropy, value[4:]); err != nil {
		return ID{}, fmt.Errorf("%w: KSUID entropy source failed", identifier.ErrEntropy)
	}

	id := FromBytes(value)
	generator.initialized = true
	generator.lastSecond = seconds
	generator.last = id

	return id, nil
}

// This package's Segment-compatible policy increments only the 128-bit payload,
// not the 32-bit epoch offset (IDENTIFIER-KSUID-DEC-001;
// TestGeneratorAcceptsTimestampAndCarryBoundaries). Work on a copy so exhaustion
// cannot replace the last successfully issued value.
func increment(value *[20]byte) bool {
	for position := 19; position >= 4; position-- {
		value[position]++
		if value[position] != 0 {
			return true
		}
	}

	return false
}
