// Package identifier defines typed domain IDs and contracts shared by UUID,
// ULID, TypeID, KSUID, and NanoID families. Start with Parse and a zero-value
// Validator, or use a family package for generation and family-specific codecs.
//
// The package does not coordinate uniqueness, authorize access, or supply
// distributed idempotency. It starts no background work and requires no cgo or
// service. Values may be read concurrently; callers must synchronize decoding
// into a shared value. Validation and clock callbacks remain caller-owned.
package identifier

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

var (
	// ErrInvalid reports malformed, non-canonical, or disallowed input.
	ErrInvalid = errors.New("invalid identifier")
	// ErrEntropy reports an entropy read failure or an exhausted rejection budget.
	// Underlying reader errors are deliberately not retained in the error chain.
	ErrEntropy = errors.New("identifier entropy failure")
	// ErrClockRollback reports a clock moving behind supported generator state.
	ErrClockRollback = errors.New("identifier clock rollback")
	// ErrOverflow reports exhausted monotonic or sequence space.
	ErrOverflow = errors.New("identifier sequence overflow")
	// ErrUnsupported reports a valid but unsupported identifier feature.
	ErrUnsupported = errors.New("unsupported identifier")
)

const (
	// MaxTypedIDBytes bounds canonical text before a Validator is invoked.
	MaxTypedIDBytes = 1024
	// MaxTypedIDJSONBytes bounds an escaped JSON string before decoding.
	MaxTypedIDJSONBytes = MaxTypedIDBytes*6 + 2
)

// Clock supplies the time observed by a generator. Implementations need not be
// monotonic: time-ordered generators detect rollback at their time resolution.
// A clock shared outside one generator must provide its own synchronization.
type Clock interface {
	// Now returns the current instant. It must not reenter the calling generator.
	Now() time.Time
}

// ClockFunc adapts a non-nil function to Clock without adding synchronization.
type ClockFunc func() time.Time

// Now calls the underlying function.
func (clock ClockFunc) Now() time.Time { return clock() }

// Generator is the common generation entry point. It does not, by itself,
// promise concurrency safety, ordering, or uniqueness across instances;
// consumers must use the concrete generator's contract.
type Generator[T any] interface {
	// New returns a generated value or an error. Callers must not use the value
	// on failure; blocking and error classifications depend on the family.
	New() (T, error)
}

// Validator validates the canonical text used by a typed ID. Implementations
// must be usable as their zero value so decoding never needs registration.
// Validate is called synchronously with bounded nonempty text; it must establish
// any family or domain policy that the tag requires. Concurrent parses may call
// it concurrently, so any state outside the zero-value tag must be synchronized.
type Validator interface {
	// Validate returns nil only for accepted canonical text. Its error details
	// and cause are hidden behind ErrInvalid; panics are not recovered.
	Validate(string) error
}

// ID is a strongly typed canonical identifier. Different Tag arguments are
// distinct Go types and cannot be mixed without an explicit conversion.
// Its zero value is unassigned. Text/binary/JSON marshaling allows that zero
// value, but parsing or decoding empty text rejects it; SQL NULL clears it.
// Unmarshal and Scan methods require a non-nil, exclusively owned receiver.
type ID[Tag Validator] struct {
	text string
}

// Parse checks nonempty text and MaxTypedIDBytes before calling Tag's zero-value
// Validate method. It preserves accepted text without normalization. Failure
// returns the zero ID and an error wrapping ErrInvalid, not the validator error.
func Parse[Tag Validator](text string) (ID[Tag], error) {
	var tag Tag

	if text == "" {
		return ID[Tag]{}, fmt.Errorf("%w: empty typed ID", ErrInvalid)
	}
	if len(text) > MaxTypedIDBytes {
		return ID[Tag]{}, fmt.Errorf("%w: typed ID text length is invalid", ErrInvalid)
	}

	if err := tag.Validate(text); err != nil {
		return ID[Tag]{}, fmt.Errorf("%w: typed ID validation failed", ErrInvalid)
	}

	return ID[Tag]{text: text}, nil
}

// String returns canonical text.
func (id ID[Tag]) String() string { return id.text }

// LogValue redacts the identifier from structured logs. Callers must opt in
// explicitly with String when an identifier has been approved for logging.
func (id ID[Tag]) LogValue() slog.Value { return slog.StringValue("[REDACTED]") }

