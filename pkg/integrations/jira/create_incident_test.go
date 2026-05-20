package jira

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func jiraTestIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: map[string]any{
			"siteUrl":  "https://test.atlassian.net",
			"email":    "test@example.com",
			"apiToken": "test-token",
		},
		Metadata: Metadata{
			CloudID:  "35273b54-3f06-40d2-880f-dd28cf6daafa",
			Projects: []Project{{ID: "1", Key: "IT", Name: "IT"}},
		},
	}
}

func Test__CreateIncident__Setup(t *testing.T) {
	component := CreateIncident{}
	baseCtx := core.SetupContext{Integration: jiraTestIntegration()}

	t.Run("missing serviceDesk", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: baseCtx.Integration,
			Configuration: map[string]any{
				"serviceDeskRequestType": "75",
				"summary":                "Outage",
			},
		})
		require.ErrorContains(t, err, "serviceDesk is required")
	})

	t.Run("missing summary", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: baseCtx.Integration,
			Configuration: map[string]any{
				"serviceDesk":            "6",
				"serviceDeskRequestType": "75",
			},
		})
		require.ErrorContains(t, err, "summary is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"6","projectName":"IT","projectKey":"IT"}],"isLastPage":true}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"75","name":"Incident","practice":"ITSM_INCIDENT"}],"isLastPage":true}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"75","name":"Incident","practice":"ITSM_INCIDENT"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{"requestTypeFields":[
						{"fieldId":"summary","name":"Summary"},
						{"fieldId":"customfield_10020","name":"Impact"},
						{"fieldId":"customfield_10021","name":"Urgency"}
					]}`)),
				},
			},
		}
		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			HTTP:        httpContext,
			Integration: baseCtx.Integration,
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"serviceDesk":            "6",
				"serviceDeskRequestType": "75",
				"summary":                "Database outage",
				"alertIds":               []any{"a1"},
			},
		})
		require.NoError(t, err)

		var meta CreateIncidentNodeMetadata
		require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &meta))
		assert.Equal(t, "IT (IT)", meta.ServiceDeskName)
		assert.Equal(t, "Incident", meta.RequestTypeName)
		assert.Equal(t, "customfield_10020", meta.ImpactFieldID)
		assert.Equal(t, "customfield_10021", meta.UrgencyFieldID)
	})

	t.Run("valid setup summary in additional fields only", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"6","projectName":"IT","projectKey":"IT"}],"isLastPage":true}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"75","name":"Incident","practice":"ITSM_INCIDENT"}],"isLastPage":true}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"75","name":"Incident","practice":"ITSM_INCIDENT"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"requestTypeFields":[{"fieldId":"summary","name":"Summary"}]}`)),
				},
			},
		}
		err := component.Setup(core.SetupContext{
			HTTP:        httpContext,
			Integration: baseCtx.Integration,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"serviceDesk":            "6",
				"serviceDeskRequestType": "75",
				"additionalFields":       map[string]any{"summary": "From additional fields"},
			},
		})
		require.NoError(t, err)
	})
}

func Test__CreateIncident__Execute(t *testing.T) {
	component := CreateIncident{}

	t.Run("successful create", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10050","key":"ITSM-30","self":"https://test.atlassian.net/rest/api/3/issue/10050"}`)),
				},
			},
		}
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"serviceDesk":            "6",
				"serviceDeskRequestType": "75",
				"summary":                "Outage",
			},
			HTTP:           httpContext,
			Integration:    jiraTestIntegration(),
			ExecutionState: execCtx,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: CreateIncidentNodeMetadata{
					ImpactFieldID:  "customfield_10020",
					UrgencyFieldID: "customfield_10021",
				},
			},
		})
		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.Equal(t, CreateJiraIncidentPayloadType, execCtx.Type)
	})
}

func Test__incidentCreateFieldsFromSpec__mergesAndScalars(t *testing.T) {
	spec := CreateIncidentSpec{
		AdditionalFields: map[string]any{"customfield_1": "x"},
		Description:      "Hello",
		DueDate:          "2026-05-20",
		Priority:         "High",
		Impact:           "1",
		Urgency:          "2",
		OriginalEstimate: "2h",
		CustomFieldValues: []IncidentCustomFieldRow{
			{FieldID: "customfield_2", ValueJSON: `{"value":"Urgent"}`},
		},
	}
	meta := CreateIncidentNodeMetadata{
		ImpactFieldID:  "customfield_impact",
		UrgencyFieldID: "customfield_urgency",
	}
	fields, err := incidentCreateFieldsFromSpec(spec, meta)
	require.NoError(t, err)
	assert.Equal(t, "x", fields["customfield_1"])
	desc := fields["description"]
	require.NotNil(t, desc)
	assert.Equal(t, "2026-05-20", fields["duedate"])
	prio := fields["priority"].(map[string]any)
	assert.Equal(t, "High", prio["name"])
	impact := fields["customfield_impact"].(map[string]any)
	assert.Equal(t, "1", impact["id"])
	urgency := fields["customfield_urgency"].(map[string]any)
	assert.Equal(t, "2", urgency["id"])
	tt := fields["timetracking"].(map[string]any)
	assert.Equal(t, "2h", tt["originalEstimate"])
	assert.Equal(t, map[string]any{"value": "Urgent"}, fields["customfield_2"])
}

func Test__incidentCreateFieldsFromSpec__priorityNoneSentinel(t *testing.T) {
	for _, prio := range []string{"", "  ", "__none__"} {
		fields, err := incidentCreateFieldsFromSpec(CreateIncidentSpec{
			Priority: prio,
		}, CreateIncidentNodeMetadata{})
		require.NoError(t, err)
		_, has := fields["priority"]
		assert.False(t, has, "priority=%q", prio)
	}
}

func Test__incidentCreateFieldsFromSpec__impactUrgencySkippedWithoutFieldIDs(t *testing.T) {
	fields, err := incidentCreateFieldsFromSpec(CreateIncidentSpec{
		Impact:  "3",
		Urgency: "4",
	}, CreateIncidentNodeMetadata{})
	require.NoError(t, err)
	_, hasImpact := fields["impact"]
	_, hasUrgency := fields["urgency"]
	assert.False(t, hasImpact)
	assert.False(t, hasUrgency)
}

func Test__incidentCreateFieldsFromSpec__invalidCustomValue(t *testing.T) {
	_, err := incidentCreateFieldsFromSpec(CreateIncidentSpec{
		CustomFieldValues: []IncidentCustomFieldRow{{FieldID: "cf", ValueJSON: `[`}},
	}, CreateIncidentNodeMetadata{})
	require.Error(t, err)
}

func Test__incidentAlertIDsFromSpec__list(t *testing.T) {
	ids := incidentAlertIDsFromSpec(CreateIncidentSpec{
		AlertIDs: []any{"  a ", "b"},
	})
	assert.Equal(t, []string{"a", "b"}, ids)
}

func Test__incidentAlertIDsFromSpec__empty(t *testing.T) {
	assert.Nil(t, incidentAlertIDsFromSpec(CreateIncidentSpec{}))
}

func Test__cloudIDFromIntegration__errors(t *testing.T) {
	app := &contexts.IntegrationContext{
		Configuration: map[string]any{"siteUrl": "https://x.net", "email": "a@b.com", "apiToken": "t"},
		Metadata:      Metadata{Projects: []Project{}},
	}
	_, err := cloudIDFromIntegration(app)
	require.Error(t, err)
}
