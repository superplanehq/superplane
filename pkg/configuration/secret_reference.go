package configuration

import (
	"fmt"
	"regexp"
	"strings"
)

// SecretReferenceRegex matches inline secret references of the form
// {{ secrets.NAME.KEY }}. NAME and KEY may contain any character except
// '.' (the separator) and '}' (the placeholder terminator).
var SecretReferenceRegex = regexp.MustCompile(`\{\{\s*secrets\.([^.}]+)\.([^.}]+?)\s*\}\}`)

// secretReferenceBodyRegex matches the body of a {{ ... }} placeholder that is
// a valid secret reference (secrets.NAME.KEY). It mirrors SecretReferenceRegex
// but anchors the whole (already trimmed) body so malformed references such as
// "secrets.foo" or "secrets.a.b.c" are rejected.
var secretReferenceBodyRegex = regexp.MustCompile(`^secrets\.[^.}]+\.[^.}]+$`)

// IsSecretReference reports whether the body of a {{ ... }} placeholder is a
// valid secret reference. The body must be the text between the braces; the
// surrounding {{ }} are not included. Only strict secrets.NAME.KEY references
// are accepted, so callers do not skip validation for placeholders that would
// never be resolved at execution time.
func IsSecretReference(body string) bool {
	return secretReferenceBodyRegex.MatchString(strings.TrimSpace(body))
}

// SecretLookup resolves an organization secret value by secret name and key.
type SecretLookup func(name, key string) ([]byte, error)

// ResolveSecretReferences replaces every {{ secrets.NAME.KEY }} occurrence in s
// with the value returned by lookup. Names and keys are trimmed of surrounding
// whitespace. The first lookup error aborts and is returned to the caller.
func ResolveSecretReferences(s string, lookup SecretLookup) (string, error) {
	if !SecretReferenceRegex.MatchString(s) {
		return s, nil
	}

	var lookupErr error
	result := SecretReferenceRegex.ReplaceAllStringFunc(s, func(match string) string {
		if lookupErr != nil {
			return match
		}

		groups := SecretReferenceRegex.FindStringSubmatch(match)
		if len(groups) != 3 {
			return match
		}

		name := strings.TrimSpace(groups[1])
		key := strings.TrimSpace(groups[2])
		value, err := lookup(name, key)
		if err != nil {
			lookupErr = fmt.Errorf("secret %q key %q: %w", name, key, err)
			return match
		}

		return string(value)
	})

	if lookupErr != nil {
		return "", lookupErr
	}

	return result, nil
}
