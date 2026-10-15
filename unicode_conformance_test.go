package identifier_test

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"

	identifier "github.com/faustbrian/go-identifier/v2"
)

func unicodeFixtureLines(t *testing.T, name string, visit func(int, []string)) {
	t.Helper()
	f, err := os.Open("testdata/unicode151/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if text == "" || strings.HasPrefix(text, "@") {
			continue
		}
		fields := strings.Split(text, ";")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		visit(line, fields)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}

func unicodeFixtureCodepoints(t *testing.T, text string) string {
	t.Helper()
	var out strings.Builder
	for _, value := range strings.Fields(text) {
		r, err := strconv.ParseUint(value, 16, 32)
		if err != nil || r > utf8.MaxRune {
			t.Fatal("invalid official fixture codepoint")
		}
		out.WriteRune(rune(r))
	}
	return out.String()
}

func fixtureProfile(t *testing.T, mode identifier.UnicodeNormalization, fold, domain bool) identifier.NormalizationProfile {
	t.Helper()
	p, err := identifier.NewNormalizationProfile(identifier.NormalizationProfileSpec{
		ID: "identity.official.fixture", Version: 1, Unicode: mode,
		CaseFold: fold, IDNA: domain, MaximumBytes: 4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUnicode151OfficialNormalizationCorpus(t *testing.T) {
	nfc := fixtureProfile(t, identifier.UnicodeNormalizationNFC, false, false)
	nfkc := fixtureProfile(t, identifier.UnicodeNormalizationNFKC, false, false)
	cases := 0
	unicodeFixtureLines(t, "NormalizationTest.txt", func(line int, fields []string) {
		if len(fields) < 5 {
			t.Fatal("incomplete official normalization row")
		}
		var columns [5]string
		for i := range columns {
			columns[i] = unicodeFixtureCodepoints(t, fields[i])
		}
		for i, text := range columns {
			for _, p := range []identifier.NormalizationProfile{nfc, nfkc} {
				want := columns[3]
				if p == nfc && i < 3 {
					want = columns[1]
				}
				input, err := identifier.NewInput(text)
				if err != nil {
					t.Fatalf("fixture row%d invalid public input", line)
				}
				value, err := identifier.Normalize(input, p)
				if err != nil || value.Record().Value != want {
					t.Fatalf("normalization corpus mismatch row%d column%d", line, i+1)
				}
				cases++
			}
		}
	})
	if cases < 190000 {
		t.Fatal("official normalization corpus was not exercised")
	}
}

func TestUnicode151OfficialFullFoldCorpus(t *testing.T) {
	p := fixtureProfile(t, identifier.UnicodeNormalizationNone, true, false)
	cases := 0
	unicodeFixtureLines(t, "CaseFolding.txt", func(line int, fields []string) {
		if fields[1] != "C" && fields[1] != "F" {
			return
		}
		text, want := unicodeFixtureCodepoints(t, fields[0]), unicodeFixtureCodepoints(t, fields[2])
		input, err := identifier.NewInput(text)
		if err != nil {
			t.Fatal(err)
		}
		value, err := identifier.Normalize(input, p)
		if err != nil || value.Record().Value != want {
			t.Fatalf("full-fold corpus mismatch row%d", line)
		}
		cases++
	})
	if cases < 1500 {
		t.Fatal("official full-fold corpus was not exercised")
	}
}

func unicodeFixtureEscapes(t *testing.T, s string) string {
	t.Helper()
	var out strings.Builder
	for len(s) != 0 {
		if strings.HasPrefix(s, `\u`) || strings.HasPrefix(s, `\x{`) {
			n, start := 6, 2
			if strings.HasPrefix(s, `\x{`) {
				n = strings.Index(s, "}") + 1
				start = 3
			}
			end := n
			if start == 3 {
				end--
			}
			if n <= start || n > len(s) {
				t.Fatal("invalid official fixture escape")
			}
			v, err := strconv.ParseUint(s[start:end], 16, 32)
			if err != nil || v > utf8.MaxRune {
				t.Fatal("invalid official fixture escape")
			}
			r := rune(v)
			s = s[n:]
			if r >= 0xd800 && r <= 0xdbff && len(s) >= 6 && strings.HasPrefix(s, `\u`) {
				low, parseErr := strconv.ParseUint(s[2:6], 16, 32)
				if parseErr == nil && low >= 0xdc00 && low <= 0xdfff {
					r = utf16.DecodeRune(r, rune(low))
					s = s[6:]
				}
			}
			out.WriteRune(r)
		} else {
			r, n := utf8.DecodeRuneInString(s)
			out.WriteRune(r)
			s = s[n:]
		}
	}
	return out.String()
}

func TestUnicode151OfficialNontransitionalIDNACorpus(t *testing.T) {
	// The IDNA stage owns its mandated NFC. None avoids applying an additional
	// generic form before UTS46, exactly matching this independent official oracle.
	p := fixtureProfile(t, identifier.UnicodeNormalizationNone, false, true)
	cases := 0
	unicodeFixtureLines(t, "IdnaTestV2.txt", func(line int, fields []string) {
		if len(fields) < 7 {
			t.Fatal("incomplete official IDNA row")
		}
		source := unicodeFixtureEscapes(t, fields[0])
		unicode := fields[1]
		if unicode == "" {
			unicode = fields[0]
		}
		ascii := fields[3]
		if ascii == "" {
			ascii = unicode
		}
		status := fields[4]
		if status == "" {
			status = fields[2]
		}
		valid := status == "" || status == "[]"
		input, err := identifier.NewInput(source)
		var value identifier.Canonical
		if err == nil {
			value, err = identifier.Normalize(input, p)
		}
		if valid {
			if err != nil || value.Record().Value != unicodeFixtureEscapes(t, ascii) {
				t.Fatalf("IDNA corpus valid-result mismatch row%d", line)
			}
		} else if err == nil || !errors.Is(err, identifier.ErrNormalization) && !errors.Is(err, identifier.ErrInvalid) || value != (identifier.Canonical{}) {
			t.Fatalf("IDNA corpus invalid-result accepted row%d", line)
		}
		cases++
	})
	if cases < 6000 {
		t.Fatal("official IDNA corpus was not exercised")
	}
}

func TestUnicode151DoesNotAdoptNewerToolchainMappings(t *testing.T) {
	p := fixtureProfile(t, identifier.UnicodeNormalizationNFKC, true, false)
	// Unicode16 added the Cyrillic TJE pair; Unicode17 outlined Latin letters
	// have compatibility mappings. Both are identity transforms under15.1.
	for _, s := range []string{"\u1c89", "\U0001ccd6"} {
		input, err := identifier.NewInput(s)
		if err != nil {
			t.Fatal(err)
		}
		value, err := identifier.Normalize(input, p)
		if err != nil || value.Record().Value != s {
			t.Fatal("toolchain changed the pinned profile")
		}
	}
	domain := fixtureProfile(t, identifier.UnicodeNormalizationNone, false, true)
	// CJK ExtensionI entered15.1, not15.0; admitting it prevents relabeling an
	// older IDNA table as15.1. Newer Unicode16 Cyrillic TJE remains prohibited.
	input, err := identifier.NewInput("\U0002ebf0.example")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := identifier.Normalize(input, domain); err != nil {
		t.Fatal("Unicode15.1 domain character was rejected")
	}
	input, err = identifier.NewInput("\u1c89.example")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := identifier.Normalize(input, domain); !errors.Is(err, identifier.ErrNormalization) {
		t.Fatal("newer Unicode domain character was admitted")
	}
}
