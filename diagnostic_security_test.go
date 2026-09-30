package identifier_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	identifier "github.com/faustbrian/go-identifier/v2"
	identifierksuid "github.com/faustbrian/go-identifier/v2/ksuid"
	identifiernanoid "github.com/faustbrian/go-identifier/v2/nanoid"
	identifiertypeid "github.com/faustbrian/go-identifier/v2/typeid"
	identifierulid "github.com/faustbrian/go-identifier/v2/ulid"
	identifieruuid "github.com/faustbrian/go-identifier/v2/uuid"
)

const hostileDiagnostic = "customer@example.com/secret-token"

type hostileTag struct{}

type panicValidatorTag struct{}

type acceptingValidatorTag struct{}

type hostileCause struct{ source string }

func (cause *hostileCause) Error() string { return hostileDiagnostic + "/" + cause.source }

var (
	hostileValidatorCause = &hostileCause{source: "validator"}
	hostileEntropyCause   = &hostileCause{source: "entropy"}
)

func (hostileTag) Validate(string) error { return hostileValidatorCause }

func (panicValidatorTag) Validate(string) error { panic("validator must not receive oversized input") }

func (acceptingValidatorTag) Validate(string) error { return nil }

type hostileReader struct{}

func (hostileReader) Read([]byte) (int, error) { return 0, hostileEntropyCause }

type sequenceClock struct {
	times []time.Time
	index int
}

func (clock *sequenceClock) Now() time.Time {
	value := clock.times[clock.index]
	clock.index++
	return value
}

func TestDiagnosticsDoNotExposeUntrustedValues(t *testing.T) {
	t.Parallel()

	_, parseErr := identifier.Parse[hostileTag]("owned-value")
	assertSafeDiagnostic(t, parseErr, identifier.ErrInvalid)
	assertCauseHidden(t, parseErr, hostileValidatorCause)

	nanoGenerator, err := identifiernanoid.NewGenerator(identifiernanoid.DefaultConfig(), hostileReader{})
	if err != nil {
		t.Fatalf("construct NanoID generator: %v", err)
	}
	typeGenerator, err := identifiertypeid.NewGenerator(
		"account",
		identifieruuid.NewV7Generator(identifier.ClockFunc(func() time.Time { return time.UnixMilli(1) }), hostileReader{}),
	)
	if err != nil {
		t.Fatalf("construct TypeID generator: %v", err)
	}

	entropyFailures := []struct {
		name     string
		generate func() error
	}{
		{"UUIDv4", func() error { _, err := identifieruuid.NewV4Generator(hostileReader{}).New(); return err }},
		{"UUIDv7", func() error {
			_, err := identifieruuid.NewV7Generator(identifier.ClockFunc(func() time.Time { return time.UnixMilli(1) }), hostileReader{}).New()
			return err
		}},
		{"ULID", func() error {
			_, err := identifierulid.NewGenerator(identifier.ClockFunc(func() time.Time { return time.UnixMilli(1) }), hostileReader{}).New()
			return err
		}},
		{"KSUID", func() error {
			_, err := identifierksuid.NewGenerator(identifier.ClockFunc(func() time.Time { return time.Unix(1_400_000_001, 0) }), hostileReader{}).New()
			return err
		}},
		{"NanoID", func() error { _, err := nanoGenerator.New(); return err }},
		{"TypeID", func() error { _, err := typeGenerator.New(); return err }},
	}
	for _, failure := range entropyFailures {
		t.Run(failure.name, func(t *testing.T) {
			err := failure.generate()
			assertSafeDiagnostic(t, err, identifier.ErrEntropy)
			assertCauseHidden(t, err, hostileEntropyCause)
		})
	}
}

func TestValidationDiagnosticsDoNotExposeDerivedCallerMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		validate  func() error
		forbidden string
	}{
		{"UUID version", func() error {
			var value [16]byte
			value[6] = 0xf0
			value[8] = 0x80
			_, err := identifieruuid.FromBytes(value)
			return err
		}, "version 15"},
		{"UUID binary length", func() error {
			var value identifieruuid.ID
			return value.UnmarshalBinary(make([]byte, 7))
		}, "length is 7"},
		{"ULID text length", func() error {
			_, err := identifierulid.Parse(strings.Repeat("0", 7))
			return err
		}, "length is 7"},
		{"ULID binary length", func() error {
			var value identifierulid.ID
			return value.UnmarshalBinary(make([]byte, 7))
		}, "length is 7"},
		{"TypeID text length", func() error {
			_, err := identifiertypeid.Parse(strings.Repeat("a", 91))
			return err
		}, "length is 91"},
		{"KSUID text length", func() error {
			_, err := identifierksuid.Parse("123456789")
			return err
		}, "length is 9"},
		{"KSUID binary length", func() error {
			var value identifierksuid.ID
			return value.UnmarshalBinary(make([]byte, 9))
		}, "length is 9"},
		{"NanoID duplicate alphabet byte", func() error {
			return (identifiernanoid.Config{Alphabet: "abca", Size: 60}).Validate()
		}, "duplicate 'a'"},
		{"NanoID entropy", func() error {
			return (identifiernanoid.Config{Alphabet: "ab", Size: 2}).Validate()
		}, "has 2.0 bits"},
		{"NanoID text length", func() error {
			_, err := identifiernanoid.Parse("123456789")
			return err
		}, "length is 9, want 21"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.validate()
			assertSafeDiagnostic(t, err, identifier.ErrInvalid)
			if strings.Contains(err.Error(), test.forbidden) {
				t.Fatalf("diagnostic exposes derived caller metadata: %q", err)
			}
		})
	}
}

