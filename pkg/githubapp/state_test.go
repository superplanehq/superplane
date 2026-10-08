package githubapp

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateStateRoundTrip(t *testing.T) {
	accountID := uuid.New()
	state, err := SignCreateState("setup-secret", "/acme/workspaces/new/setup?step=vcs", accountID)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(state, createAppPrefix))
	assert.NotContains(t, state, "/acme/workspaces")
	assert.NotContains(t, state, accountID.String())

	path, gotAccountID, err := VerifyCreateState("setup-secret", state)
	require.NoError(t, err)
	assert.Equal(t, "/acme/workspaces/new/setup?step=vcs", path)
	assert.Equal(t, accountID, gotAccountID)

	_, _, err = VerifyCreateState("other-secret", state)
	assert.Error(t, err)
}

func TestSignCreateStateRequiresAccount(t *testing.T) {
	_, err := SignCreateState("setup-secret", "/setup", uuid.Nil)
	assert.Error(t, err)
}
