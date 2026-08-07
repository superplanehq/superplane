package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__IsExpression(t *testing.T) {
	assert.True(t, isExpression("{{ $['On Merge Request'].data.project.id }}"))
	assert.True(t, isExpression("prefix-{{ inputs.project }}-suffix"))
	assert.True(t, isExpression("{{\n  inputs.project\n}}"))
	assert.False(t, isExpression("123456"))
	assert.False(t, isExpression("my-group/my-project"))
	assert.False(t, isExpression(""))
	assert.False(t, isExpression("{ not an expression }"))
}

func Test__EnsureConcreteProject(t *testing.T) {
	t.Run("concrete project is accepted", func(t *testing.T) {
		assert.NoError(t, ensureConcreteProject("123456"))
		assert.NoError(t, ensureConcreteProject("my-group/my-project"))
		assert.NoError(t, ensureConcreteProject(""))
	})

	t.Run("expression project is rejected", func(t *testing.T) {
		err := ensureConcreteProject("{{ root().data.project.id }}")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project does not support expressions")
	})
}

func Test__EnsureProjectInMetadata(t *testing.T) {
	integration := &contexts.IntegrationContext{
		Metadata: Metadata{
			Projects: []ProjectMetadata{
				{ID: 123, Name: "group/project", URL: "https://gitlab.com/group/project"},
			},
		},
	}

	t.Run("empty project", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := ensureProjectInMetadata(metadata, integration, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("expression project skips validation and metadata caching", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := ensureProjectInMetadata(metadata, integration, "{{ $['On Merge Request'].data.project.id }}")
		require.NoError(t, err)
		assert.Nil(t, metadata.Metadata)
	})

	t.Run("accessible project is cached in node metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := ensureProjectInMetadata(metadata, integration, "123")
		require.NoError(t, err)

		nodeMetadata, ok := metadata.Metadata.(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, nodeMetadata.Project)
		assert.Equal(t, 123, nodeMetadata.Project.ID)
		assert.Equal(t, "group/project", nodeMetadata.Project.Name)
	})

	t.Run("inaccessible project", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := ensureProjectInMetadata(metadata, integration, "999")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not accessible to integration")
	})
}