func TestTypedIdentifierRejectsOversizedInputBeforeValidation(t *testing.T) {
	t.Parallel()

	maximumText := strings.Repeat("x", identifier.MaxTypedIDBytes)
	if _, err := identifier.Parse[acceptingValidatorTag](maximumText); err != nil {
		t.Fatalf("maximum typed identifier error = %v", err)
	}

	oversizedText := strings.Repeat("x", identifier.MaxTypedIDBytes+1)
	if _, err := identifier.Parse[panicValidatorTag](oversizedText); !errors.Is(err, identifier.ErrInvalid) {
		t.Fatalf("oversized typed identifier error = %v", err)
	}
	var id identifier.ID[panicValidatorTag]
	if err := id.UnmarshalText([]byte(oversizedText)); !errors.Is(err, identifier.ErrInvalid) || !id.IsZero() {
		t.Fatalf("oversized typed text = %v, %v", id, err)
	}
	if err := id.Scan(oversizedText); !errors.Is(err, identifier.ErrInvalid) || !id.IsZero() {
		t.Fatalf("oversized typed SQL text = %v, %v", id, err)
	}

	oversizedJSON := []byte(`"` + strings.Repeat(`\u0078`, identifier.MaxTypedIDBytes+1) + `"`)
	if len(oversizedJSON) <= identifier.MaxTypedIDJSONBytes {
		t.Fatalf("oversized JSON fixture length = %d", len(oversizedJSON))
	}
	maximumJSON := []byte(`"` + strings.Repeat(`\u0078`, identifier.MaxTypedIDBytes) + `"`)
	if len(maximumJSON) != identifier.MaxTypedIDJSONBytes {
		t.Fatalf("maximum JSON fixture length = %d", len(maximumJSON))
	}
	var maximumID identifier.ID[acceptingValidatorTag]
	if err := maximumID.UnmarshalJSON(maximumJSON); err != nil {
		t.Fatalf("maximum typed identifier JSON error = %v", err)
	}

	if err := id.UnmarshalJSON(oversizedJSON); !errors.Is(err, identifier.ErrInvalid) {
		t.Fatalf("oversized typed identifier JSON error = %v", err)
	}
}

func TestConcreteIdentifierDecodersRejectInvalidCanonicalPayloadsWithoutMutation(t *testing.T) {
	t.Parallel()

	type decoder interface {
		UnmarshalText([]byte) error
		UnmarshalJSON([]byte) error
		String() string
	}
	decoders := []struct {
		name      string
		valid     string
		invalid   string
		construct func() decoder
	}{
		{"UUID", "017f22e2-79b0-7cc3-98c4-dc0c0c07398f", strings.Repeat("!", 36), func() decoder { return new(identifieruuid.ID) }},
		{"ULID", "01ARZ3NDEKTSV4RRFFQ69G5FAV", strings.Repeat("!", 26), func() decoder { return new(identifierulid.ID) }},
		{"TypeID", "user_" + strings.Repeat("0", 26), strings.Repeat("!", 26), func() decoder { return new(identifiertypeid.ID) }},
		{"KSUID", "0ujtsYcgvSTl8PAuAdqWYSMnLOv", strings.Repeat("!", 27), func() decoder { return new(identifierksuid.ID) }},
		{"NanoID", strings.Repeat("_", identifiernanoid.DefaultSize), strings.Repeat("!", identifiernanoid.DefaultSize), func() decoder { return new(identifiernanoid.ID) }},
	}
	for _, test := range decoders {
		t.Run(test.name, func(t *testing.T) {
			textValue := test.construct()
			if err := textValue.UnmarshalText([]byte(test.valid)); err != nil {
				t.Fatalf("valid text: %v", err)
			}
			assertSafeDiagnostic(t, textValue.UnmarshalText([]byte(test.invalid)), identifier.ErrInvalid)
			if textValue.String() != test.valid {
				t.Fatal("invalid text changed identifier")
			}

			jsonValue := test.construct()
			if err := jsonValue.UnmarshalJSON([]byte(`"` + test.valid + `"`)); err != nil {
				t.Fatalf("valid JSON: %v", err)
			}
			assertSafeDiagnostic(t, jsonValue.UnmarshalJSON([]byte(`"`+test.invalid+`"`)), identifier.ErrInvalid)
			if jsonValue.String() != test.valid {
				t.Fatal("invalid JSON changed identifier")
			}
		})
	}
}

