// Package nanoid implements the Nano ID reference profile pinned at revision
// 07a39d62d84c21af5046fe6b2ef7b3e36ee557db, restricted to unique printable ASCII
// alphabets, bounded sizes, and a 120-bit ideal entropy floor.
// Start with DefaultConfig and NewGenerator, or Parse for default text.
// Custom decoding requires Prepare. The package requires no service or cgo
// and starts no background work.
//
// NanoIDs provide neither time order nor authorization. Randomness quality
// depends on the supplied reader; the size floor alone cannot ensure uniqueness
// or secrecy. IDENTIFIER-NANOID-DEC-001 in docs/specification-decisions.md
// records the deliberate restrictions on the reference profile.
package nanoid

import (
	"bytes"
	cryptorand "crypto/rand"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/bits"
	"strings"
	"sync"

	identifier "github.com/faustbrian/go-identifier/v2"
)

const (
	// DefaultAlphabet is the URL-safe alphabet used by the reference NanoID.
	DefaultAlphabet = "_-0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// DefaultSize is the reference NanoID length.
	DefaultSize = 21
	// MinimumEntropyBits prevents configurations below the package's security
	// floor. NanoIDs remain identifiers, not secrets or authorization evidence.
	MinimumEntropyBits = 120
	maximumSize        = 1024
)

// Config defines a copied alphabet and fixed byte length. The zero Config is
// invalid; use DefaultConfig or an explicitly validated custom configuration.
type Config struct {
	// Alphabet is an ordered set of 2 through 94 distinct printable ASCII bytes
	// (0x21 through 0x7e). Its order affects seeded generation, not uniformity.
	Alphabet string
	// Size is the output byte count, at most 1024 and large enough to provide
	// Size*log2(len(Alphabet)) >= MinimumEntropyBits of ideal entropy.
	Size int
}

// DefaultConfig returns the 21-character URL-safe NanoID configuration.
func DefaultConfig() Config { return Config{Alphabet: DefaultAlphabet, Size: DefaultSize} }

// Validate rejects alphabets that could bias or ambiguously encode output and
// configurations below 120 bits of ideal entropy. Failure wraps
// identifier.ErrInvalid; no defaults are applied to an invalid Config.
func (config Config) Validate() error {
	if len(config.Alphabet) < 2 {
		return fmt.Errorf("%w: NanoID alphabet must contain at least 2 bytes", identifier.ErrInvalid)
	}
	var seen [128]bool
	for index := range len(config.Alphabet) {
		character := config.Alphabet[index]
		if character < 0x21 || character > 0x7e {
			return fmt.Errorf("%w: NanoID alphabet must use printable ASCII", identifier.ErrInvalid)
		}
		if seen[character] {
			return fmt.Errorf("%w: NanoID alphabet contains a duplicate byte", identifier.ErrInvalid)
		}
		seen[character] = true
	}
	if config.Size > maximumSize {
		return fmt.Errorf("%w: NanoID size must not exceed %d", identifier.ErrInvalid, maximumSize)
	}
	entropy := float64(config.Size) * math.Log2(float64(len(config.Alphabet)))
	if entropy < MinimumEntropyBits {
		return fmt.Errorf("%w: NanoID entropy is below the minimum", identifier.ErrInvalid)
	}

	return nil
}

// ID is an immutable NanoID carrying the configuration required to validate
// subsequent decoding. Its zero value decodes with DefaultConfig.
// Read-only methods may run concurrently. Decode methods require a non-nil
// receiver and exclusive access; they leave it unchanged on invalid input.
type ID struct {
	text   string
	config Config
}

// Parse validates exactly DefaultSize bytes drawn from DefaultAlphabet.
// Invalid text returns a zero ID with identifier.ErrInvalid.
func Parse(text string) (ID, error) { return ParseWithConfig(text, DefaultConfig()) }

// ParseWithConfig validates config first, then requires its exact size and
// alphabet. It returns a zero ID with identifier.ErrInvalid on failure; accepted
// text and configuration are retained by value without normalization.
func ParseWithConfig(text string, config Config) (ID, error) {
	if err := config.Validate(); err != nil {
		return ID{}, err
	}
	if len(text) != config.Size {
		return ID{}, fmt.Errorf("%w: NanoID text length does not match configuration", identifier.ErrInvalid)
	}
	for index := range len(text) {
		if !strings.ContainsRune(config.Alphabet, rune(text[index])) {
			return ID{}, fmt.Errorf("%w: NanoID contains a byte outside its alphabet", identifier.ErrInvalid)
		}
	}

	return ID{text: text, config: config}, nil
}

// Prepare validates config and returns an unassigned decoding target carrying
// it. Invalid config returns a zero ID with identifier.ErrInvalid. JSON null or
// SQL NULL clears text but keeps this custom decoding configuration.
func Prepare(config Config) (ID, error) {
	if err := config.Validate(); err != nil {
		return ID{}, err
	}

	return ID{config: config}, nil
}

// Config returns the carried configuration by value, or DefaultConfig when
// no custom configuration has been assigned.
func (id ID) Config() Config {
	if id.config.Alphabet == "" {
		return DefaultConfig()
	}

	return id.config
}

// String returns the NanoID text.
func (id ID) String() string { return id.text }

// LogValue redacts the NanoID from structured logs.
func (id ID) LogValue() slog.Value { return slog.StringValue("[REDACTED]") }

// IsZero reports whether no NanoID has been assigned.
func (id ID) IsZero() bool { return id.text == "" }

// Compare returns lexical ordering. NanoID generation itself is not sortable.
func (id ID) Compare(other ID) int { return strings.Compare(id.text, other.text) }

