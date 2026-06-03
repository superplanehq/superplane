package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__parseOrgAppTarget(t *testing.T) {
	t.Parallel()

	org, app, err := parseOrgAppTarget("acme/widget-app")
	require.NoError(t, err)
	assert.Equal(t, "acme", org)
	assert.Equal(t, "widget-app", app)

	_, _, err = parseOrgAppTarget("widget-app")
	require.Error(t, err)

	_, _, err = parseOrgAppTarget("acme/")
	require.Error(t, err)

	_, _, err = parseOrgAppTarget("/widget-app")
	require.Error(t, err)
}

func Test__cloneRepoDir(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "widget-app", cloneRepoDir("widget-app", nil))
	assert.Equal(t, "./custom", cloneRepoDir("widget-app", []string{"./custom"}))
	assert.Equal(t, ".", cloneRepoDir("widget-app", []string{"."}))
}