func TestScanDiagnosticsDoNotExposeRuntimeTypes(t *testing.T) {
	t.Parallel()

	input := struct {
		Value string `secret:"customer@example.com/secret-token"`
	}{}

	var typed identifier.ID[hostileTag]
	var uuid identifieruuid.ID
	var ulid identifierulid.ID
	var typeID identifiertypeid.ID
	var ksuid identifierksuid.ID
	var nanoID identifiernanoid.ID

	scanners := []struct {
		name string
		scan func() error
	}{
		{"typed ID", func() error { return typed.Scan(input) }},
		{"UUID", func() error { return uuid.Scan(input) }},
		{"ULID", func() error { return ulid.Scan(input) }},
		{"TypeID", func() error { return typeID.Scan(input) }},
		{"KSUID", func() error { return ksuid.Scan(input) }},
		{"NanoID", func() error { return nanoID.Scan(input) }},
	}
	for _, scanner := range scanners {
		t.Run(scanner.name, func(t *testing.T) {
			assertSafeDiagnostic(t, scanner.scan(), identifier.ErrInvalid)
		})
	}
}

func TestJSONDiagnosticsDoNotExposeInvalidTokens(t *testing.T) {
	t.Parallel()

	data := []byte("@customer-data")
	var typed identifier.ID[hostileTag]
	var uuid identifieruuid.ID
	var ulid identifierulid.ID
	var typeID identifiertypeid.ID
	var ksuid identifierksuid.ID
	var nanoID identifiernanoid.ID

	decoders := []struct {
		name   string
		decode func() error
	}{
		{"typed ID", func() error { return typed.UnmarshalJSON(data) }},
		{"UUID", func() error { return uuid.UnmarshalJSON(data) }},
		{"ULID", func() error { return ulid.UnmarshalJSON(data) }},
		{"TypeID", func() error { return typeID.UnmarshalJSON(data) }},
		{"KSUID", func() error { return ksuid.UnmarshalJSON(data) }},
		{"NanoID", func() error { return nanoID.UnmarshalJSON(data) }},
	}
	for _, decoder := range decoders {
		t.Run(decoder.name, func(t *testing.T) {
			err := decoder.decode()
			assertSafeDiagnostic(t, err, identifier.ErrInvalid)
			if strings.Contains(err.Error(), "@") {
				t.Fatalf("diagnostic exposes invalid JSON token: %q", err)
			}
		})
	}
}

func TestConcreteIdentifierJSONRejectsOversizedInputBeforeDecoding(t *testing.T) {
	t.Parallel()

	decoders := []struct {
		name    string
		maximum string
		decode  func([]byte) error
	}{
		{"UUID", "017f22e2-79b0-7cc3-98c4-dc0c0c07398f", func(data []byte) error { var value identifieruuid.ID; return value.UnmarshalJSON(data) }},
		{"ULID", "01ARZ3NDEKTSV4RRFFQ69G5FAV", func(data []byte) error { var value identifierulid.ID; return value.UnmarshalJSON(data) }},
		{"TypeID", strings.Repeat("a", 63) + "_" + strings.Repeat("0", 26), func(data []byte) error { var value identifiertypeid.ID; return value.UnmarshalJSON(data) }},
		{"KSUID", "0ujtsYcgvSTl8PAuAdqWYSMnLOv", func(data []byte) error { var value identifierksuid.ID; return value.UnmarshalJSON(data) }},
		{"NanoID", strings.Repeat("_", identifiernanoid.DefaultSize), func(data []byte) error { var value identifiernanoid.ID; return value.UnmarshalJSON(data) }},
	}
	for _, decoder := range decoders {
		t.Run(decoder.name, func(t *testing.T) {
			if err := decoder.decode(escapedASCIIJSON(decoder.maximum)); err != nil {
				t.Fatalf("maximum encoded JSON error = %v", err)
			}

			err := decoder.decode(escapedASCIIJSON(decoder.maximum + "x"))
			assertSafeDiagnostic(t, err, identifier.ErrInvalid)
			if !strings.Contains(err.Error(), "JSON length is invalid") {
				t.Fatalf("oversized JSON error = %q", err)
			}
		})
	}
}

