// Package slug provides the frozen Laravel 13 English slug profile used by
// spatie/laravel-sluggable 4.0.2. Start with LaravelEnglish for deterministic
// normalization; no PHP runtime, cgo, service, or background work is required.
// The package owns normalization, not persistence uniqueness or suffix selection.
//
// Slugs are public lookup identifiers, not authorization facts or secrets.
// The pinned source, transliteration data and differential corpus are recorded
// in specification/manifest.tsv and IDENTIFIER-SLUG-DEC-001 in
// docs/specification-decisions.md; newer live Laravel dependency graphs are not
// implied by this compatibility profile.
package slug
