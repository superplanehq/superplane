package grafana

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

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
		},
	}
	client := &Client{
		BaseURL:  "https://grafana.example.com",
		APIToken: "token",
		http:     httpContext,
	}

	incident, err := client.DeclareIncident(" API latency ", " minor ", "pool exhaustion", []string{" prod ", "", "api"}, true)
	require.NoError(t, err)
	require.Equal(t, "incident-123", incident.IncidentID)
	require.Equal(t, "https://grafana.example.com/a/grafana-irm-app/incidents/incident-123", incident.IncidentURL)
	require.Equal(t, "https://grafana.example.com/a/grafana-irm-app/incidents/incident-123/title", incident.OverviewURL)

	require.Len(t, httpContext.Requests, 1)
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
	require.Equal(t, "active", payload["status"])
	require.Equal(t, "incident", payload["roomPrefix"])
	require.Equal(t, true, payload["isDrill"])
	require.Equal(t, "pool exhaustion", payload["initialStatusUpdate"])
	require.Len(t, payload["labels"], 2)
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
