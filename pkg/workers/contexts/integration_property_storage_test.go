package contexts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

func Test__IntegrationPropertyStorage(t *testing.T) {
	t.Run("gets existing properties", func(t *testing.T) {
		storage := NewIntegrationPropertyStorage(&models.Integration{
			Properties: datatypes.NewJSONSlice([]core.IntegrationPropertyDefinition{
				{Name: "region", Value: "us-east-1"},
				{Name: "retries", Value: float64(3)},
			}),
		})

		value, err := storage.Get("region")
		require.NoError(t, err)
		assert.Equal(t, "us-east-1", value)

		value, err = storage.GetString("region")
		require.NoError(t, err)
		assert.Equal(t, "us-east-1", value)
	})

	t.Run("returns errors for missing and non-string properties", func(t *testing.T) {
		storage := NewIntegrationPropertyStorage(&models.Integration{
			Properties: datatypes.NewJSONSlice([]core.IntegrationPropertyDefinition{
				{Name: "retries", Value: float64(3)},
			}),
		})

		value, err := storage.Get("missing")
		require.Error(t, err)
		assert.Nil(t, value)
		assert.Contains(t, err.Error(), "property missing not found")

		stringValue, err := storage.GetString("retries")
		require.Error(t, err)
		assert.Empty(t, stringValue)
		assert.Contains(t, err.Error(), "property retries is not a string")
	})

	t.Run("creates and deletes properties", func(t *testing.T) {
		integration := &models.Integration{
			Properties: datatypes.NewJSONSlice([]core.IntegrationPropertyDefinition{
				{Name: "region", Value: "us-east-1"},
				{Name: "project", Value: "superplane"},
			}),
		}

		storage := NewIntegrationPropertyStorage(integration)
		require.NoError(t, storage.Create(core.IntegrationPropertyDefinition{Name: "team", Value: "delivery"}))

		value, err := storage.Get("team")
		require.NoError(t, err)
		assert.Equal(t, "delivery", value)

		err = storage.Create(core.IntegrationPropertyDefinition{Name: "team", Value: "platform"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "property team already exists")

		require.NoError(t, storage.Delete("region", "missing"))

		_, err = storage.Get("region")
		require.Error(t, err)

		value, err = storage.Get("project")
		require.NoError(t, err)
		assert.Equal(t, "superplane", value)
		assert.Len(t, integration.Properties, 2)
	})

	t.Run("creates many properties and stops on duplicates", func(t *testing.T) {
		integration := &models.Integration{}
		storage := NewIntegrationPropertyStorage(integration)

		err := storage.CreateMany([]core.IntegrationPropertyDefinition{
			{Name: "region", Value: "us-east-1"},
			{Name: "project", Value: "superplane"},
		})
		require.NoError(t, err)
		assert.Len(t, integration.Properties, 2)

		err = storage.CreateMany([]core.IntegrationPropertyDefinition{
			{Name: "team", Value: "delivery"},
			{Name: "project", Value: "duplicate"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "property project already exists")

		value, err := storage.Get("team")
		require.NoError(t, err)
		assert.Equal(t, "delivery", value)
		assert.Len(t, integration.Properties, 3)
	})
}
