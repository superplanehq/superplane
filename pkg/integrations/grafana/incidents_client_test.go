package grafana

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Client__DeclareIncident__UsesGrafanaIRMRPC(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident": {
						"incidentID": "incident-123",
						"title": "API latency",
						"severity": "minor",
						"status": "active",
						"overviewURL": "/a/grafana-irm-app/incidents/incident-123/title"
					}
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			},
		},
	}
	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	startTime := time.Date(2026, time.April, 20, 9, 45, 0, 0, time.UTC)

	incident, err := client.DeclareIncident(DeclareIncidentInput{
		Title:               " API latency ",
		Severity:            " minor ",
		InitialStatusUpdate: "pool exhaustion",
		Labels:              []string{" prod ", "", "api"},
		IsDrill:             true,
		Status:              incidentStatusResolved,
		StartTime:           &startTime,
	})
	require.NoError(t, err)
	require.Equal(t, "incident-123", incident.IncidentID)
	require.Equal(t, "https://grafana.example.com/a/grafana-irm-app/incidents/incident-123", incident.IncidentURL)
	require.Equal(t, "https://grafana.example.com/a/grafana-irm-app/incidents/incident-123/title", incident.OverviewURL)
	require.Equal(t, "2026-04-20T09:45:00Z", incident.IncidentStart)

	require.Len(t, httpContext.Requests, 2)
	request := httpContext.Requests[0]
	require.Equal(t, http.MethodPost, request.Method)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.CreateIncident", request.URL.Path)
	require.Equal(t, "Bearer token", request.Header.Get("Authorization"))

	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "API latency", payload["title"])
	require.Equal(t, "minor", payload["severity"])
	require.Equal(t, "resolved", payload["status"])
	require.Equal(t, "incident", payload["roomPrefix"])
	require.Equal(t, true, payload["isDrill"])
	require.Equal(t, "pool exhaustion", payload["initialStatusUpdate"])
	require.Len(t, payload["labels"], 2)

	body, err = io.ReadAll(httpContext.Requests[1].Body)
	require.NoError(t, err)

	var startPayload map[string]string
	require.NoError(t, json.Unmarshal(body, &startPayload))
	require.Equal(t, "incident-123", startPayload["incidentID"])
	require.Equal(t, "incidentStart", startPayload["activityItemKind"])
	require.Equal(t, "incidentStart", startPayload["eventName"])
	require.Equal(t, "2026-04-20T09:45:00Z", startPayload["eventTime"])
}

func Test__Client__ResolveIncident__SetsResolvedStatus(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident": {
						"incidentID": "incident-123",
						"title": "API latency",
						"severity": "minor",
						"status": "resolved"
					}
				}`)),
			},
		},
	}
	client := &Client{BaseURL: "https://grafana.example.com", APIToken: "token", http: httpContext}

	incident, err := client.ResolveIncident("incident-123")
	require.NoError(t, err)
	require.Equal(t, "resolved", incident.Status)

	require.Len(t, httpContext.Requests, 1)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.UpdateStatus", httpContext.Requests[0].URL.Path)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "incident-123", payload["incidentID"])
	require.Equal(t, "resolved", payload["status"])
}

func Test__Client__UpdateIncident__AddsLabels(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"field":{"slug":"tags"}}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident": {
						"incidentID": "incident-123",
						"title": "API latency",
						"severity": "minor",
						"status": "active",
						"labels": [{"label": "prod"}]
					}
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"field":{"slug":"tags"}}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident": {
						"incidentID": "incident-123",
						"title": "API latency",
						"severity": "minor",
						"status": "active",
						"labels": [{"label": "prod"}, {"label": "api"}]
					}
				}`)),
			},
		},
	}
	client := &Client{BaseURL: "https://grafana.example.com", APIToken: "token", http: httpContext}

	incident, err := client.UpdateIncident("incident-123", nil, nil, []string{" prod ", "", "api"}, nil)
	require.NoError(t, err)
	require.Len(t, incident.Labels, 2)

	require.Len(t, httpContext.Requests, 4)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/FieldsService.AddLabelValue", httpContext.Requests[0].URL.Path)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.AddLabel", httpContext.Requests[1].URL.Path)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/FieldsService.AddLabelValue", httpContext.Requests[2].URL.Path)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.AddLabel", httpContext.Requests[3].URL.Path)

	body, err := io.ReadAll(httpContext.Requests[1].Body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "incident-123", payload["incidentID"])
	require.Equal(t, map[string]any{"key": "tags", "label": "prod"}, payload["label"])
}

