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
		appCtx := &contexts.AppInstallationContext{}
		require.NoError(t, g.Sync(core.SyncContext{AppInstallation: appCtx}))

		//
		// Browser action is created
		//
		require.NotNil(t, appCtx.BrowserAction)
		assert.Equal(t, appCtx.BrowserAction.Method, "POST")
		assert.NotEmpty(t, appCtx.BrowserAction.Description)
		assert.Equal(t, appCtx.BrowserAction.URL, "https://github.com/settings/apps/new")

		//
		// Metadata is set
		//
		require.NotNil(t, appCtx.Metadata)
		metadata := appCtx.Metadata.(Metadata)
		assert.Empty(t, metadata.Owner)
		assert.NotEmpty(t, metadata.State)
	})

	t.Run("organization scope", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		require.NoError(t, g.Sync(core.SyncContext{
			Configuration:   Configuration{Organization: "testhq"},
			AppInstallation: appCtx,
		}))

		//
		// Browser action is created
		//
		require.NotNil(t, appCtx.BrowserAction)
		assert.Equal(t, appCtx.BrowserAction.Method, "POST")
		assert.NotEmpty(t, appCtx.BrowserAction.Description)
		assert.Equal(t, appCtx.BrowserAction.URL, "https://github.com/organizations/testhq/settings/apps/new")

		//
		// Metadata is set
		//
		require.NotNil(t, appCtx.Metadata)
		metadata := appCtx.Metadata.(Metadata)
		assert.Equal(t, metadata.Owner, "testhq")
		assert.NotEmpty(t, metadata.State)
	})
}
