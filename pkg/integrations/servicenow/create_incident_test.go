package servicenow

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateIncident__Setup(t *testing.T) {
	component := &CreateIncident{}

	t.Run("valid configuration with successful connection", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": []}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"authType":    "basicAuth",
				"username":    "admin",
				"password":    "password",
			},
		}

		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"shortDescription": "Test Incident",
				"urgency":          "2",
				"impact":           "2",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://dev12345.service-now.com/api/now/table/incident?sysparm_limit=1", httpContext.Requests[0].URL.String())

		metadata := metadataCtx.Metadata.(NodeMetadata)
		assert.Equal(t, "https://dev12345.service-now.com", metadata.InstanceURL)
	})

	t.Run("valid configuration with resources verifies them", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// ValidateConnection
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": []}`)),
				},
				// GetAssignmentGroup
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": {"sys_id": "grp1", "name": "Network"}}`)),
				},
				// GetUser (assignedTo)
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": {"sys_id": "user1", "name": "John Smith", "email": "john@example.com"}}`)),
				},
				// GetUser (caller)
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": {"sys_id": "user2", "name": "Jane Doe", "email": "jane@example.com"}}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"authType":    "basicAuth",
				"username":    "admin",
				"password":    "password",
			},
		}

		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"shortDescription": "Test Incident",
				"urgency":          "2",
				"impact":           "2",
				"assignmentGroup":  "grp1",
				"assignedTo":       "user1",
				"caller":           "user2",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 4)

		metadata := metadataCtx.Metadata.(NodeMetadata)
		assert.Equal(t, "https://dev12345.service-now.com", metadata.InstanceURL)
		require.NotNil(t, metadata.AssignmentGroup)
		assert.Equal(t, "grp1", metadata.AssignmentGroup.ID)
		assert.Equal(t, "Network", metadata.AssignmentGroup.Name)
		require.NotNil(t, metadata.AssignedTo)
		assert.Equal(t, "user1", metadata.AssignedTo.ID)
		assert.Equal(t, "John Smith", metadata.AssignedTo.Name)
		require.NotNil(t, metadata.Caller)
		assert.Equal(t, "user2", metadata.Caller.ID)
		assert.Equal(t, "Jane Doe", metadata.Caller.Name)
	})

	t.Run("invalid assignment group returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": []}`)),
				},
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": "not found"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"authType":    "basicAuth",
				"username":    "admin",
				"password":    "password",
			},
		}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"shortDescription": "Test Incident",
				"urgency":          "2",
				"impact":           "2",
				"assignmentGroup":  "invalid-group",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error verifying assignment group")
	})

	t.Run("missing shortDescription returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"urgency": "2",
				"impact":  "2",
			},
		})

		require.ErrorContains(t, err, "shortDescription is required")
	})

	t.Run("missing urgency returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"shortDescription": "Test Incident",
				"impact":           "2",
			},
		})

		require.ErrorContains(t, err, "urgency is required")
	})

	t.Run("missing impact returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"shortDescription": "Test Incident",
				"urgency":          "2",
			},
		})

		require.ErrorContains(t, err, "impact is required")
	})

	t.Run("invalid ServiceNow connection returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error": "unauthorized"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"authType":    "basicAuth",
				"username":    "admin",
				"password":    "wrong",
			},
		}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"shortDescription": "Test Incident",
				"urgency":          "2",
				"impact":           "2",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error validating ServiceNow connection")
	})
}

func Test__CreateIncident__Execute(t *testing.T) {
	component := &CreateIncident{}

	t.Run("successful incident creation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"result": {
							"sys_id": "abc123",
							"number": "INC0010001",
							"short_description": "Test Incident",
							"state": "1",
							"urgency": "2",
							"impact": "2"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"authType":    "basicAuth",
				"username":    "admin",
				"password":    "password",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"shortDescription": "Test Incident",
				"description":      "Detailed description",
				"urgency":          "2",
				"impact":           "2",
				"category":         "software",
				"subcategory":      "email",
				"assignmentGroup":  "abc123def456",
				"assignedTo":       "789abc012def",
				"caller":           "def456abc123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "servicenow.incident", executionState.Type)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://dev12345.service-now.com/api/now/table/incident", httpContext.Requests[0].URL.String())

		// Verify the request body contains all configured fields
		reqBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		var sentParams map[string]any
		err = json.Unmarshal(reqBody, &sentParams)
		require.NoError(t, err)

		assert.Equal(t, "Test Incident", sentParams["short_description"])
		assert.Equal(t, "Detailed description", sentParams["description"])
		assert.Equal(t, "2", sentParams["urgency"])
		assert.Equal(t, "2", sentParams["impact"])
		assert.Equal(t, "software", sentParams["category"])
		assert.Equal(t, "email", sentParams["subcategory"])
		assert.Equal(t, "abc123def456", sentParams["assignment_group"])
		assert.Equal(t, "789abc012def", sentParams["assigned_to"])
		assert.Equal(t, "def456abc123", sentParams["caller_id"])
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error": "unauthorized"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"authType":    "basicAuth",
				"username":    "admin",
				"password":    "wrong",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"shortDescription": "Test Incident",
				"urgency":          "2",
				"impact":           "2",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create incident")
	})
}
