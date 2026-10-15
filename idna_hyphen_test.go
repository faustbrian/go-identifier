package identifier_test

import (
	"errors"
	"strings"
	"testing"

	identifier "github.com/faustbrian/go-identifier/v2"
)

func TestIdentityIDNAHyphenPositionsUseCodepoints(t *testing.T) {
	for _, mode := range []identifier.UnicodeNormalization{identifier.UnicodeNormalizationNone, identifier.UnicodeNormalizationNFC, identifier.UnicodeNormalizationNFKC} {
		profile := fixtureProfile(t, mode, false, true)
		for _, test := range []struct {
			name  string
			text  string
			valid bool
		}{
			{"two-byte-valid", "é--a.example", true},
			{"three-byte-valid", "あ--a.example", true},
			{"two-byte-forbidden", "aé--b.example", false},
			{"three-byte-forbidden", "aあ--b.example", false},
			{"ascii-forbidden", "ab--c.example", false},
		} {
			t.Run(test.name, func(t *testing.T) {
				input, err := identifier.NewInput(test.text)
				if err != nil {
					t.Fatal("synthetic input construction failed")
				}
				value, err := identifier.Normalize(input, profile)
				if !test.valid {
					if !errors.Is(err, identifier.ErrNormalization) || value != (identifier.Canonical{}) {
						t.Fatal("codepoint positions three and four must deny with zero canonical custody")
					}
					return
				}
				if err != nil || !strings.HasPrefix(value.Record().Value, "xn--") {
					t.Fatal("allowed Unicode hyphen placement must produce its ASCII domain")
				}
				restored, err := identifier.RestoreCanonical(value.Record(), profile)
				if err != nil || !restored.Equal(value) {
					t.Fatal("allowed domain must remain a canonical fixed point")
				}
			})
		}
	}
}
