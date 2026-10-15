package identifier_test

import (
	"errors"
	"strings"
	"testing"

	identifier "github.com/faustbrian/go-identifier/v2"
)

func TestIdentityInputPreservesBytesAndRejectsInvalidBounds(t *testing.T) {
	for _, text := range []string{"x", " e\u0301 ", strings.Repeat("é", 2048)} {
		input, err := identifier.NewInput(text)
		if err != nil || string(input) != text {
			t.Fatal("input changed before selecting its profile")
		}
	}
	for _, text := range []string{"", strings.Repeat("x", 4097), string([]byte{0xff})} {
		input, err := identifier.NewInput(text)
		if !errors.Is(err, identifier.ErrInvalid) || input != "" {
			t.Fatal("invalid input became usable")
		}
	}
}

func TestIdentityCanonicalRetainsExactProfileAndRequiresFixedPoint(t *testing.T) {
	spec := identifier.NormalizationProfileSpec{
		ID: "identity.test.raw", Version: 1,
		Unicode: identifier.UnicodeNormalizationNone, MaximumBytes: 4096,
	}
	profile, err := identifier.NewNormalizationProfile(spec)
	if err != nil {
		t.Fatal(err)
	}
	input, err := identifier.NewInput(" synthetic-identity-é ")
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := identifier.Normalize(input, profile)
	if err != nil {
		t.Fatal(err)
	}
	want := identifier.CanonicalRecord{Value: string(input), ProfileID: spec.ID, ProfileVersion: 1}
	if canonical.Record() != want {
		t.Fatal("canonical projection lost bytes or profile binding")
	}
	restored, err := identifier.RestoreCanonical(want, profile)
	if err != nil || !canonical.Equal(restored) {
		t.Fatal("exact profile round trip changed identity")
	}
	// Construction must copy the spec; a caller cannot mutate a live profile.
	spec.MaximumBytes = 1
	spec.Version = 2
	next, err := identifier.Normalize(input, profile)
	if err != nil || !canonical.Equal(next) {
		t.Fatal("caller mutation changed immutable profile")
	}
	other, err := identifier.NewNormalizationProfile(identifier.NormalizationProfileSpec{
		ID: want.ProfileID, Version: 2,
		Unicode: identifier.UnicodeNormalizationNone, MaximumBytes: 4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	otherValue, err := identifier.Normalize(input, other)
	if err != nil || canonical.Equal(otherValue) {
		t.Fatal("profile version was not part of canonical identity")
	}
	if _, err := identifier.RestoreCanonical(want, other); !errors.Is(err, identifier.ErrInvalid) {
		t.Fatal("wrong profile restored canonical bytes")
	}
	projection := canonical.Record()
	projection.Value = "changed"
	if projection.Value != "changed" || canonical.Record() != want {
		t.Fatal("record mutation changed canonical ownership")
	}
	for _, forged := range []identifier.Input{"", identifier.Input(strings.Repeat("x", 4097)), identifier.Input(string([]byte{0xff}))} {
		value, err := identifier.Normalize(forged, profile)
		if !errors.Is(err, identifier.ErrInvalid) || value != (identifier.Canonical{}) {
			t.Fatal("defined-string conversion bypassed input validation")
		}
	}
}

func TestIdentityUnicodeProfilesHaveFixedObservableSemantics(t *testing.T) {
	for _, test := range []struct {
		name, input, want  string
		mode               identifier.UnicodeNormalization
		fold, trim, domain bool
	}{
		{"none", "e\u0301ﬃ", "e\u0301ﬃ", identifier.UnicodeNormalizationNone, false, false, false},
		{"nfc", "e\u0301ﬃ", "éﬃ", identifier.UnicodeNormalizationNFC, false, false, false},
		{"nfkc", "e\u0301ﬃ", "éffi", identifier.UnicodeNormalizationNFKC, false, false, false},
		{"hangul", "\u1100\u1161\u11a8", "각", identifier.UnicodeNormalizationNFC, false, false, false},
		{"fold", "Straße Σς İ", "strasse σσ i\u0307", identifier.UnicodeNormalizationNFC, true, false, false},
		{"trim", "\u2003 name interior \u00a0", "name interior", identifier.UnicodeNormalizationNone, false, true, false},
		{"domain", "BÜCHER.example", "xn--bcher-kva.example", identifier.UnicodeNormalizationNFC, false, false, true},
		{"nontransitional", "faß.de", "xn--fa-hia.de", identifier.UnicodeNormalizationNFC, false, false, true},
		{"mapped-dot", "ＥＸＡＭＰＬＥ。COM", "example.com", identifier.UnicodeNormalizationNFC, false, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			profile, err := identifier.NewNormalizationProfile(identifier.NormalizationProfileSpec{
				ID: identifier.NormalizationProfileID("identity.test." + test.name), Version: 1,
				Unicode: test.mode, CaseFold: test.fold, TrimSpace: test.trim, IDNA: test.domain, MaximumBytes: 4096,
			})
			if err != nil {
				t.Fatal(err)
			}
			input, err := identifier.NewInput(test.input)
			if err != nil {
				t.Fatal(err)
			}
			value, err := identifier.Normalize(input, profile)
			if err != nil || value.Record().Value != test.want {
				t.Fatal("normalization changed pinned expected bytes", err)
			}
			restored, err := identifier.RestoreCanonical(value.Record(), profile)
			if err != nil || !value.Equal(restored) {
				t.Fatal("normalization was not a fixed point", err)
			}
		})
	}
}
