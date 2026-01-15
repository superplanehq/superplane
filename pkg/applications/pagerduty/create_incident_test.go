package pagerduty

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateIncident__Configuration(t *testing.T) {
	component := &CreateIncident{}

	t.Run("returns correct configuration fields", func(t *testing.T) {
		fields := component.Configuration()

		require.Len(t, fields, 4)

		// Verify title field
		titleField := fields[0]
		assert.Equal(t, "title", titleField.Name)
		assert.Equal(t, "Incident Title", titleField.Label)
		assert.Equal(t, configuration.FieldTypeString, titleField.Type)
		assert.True(t, titleField.Required)

		// Verify description field
		descField := fields[1]
		assert.Equal(t, "description", descField.Name)
		assert.Equal(t, "Description", descField.Label)
		assert.Equal(t, configuration.FieldTypeString, descField.Type)
		assert.False(t, descField.Required)

		// Verify urgency field
		urgencyField := fields[2]
		assert.Equal(t, "urgency", urgencyField.Name)
		assert.Equal(t, "Urgency", urgencyField.Label)
		assert.Equal(t, configuration.FieldTypeSelect, urgencyField.Type)
		assert.True(t, urgencyField.Required)
		assert.Equal(t, "high", urgencyField.Default)

		// Verify service field - this is the important one for our changes
		serviceField := fields[3]
		assert.Equal(t, "service", serviceField.Name)
		assert.Equal(t, "Service", serviceField.Label)
		assert.Equal(t, "app-installation-resource", string(serviceField.Type))
		assert.True(t, serviceField.Required)
		assert.Equal(t, "The PagerDuty service to create the incident for", serviceField.Description)
		assert.Equal(t, "Select a service", serviceField.Placeholder)
		require.NotNil(t, serviceField.TypeOptions)
		require.NotNil(t, serviceField.TypeOptions.Resource)
		assert.Equal(t, "service", serviceField.TypeOptions.Resource.Type)
	})
}

func Test__CreateIncident__Setup(t *testing.T) {
	component := &CreateIncident{}

	t.Run("valid configuration with existing service", func(t *testing.T) {
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
				"title":       "Test Incident",
				"urgency":     "high",
				"service":     "PX123456",
				"description": "Test description",
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			Metadata:        metadataCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.pagerduty.com/services/PX123456", httpContext.Requests[0].URL.String())

		metadata := metadataCtx.Metadata.(NodeMetadata)
		require.NotNil(t, metadata.Service)
		assert.Equal(t, "PX123456", metadata.Service.ID)
		assert.Equal(t, "Production API", metadata.Service.Name)
	})

	t.Run("missing title returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"urgency": "high",
				"service": "PX123456",
			},
		})

		require.ErrorContains(t, err, "title is required")
	})

	t.Run("missing urgency returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"title":   "Test Incident",
				"service": "PX123456",
			},
		})

		require.ErrorContains(t, err, "urgency is required")
	})

	t.Run("missing service returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"title":   "Test Incident",
				"urgency": "high",
			},
		})

		require.ErrorContains(t, err, "service is required")
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
				"title":   "Test Incident",
				"urgency": "high",
				"service": "INVALID",
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error finding service")
	})
}
