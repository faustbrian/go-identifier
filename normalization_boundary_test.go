package identifier_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	identifier "github.com/faustbrian/go-identifier/v2"
)

func TestIdentityProfileAndReferenceBounds(t *testing.T) {
	base := identifier.NormalizationProfileSpec{ID: "identity.bounds", Version: 1, Unicode: identifier.UnicodeNormalizationNFC, MaximumBytes: 4096}
	for _, change := range []func(*identifier.NormalizationProfileSpec){
		func(p *identifier.NormalizationProfileSpec) { p.ID = "" },
		func(p *identifier.NormalizationProfileSpec) { p.ID = ".name" },
		func(p *identifier.NormalizationProfileSpec) { p.ID = "Upper" },
		func(p *identifier.NormalizationProfileSpec) { p.ID = "has space" },
		func(p *identifier.NormalizationProfileSpec) {
			p.ID = identifier.NormalizationProfileID(strings.Repeat("x", 129))
		},
		func(p *identifier.NormalizationProfileSpec) { p.Version = 0 },
		func(p *identifier.NormalizationProfileSpec) { p.Unicode = identifier.UnicodeNormalizationUnspecified },
		func(p *identifier.NormalizationProfileSpec) { p.MaximumBytes = 0 },
		func(p *identifier.NormalizationProfileSpec) { p.MaximumBytes = 4097 },
	} {
		spec := base
		change(&spec)
		p, err := identifier.NewNormalizationProfile(spec)
		if !errors.Is(err, identifier.ErrInvalid) || p != (identifier.NormalizationProfile{}) {
			t.Fatal("invalid profile became usable")
		}
	}
	unknown := base
	unknown.Unicode = 255
	if _, err := identifier.NewNormalizationProfile(unknown); !errors.Is(err, identifier.ErrUnsupported) {
		t.Fatal("unknown Unicode mode did not report unsupported")
	}
	base.ID = identifier.NormalizationProfileID(strings.Repeat("x", 128))
	p, err := identifier.NewNormalizationProfile(base)
	if err != nil {
		t.Fatal(err)
	}
	value, err := identifier.Normalize("é", p)
	if err != nil {
		t.Fatal(err)
	}
	if (identifier.Canonical{}).Equal(identifier.Canonical{}) {
		t.Fatal("zero canonical granted equality")
	}
	for _, kind := range []string{"x", strings.Repeat("x", 64)} {
		ref, err := identifier.NewReference(kind, value, strings.Repeat("é", 2048))
		if err != nil || ref.Kind != kind || ref.Display != strings.Repeat("é", 2048) || !ref.Canonical.Equal(value) {
			t.Fatal("reference lost validated fields")
		}
	}
	for _, kind := range []string{"", strings.Repeat("x", 65), "Upper", "bad:kind"} {
		if _, err := identifier.NewReference(kind, value, ""); !errors.Is(err, identifier.ErrInvalid) {
			t.Fatal("invalid kind accepted")
		}
	}
	for _, display := range []string{strings.Repeat("x", 4097), string([]byte{0xff})} {
		if _, err := identifier.NewReference("name", value, display); !errors.Is(err, identifier.ErrInvalid) {
			t.Fatal("invalid display accepted")
		}
	}
	if _, err := identifier.NewReference("name", identifier.Canonical{}, ""); !errors.Is(err, identifier.ErrInvalid) {
		t.Fatal("zero reference accepted")
	}
	if _, err := identifier.Normalize("x", identifier.NormalizationProfile{}); !errors.Is(err, identifier.ErrInvalid) {
		t.Fatal("zero profile accepted")
	}
}