func Test__Client__UpdateIncident__DedupesConfiguredLabels(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"field":{"slug":"tags"}}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident": {
						"incidentID": "incident-123",
						"title": "API latency",
						"severity": "minor",
						"status": "active",
						"labels": [{"label": "prod"}]
					}
				}`)),
			},
		},
	}
	client := &Client{BaseURL: "https://grafana.example.com", APIToken: "token", http: httpContext}

	incident, err := client.UpdateIncident("incident-123", nil, nil, []string{" prod ", "prod"}, nil)
	require.NoError(t, err)
	require.Len(t, incident.Labels, 1)

	require.Len(t, httpContext.Requests, 2)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/FieldsService.AddLabelValue", httpContext.Requests[0].URL.Path)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.AddLabel", httpContext.Requests[1].URL.Path)
}

func Test__Client__UpdateIncident__IgnoresDuplicateLabelErrors(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(`{"error":"duplicate field option"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"error":"label already exists"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident": {
						"incidentID": "incident-123",
						"title": "API latency",
						"severity": "minor",
						"status": "active",
						"labels": [{"label": "prod"}],
						"isDrill": true
					}
				}`)),
			},
		},
	}
	client := &Client{BaseURL: "https://grafana.example.com", APIToken: "token", http: httpContext}

	incident, err := client.UpdateIncident("incident-123", nil, nil, []string{"prod"}, nil)
	require.NoError(t, err)
	require.Equal(t, "incident-123", incident.IncidentID)
	require.Equal(t, "API latency", incident.Title)
	require.Equal(t, "minor", incident.Severity)
	require.Equal(t, "active", incident.Status)
	require.True(t, incident.IsDrill)
	require.Len(t, incident.Labels, 1)
	require.Equal(t, "prod", incident.Labels[0].Label)

	require.Len(t, httpContext.Requests, 3)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/FieldsService.AddLabelValue", httpContext.Requests[0].URL.Path)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.AddLabel", httpContext.Requests[1].URL.Path)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.GetIncident", httpContext.Requests[2].URL.Path)
}

func Test__Client__AddIncidentActivity__UsesUserNote(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"activityItem": {
						"activityItemID": "activity-123",
						"incidentID": "incident-123",
						"activityKind": "userNote",
						"body": "Root cause found"
					}
				}`)),
			},
		},
	}
	client := &Client{BaseURL: "https://grafana.example.com", APIToken: "token", http: httpContext}

	activity, err := client.AddIncidentActivity("incident-123", " Root cause found ")
	require.NoError(t, err)
	require.Equal(t, "activity-123", activity.ActivityItemID)

	require.Len(t, httpContext.Requests, 1)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/ActivityService.AddActivity", httpContext.Requests[0].URL.Path)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "incident-123", payload["incidentID"])
	require.Equal(t, "userNote", payload["activityKind"])
	require.Equal(t, "Root cause found", payload["body"])
}

func Test__Client__ListIncidents__UsesQueryIncidentPreviews(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incidentPreviews": [
						{"incidentID":"incident-1","title":"API latency","status":"active","severity":"minor"}
					]
				}`)),
			},
		},
	}
	client := &Client{BaseURL: "https://grafana.example.com", APIToken: "token", http: httpContext}

	incidents, err := client.ListIncidents(25)
	require.NoError(t, err)
	require.Len(t, incidents, 1)
	require.Equal(t, "incident-1", incidents[0].IncidentID)
	require.Equal(t, "https://grafana.example.com/a/grafana-irm-app/incidents/incident-1", incidents[0].IncidentURL)

	require.Len(t, httpContext.Requests, 1)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.QueryIncidentPreviews", httpContext.Requests[0].URL.Path)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var payload struct {
		Query struct {
			Limit          int    `json:"limit"`
			OrderField     string `json:"orderField"`
			OrderDirection string `json:"orderDirection"`
		} `json:"query"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, 25, payload.Query.Limit)
	require.Equal(t, "createdTime", payload.Query.OrderField)
	require.Equal(t, "DESC", payload.Query.OrderDirection)
}

func Test__Client__GrafanaIRMRPCErrorFieldIsReturned(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"error":"incident not found"}`)),
			},
		},
	}
	client := &Client{BaseURL: "https://grafana.example.com", APIToken: "token", http: httpContext}

	_, err := client.GetIncident("missing")
	require.ErrorContains(t, err, "incident not found")
}
