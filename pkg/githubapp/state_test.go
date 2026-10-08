package githubapp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateStateRoundTrip(t *testing.T) {
	state, err := SignCreateState("setup-secret", "/acme/workspaces/new/setup?step=vcs")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(state, createAppPrefix))
	assert.NotContains(t, state, "/acme/workspaces")

	path, err := VerifyCreateState("setup-secret", state)
	require.NoError(t, err)
	assert.Equal(t, "/acme/workspaces/new/setup?step=vcs", path)

	_, err = VerifyCreateState("other-secret", state)
	assert.Error(t, err)
}