func TestIdentityFinalBoundsAndRestoreRejectForgery(t *testing.T) {
	for _, test := range []struct {
		text       string
		max        uint32
		fold, trim bool
		valid      bool
	}{
		{"é", 2, false, false, true}, {"é", 1, false, false, false},
		{"ß", 2, true, false, true}, {"ß", 1, true, false, false},
		{"e\u0301", 2, false, false, true}, {"  ", 4, false, true, false},
	} {
		p, err := identifier.NewNormalizationProfile(identifier.NormalizationProfileSpec{ID: "identity.bound", Version: 1, Unicode: identifier.UnicodeNormalizationNFC, MaximumBytes: test.max, CaseFold: test.fold, TrimSpace: test.trim})
		if err != nil {
			t.Fatal(err)
		}
		input, err := identifier.NewInput(test.text)
		if err != nil {
			t.Fatal(err)
		}
		value, err := identifier.Normalize(input, p)
		if test.valid {
			if err != nil || len(value.Record().Value) > int(test.max) {
				t.Fatal("valid final size rejected")
			}
		} else if !errors.Is(err, identifier.ErrNormalization) || value != (identifier.Canonical{}) {
			t.Fatal("unusable transformed value accepted")
		}
	}
	p := fixtureProfile(t, identifier.UnicodeNormalizationNFC, false, false)
	value, err := identifier.Normalize("é", p)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*identifier.CanonicalRecord){
		func(r *identifier.CanonicalRecord) { r.Value = "e\u0301" },
		func(r *identifier.CanonicalRecord) { r.Value = "" },
		func(r *identifier.CanonicalRecord) { r.Value = string([]byte{0xff}) },
		func(r *identifier.CanonicalRecord) { r.Value = strings.Repeat("x", 4097) },
		func(r *identifier.CanonicalRecord) { r.ProfileID = "unknown" },
		func(r *identifier.CanonicalRecord) { r.ProfileVersion++ },
	} {
		record := value.Record()
		change(&record)
		if result, restoreErr := identifier.RestoreCanonical(record, p); !errors.Is(restoreErr, identifier.ErrInvalid) || result != (identifier.Canonical{}) {
			t.Fatal("forged record restored identity")
		}
	}
	// Same bytes with a different ID also remain distinct.
	other, err := identifier.NewNormalizationProfile(identifier.NormalizationProfileSpec{ID: "different.id", Version: 1, Unicode: identifier.UnicodeNormalizationNone, MaximumBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	different, err := identifier.Normalize("é", other)
	if err != nil || value.Equal(different) {
		t.Fatal("profile ID was ignored")
	}
}

func TestIdentitySensitiveDiagnosticsAndExplicitProjection(t *testing.T) {
	marker := "synthetic-sensitive-identity"
	input, err := identifier.NewInput(marker)
	if err != nil {
		t.Fatal(err)
	}
	p := fixtureProfile(t, identifier.UnicodeNormalizationNone, false, false)
	value, err := identifier.Normalize(input, p)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := identifier.NewReference("name", value, marker)
	if err != nil {
		t.Fatal(err)
	}
	record := value.Record()
	for _, item := range []any{input, &input, value, &value, ref, &ref, record, &record} {
		for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			text := fmt.Sprintf(verb, item)
			if strings.Contains(text, marker) || !strings.Contains(text, "redacted") {
				t.Fatal("diagnostic disclosed identifier")
			}
		}
		var output strings.Builder
		logger := slog.New(slog.NewJSONHandler(&output, nil))
		logger.Info("synthetic", "value", item)
		if strings.Contains(output.String(), marker) {
			t.Fatal("structured log disclosed identifier")
		}
	}
	for _, item := range []any{input, &input, value, &value, ref, &ref} {
		if data, encodeErr := json.Marshal(item); !errors.Is(encodeErr, identifier.ErrUnsupported) || len(data) != 0 || strings.Contains(encodeErr.Error(), marker) {
			t.Fatal("implicit serialization disclosed identifier")
		}
	}
	for _, item := range []interface{ MarshalText() ([]byte, error) }{input, &input, value, &value, ref, &ref} {
		data, textErr := item.MarshalText()
		if !errors.Is(textErr, identifier.ErrUnsupported) || len(data) != 0 {
			t.Fatal("text serialization disclosed identifier")
		}
	}
	data, err := json.Marshal(value.Record())
	if err != nil {
		t.Fatal(err)
	}
	var restored identifier.CanonicalRecord
	if decodeErr := json.Unmarshal(data, &restored); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	roundtrip, err := identifier.RestoreCanonical(restored, p)
	if err != nil || !roundtrip.Equal(value) {
		t.Fatal("explicit versioned projection could not round trip")
	}
}

func TestIdentityImmutableProfileConcurrentReuse(t *testing.T) {
	p := fixtureProfile(t, identifier.UnicodeNormalizationNFKC, true, false)
	want, err := identifier.Normalize("Straße Ｓ", p)
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			for range 64 {
				value, err := identifier.Normalize("Straße Ｓ", p)
				if err != nil || !value.Equal(want) {
					t.Error("concurrent policy reuse changed identity")
					return
				}
				restored, err := identifier.RestoreCanonical(value.Record(), p)
				if err != nil || !value.Equal(restored) {
					t.Error("concurrent restore changed identity")
					return
				}
			}
		})
	}
	workers.Wait()
}

