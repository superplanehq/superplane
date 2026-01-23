package pagerduty

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

func Test__ListIncidents__Setup(t *testing.T) {
	component := &ListIncidents{}

	t.Run("valid setup without services", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration:   map[string]any{},
			HTTP:            &contexts.HTTPContext{},
			AppInstallation: appCtx,
			Metadata:        metadataCtx,
		})

		require.NoError(t, err)
	})

	t.Run("valid setup with services", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"service": {
								"id": "PX123456",
								"name": "Production API",
								"html_url": "https://example.pagerduty.com/services/PX123456"
							}
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"services": []string{"PX123456"},
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			Metadata:        metadataCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.pagerduty.com/services/PX123456", httpContext.Requests[0].URL.String())

		metadata := metadataCtx.Metadata.(ListIncidentsNodeMetadata)
		require.Len(t, metadata.Services, 1)
		assert.Equal(t, "PX123456", metadata.Services[0].ID)
		assert.Equal(t, "Production API", metadata.Services[0].Name)
	})

	t.Run("skips setup when metadata already exists", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		err := metadataCtx.Set(ListIncidentsNodeMetadata{
			Services: []Service{
				{ID: "PX123456", Name: "Production API"},
			},
		})
		require.NoError(t, err)

		httpContext := &contexts.HTTPContext{}
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		err = component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"services": []string{"PX123456"},
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			Metadata:        metadataCtx,
		})

		require.NoError(t, err)
		// Should not have made any HTTP requests since metadata already exists
		assert.Len(t, httpContext.Requests, 0)
	})

	t.Run("invalid service ID returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Service not found"}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"services": []string{"INVALID"},
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error fetching service")
	})
}

func Test__ListIncidents__Execute(t *testing.T) {
	component := &ListIncidents{}

	t.Run("high urgency incidents emit to high channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"incidents": [
								{
									"id": "PT4KHLK",
									"incident_number": 1234,
									"title": "Server is on fire",
									"status": "triggered",
									"urgency": "high",
									"html_url": "https://example.pagerduty.com/incidents/PT4KHLK",
									"created_at": "2024-01-15T12:00:00Z",
									"service": {
										"id": "PX123456",
										"type": "service_reference",
										"summary": "Production API",
										"html_url": "https://example.pagerduty.com/services/PX123456"
									}
								},
								{
									"id": "PT4KHLM",
									"incident_number": 1235,
									"title": "Database connection issues",
									"status": "acknowledged",
									"urgency": "high",
									"html_url": "https://example.pagerduty.com/incidents/PT4KHLM",
									"created_at": "2024-01-15T12:30:00Z",
									"service": {
										"id": "PX123456",
										"type": "service_reference",
										"summary": "Production API",
										"html_url": "https://example.pagerduty.com/services/PX123456"
									}
								}
							]
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{}
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:   map[string]any{},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
			NodeMetadata:    nodeMetadataCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "pagerduty.incidents.list", execCtx.Type)
		assert.Equal(t, ChannelNameHigh, execCtx.Channel)
		require.Len(t, execCtx.Payloads, 1)

		// Verify the request was made correctly
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/incidents")
		assert.Contains(t, httpContext.Requests[0].URL.String(), "statuses[]=triggered")
		assert.Contains(t, httpContext.Requests[0].URL.String(), "statuses[]=acknowledged")
	})

	t.Run("low urgency incidents emit to low channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"incidents": [
								{
									"id": "PT4KHLK",
									"incident_number": 1234,
									"title": "Minor issue",
									"status": "triggered",
									"urgency": "low",
									"html_url": "https://example.pagerduty.com/incidents/PT4KHLK",
									"created_at": "2024-01-15T12:00:00Z",
									"service": {
										"id": "PX123456",
										"type": "service_reference",
										"summary": "Production API",
										"html_url": "https://example.pagerduty.com/services/PX123456"
									}
								}
							]
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{}
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:   map[string]any{},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
			NodeMetadata:    nodeMetadataCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ChannelNameLow, execCtx.Channel)
	})

	t.Run("no incidents emit to clear channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"incidents": []
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{}
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:   map[string]any{},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
			NodeMetadata:    nodeMetadataCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ChannelNameClear, execCtx.Channel)
	})

	t.Run("mixed urgency emits to high channel (highest severity)", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"incidents": [
								{
									"id": "PT4KHLK",
									"incident_number": 1234,
									"title": "Minor issue",
									"status": "triggered",
									"urgency": "low",
									"html_url": "https://example.pagerduty.com/incidents/PT4KHLK",
									"created_at": "2024-01-15T12:00:00Z",
									"service": {
										"id": "PX123456",
										"type": "service_reference",
										"summary": "Production API",
										"html_url": "https://example.pagerduty.com/services/PX123456"
									}
								},
								{
									"id": "PT4KHLM",
									"incident_number": 1235,
									"title": "Server is on fire",
									"status": "triggered",
									"urgency": "high",
									"html_url": "https://example.pagerduty.com/incidents/PT4KHLM",
									"created_at": "2024-01-15T12:30:00Z",
									"service": {
										"id": "PX123456",
										"type": "service_reference",
										"summary": "Production API",
										"html_url": "https://example.pagerduty.com/services/PX123456"
									}
								}
							]
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{}
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:   map[string]any{},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
			NodeMetadata:    nodeMetadataCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ChannelNameHigh, execCtx.Channel)
	})

	t.Run("filters by services when specified", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"incidents": [
								{
									"id": "PT4KHLK",
									"incident_number": 1234,
									"title": "Server is on fire",
									"status": "triggered",
									"urgency": "high",
									"html_url": "https://example.pagerduty.com/incidents/PT4KHLK",
									"created_at": "2024-01-15T12:00:00Z",
									"service": {
										"id": "PX123456",
										"type": "service_reference",
										"summary": "Production API",
										"html_url": "https://example.pagerduty.com/services/PX123456"
									}
								}
							]
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{}
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"services": []string{"PX123456", "PX789012"},
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
			NodeMetadata:    nodeMetadataCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ChannelNameHigh, execCtx.Channel)

		// Verify service IDs were included in the request
		require.Len(t, httpContext.Requests, 1)
		requestURL := httpContext.Requests[0].URL.String()
		assert.Contains(t, requestURL, "service_ids[]=PX123456")
		assert.Contains(t, requestURL, "service_ids[]=PX789012")
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Invalid API key"}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "invalid-token",
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{}
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:   map[string]any{},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
			NodeMetadata:    nodeMetadataCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list incidents")
	})
}

