package models

import (
	"fmt"
	"slices"
)

// ValidateAllowedOAuthProviders returns an error if any value is not a supported OAuth provider name.
func ValidateAllowedOAuthProviders(providers []string) error {
	for _, p := range providers {
		if p != ProviderGitHub && p != ProviderGoogle {
			return fmt.Errorf("unsupported oauth provider %q (allowed: %s, %s)", p, ProviderGitHub, ProviderGoogle)
		}
	}
	return nil
}

// NormalizeAllowedOAuthProviders returns a copy with duplicates removed, order preserved.
func NormalizeAllowedOAuthProviders(providers []string) []string {
	if len(providers) == 0 {
		return nil
	}
	out := make([]string, 0, len(providers))
	for _, p := range providers {
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}
