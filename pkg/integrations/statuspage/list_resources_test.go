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

func Test__ListResources__Component_expression_page_id(t *testing.T) {
	// When page_id is an expression (e.g. {{ previous().data.page_id }}), return empty list
	// instead of calling the Statuspage API with invalid data (which would cause 500).
	s := &Statuspage{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "test-key"},
	}
	ctx := core.ListResourcesContext{
		HTTP:        nil,
		Integration: integrationCtx,
		Parameters:  map[string]string{"page_id": "{{ previous().data.page_id }}"},
	}

	resources, err := s.ListResources(ResourceTypeComponent, ctx)
	require.NoError(t, err)
	assert.Empty(t, resources)
}

func Test__ListResources__Incident_with_page_id(t *testing.T) {
	s := &Statuspage{}
	incidentsJSON := `[{"id":"inc1","name":"Outage"},{"id":"inc2","name":"Scheduled Maintenance"}]`
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
	assert.Equal(t, "Outage", resources[0].Name)
	assert.Equal(t, "inc1", resources[0].ID)
	assert.Equal(t, ResourceTypeIncident, resources[1].Type)
	assert.Equal(t, "Scheduled Maintenance", resources[1].Name)
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
	require.Len(t, resources, 1)
	assert.Equal(t, IncidentUseExpressionID, resources[0].ID)
}

func Test__ListResources__Incident_expression_page_id(t *testing.T) {
	s := &Statuspage{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "test-key"},
	}
	ctx := core.ListResourcesContext{
		HTTP:        nil,
		Integration: integrationCtx,
		Parameters:  map[string]string{"page_id": "{{ previous().data.page_id }}"},
	}

	resources, err := s.ListResources(ResourceTypeIncident, ctx)
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, IncidentUseExpressionID, resources[0].ID)
}

func Test__ListResources__Incident_invalid_page_id_returns_use_expression(t *testing.T) {
	// When page_id is invalid (e.g. from expression or typo like "bla bla"), API returns 404.
	// Return "Use expression" option so user can select it and type expression in incidentExpression field.
	s := &Statuspage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`{"error":"Not found"}`)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "test-key"},
	}
	ctx := core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: integrationCtx,
		Parameters:  map[string]string{"page_id": "bla bla"},
	}

	resources, err := s.ListResources(ResourceTypeIncident, ctx)
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, IncidentUseExpressionID, resources[0].ID)
	assert.Equal(t, "Use expression for incident", resources[0].Name)
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

func Test__ListResources__Impact(t *testing.T) {
	s := &Statuspage{}
	ctx := core.ListResourcesContext{Parameters: map[string]string{}}

	resources, err := s.ListResources(ResourceTypeImpact, ctx)
	require.NoError(t, err)
	require.Len(t, resources, 4)
	assert.Equal(t, ResourceTypeImpact, resources[0].Type)
	assert.Equal(t, "None", resources[0].Name)
	assert.Equal(t, "none", resources[0].ID)
	assert.Equal(t, "Minor", resources[1].Name)
	assert.Equal(t, "minor", resources[1].ID)
	assert.Equal(t, "Major", resources[2].Name)
	assert.Equal(t, "major", resources[2].ID)
	assert.Equal(t, "Critical", resources[3].Name)
	assert.Equal(t, "critical", resources[3].ID)
}

func Test__ListResources__ImpactUpdate(t *testing.T) {
	s := &Statuspage{}
	ctx := core.ListResourcesContext{Parameters: map[string]string{}}

	resources, err := s.ListResources(ResourceTypeImpactUpdate, ctx)
	require.NoError(t, err)
	require.Len(t, resources, 6)
	assert.Equal(t, "Don't override", resources[0].Name)
	assert.Equal(t, "__none__", resources[0].ID)
	assert.Equal(t, "None", resources[1].Name)
	assert.Equal(t, "none", resources[1].ID)
	assert.Equal(t, "Maintenance", resources[2].Name)
	assert.Equal(t, "maintenance", resources[2].ID)
	assert.Equal(t, "Minor", resources[3].Name)
	assert.Equal(t, "Critical", resources[5].Name)
	assert.Equal(t, "critical", resources[5].ID)
}

func Test__ListResources__IncidentStatusRealtime(t *testing.T) {
	s := &Statuspage{}
	ctx := core.ListResourcesContext{Parameters: map[string]string{}}

	resources, err := s.ListResources(ResourceTypeIncidentStatusRealtime, ctx)
	require.NoError(t, err)
	require.Len(t, resources, 4)
	assert.Equal(t, "Investigating", resources[0].Name)
	assert.Equal(t, "investigating", resources[0].ID)
	assert.Equal(t, "Identified", resources[1].Name)
	assert.Equal(t, "identified", resources[1].ID)
	assert.Equal(t, "Monitoring", resources[2].Name)
	assert.Equal(t, "monitoring", resources[2].ID)
	assert.Equal(t, "Resolved", resources[3].Name)
	assert.Equal(t, "resolved", resources[3].ID)
}

func Test__ListResources__IncidentStatusScheduled(t *testing.T) {
	s := &Statuspage{}
	ctx := core.ListResourcesContext{Parameters: map[string]string{}}

	resources, err := s.ListResources(ResourceTypeIncidentStatusScheduled, ctx)
	require.NoError(t, err)
	require.Len(t, resources, 4)
	assert.Equal(t, "Scheduled", resources[0].Name)
	assert.Equal(t, "scheduled", resources[0].ID)
	assert.Equal(t, "In progress", resources[1].Name)
	assert.Equal(t, "in_progress", resources[1].ID)
	assert.Equal(t, "Verifying", resources[2].Name)
	assert.Equal(t, "verifying", resources[2].ID)
	assert.Equal(t, "Completed", resources[3].Name)
	assert.Equal(t, "completed", resources[3].ID)
}

func Test__ListResources__ComponentStatus(t *testing.T) {
	s := &Statuspage{}
	ctx := core.ListResourcesContext{Parameters: map[string]string{}}

	resources, err := s.ListResources(ResourceTypeComponentStatus, ctx)
	require.NoError(t, err)
	require.Len(t, resources, 5)
	assert.Equal(t, "Operational", resources[0].Name)
	assert.Equal(t, "operational", resources[0].ID)
	assert.Equal(t, "Degraded performance", resources[1].Name)
	assert.Equal(t, "degraded_performance", resources[1].ID)
	assert.Equal(t, "Partial outage", resources[2].Name)
	assert.Equal(t, "partial_outage", resources[2].ID)
	assert.Equal(t, "Major outage", resources[3].Name)
	assert.Equal(t, "major_outage", resources[3].ID)
	assert.Equal(t, "Under maintenance", resources[4].Name)
	assert.Equal(t, "under_maintenance", resources[4].ID)
}
