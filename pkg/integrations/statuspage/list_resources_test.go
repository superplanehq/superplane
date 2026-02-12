package statuspage

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ListResources__Page(t *testing.T) {
	s := &Statuspage{}
	pagesJSON := `[{"id":"page1","name":"My Page"},{"id":"page2","name":"Other Page"}]`
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(pagesJSON)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "test-key"},
	}
	ctx := core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: integrationCtx,
		Parameters:  map[string]string{},
	}

	resources, err := s.ListResources(ResourceTypePage, ctx)
	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, ResourceTypePage, resources[0].Type)
	assert.Equal(t, "My Page", resources[0].Name)
	assert.Equal(t, "page1", resources[0].ID)
	assert.Equal(t, ResourceTypePage, resources[1].Type)
	assert.Equal(t, "Other Page", resources[1].Name)
	assert.Equal(t, "page2", resources[1].ID)
	require.Len(t, httpContext.Requests, 1)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/pages")
}

func Test__ListResources__Component_with_page_id(t *testing.T) {
	s := &Statuspage{}
	componentsJSON := `[{"id":"comp1","name":"API","status":"operational"},{"id":"comp2","name":"DB","status":"operational"}]`
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(componentsJSON)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "test-key"},
	}
	ctx := core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: integrationCtx,
		Parameters:  map[string]string{"page_id": "page1"},
	}

	resources, err := s.ListResources(ResourceTypeComponent, ctx)
	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, ResourceTypeComponent, resources[0].Type)
	assert.Equal(t, "API", resources[0].Name)
	assert.Equal(t, "comp1", resources[0].ID)
	assert.Equal(t, "DB", resources[1].Name)
	assert.Equal(t, "comp2", resources[1].ID)
	require.Len(t, httpContext.Requests, 1)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/pages/page1/components")
}

func Test__ListResources__Component_empty_page_id(t *testing.T) {
	s := &Statuspage{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "test-key"},
	}
	ctx := core.ListResourcesContext{
		HTTP:        nil,
		Integration: integrationCtx,
		Parameters:  map[string]string{},
	}

	resources, err := s.ListResources(ResourceTypeComponent, ctx)
	require.NoError(t, err)
	assert.Empty(t, resources)
}

func Test__ListResources__Incident_with_page_id(t *testing.T) {
	s := &Statuspage{}
	incidentsJSON := `[
		{"id":"inc1","name":"Database Connection Issues"},
		{"id":"inc2","name":"API Latency"}
	]`
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(incidentsJSON)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "test-key"},
	}
	ctx := core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: integrationCtx,
		Parameters:  map[string]string{"page_id": "page1"},
	}

	resources, err := s.ListResources(ResourceTypeIncident, ctx)
	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, ResourceTypeIncident, resources[0].Type)
	assert.Equal(t, "Database Connection Issues", resources[0].Name)
	assert.Equal(t, "inc1", resources[0].ID)
	assert.Equal(t, "API Latency", resources[1].Name)
	assert.Equal(t, "inc2", resources[1].ID)
	require.Len(t, httpContext.Requests, 1)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/pages/page1/incidents")
}

func Test__ListResources__Incident_empty_page_id(t *testing.T) {
	s := &Statuspage{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "test-key"},
	}
	ctx := core.ListResourcesContext{
		HTTP:        nil,
		Integration: integrationCtx,
		Parameters:  map[string]string{},
	}

	resources, err := s.ListResources(ResourceTypeIncident, ctx)
	require.NoError(t, err)
	assert.Empty(t, resources)
}

func Test__ListResources__Unknown_type(t *testing.T) {
	s := &Statuspage{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "test-key"},
	}
	ctx := core.ListResourcesContext{
		Integration: integrationCtx,
		Parameters:  map[string]string{},
	}

	resources, err := s.ListResources("unknown", ctx)
	require.NoError(t, err)
	assert.Empty(t, resources)
}