// Inspect reports that NanoIDs define neither time nor sortable generation.
func (id ID) Inspect() identifier.Inspection {
	return identifier.Inspection{Family: identifier.FamilyNanoID}
}

// MarshalText returns newly allocated text or identifier.ErrInvalid when unassigned.
func (id ID) MarshalText() ([]byte, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("%w: unassigned NanoID", identifier.ErrInvalid)
	}

	return []byte(id.text), nil
}

// UnmarshalText parses text and replaces id only on success. Invalid input
// wraps identifier.ErrInvalid; input bytes are not retained.
// Validation uses the carried configuration, or DefaultConfig for a zero ID.
func (id *ID) UnmarshalText(text []byte) error {
	config := id.Config()
	if len(text) != config.Size {
		return fmt.Errorf("%w: NanoID text length does not match configuration", identifier.ErrInvalid)
	}

	parsed, err := ParseWithConfig(string(text), config)
	if err != nil {
		return err
	}
	*id = parsed

	return nil
}

// MarshalBinary returns a new text slice, with the same errors as MarshalText.
func (id ID) MarshalBinary() ([]byte, error) { return id.MarshalText() }

// UnmarshalBinary uses UnmarshalText; bytes are text, not a separate raw payload.
func (id *ID) UnmarshalBinary(data []byte) error { return id.UnmarshalText(data) }

// MarshalJSON encodes a NanoID string or null when unassigned.
func (id ID) MarshalJSON() ([]byte, error) {
	if id.IsZero() {
		return []byte("null"), nil
	}

	return json.Marshal(id.text)
}

// UnmarshalJSON validates a string and accepts null as unassigned while
// retaining a prepared custom configuration. Encoded input is bounded to
// Config().Size*6+2 bytes; failure wraps identifier.ErrInvalid without changing id.
// Only the exact null token clears id in a direct call.
func (id *ID) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		id.text = ""

		return nil
	}
	config := id.Config()
	if len(data) > config.Size*6+2 {
		return fmt.Errorf("%w: NanoID JSON length is invalid", identifier.ErrInvalid)
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("%w: malformed NanoID JSON", identifier.ErrInvalid)
	}

	parsed, err := ParseWithConfig(text, config)
	if err != nil {
		return err
	}
	*id = parsed

	return nil
}

// Value returns identifier text for database/sql, or nil (SQL NULL) when unassigned.
func (id ID) Value() (driver.Value, error) {
	if id.IsZero() {
		return nil, nil
	}

	return id.text, nil
}

// Scan validates strings or text byte slices against Config(). Nil clears text
// while preserving configuration. Invalid sources wrap identifier.ErrInvalid
// without changing id; accepted bytes are not retained.
func (id *ID) Scan(src any) error {
	switch value := src.(type) {
	case nil:
		id.text = ""

		return nil
	case string:
		parsed, err := ParseWithConfig(value, id.Config())
		if err != nil {
			return err
		}
		*id = parsed

		return nil
	case []byte:
		return id.UnmarshalText(value)
	default:
		return fmt.Errorf("%w: unsupported NanoID scan source", identifier.ErrInvalid)
	}
}

// Generator serializes New calls over a copied configuration and borrowed
// reader. Construct it with NewGenerator; its zero value is not ready for use
// and it must not be copied after first use. The reader is never closed and
// runs under the lock; it must not reenter the generator and needs its own
// synchronization if shared elsewhere.
type Generator struct {
	mutex   sync.Mutex
	config  Config
	entropy io.Reader
	mask    byte
	step    int
}

// NewGenerator constructs a bias-free NanoID generator. A nil reader selects
// crypto/rand.Reader for this generator instance. Invalid configuration returns
// nil and identifier.ErrInvalid; caller-supplied entropy quality remains the
// caller's responsibility.
func NewGenerator(config Config, entropy io.Reader) (*Generator, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if entropy == nil {
		entropy = cryptorand.Reader
	}

	// The pinned Nano ID mask/rejection algorithm avoids modulo bias for
	// non-power-of-two alphabets. The 1.6 factor sizes each batch; it is not a
	// completeness guarantee (IDENTIFIER-NANOID-DEC-001;
	// TestGeneratorMaskUsesSmallestCoveringBitRange and
	// TestNanoIDRejectionSamplingHasNoModuloBias).
	mask := byte(1<<bits.Len(uint(len(config.Alphabet)-1))) - 1
	step := int(math.Ceil(1.6 * float64(int(mask)*config.Size) / float64(len(config.Alphabet))))

	return &Generator{config: config, entropy: entropy, mask: mask, step: step}, nil
}

// New samples alphabet positions uniformly when entropy bytes are uniform.
// It allows at most 128 batches; read failure or rejection-budget exhaustion
// returns a zero ID with identifier.ErrEntropy and hides the reader's cause.
// Reads can still block without cancellation: the batch bound is not a deadline.
func (generator *Generator) New() (ID, error) {
	generator.mutex.Lock()
	defer generator.mutex.Unlock()

	output := make([]byte, 0, generator.config.Size)
	buffer := make([]byte, generator.step)
	for range 128 {
		if _, err := io.ReadFull(generator.entropy, buffer); err != nil {
			return ID{}, fmt.Errorf("%w: NanoID entropy source failed", identifier.ErrEntropy)
		}
		for _, random := range buffer {
			index := int(random & generator.mask)
			if index >= len(generator.config.Alphabet) {
				continue
			}
			output = append(output, generator.config.Alphabet[index])
			if len(output) == generator.config.Size {
				return ID{text: string(output), config: generator.config}, nil
			}
		}
	}

	return ID{}, fmt.Errorf("%w: NanoID rejection limit", identifier.ErrEntropy)
}
