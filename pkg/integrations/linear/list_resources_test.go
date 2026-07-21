package linear

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ListResources__Teams(t *testing.T) {
	integration := &Linear{}

	resources, err := integration.ListResources(ResourceTypeTeam, core.ListResourcesContext{
		Integration: integrationWithTeam(),
	})

	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "t1", resources[0].ID)
	assert.Equal(t, "Engineering (ENG)", resources[0].Name)
}

func Test__ListResources__UnknownType(t *testing.T) {
	integration := &Linear{}

	resources, err := integration.ListResources("unknown", core.ListResourcesContext{
		Integration: integrationWithTeam(),
	})

	require.NoError(t, err)
	assert.Empty(t, resources)
}

func Test__ListResources__TeamScopedTypesRequireTeam(t *testing.T) {
	integration := &Linear{}

	for _, resourceType := range []string{ResourceTypeWorkflowState, ResourceTypeMember, ResourceTypeLabel, ResourceTypeProject} {
		t.Run(resourceType, func(t *testing.T) {
			resources, err := integration.ListResources(resourceType, core.ListResourcesContext{
				Integration: integrationWithTeam(),
				Parameters:  map[string]string{},
			})

			require.NoError(t, err)
			assert.Empty(t, resources)
		})
	}
}

func Test__ListResources__WorkflowStates(t *testing.T) {
	integration := &Linear{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"workflowStates":{"nodes":[{"id":"s1","name":"Todo","type":"unstarted"}]}}}`),
		},
	}

	resources, err := integration.ListResources(ResourceTypeWorkflowState, core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: integrationWithTeam(),
		Parameters:  map[string]string{"team": "t1"},
	})

	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "Todo", resources[0].Name)
	assert.Equal(t, "s1", resources[0].ID)
}

func Test__ListResources__Members(t *testing.T) {
	integration := &Linear{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"team":{"members":{"nodes":[{"id":"u1","name":"Jane Doe","displayName":"jane"},{"id":"u2","name":"John Doe"}]}}}}`),
		},
	}

	resources, err := integration.ListResources(ResourceTypeMember, core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: integrationWithTeam(),
		Parameters:  map[string]string{"team": "t1"},
	})

	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, "Jane Doe (@jane)", resources[0].Name)
	assert.Equal(t, "John Doe", resources[1].Name)
}

func Test__ListResources__Projects(t *testing.T) {
	integration := &Linear{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"team":{"projects":{"nodes":[{"id":"p1","name":"Q3 Reliability"}],"pageInfo":{"hasNextPage":false}}}}}`),
		},
	}

	resources, err := integration.ListResources(ResourceTypeProject, core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: integrationWithTeam(),
		Parameters:  map[string]string{"team": "t1"},
	})

	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "Q3 Reliability", resources[0].Name)
	assert.Equal(t, "p1", resources[0].ID)
}

func Test__ListResources__Labels(t *testing.T) {
	integration := &Linear{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"issueLabels":{"nodes":[{"id":"l1","name":"bug"}]}}}`),
		},
	}

	resources, err := integration.ListResources(ResourceTypeLabel, core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: integrationWithTeam(),
		Parameters:  map[string]string{"team": "t1"},
	})

	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "bug", resources[0].Name)
}
