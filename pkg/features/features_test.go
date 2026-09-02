package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__Get(t *testing.T) {
	t.Run("known id returns feature and true", func(t *testing.T) {
		f, ok := Get(FeatureClaudeManagedAgents)
		assert.True(t, ok)
		assert.Equal(t, FeatureClaudeManagedAgents, f.ID)
		assert.Equal(t, "Claude Managed Agents", f.Label)
		assert.Equal(t, "Chat with a Claude-powered agent against the canvas", f.Description)
	})

	t.Run("known id returns factory velocity feature", func(t *testing.T) {
		f, ok := Get(FeatureFactoryVelocity)
		assert.True(t, ok)
		assert.Equal(t, FeatureFactoryVelocity, f.ID)
		assert.Equal(t, "Factory Velocity", f.Label)
		assert.Equal(t, "Show the Velocity view for a factory organization", f.Description)
	})

	t.Run("known id returns factory sentry intake feature", func(t *testing.T) {
		f, ok := Get(FeatureFactorySentryIntake)
		assert.True(t, ok)
		assert.Equal(t, FeatureFactorySentryIntake, f.ID)
		assert.Equal(t, "Factory Sentry Intake", f.Label)
		assert.Equal(t, "Add Sentry intake from the Backlog column menu", f.Description)
	})

	t.Run("known id returns workspace models feature", func(t *testing.T) {
		f, ok := Get(FeatureWorkspaceModels)
		assert.True(t, ok)
		assert.Equal(t, FeatureWorkspaceModels, f.ID)
		assert.Equal(t, "Workspace Models", f.Label)
		assert.Equal(t, "Show the in-progress workspace Models settings page", f.Description)
	})

	t.Run("unknown id returns zero value and false", func(t *testing.T) {
		f, ok := Get("does-not-exist")
		assert.False(t, ok)
		assert.Equal(t, Feature{}, f)
	})

	t.Run("empty id returns false", func(t *testing.T) {
		_, ok := Get("")
		assert.False(t, ok)
	})
}

func Test__Exists(t *testing.T) {
	assert.True(t, Exists(FeatureClaudeManagedAgents))
	assert.True(t, Exists(FeatureFactories))
	assert.True(t, Exists(FeatureFactoryVelocity))
	assert.True(t, Exists(FeatureFactorySentryIntake))
	assert.True(t, Exists(FeatureWorkspaceModels))
	assert.False(t, Exists("does-not-exist"))
	assert.False(t, Exists(""))
}

func Test__All_isCopy(t *testing.T) {
	a := All()
	require := len(a)
	a[0] = Feature{ID: "mutated"}

	b := All()
	assert.Equal(t, require, len(b))
	assert.NotEqual(t, "mutated", b[0].ID)
}

func Test__IsReleased(t *testing.T) {
	t.Run("unknown id is not released", func(t *testing.T) {
		assert.False(t, IsReleased("does-not-exist"))
	})

	t.Run("registered id with nil Released is not released", func(t *testing.T) {
		original := registry
		t.Cleanup(func() { registry = original })

		registry = []Feature{{ID: "preview", Label: "Preview", Description: "Preview feature"}}
		assert.False(t, IsReleased("preview"))
	})

	t.Run("registered id with Released=&true is released", func(t *testing.T) {
		original := registry
		t.Cleanup(func() { registry = original })

		v := true
		registry = []Feature{{ID: "graduated", Label: "Graduated", Released: &v}}
		assert.True(t, IsReleased("graduated"))
	})

	t.Run("registered id with Released=&false is not released", func(t *testing.T) {
		original := registry
		t.Cleanup(func() { registry = original })

		v := false
		registry = []Feature{{ID: "explicit-false", Label: "X", Released: &v}}
		assert.False(t, IsReleased("explicit-false"))
	})
}