// IsZero reports whether no identifier has been assigned.
func (id ID[Tag]) IsZero() bool { return id.text == "" }

// Compare returns -1, 0, or 1 according to canonical lexical ordering.
func (id ID[Tag]) Compare(other ID[Tag]) int {
	if id.text < other.text {
		return -1
	}
	if id.text > other.text {
		return 1
	}

	return 0
}

// MarshalText returns a new slice of canonical text, empty when unassigned.
func (id ID[Tag]) MarshalText() ([]byte, error) { return []byte(id.text), nil }

// UnmarshalText validates copied text through Parse and replaces id only on success.
func (id *ID[Tag]) UnmarshalText(text []byte) error {
	if len(text) > MaxTypedIDBytes {
		return fmt.Errorf("%w: typed ID text length is invalid", ErrInvalid)
	}

	parsed, err := Parse[Tag](string(text))
	if err != nil {
		return err
	}

	*id = parsed

	return nil
}

// MarshalBinary uses canonical text so typed values remain family-neutral.
func (id ID[Tag]) MarshalBinary() ([]byte, error) { return id.MarshalText() }

// UnmarshalBinary has the same validation and receiver semantics as UnmarshalText.
func (id *ID[Tag]) UnmarshalBinary(data []byte) error { return id.UnmarshalText(data) }

// MarshalJSON encodes text as a JSON string, including "" for an unassigned ID.
func (id ID[Tag]) MarshalJSON() ([]byte, error) { return json.Marshal(id.text) }

// UnmarshalJSON decodes and validates a canonical JSON string no larger than
// MaxTypedIDJSONBytes. Empty strings and null are invalid. Errors wrap
// ErrInvalid and leave id unchanged; JSON parser diagnostics are not exposed.
func (id *ID[Tag]) UnmarshalJSON(data []byte) error {
	if len(data) > MaxTypedIDJSONBytes {
		return fmt.Errorf("%w: typed ID JSON length is invalid", ErrInvalid)
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("%w: malformed typed identifier JSON", ErrInvalid)
	}

	return id.UnmarshalText([]byte(text))
}

// Value implements driver.Valuer. An unassigned ID is stored as SQL NULL.
func (id ID[Tag]) Value() (driver.Value, error) {
	if id.IsZero() {
		return nil, nil
	}

	return id.text, nil
}

// Scan accepts strings or text byte slices through Parse; nil clears id.
// Other sources and invalid text return ErrInvalid without replacing id.
// Accepted bytes are not retained.
func (id *ID[Tag]) Scan(src any) error {
	switch value := src.(type) {
	case nil:
		*id = ID[Tag]{}

		return nil
	case string:
		parsed, err := Parse[Tag](value)
		if err != nil {
			return err
		}
		*id = parsed

		return nil
	case []byte:
		return id.UnmarshalText(value)
	default:
		return fmt.Errorf("%w: unsupported typed ID scan source", ErrInvalid)
	}
}

// Family identifies an encoding without implying equivalent guarantees.
type Family string

const (
	// FamilyUUID identifies UUID values in the RFC 9562 profile.
	FamilyUUID Family = "uuid"
	// FamilyULID identifies canonical ULID values.
	FamilyULID Family = "ulid"
	// FamilyTypeID identifies TypeID values.
	FamilyTypeID Family = "typeid"
	// FamilyKSUID identifies Segment-compatible KSUID values.
	FamilyKSUID Family = "ksuid"
	// FamilyNanoID identifies configured NanoID values.
	FamilyNanoID Family = "nanoid"
)

// Inspection exposes only properties actually defined by an identifier family.
type Inspection struct {
	// Family identifies the representation inspected, even for a zero value.
	Family Family
	// Version is the UUID version nibble for UUIDs or TypeID suffixes;
	// it is zero for families that do not define a UUID version.
	Version int
	// Timestamp is the embedded UTC instant, meaningful only when HasTime is true.
	Timestamp time.Time
	// HasTime reports whether this value exposes a recognized timestamp.
	HasTime bool
	// Sortable reports the family's ordering capability, not global issuance
	// order, uniqueness, or necessarily the presence of a timestamp.
	Sortable bool
}
