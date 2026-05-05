package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAllowedOAuthProviders(t *testing.T) {
	require.NoError(t, ValidateAllowedOAuthProviders(nil))
	require.NoError(t, ValidateAllowedOAuthProviders([]string{}))
	require.NoError(t, ValidateAllowedOAuthProviders([]string{ProviderGitHub}))
	require.NoError(t, ValidateAllowedOAuthProviders([]string{ProviderGitHub, ProviderGoogle}))

	err := ValidateAllowedOAuthProviders([]string{"gitlab"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gitlab")
}

func TestNormalizeAllowedOAuthProviders(t *testing.T) {
	assert.Equal(t, []string{}, NormalizeAllowedOAuthProviders(nil))
	assert.Equal(t, []string{}, NormalizeAllowedOAuthProviders([]string{}))
	assert.Equal(t, []string{ProviderGitHub}, NormalizeAllowedOAuthProviders([]string{ProviderGitHub, ProviderGitHub}))
}