func TestConcreteIdentifierScannersRejectOversizedText(t *testing.T) {
	t.Parallel()

	scanners := []struct {
		name    string
		maximum string
		scan    func(any) error
	}{
		{"UUID", "017f22e2-79b0-7cc3-98c4-dc0c0c07398f", func(input any) error { var value identifieruuid.ID; return value.Scan(input) }},
		{"ULID", "01ARZ3NDEKTSV4RRFFQ69G5FAV", func(input any) error { var value identifierulid.ID; return value.Scan(input) }},
		{"TypeID", strings.Repeat("a", 63) + "_" + strings.Repeat("0", 26), func(input any) error { var value identifiertypeid.ID; return value.Scan(input) }},
		{"KSUID", "0ujtsYcgvSTl8PAuAdqWYSMnLOv", func(input any) error { var value identifierksuid.ID; return value.Scan(input) }},
		{"NanoID", strings.Repeat("_", identifiernanoid.DefaultSize), func(input any) error { var value identifiernanoid.ID; return value.Scan(input) }},
	}
	for _, scanner := range scanners {
		t.Run(scanner.name, func(t *testing.T) {
			for _, input := range []any{scanner.maximum, []byte(scanner.maximum)} {
				if err := scanner.scan(input); err != nil {
					t.Fatalf("maximum scanner input error = %v", err)
				}
			}
			oversized := scanner.maximum + "x"
			for _, input := range []any{oversized, []byte(oversized)} {
				assertSafeDiagnostic(t, scanner.scan(input), identifier.ErrInvalid)
			}
		})
	}
}

func escapedASCIIJSON(text string) []byte {
	var encoded strings.Builder
	encoded.Grow(len(text)*6 + 2)
	encoded.WriteByte('"')
	for _, character := range []byte(text) {
		_, _ = fmt.Fprintf(&encoded, `\u%04x`, character)
	}
	encoded.WriteByte('"')

	return []byte(encoded.String())
}

func TestClockRollbackDiagnosticsDoNotExposeTimestamps(t *testing.T) {
	t.Parallel()

	const current = int64(1_750_000_123_456)
	const earlier = current - 1
	clockFactories := []struct {
		name     string
		generate func() error
	}{
		{"UUIDv7", rollbackUUID(current, earlier)},
		{"ULID", rollbackULID(current, earlier)},
		{"KSUID", rollbackKSUID(current/1000, earlier/1000-1)},
	}
	for _, factory := range clockFactories {
		t.Run(factory.name, func(t *testing.T) {
			err := factory.generate()
			assertSafeDiagnostic(t, err, identifier.ErrClockRollback)
			for _, timestamp := range []int64{current, earlier, current / 1000, earlier/1000 - 1} {
				if strings.Contains(err.Error(), fmt.Sprint(timestamp)) {
					t.Fatalf("diagnostic exposes clock value: %q", err)
				}
			}
		})
	}
}

func rollbackUUID(current, earlier int64) func() error {
	generator := identifieruuid.NewV7Generator(
		&sequenceClock{times: []time.Time{time.UnixMilli(current), time.UnixMilli(earlier)}},
		strings.NewReader(strings.Repeat("x", 20)),
	)
	return func() error { _, _ = generator.New(); _, err := generator.New(); return err }
}

func rollbackULID(current, earlier int64) func() error {
	generator := identifierulid.NewGenerator(
		&sequenceClock{times: []time.Time{time.UnixMilli(current), time.UnixMilli(earlier)}},
		strings.NewReader(strings.Repeat("x", 20)),
	)
	return func() error { _, _ = generator.New(); _, err := generator.New(); return err }
}

func rollbackKSUID(current, earlier int64) func() error {
	generator := identifierksuid.NewGenerator(
		&sequenceClock{times: []time.Time{time.Unix(current, 0), time.Unix(earlier, 0)}},
		strings.NewReader(strings.Repeat("x", 32)),
	)
	return func() error { _, _ = generator.New(); _, err := generator.New(); return err }
}

func assertSafeDiagnostic(t *testing.T, err error, classification error) {
	t.Helper()
	if !errors.Is(err, classification) {
		t.Fatalf("error %q does not preserve %v", err, classification)
	}
	if strings.Contains(err.Error(), hostileDiagnostic) {
		t.Fatalf("diagnostic exposes attacker-controlled value: %q", err)
	}
	if len(err.Error()) > 256 {
		t.Fatalf("diagnostic length = %d, want at most 256", len(err.Error()))
	}
}

func assertCauseHidden(t *testing.T, err error, cause error) {
	t.Helper()
	if errors.Is(err, cause) {
		t.Fatalf("error chain exposes injected cause %T", cause)
	}
	var target *hostileCause
	if errors.As(err, &target) {
		t.Fatalf("error chain exposes injected cause type %T", target)
	}
}
