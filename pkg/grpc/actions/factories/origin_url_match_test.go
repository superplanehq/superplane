package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkOrderOriginMatchPreservesGitHubHost(t *testing.T) {
	dotcom, err := workOrderOriginMatch("https://github.com/acme/app/issues/42")
	require.NoError(t, err)
	require.Len(t, dotcom.Args, 1)
	assert.Contains(t, dotcom.Args[0], `github\.com`)
	assert.NotContains(t, dotcom.Args[0], "ghe.com")

	enterprise, err := workOrderOriginMatch("https://acme.ghe.com/acme/app/issues/42")
	require.NoError(t, err)
	require.Len(t, enterprise.Args, 1)
	assert.Contains(t, enterprise.Args[0], `acme\.ghe\.com`)
	assert.NotContains(t, enterprise.Args[0], `github\.com`)
}
