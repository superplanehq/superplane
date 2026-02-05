package flyio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__FlyIO__Components(t *testing.T) {
	f := &FlyIO{}
	components := f.Components()

	require.Len(t, components, 1)
	assert.Equal(t, "flyio.listApps", components[0].Name())
}

func Test__FlyIO__Triggers(t *testing.T) {
	f := &FlyIO{}
	triggers := f.Triggers()

	require.Len(t, triggers, 1)
	assert.Equal(t, "flyio.onAppStateChange", triggers[0].Name())
}

func Test__FlyIO__Configuration(t *testing.T) {
	f := &FlyIO{}
	config := f.Configuration()

	require.Len(t, config, 2)

	// Check for apiToken field
	var apiTokenField, orgSlugField bool
	for _, field := range config {
		if field.Name == "apiToken" {
			apiTokenField = true
			assert.True(t, field.Required)
			assert.True(t, field.Sensitive)
		}
		if field.Name == "orgSlug" {
			orgSlugField = true
			assert.False(t, field.Required)
		}
	}

	assert.True(t, apiTokenField, "apiToken field should exist")
	assert.True(t, orgSlugField, "orgSlug field should exist")
}

func Test__FlyIO__Actions(t *testing.T) {
	f := &FlyIO{}
	actions := f.Actions()
	assert.Empty(t, actions)
}

func Test__FlyIO__Cleanup(t *testing.T) {
	f := &FlyIO{}
	err := f.Cleanup(core.IntegrationCleanupContext{})
	assert.NoError(t, err)
}
