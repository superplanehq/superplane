package servicenow

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

func Test__GetIncidents__Setup(t *testing.T) {
	component := &GetIncidents{}

	t.Run("empty configuration sets instance url in metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			HTTP:          httpContext,
			Integration:   oauthIntegrationContext(),
			Metadata:      metadataCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 0)

		metadata := metadataCtx.Metadata.(NodeMetadata)
		assert.Equal(t, "https://dev12345.service-now.com", metadata.InstanceURL)
	})

	t.Run("valid configuration with resources verifies them", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
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

		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"assignmentGroup": "grp1",
				"assignedTo":      "user1",
				"caller":          "user2",
			},
			HTTP:        httpContext,
			Integration: oauthIntegrationContext(),
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 3)

		metadata := metadataCtx.Metadata.(NodeMetadata)
		require.NotNil(t, metadata.AssignmentGroup)
		assert.Equal(t, "grp1", metadata.AssignmentGroup.ID)
		assert.Equal(t, "Network", metadata.AssignmentGroup.Name)
		require.NotNil(t, metadata.AssignedTo)
		assert.Equal(t, "user1", metadata.AssignedTo.ID)
		require.NotNil(t, metadata.Caller)
		assert.Equal(t, "user2", metadata.Caller.ID)
	})

	t.Run("invalid resource returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": "not found"}`)),
				},
			},
		}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"assignmentGroup": "invalid-grp"},
			HTTP:          httpContext,
			Integration:   oauthIntegrationContext(),
			Metadata:      &contexts.MetadataContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error verifying assignment group")
	})
}

func Test__GetIncidents__Execute(t *testing.T) {
	component := &GetIncidents{}

	t.Run("high urgency incidents emit to high channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"result": [
							{
								"sys_id": "abc123",
								"number": "INC0010001",
								"short_description": "Server down",
								"state": "1",
								"urgency": "1",
								"impact": "1"
							}
						]
					}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"state": "1,2",
			},
			HTTP:           httpContext,
			Integration:    oauthIntegrationContext(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, PayloadTypeIncidents, execCtx.Type)
		assert.Equal(t, ChannelNameHigh, execCtx.Channel)
	})

	t.Run("low urgency incidents emit to low channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"result": [
							{
								"sys_id": "abc123",
								"number": "INC0010001",
								"short_description": "Minor issue",
								"state": "1",
								"urgency": "3",
								"impact": "3"
							}
						]
					}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{},
			HTTP:           httpContext,
			Integration:    oauthIntegrationContext(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ChannelNameLow, execCtx.Channel)
	})

	t.Run("no incidents emit to clear channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": []}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{},
			HTTP:           httpContext,
			Integration:    oauthIntegrationContext(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ChannelNameClear, execCtx.Channel)
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

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{},
			HTTP:           httpContext,
			Integration:    oauthIntegrationContext(),
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get incidents")
	})

	t.Run("builds query with all filters", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": []}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"assignmentGroup": "grp1",
				"assignedTo":      "user1",
				"caller":          "user2",
				"category":        "software",
				"state":           "1,2",
				"urgency":         "1",
				"impact":          "1,2",
				"limit":           20,
			},
			HTTP:           httpContext,
			Integration:    oauthIntegrationContext(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		requestURL := httpContext.Requests[0].URL.String()
		assert.Contains(t, requestURL, "assignment_group%3Dgrp1")
		assert.Contains(t, requestURL, "assigned_to%3Duser1")
		assert.Contains(t, requestURL, "caller_id%3Duser2")
		assert.Contains(t, requestURL, "category%3Dsoftware")
		assert.Contains(t, requestURL, "stateIN1%2C2")
		assert.Contains(t, requestURL, "urgencyIN1")
		assert.Contains(t, requestURL, "impactIN1%2C2")
		assert.Contains(t, requestURL, "sysparm_limit=20")
	})

	t.Run("builds query with subcategory, priority, and service filters", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": []}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"category":    "software",
				"subcategory": "email",
				"priority":    "1,2",
				"service":     "svc123",
			},
			HTTP:           httpContext,
			Integration:    oauthIntegrationContext(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		requestURL := httpContext.Requests[0].URL.String()
		assert.Contains(t, requestURL, "category%3Dsoftware")
		assert.Contains(t, requestURL, "subcategory%3Demail")
		assert.Contains(t, requestURL, "priorityIN1%2C2")
		assert.Contains(t, requestURL, "business_service%3Dsvc123")
	})
}

func Test__GetIncidents__DetermineOutputChannel(t *testing.T) {
	component := &GetIncidents{}

	t.Run("returns clear when no incidents", func(t *testing.T) {
		channel := component.determineOutputChannel([]IncidentRecord{})
		assert.Equal(t, ChannelNameClear, channel)
	})

	t.Run("returns high when high urgency incident exists", func(t *testing.T) {
		incidents := []IncidentRecord{
			{SysID: "abc", Urgency: "1"},
		}
		channel := component.determineOutputChannel(incidents)
		assert.Equal(t, ChannelNameHigh, channel)
	})

	t.Run("returns low when only medium/low urgency", func(t *testing.T) {
		incidents := []IncidentRecord{
			{SysID: "abc", Urgency: "2"},
			{SysID: "def", Urgency: "3"},
		}
		channel := component.determineOutputChannel(incidents)
		assert.Equal(t, ChannelNameLow, channel)
	})

	t.Run("returns high when mixed urgency (highest wins)", func(t *testing.T) {
		incidents := []IncidentRecord{
			{SysID: "abc", Urgency: "3"},
			{SysID: "def", Urgency: "1"},
		}
		channel := component.determineOutputChannel(incidents)
		assert.Equal(t, ChannelNameHigh, channel)
	})
}