func TestIdentityDomainLengthsAndCombiningRuns(t *testing.T) {
	p, err := identifier.NewNormalizationProfile(identifier.NormalizationProfileSpec{
		ID: "domain.bounds", Version: 1, Unicode: identifier.UnicodeNormalizationNone,
		IDNA: true, MaximumBytes: 4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	domain := strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 61)
	for _, text := range []string{domain, domain + ".", "example."} {
		value, normalizeErr := identifier.Normalize(identifier.Input(text), p)
		if normalizeErr != nil || value.Record().Value != text {
			t.Fatal("valid DNS boundary lost bytes or root dot")
		}
	}
	for _, text := range []string{domain + "x", domain + "x.", strings.Repeat("a", 64) + ".example", ".example", "example..", "user@example", "xn--", "-example", "example-"} {
		value, normalizeErr := identifier.Normalize(identifier.Input(text), p)
		if !errors.Is(normalizeErr, identifier.ErrNormalization) || value != (identifier.Canonical{}) {
			t.Fatal("invalid domain became usable")
		}
	}
	// More than a stream-safe segment: normalization must not insert CGJ.
	text := "a" + strings.Repeat("\u0315\u0300", 100)
	nfc := fixtureProfile(t, identifier.UnicodeNormalizationNFC, false, false)
	value, err := identifier.Normalize(identifier.Input(text), nfc)
	want := "à" + strings.Repeat("\u0300", 99) + strings.Repeat("\u0315", 100)
	if err != nil || value.Record().Value != want {
		t.Fatal("long combining run was truncated, rewritten or incorrectly ordered")
	}
}

func TestIdentityPolicyMigrationExposesCollisions(t *testing.T) {
	spec := identifier.NormalizationProfileSpec{ID: "account.migration", Version: 1, Unicode: identifier.UnicodeNormalizationNone, MaximumBytes: 4096}
	exact, err := identifier.NewNormalizationProfile(spec)
	if err != nil {
		t.Fatal(err)
	}
	spec.Version = 2
	spec.Unicode = identifier.UnicodeNormalizationNFKC
	spec.CaseFold = true
	folded, err := identifier.NewNormalizationProfile(spec)
	if err != nil {
		t.Fatal(err)
	}
	left, err := identifier.Normalize("Straße", exact)
	if err != nil {
		t.Fatal(err)
	}
	right, err := identifier.Normalize("STRASSE", exact)
	if err != nil || left.Equal(right) {
		t.Fatal("exact profile lost distinct identities")
	}
	newLeft, err := identifier.Normalize("Straße", folded)
	if err != nil {
		t.Fatal(err)
	}
	newRight, err := identifier.Normalize("STRASSE", folded)
	if err != nil || !newLeft.Equal(newRight) {
		t.Fatal("folding migration failed to expose collision")
	}
	if left.Equal(newLeft) || right.Equal(newRight) {
		t.Fatal("mixed policies silently reinterpreted identity")
	}
}

func FuzzIdentityNormalization(f *testing.F) {
	for _, text := range []string{"e\u0301", "Straße", "BÜCHER.example", "é--a.example", "aé--b.example", "あ--a.example", "aあ--b.example", "\u200c", "\u1c89", "\U0001ccd6", "  x  ", string([]byte{0xff})} {
		for _, flags := range []uint8{0, 1, 2, 4, 8, 16} {
			f.Add(text, flags, uint16(4095))
		}
	}
	f.Fuzz(func(t *testing.T, text string, flags uint8, limit uint16) {
		input, inputErr := identifier.NewInput(text)
		p, err := identifier.NewNormalizationProfile(identifier.NormalizationProfileSpec{ID: "identity.fuzz", Version: 1, Unicode: identifier.UnicodeNormalization(1 + flags%3), CaseFold: flags&4 != 0, TrimSpace: flags&8 != 0, IDNA: flags&16 != 0, MaximumBytes: 1 + uint32(limit)%4096})
		if err != nil {
			t.Fatal("bounded fuzz profile rejected")
		}
		value, err := identifier.Normalize(identifier.Input(text), p)
		if inputErr != nil {
			if !errors.Is(err, identifier.ErrInvalid) || input != "" || value != (identifier.Canonical{}) {
				t.Fatal("invalid fuzz input bypassed validation")
			}
			return
		}
		if err != nil {
			if value != (identifier.Canonical{}) {
				t.Fatal("failure exposed usable identity")
			}
			return
		}
		record := value.Record()
		if record.Value == "" || len(record.Value) > int(1+uint32(limit)%4096) || !utf8.ValidString(record.Value) || record.ProfileID != "identity.fuzz" || record.ProfileVersion != 1 {
			t.Fatal("fuzz result broke canonical invariants")
		}
		restored, err := identifier.RestoreCanonical(record, p)
		if err != nil || !value.Equal(restored) {
			t.Fatal("fuzz result was not a fixed point")
		}
	})
}
