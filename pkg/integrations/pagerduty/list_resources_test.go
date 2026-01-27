package pagerduty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__PagerDuty__ListResources(t *testing.T) {
	p := &PagerDuty{}

	t.Run("unknown resource type returns empty list", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}

		resources, err := p.ListResources("unknown", core.ListResourcesContext{
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("service resource type returns services from metadata", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Metadata: Metadata{
				Services: []Service{
					{ID: "PX123", Name: "Production API"},
					{ID: "PX456", Name: "Staging Database"},
					{ID: "PX789", Name: "Frontend App"},
				},
			},
		}

		resources, err := p.ListResources("service", core.ListResourcesContext{
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 3)

		assert.Equal(t, "service", resources[0].Type)
		assert.Equal(t, "PX123", resources[0].ID)
		assert.Equal(t, "Production API", resources[0].Name)

		assert.Equal(t, "service", resources[1].Type)
		assert.Equal(t, "PX456", resources[1].ID)
		assert.Equal(t, "Staging Database", resources[1].Name)

		assert.Equal(t, "service", resources[2].Type)
		assert.Equal(t, "PX789", resources[2].ID)
		assert.Equal(t, "Frontend App", resources[2].Name)
	})

	t.Run("empty services metadata returns empty list", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Metadata: Metadata{
				Services: []Service{},
			},
		}

		resources, err := p.ListResources("service", core.ListResourcesContext{
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("nil metadata returns empty list", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}

		resources, err := p.ListResources("service", core.ListResourcesContext{
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}
