package pathfilter

import (
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// TrimNonEmptyStrings returns trimmed elements, skipping those that trim to empty.
func TrimNonEmptyStrings(elems []string) []string {
	var out []string
	for _, s := range elems {
		t := strings.TrimSpace(s)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}

// splitIncludesExcludes splits trimmed patterns into positive globs and exclude globs
// (leading "!" stripped). Does not add implicit "**".
func splitIncludesExcludes(patterns []string) (includes []string, excludes []string) {
	for _, raw := range patterns {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "!") {
			exc := strings.TrimPrefix(p, "!")
			exc = strings.TrimSpace(exc)
			if exc != "" {
				excludes = append(excludes, exc)
			}
			continue
		}
		includes = append(includes, p)
	}
	return includes, excludes
}

func filterValidGlobs(patterns []string, onInvalid func(pattern string)) []string {
	var out []string
	for _, p := range patterns {
		if doublestar.ValidatePattern(p) {
			out = append(out, p)
			continue
		}
		if onInvalid != nil {
			onInvalid(p)
		}
	}
	return out
}

// EvaluatePushPathGlobFilter returns true if the push should pass the path gate.
//
// patterns should already be trimmed (callers normally use TrimNonEmptyStrings first).
// If patterns is empty, the function returns true (no path restriction) so accidental
// misuse does not block every webhook; github.onPush only calls when len(patterns) > 0.
//
// onBypassFilter, if non-nil, is invoked when the path filter is not applied because
// configured patterns are unusable (fail-open), with a short English reason for logs.
//
// Semantics:
//   - If the user configured positive globs but none are valid, and there are no
//     valid excludes, the filter is bypassed (returns true).
//   - If every positive glob is invalid but at least one exclude is valid,
//     an implicit include of "**" is used so excludes still apply.
//   - If, after validation, there is no effective restriction (no valid includes
//     and no valid excludes), the filter is bypassed (returns true).
//   - Otherwise: true when some changed file matches a valid include and matches
//     no valid exclude. Exclude-only lists use an implicit include of "**".
//
// onMatchError is called when doublestar.Match returns an error (should be rare
// after ValidatePattern).
func EvaluatePushPathGlobFilter(
	patterns []string,
	changedFiles []string,
	onInvalidPattern func(pattern string),
	onMatchError func(pattern string, err error),
	onBypassFilter func(reason string),
) bool {
	if len(patterns) == 0 {
		return true
	}

	incRaw, excRaw := splitIncludesExcludes(patterns)
	validInc := filterValidGlobs(incRaw, onInvalidPattern)
	validExc := filterValidGlobs(excRaw, onInvalidPattern)

	// User configured positive globs but every one is invalid, and there is no
	// working exclude — fail-open so a typo does not drop every event.
	if len(incRaw) > 0 && len(validInc) == 0 && len(validExc) == 0 {
		if onBypassFilter != nil {
			onBypassFilter("all include globs invalid and no valid excludes; path filter disabled for this event")
		}
		return true
	}

	includes := validInc
	excludes := validExc
	if len(includes) == 0 && len(excludes) > 0 {
		includes = []string{"**"}
	}
	if len(includes) == 0 && len(excludes) == 0 {
		if onBypassFilter != nil {
			onBypassFilter("no valid path globs after validation; path filter disabled for this event")
		}
		return true
	}

	if len(changedFiles) == 0 {
		return false
	}

	for _, path := range changedFiles {
		if path == "" {
			continue
		}
		if !matchesAnyPattern(includes, path, onMatchError) {
			continue
		}
		if matchesAnyPattern(excludes, path, onMatchError) {
			continue
		}
		return true
	}

	return false
}

// changedFilesMatchPushPathGlobs returns true when at least one entry in
// changedFiles satisfies the glob filter described by patterns.
//
// Unlike EvaluatePushPathGlobFilter with an empty pattern list (which passes
// every event), an all-blank pattern list after trim returns false.
//
// Patterns follow GitHub Actions style for path filters:
//   - Use glob syntax (see doublestar / bash globstar).
//   - A pattern starting with "!" is an exclude (the "!" is stripped before matching).
//   - If every pattern is an exclude, an implicit include of "**" is assumed so
//     negation-only lists still make sense (e.g. "!docs/**").
func changedFilesMatchPushPathGlobs(patterns []string, changedFiles []string, onBadPattern func(pattern string, err error)) bool {
	trimmed := TrimNonEmptyStrings(patterns)
	if len(trimmed) == 0 {
		return false
	}

	return EvaluatePushPathGlobFilter(
		trimmed,
		changedFiles,
		func(pat string) {
			if onBadPattern != nil {
				onBadPattern(pat, doublestar.ErrBadPattern)
			}
		},
		onBadPattern,
		nil,
	)
}

func matchesAnyPattern(patterns []string, path string, onBadPattern func(pattern string, err error)) bool {
	for _, pat := range patterns {
		ok, err := doublestar.Match(pat, path)
		if err != nil {
			if onBadPattern != nil {
				onBadPattern(pat, err)
			}
			continue
		}
		if ok {
			return true
		}
	}
	return false
}
