package slug

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxSourceRunes is the Laravel-compatible source limit applied before
// transliteration.
const MaxSourceRunes = 250

// LaravelEnglish reproduces Laravel 13 Str::slug source normalization with
// the default English profile used by spatie/laravel-sluggable 4.0.2.
//
// The function is deterministic and safe for concurrent use. It may return an
// empty string when the source has no supported letters or digits. Database
// uniqueness and numeric suffix selection remain persistence concerns.
// At most MaxSourceRunes UTF-8 decoded runes are considered before expansion.
// Unsupported or malformed UTF-8 is discarded; output contains only lowercase
// ASCII letters, digits and single interior hyphens. The frozen Laravel source
// is revision 138e5806ed7e4e21591948b661119d3810f052cf; normalization order is
// protected by TestLaravelEnglishMatchesFrozenLaravelDifferentialVectors.
func LaravelEnglish(source string) string {
	source = truncateSource(source)

	var transliterated strings.Builder
	offset := 0
	for range len(source) {
		if offset < len(source) {
			if replacement, consumed := longestReplacement(source[offset:]); consumed > 0 {
				transliterated.WriteString(replacement)
				offset += consumed
				continue
			}

			character, size := utf8.DecodeRuneInString(source[offset:])
			switch {
			case character == '\r' || character == '\n' || character == '\t':
				transliterated.WriteByte(' ')
			case printableASCII(character):
				transliterated.WriteRune(character)
			}
			offset += size
		}
	}

	normalized := strings.NewReplacer(
		"_", "-",
		"@", "-at-",
	).Replace(strings.ToLower(transliterated.String()))

	var result strings.Builder
	pendingSeparator := false
	for _, character := range normalized {
		switch {
		case character >= 'a' && character <= 'z',
			character >= '0' && character <= '9':
			if pendingSeparator {
				result.WriteByte('-')
			}
			result.WriteRune(character)
			pendingSeparator = false
		case character == '-' || unicode.IsSpace(character):
			pendingSeparator = result.Len() != 0
		}
	}

	return result.String()
}

func printableASCII(character rune) bool {
	return character >= 0x20 && character <= 0x7e
}

// Limit the source, not the transliterated result: one rune can expand to
// several output letters, so truncating afterward would change stored slugs
// (IDENTIFIER-SLUG-DEC-001; TestLaravelEnglishTruncatesTheSourceBeforeTransliteration).
func truncateSource(source string) string {
	offset := 0
	for range MaxSourceRunes {
		if offset == len(source) {
			return source
		}
		_, size := utf8.DecodeRuneInString(source[offset:])
		offset += size
	}

	return source[:offset]
}

// The generated table groups by first UTF-8 byte and orders longer sequences
// first. Preserve that ordering: matching a base character before its combining
// sequence changes the frozen profile. Regenerate from the pinned portable-ascii
// 2.1.1 source using slug/scripts/generate-table.php; the generator owns ordering.
func longestReplacement(source string) (string, int) {
	for _, candidate := range laravelEnglishReplacements[source[0]] {
		if !strings.HasPrefix(source, candidate.source) {
			continue
		}
		return candidate.value, len(candidate.source)
	}

	return "", 0
}
