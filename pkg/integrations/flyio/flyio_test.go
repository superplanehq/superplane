package flyio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__FlyIO__Name(t *testing.T) {
	f := &FlyIO{}
	assert.Equal(t, "flyio", f.Name())
}

func Test__FlyIO__Label(t *testing.T) {
	f := &FlyIO{}
	assert.Equal(t, "Fly.io", f.Label())
}

func Test__FlyIO__Description(t *testing.T) {
	f := &FlyIO{}
	assert.Contains(t, f.Description(), "Fly.io")
}

func Test__FlyIO__Components(t *testing.T) {
	f := &FlyIO{}
	components := f.Components()

	require.Len(t, components, 5)

	names := make([]string, len(components))
	for i, c := range components {
		names[i] = c.Name()
	}

	assert.Contains(t, names, "flyio.startMachine")
	assert.Contains(t, names, "flyio.stopMachine")
	assert.Contains(t, names, "flyio.createMachine")
	assert.Contains(t, names, "flyio.deleteMachine")
	assert.Contains(t, names, "flyio.listMachines")
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

func Test__FlyIO__Triggers(t *testing.T) {
	f := &FlyIO{}
	triggers := f.Triggers()
	assert.Empty(t, triggers)
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

func Test__FlyIO__Instructions(t *testing.T) {
	f := &FlyIO{}
	instructions := f.Instructions()

	assert.Contains(t, instructions, "API Token")
	assert.Contains(t, instructions, "fly tokens")
}