func Test__ListIncidents__DetermineOutputChannel(t *testing.T) {
	component := &ListIncidents{}

	t.Run("returns clear when no incidents", func(t *testing.T) {
		incidents := []Incident{}
		channel := component.determineOutputChannel(incidents)
		assert.Equal(t, ChannelNameClear, channel)
	})

	t.Run("returns high when high urgency incidents exist", func(t *testing.T) {
		incidents := []Incident{
			{ID: "PT4KHLK", Title: "Test incident", Status: "triggered", Urgency: "high"},
		}
		channel := component.determineOutputChannel(incidents)
		assert.Equal(t, ChannelNameHigh, channel)
	})

	t.Run("returns low when only low urgency incidents exist", func(t *testing.T) {
		incidents := []Incident{
			{ID: "PT4KHLK", Title: "Test incident", Status: "triggered", Urgency: "low"},
		}
		channel := component.determineOutputChannel(incidents)
		assert.Equal(t, ChannelNameLow, channel)
	})

	t.Run("returns high when mixed urgency (highest wins)", func(t *testing.T) {
		incidents := []Incident{
			{ID: "PT4KHLK", Title: "Test incident 1", Status: "triggered", Urgency: "low"},
			{ID: "PT4KHLM", Title: "Test incident 2", Status: "acknowledged", Urgency: "high"},
		}
		channel := component.determineOutputChannel(incidents)
		assert.Equal(t, ChannelNameHigh, channel)
	})

	t.Run("returns low with multiple low urgency incidents", func(t *testing.T) {
		incidents := []Incident{
			{ID: "PT4KHLK", Title: "Test incident 1", Status: "triggered", Urgency: "low"},
			{ID: "PT4KHLM", Title: "Test incident 2", Status: "acknowledged", Urgency: "low"},
		}
		channel := component.determineOutputChannel(incidents)
		assert.Equal(t, ChannelNameLow, channel)
	})
}
