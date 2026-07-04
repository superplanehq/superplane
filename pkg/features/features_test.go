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
