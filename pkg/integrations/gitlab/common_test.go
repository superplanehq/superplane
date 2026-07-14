package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__isExpressionValue(t *testing.T) {
	assert.True(t, isExpressionValue("{{ event.data.project_id }}"))
	assert.True(t, isExpressionValue("  {{ project }}  "))
	assert.True(t, isExpressionValue(`$["Run Pipeline"].data.project_id`))
	assert.False(t, isExpressionValue("123"))
	assert.False(t, isExpressionValue(""))
	assert.False(t, isExpressionValue("my-group/my-project"))
}

func Test__ensureProjectInMetadata(t *testing.T) {
	integration := &contexts.IntegrationContext{
		Metadata: Metadata{
			Projects: []ProjectMetadata{
				{ID: 123, Name: "repo", URL: "http://repo"},
			},
		},
	}

	t.Run("empty project is rejected", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := ensureProjectInMetadata(metadata, integration, "  ")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("literal project not accessible to integration is rejected", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := ensureProjectInMetadata(metadata, integration, "456")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not accessible to integration")
	})

	t.Run("accessible literal project is stored in metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := ensureProjectInMetadata(metadata, integration, "123")
		require.NoError(t, err)

		nodeMetadata, ok := metadata.Get().(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, nodeMetadata.Project)
		assert.Equal(t, 123, nodeMetadata.Project.ID)
	})

	t.Run("expression project defers validation to runtime", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := ensureProjectInMetadata(metadata, integration, "{{ event.data.object_attributes.project_id }}")
		require.NoError(t, err)

		nodeMetadata, ok := metadata.Get().(NodeMetadata)
		require.True(t, ok)
		assert.Nil(t, nodeMetadata.Project)
	})
}
