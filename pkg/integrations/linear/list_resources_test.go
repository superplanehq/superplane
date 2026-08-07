package linear

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

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

func Test__ListResources__Comments(t *testing.T) {
	integration := &Linear{}

	t.Run("labels comments by author and body preview", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issue":{"comments":{"nodes":[{"id":"c1","body":"Deploy is green","user":{"id":"u1","name":"Jane Doe","displayName":"jane"}},{"id":"c2","body":"No author here"}]}}}}`),
			},
		}

		resources, err := integration.ListResources(ResourceTypeComment, core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationWithTeam(),
			Parameters:  map[string]string{"issue": "ENG-142"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "c1", resources[0].ID)
		assert.Equal(t, "Jane Doe (@jane): Deploy is green", resources[0].Name)
		assert.Equal(t, "No author here", resources[1].Name)
	})

	t.Run("truncates a long body and collapses whitespace", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issue":{"comments":{"nodes":[{"id":"c1","body":"line one\nline two that keeps going on and on and on until it is far too long to show"}]}}}}`),
			},
		}

		resources, err := integration.ListResources(ResourceTypeComment, core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationWithTeam(),
			Parameters:  map[string]string{"issue": "ENG-142"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "line one line two that keeps going on and on and on until it…", resources[0].Name)
	})

	t.Run("cuts a multi-byte body on a rune boundary", func(t *testing.T) {
		//
		// 59 ASCII characters followed by an emoji, so a byte-indexed cut would
		// land inside the emoji and corrupt the label.
		//
		body := strings.Repeat("a", 59) + "🚀 and more text after the limit"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(fmt.Sprintf(`{"data":{"issue":{"comments":{"nodes":[{"id":"c1","body":%q}]}}}}`, body)),
			},
		}

		resources, err := integration.ListResources(ResourceTypeComment, core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationWithTeam(),
			Parameters:  map[string]string{"issue": "ENG-142"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, strings.Repeat("a", 59)+"🚀…", resources[0].Name)
		assert.True(t, utf8.ValidString(resources[0].Name))
		assert.NotContains(t, resources[0].Name, "�")
	})

	t.Run("stays empty without an issue", func(t *testing.T) {
		resources, err := integration.ListResources(ResourceTypeComment, core.ListResourcesContext{
			Integration: integrationWithTeam(),
			Parameters:  map[string]string{},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("stays empty when the issue is an expression", func(t *testing.T) {
		resources, err := integration.ListResources(ResourceTypeComment, core.ListResourcesContext{
			Integration: integrationWithTeam(),
			Parameters:  map[string]string{"issue": "{{ inputs.issue }}"},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}
