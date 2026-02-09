package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GitHub__Setup(t *testing.T) {
	g := &GitHub{}

	t.Run("personal scope", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		require.NoError(t, g.Sync(core.SyncContext{Integration: integrationCtx}))

		//
		// Browser action is created
		//
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Equal(t, integrationCtx.BrowserAction.Method, "POST")
		assert.NotEmpty(t, integrationCtx.BrowserAction.Description)
		assert.Equal(t, integrationCtx.BrowserAction.URL, "https://github.com/settings/apps/new")

		//
		// Metadata is set
		//
		require.NotNil(t, integrationCtx.Metadata)
		metadata := integrationCtx.Metadata.(Metadata)
		assert.Empty(t, metadata.Owner)
		assert.NotEmpty(t, metadata.State)
	})

	t.Run("organization scope", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		require.NoError(t, g.Sync(core.SyncContext{
			Configuration: Configuration{Organization: "testhq"},
			Integration:   integrationCtx,
		}))

		//
		// Browser action is created
		//
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Equal(t, integrationCtx.BrowserAction.Method, "POST")
		assert.NotEmpty(t, integrationCtx.BrowserAction.Description)
		assert.Equal(t, integrationCtx.BrowserAction.URL, "https://github.com/organizations/testhq/settings/apps/new")

		//
		// Metadata is set
		//
		require.NotNil(t, integrationCtx.Metadata)
		metadata := integrationCtx.Metadata.(Metadata)
		assert.Equal(t, metadata.Owner, "testhq")
		assert.NotEmpty(t, metadata.State)
	})
}
