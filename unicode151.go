package identifier

import (
	"slices"
	"sort"
	"strings"
)

// Generated arrays have one synchronization owner: immutable process-lifetime
// initialization. No code mutates them; every transform owns its working memory.
type unicodeDecomposition struct {
	r             rune
	compatibility bool
	value         string
}
type unicodeComposition struct {
	pair uint64
	r    rune
}
type unicodeClass struct {
	r     rune
	class uint8
}
type unicodeMapping struct {
	r     rune
	value string
}
type unicodeRange struct{ lo, hi rune }
type unicodeBidi struct {
	r     rune
	class string
}
type unicodeJoining struct {
	lo, hi rune
	class  string
}
type unicodeIDNA struct {
	lo, hi rune
	status uint8
	value  string
}

// Full compatibility decomposition expands at most18 runes per source rune in
// the pinned data; folding expands at most3. The extra hard ceiling bounds
// intermediate allocations independently of the caller's smaller final limit.
const unicodeWorkRunes = 4096 * 18 * 3

func unicodeCCC(r rune) uint8 {
	i := sort.Search(len(unicode151Classes), func(i int) bool { return unicode151Classes[i].r >= r })
	if i < len(unicode151Classes) && unicode151Classes[i].r == r {
		return unicode151Classes[i].class
	}
	return 0
}
func unicodeSpace(r rune) bool {
	i := sort.Search(len(unicode151Spaces), func(i int) bool { return unicode151Spaces[i].hi >= r })
	return i < len(unicode151Spaces) && unicode151Spaces[i].lo <= r
}
func unicodeFold(s string) (string, error) {
	var out strings.Builder
	for _, r := range s {
		i := sort.Search(len(unicode151Folds), func(i int) bool { return unicode151Folds[i].r >= r })
		if i < len(unicode151Folds) && unicode151Folds[i].r == r {
			out.WriteString(unicode151Folds[i].value)
		} else {
			out.WriteRune(r)
		}
		if out.Len() > unicodeWorkRunes*4 {
			return "", ErrNormalization
		}
	}
	return out.String(), nil
}

func unicodeDecompose(out []rune, r rune, compatibility bool, depth int) ([]rune, error) {
	if depth > 32 || len(out) >= unicodeWorkRunes {
		return nil, ErrNormalization
	}
	// UAX15 algorithmic Hangul decomposition; these constants never depend on Go.
	if r >= 0xac00 && r < 0xac00+11172 {
		s := r - 0xac00
		out = append(out, 0x1100+s/588, 0x1161+s%588/28)
		if s%28 != 0 {
			out = append(out, 0x11a7+s%28)
		}
		return out, nil
	}
	i := sort.Search(len(unicode151Decompositions), func(i int) bool { return unicode151Decompositions[i].r >= r })
	if i == len(unicode151Decompositions) || unicode151Decompositions[i].r != r || unicode151Decompositions[i].compatibility && !compatibility {
		return append(out, r), nil
	}
	for _, d := range unicode151Decompositions[i].value {
		var err error
		out, err = unicodeDecompose(out, d, compatibility, depth+1)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func unicodeCompose(a, b rune) (rune, bool) {
	if a < 0 || b < 0 {
		return 0, false
	}
	if a >= 0x1100 && a < 0x1100+19 && b >= 0x1161 && b < 0x1161+21 {
		return 0xac00 + (a-0x1100)*588 + (b-0x1161)*28, true
	}
	if a >= 0xac00 && a < 0xac00+11172 && (a-0xac00)%28 == 0 && b > 0x11a7 && b < 0x11a7+28 {
		return a + b - 0x11a7, true
	}
	pair := uint64(a)<<21 | uint64(b)
	i := sort.Search(len(unicode151Compositions), func(i int) bool { return unicode151Compositions[i].pair >= pair })
	if i < len(unicode151Compositions) && unicode151Compositions[i].pair == pair {
		return unicode151Compositions[i].r, true
	}
	return 0, false
}

// UAX15 decomposition, stable canonical ordering, and blocked composition.
// Unlike stream-safe transforms, no CGJ is inserted into a valid combining run.
func unicodeNormalize(s string, mode UnicodeNormalization) (string, error) {
	if mode == UnicodeNormalizationNone {
		return s, nil
	}
	var decomposed []rune
	for _, r := range s {
		var err error
		decomposed, err = unicodeDecompose(decomposed, r, mode == UnicodeNormalizationNFKC, 0)
		if err != nil {
			return "", err
		}
	}
	for start := 0; start < len(decomposed); {
		end := start + 1
		if unicodeCCC(decomposed[start]) == 0 {
			start++
		}
		for end < len(decomposed) && unicodeCCC(decomposed[end]) != 0 {
			end++
		}
		slices.SortStableFunc(decomposed[start:end], func(a, b rune) int { return int(unicodeCCC(a)) - int(unicodeCCC(b)) })
		start = end
	}
	out := make([]rune, 0, len(decomposed))
	starter, previous := -1, uint8(0)
	for _, r := range decomposed {
		class := unicodeCCC(r)
		if starter >= 0 && (previous < class || previous == 0) {
			if joined, ok := unicodeCompose(out[starter], r); ok {
				out[starter] = joined
				continue
			}
		}
		if class == 0 {
			starter = len(out)
		}
		out = append(out, r)
		previous = class
	}
	return string(out), nil
}
