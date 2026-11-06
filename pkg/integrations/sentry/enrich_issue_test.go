package sentry

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type failingEnrichOutput struct{ contexts.ExecutionStateContext }

func (failingEnrichOutput) Emit(string, string, []any) error {
	return errors.New("output store unavailable")
}

type deadlineSentryHTTP struct{ calls int }

func (h *deadlineSentryHTTP) Do(*http.Request) (*http.Response, error) {
	h.calls++
	return nil, context.DeadlineExceeded
}

func TestEnrichIssueUsesSentryDetailsAndFallsBackToWebhook(t *testing.T) {
	issue := map[string]any{
		"id": "123", "title": "Broken deploy",
		"permalink": "https://sentry.io/issues/123/",
	}
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{"baseUrl": "https://sentry.io", "userToken": "token"},
		Metadata:      Metadata{Organization: &OrganizationSummary{Slug: "example"}},
	}

	for _, test := range []struct {
		name      string
		responses []*http.Response
		want      string
	}{
		{
			name: "rich description",
			responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `{"id":"123","title":"Broken deploy","permalink":"https://sentry.io/issues/123/","count":"8"}`),
				sentryMockResponse(http.StatusOK, `{"eventID":"evt-1","entries":[{"type":"exception","data":{"values":[{"type":"TypeError","value":"boom","stacktrace":{"frames":[{"filename":"app.go","function":"Handle","lineNo":22,"inApp":true}]}}]}}]}`),
			},
			want: "Handle (app.go:22) [in app]",
		},
		{
			name: "Sentry API error",
			responses: []*http.Response{
				sentryMockResponse(http.StatusServiceUnavailable, `{"detail":"unavailable"}`),
				sentryMockResponse(http.StatusServiceUnavailable, `{"detail":"unavailable"}`),
				sentryMockResponse(http.StatusServiceUnavailable, `{"detail":"unavailable"}`),
			},
			want: "[View in Sentry](https://sentry.io/issues/123/)",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{Responses: test.responses}
			state := &contexts.ExecutionStateContext{}
			err := (&EnrichIssue{}).Execute(core.ExecutionContext{
				Configuration: map[string]any{"issue": issue},
				HTTP:          httpCtx, Integration: integration,
				ExecutionState: state, Logger: logrus.NewEntry(logrus.New()),
			})
			require.NoError(t, err)
			require.True(t, state.Passed)
			require.Len(t, state.Payloads, 1)
			data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
			assert.Contains(t, data["description"], test.want)
			if test.name == "Sentry API error" {
				assert.Len(t, httpCtx.Requests, 3)
			} else {
				assert.Len(t, httpCtx.Requests, 2)
			}
			if test.name == "Sentry API error" {
				assert.Equal(t, IssueDescription(issue, nil), data["description"])
			}
		})
	}
}

func TestEnrichIssueRequiresIssueConfiguration(t *testing.T) {
	component := &EnrichIssue{}
	require.Error(t, component.Setup(core.SetupContext{Configuration: map[string]any{}}))
	require.Error(t, component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{},
		ExecutionState: &contexts.ExecutionStateContext{},
	}))
	require.ErrorContains(t, component.Execute(core.ExecutionContext{
		Configuration: map[string]any{"issue": map[string]any{"id": "123"}},
		HTTP:          &contexts.HTTPContext{},
	}), "Sentry integration is not connected")
}

func TestEnrichIssuePropagatesOutputPersistenceError(t *testing.T) {
	err := (&EnrichIssue{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{"issue": map[string]any{"id": "123"}},
		HTTP:          &contexts.HTTPContext{},
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"baseUrl": "https://sentry.io", "userToken": "token"},
		},
		ExecutionState: &failingEnrichOutput{},
	})
	require.ErrorContains(t, err, "output store unavailable")
}

func TestEnrichIssueContinuesAfterSentryTimeout(t *testing.T) {
	issue := map[string]any{"id": "123", "title": "Broken deploy"}
	state := &contexts.ExecutionStateContext{}
	httpClient := &deadlineSentryHTTP{}
	err := (&EnrichIssue{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{"issue": issue},
		HTTP:          httpClient,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"baseUrl": "https://sentry.io", "userToken": "token"},
			Metadata:      Metadata{Organization: &OrganizationSummary{Slug: "example"}},
		},
		ExecutionState: state,
		Logger:         logrus.NewEntry(logrus.New()),
	})
	require.NoError(t, err)
	assert.Positive(t, httpClient.calls, "Sentry API was not called")
	require.Len(t, state.Payloads, 1)
	data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, IssueDescription(issue, nil), data["description"])
}
