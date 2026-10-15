package identifier

import (
	"slices"
	"sort"
	"strings"
	"unicode/utf8"
)

func unicodeIDNAInfo(r rune) unicodeIDNA {
	i := sort.Search(len(unicode151IDNA), func(i int) bool { return unicode151IDNA[i].hi >= r })
	if i == len(unicode151IDNA) || unicode151IDNA[i].lo > r {
		return unicodeIDNA{status: 4}
	}
	return unicode151IDNA[i]
}
func unicodeBidiClass(r rune) string {
	i := sort.Search(len(unicode151Bidi), func(i int) bool { return unicode151Bidi[i].r >= r })
	if i < len(unicode151Bidi) && unicode151Bidi[i].r == r {
		return unicode151Bidi[i].class
	}
	return "L"
}
func unicodeJoiningClass(r rune) string {
	i := sort.Search(len(unicode151Joining), func(i int) bool { return unicode151Joining[i].hi >= r })
	if i < len(unicode151Joining) && unicode151Joining[i].lo <= r {
		return unicode151Joining[i].class
	}
	return "U"
}

// UTS46 nontransitional mapping, followed by NFC and the validity criteria.
// No runtime unicode.Version, locale or standard-library character property is
// consulted. The borrowed upstream RFC3492 codec sees only bounded labels.
func unicodeDomain(input string) (string, error) {
	var mapped strings.Builder
	for _, r := range input {
		info := unicodeIDNAInfo(r)
		switch info.status {
		case 0, 3, 4:
			// UTS46 revision31 Mapping leaves disallowed codepoints unchanged;
			// Convert/Validate checks them after NFC. For example '=' plus
			// an overlay composes to allowed U+2260 under the15.1 table.
			mapped.WriteRune(r)
		case 1:
			mapped.WriteString(info.value)
		case 2: // An explicitly ignored codepoint contributes no bytes.
		}
		if mapped.Len() > unicodeWorkRunes*4 {
			return "", ErrNormalization
		}
	}
	s, err := unicodeNormalize(mapped.String(), UnicodeNormalizationNFC)
	if err != nil || len(s) == 0 || len(s) > 4*254 {
		return "", ErrNormalization
	}
	labels := strings.Split(s, ".")
	root := strings.HasSuffix(s, ".")
	if root {
		labels = labels[:len(labels)-1]
	}
	bidi := false
	for i, label := range labels {
		if len(label) == 0 || utf8.RuneCountInString(label) > 63 {
			return "", ErrNormalization
		}
		if strings.HasPrefix(label, "xn--") {
			if len(label) > 63 {
				return "", ErrNormalization
			}
			label, err = punyDecode(label[4:])
			if err != nil || label == "" || utf8.RuneCountInString(label) == len(label) {
				return "", ErrNormalization
			}
			encoded, encodeErr := punyEncode("xn--", label)
			if encodeErr != nil || encoded != labels[i] {
				return "", ErrNormalization
			}
		}
		if !unicodeValidLabel(label) {
			return "", ErrNormalization
		}
		labels[i] = label
		for _, r := range label {
			class := unicodeBidiClass(r)
			bidi = bidi || class == "R" || class == "AL" || class == "AN"
		}
	}
	if bidi {
		for _, label := range labels {
			if !unicodeValidBidi(label) {
				return "", ErrNormalization
			}
		}
	}
	for i, label := range labels {
		if utf8.RuneCountInString(label) != len(label) {
			labels[i], err = punyEncode("xn--", label)
			if err != nil {
				return "", ErrNormalization
			}
		}
		if len(labels[i]) > 63 {
			return "", ErrNormalization
		}
	}
	result := strings.Join(labels, ".")
	if len(result) == 0 || len(result) > 253 {
		return "", ErrNormalization
	}
	if root {
		result += "."
	}
	return result, nil
}

func unicodeValidLabel(label string) bool {
	if label == "" || label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	normal, err := unicodeNormalize(label, UnicodeNormalizationNFC)
	if err != nil || normal != label {
		return false
	}
	runes := []rune(label)
	// UTS #46 positions are Unicode codepoints, not UTF-8 byte offsets.
	if len(runes) >= 4 && runes[2] == '-' && runes[3] == '-' {
		return false
	}
	if _, mark := slices.BinarySearch(unicode151Marks[:], runes[0]); mark {
		return false
	}
	for i, r := range runes {
		status := unicodeIDNAInfo(r).status
		if status != 0 && status != 3 {
			return false
		}
		if (r == 0x200c || r == 0x200d) && !unicodeValidJoiner(runes, i) {
			return false
		}
	}
	return true
}

// RFC5892 AppendixA ContextJ. A virama immediately before either joiner is
// sufficient; only ZWNJ has the additional joining-type alternative.
func unicodeValidJoiner(runes []rune, i int) bool {
	if i == 0 {
		return false
	}
	if unicodeCCC(runes[i-1]) == 9 {
		return true
	}
	if runes[i] == 0x200d {
		return false
	}
	left, right := i-1, i+1
	for left >= 0 && unicodeJoiningClass(runes[left]) == "T" {
		left--
	}
	for right < len(runes) && unicodeJoiningClass(runes[right]) == "T" {
		right++
	}
	if left < 0 || right == len(runes) {
		return false
	}
	l, r := unicodeJoiningClass(runes[left]), unicodeJoiningClass(runes[right])
	return (l == "L" || l == "D") && (r == "R" || r == "D")
}

// RFC5893 section2, applied to every label only in a bidi domain. Trailing
// nonspacing marks inherit the preceding terminal class; EN and AN cannot mix.
func unicodeValidBidi(label string) bool {
	runes := []rune(label)
	first := unicodeBidiClass(runes[0])
	rtl := first == "R" || first == "AL"
	if !rtl && first != "L" {
		return false
	}
	en, an := false, false
	last := first
	for _, r := range runes {
		class := unicodeBidiClass(r)
		if class != "NSM" {
			last = class
		}
		en, an = en || class == "EN", an || class == "AN"
		if rtl {
			switch class {
			case "R", "AL", "AN", "EN", "ES", "CS", "ET", "ON", "BN", "NSM":
			default:
				return false
			}
		} else {
			switch class {
			case "L", "EN", "ES", "CS", "ET", "ON", "BN", "NSM":
			default:
				return false
			}
		}
	}
	if rtl {
		return !(en && an) && (last == "R" || last == "AL" || last == "EN" || last == "AN")
	}
	return last == "L" || last == "EN"
}
