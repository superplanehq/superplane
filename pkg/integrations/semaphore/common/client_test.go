package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const testOrgURL = "https://acme.semaphoreci.com"

func newTestClient(responses ...*http.Response) (*Client, *contexts.HTTPContext) {
	httpCtx := &contexts.HTTPContext{Responses: responses}
	client := &Client{OrgURL: testOrgURL, APIToken: "token-123", http: httpCtx}
	return client, httpCtx
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// closeTrackingBody records whether the response body was closed.
type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true
	return nil
}

func requestBody(t *testing.T, req *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	return body
}

func Test__Semaphore__Common__IsNotFoundError(t *testing.T) {
	t.Run("404 HTTP error", func(t *testing.T) {
		assert.True(t, IsNotFoundError(&HTTPError{StatusCode: http.StatusNotFound, Body: "not found"}))
	})

	t.Run("wrapped 404 HTTP error", func(t *testing.T) {
		err := fmt.Errorf("error deleting notification: %w", &HTTPError{StatusCode: http.StatusNotFound})
		assert.True(t, IsNotFoundError(err))
	})

	t.Run("HTTP error with another status", func(t *testing.T) {
		assert.False(t, IsNotFoundError(&HTTPError{StatusCode: http.StatusInternalServerError}))
	})

	t.Run("plain error", func(t *testing.T) {
		assert.False(t, IsNotFoundError(errors.New("not found")))
	})

	t.Run("nil error", func(t *testing.T) {
		assert.False(t, IsNotFoundError(nil))
	})
}

func Test__Semaphore__Common__HTTPError(t *testing.T) {
	err := &HTTPError{StatusCode: http.StatusForbidden, Body: `{"message":"forbidden"}`}
	assert.Equal(t, `request got 403 code: {"message":"forbidden"}`, err.Error())
}

func Test__Semaphore__Common__NewClientWithAPIToken(t *testing.T) {
	t.Run("builds client from organization URL property", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{
			CurrentProperties: map[string]any{"organizationUrl": testOrgURL},
		}

		client, err := NewClientWithAPIToken(&contexts.HTTPContext{}, intCtx.Properties(), "api-token")
		require.NoError(t, err)
		assert.Equal(t, testOrgURL, client.OrgURL)
		assert.Equal(t, "api-token", client.APIToken)
	})

	t.Run("missing organization URL", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}

		_, err := NewClientWithAPIToken(&contexts.HTTPContext{}, intCtx.Properties(), "api-token")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error getting organization URL")
	})
}

func Test__Semaphore__Common__NewClientWithStorageContexts(t *testing.T) {
	t.Run("builds client from property and secret storage", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{
			CurrentProperties: map[string]any{"organizationUrl": testOrgURL},
			CurrentSecrets: map[string]core.IntegrationSecret{
				"apiToken": {Name: "apiToken", Value: []byte("secret-token")},
			},
		}

		client, err := NewClientWithStorageContexts(&contexts.HTTPContext{}, intCtx.Properties(), intCtx.Secrets())
		require.NoError(t, err)
		assert.Equal(t, testOrgURL, client.OrgURL)
		assert.Equal(t, "secret-token", client.APIToken)
	})

	t.Run("missing organization URL", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				"apiToken": {Name: "apiToken", Value: []byte("secret-token")},
			},
		}

		_, err := NewClientWithStorageContexts(&contexts.HTTPContext{}, intCtx.Properties(), intCtx.Secrets())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error getting organization URL")
	})

	t.Run("missing API token secret", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{
			CurrentProperties: map[string]any{"organizationUrl": testOrgURL},
		}

		_, err := NewClientWithStorageContexts(&contexts.HTTPContext{}, intCtx.Properties(), intCtx.Secrets())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret not found: apiToken")
	})
}

func Test__Semaphore__Common__NewClient(t *testing.T) {
	t.Run("legacy setup reads the integration configuration", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": testOrgURL,
				"apiToken":        "legacy-token",
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, intCtx)
		require.NoError(t, err)
		assert.Equal(t, testOrgURL, client.OrgURL)
		assert.Equal(t, "legacy-token", client.APIToken)
	})

	t.Run("legacy setup without organization URL", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "legacy-token"},
		}

		_, err := NewClient(&contexts.HTTPContext{}, intCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "config not found: organizationUrl")
	})

	t.Run("legacy setup without API token", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"organizationUrl": testOrgURL},
		}

		_, err := NewClient(&contexts.HTTPContext{}, intCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "config not found: apiToken")
	})

	t.Run("new setup flow reads properties and secrets", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{
			NewSetupFlow: true,
			// Configuration is ignored in the new flow.
			Configuration: map[string]any{
				"organizationUrl": "https://ignored.semaphoreci.com",
				"apiToken":        "ignored",
			},
			CurrentProperties: map[string]any{"organizationUrl": testOrgURL},
			CurrentSecrets: map[string]core.IntegrationSecret{
				"apiToken": {Name: "apiToken", Value: []byte("stored-token")},
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, intCtx)
		require.NoError(t, err)
		assert.Equal(t, testOrgURL, client.OrgURL)
		assert.Equal(t, "stored-token", client.APIToken)
	})
}

func Test__Semaphore__Common__ExecRequest(t *testing.T) {
	t.Run("sets JSON and token authorization headers", func(t *testing.T) {
		client, httpCtx := newTestClient(jsonResponse(http.StatusOK, `{"pipeline":{"ppl_id":"p1"}}`))

		_, err := client.GetPipeline("p1")
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, testOrgURL+"/api/v1alpha/pipelines/p1", req.URL.String())
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		assert.Equal(t, "Token token-123", req.Header.Get("Authorization"))
	})

	t.Run("closes the response body", func(t *testing.T) {
		body := &closeTrackingBody{Reader: strings.NewReader(`{"pipeline":{"ppl_id":"p1"}}`)}
		client, _ := newTestClient(&http.Response{StatusCode: http.StatusOK, Body: body})

		_, err := client.GetPipeline("p1")
		require.NoError(t, err)
		assert.True(t, body.closed)
	})

	t.Run("closes the response body on HTTP error status", func(t *testing.T) {
		body := &closeTrackingBody{Reader: strings.NewReader("boom")}
		client, _ := newTestClient(&http.Response{StatusCode: http.StatusInternalServerError, Body: body})

		_, err := client.GetPipeline("p1")
		require.Error(t, err)
		assert.True(t, body.closed)
	})

	t.Run("non-2xx status returns HTTPError with status and body", func(t *testing.T) {
		client, _ := newTestClient(jsonResponse(http.StatusNotFound, `{"message":"missing"}`))

		_, err := client.GetPipeline("p1")
		require.Error(t, err)

		var httpErr *HTTPError
		require.ErrorAs(t, err, &httpErr)
		assert.Equal(t, http.StatusNotFound, httpErr.StatusCode)
		assert.Equal(t, `{"message":"missing"}`, httpErr.Body)
		assert.True(t, IsNotFoundError(err))
	})

	t.Run("204 is accepted as success", func(t *testing.T) {
		client, _ := newTestClient(jsonResponse(http.StatusNoContent, ""))

		require.NoError(t, client.DeleteNotification("n1"))
	})

	t.Run("transport error is wrapped", func(t *testing.T) {
		client, _ := newTestClient() // no responses mocked → Do returns an error

		_, err := client.GetPipeline("p1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error executing request")
	})

	t.Run("invalid JSON response is reported", func(t *testing.T) {
		client, _ := newTestClient(jsonResponse(http.StatusOK, `not json`))

		_, err := client.GetPipeline("p1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error unmarshaling response")
	})
}

func Test__Semaphore__Common__GetProject(t *testing.T) {
	const projectID = "5e2f1c3a-9d1b-4f9f-8a0c-3b6e5f8d1a2b"
	projectsJSON := `[
		{"metadata":{"name":"first","id":"11111111-1111-1111-1111-111111111111"},"spec":{"repository":{"url":"git@github.com:acme/first.git"}}},
		{"metadata":{"name":"second","id":"` + projectID + `"},"spec":{"repository":{"url":"git@github.com:acme/second.git"}}}
	]`

	t.Run("by name fetches the project endpoint directly", func(t *testing.T) {
		client, httpCtx := newTestClient(jsonResponse(http.StatusOK, `{"metadata":{"name":"my-project","id":"`+projectID+`"},"spec":{"repository":{"url":"git@github.com:acme/my-project.git"}}}`))

		project, err := client.GetProject("my-project")
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, testOrgURL+"/api/v1alpha/projects/my-project", httpCtx.Requests[0].URL.String())
		assert.Equal(t, "my-project", project.Metadata.ProjectName)
		assert.Equal(t, projectID, project.Metadata.ProjectID)
		assert.Equal(t, "git@github.com:acme/my-project.git", project.Spec.Repository.URL)
	})

	t.Run("by ID lists projects and matches on ID", func(t *testing.T) {
		client, httpCtx := newTestClient(jsonResponse(http.StatusOK, projectsJSON))

		project, err := client.GetProject(projectID)
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, testOrgURL+"/api/v1alpha/projects", httpCtx.Requests[0].URL.String())
		assert.Equal(t, "second", project.Metadata.ProjectName)
		assert.Equal(t, projectID, project.Metadata.ProjectID)
	})

	t.Run("by ID not in the list", func(t *testing.T) {
		client, _ := newTestClient(jsonResponse(http.StatusOK, projectsJSON))

		_, err := client.GetProject("99999999-9999-9999-9999-999999999999")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project 99999999-9999-9999-9999-999999999999 not found")
	})

	t.Run("by ID propagates list errors", func(t *testing.T) {
		client, _ := newTestClient(jsonResponse(http.StatusUnauthorized, "unauthorized"))

		_, err := client.GetProject(projectID)
		require.Error(t, err)

		var httpErr *HTTPError
		require.ErrorAs(t, err, &httpErr)
		assert.Equal(t, http.StatusUnauthorized, httpErr.StatusCode)
	})
}

func Test__Semaphore__Common__ListProjects(t *testing.T) {
	client, httpCtx := newTestClient(jsonResponse(http.StatusOK, `[
		{"metadata":{"name":"a","id":"id-a"},"spec":{"repository":{"url":"url-a"}}},
		{"metadata":{"name":"b","id":"id-b"},"spec":{"repository":{"url":"url-b"}}}
	]`))

	projects, err := client.ListProjects()
	require.NoError(t, err)

	assert.Equal(t, testOrgURL+"/api/v1alpha/projects", httpCtx.Requests[0].URL.String())
	require.Len(t, projects, 2)
	assert.Equal(t, "a", projects[0].Metadata.ProjectName)
	assert.Equal(t, "id-b", projects[1].Metadata.ProjectID)
	assert.Equal(t, "url-b", projects[1].Spec.Repository.URL)
}

func Test__Semaphore__Common__GetPipeline(t *testing.T) {
	client, _ := newTestClient(jsonResponse(http.StatusOK, `{"pipeline":{
		"name":"Build","ppl_id":"ppl-1","wf_id":"wf-1","state":"done","result":"passed",
		"result_reason":"test","branch_name":"main","commit_sha":"abc123","commit_message":"msg",
		"yaml_file_name":".semaphore/semaphore.yml","working_directory":".semaphore","project_id":"proj-1",
		"created_at":"2026-01-01T00:00:00Z","done_at":"2026-01-01T00:10:00Z","running_at":"2026-01-01T00:01:00Z",
		"error_description":"","terminated_by":"","promotion_of":"ppl-0"
	}}`))

	pipeline, err := client.GetPipeline("ppl-1")
	require.NoError(t, err)

	assert.Equal(t, "Build", pipeline.PipelineName)
	assert.Equal(t, "ppl-1", pipeline.PipelineID)
	assert.Equal(t, "wf-1", pipeline.WorkflowID)
	assert.Equal(t, "done", pipeline.State)
	assert.Equal(t, "passed", pipeline.Result)
	assert.Equal(t, "test", pipeline.ResultReason)
	assert.Equal(t, "main", pipeline.BranchName)
	assert.Equal(t, "abc123", pipeline.CommitSHA)
	assert.Equal(t, "msg", pipeline.CommitMessage)
	assert.Equal(t, ".semaphore/semaphore.yml", pipeline.YAMLFileName)
	assert.Equal(t, ".semaphore", pipeline.WorkingDirectory)
	assert.Equal(t, "proj-1", pipeline.ProjectID)
	assert.Equal(t, "2026-01-01T00:00:00Z", pipeline.CreatedAt)
	assert.Equal(t, "2026-01-01T00:10:00Z", pipeline.DoneAt)
	assert.Equal(t, "2026-01-01T00:01:00Z", pipeline.RunningAt)
	assert.Equal(t, "ppl-0", pipeline.PromotionOf)
}

func Test__Semaphore__Common__ListPipelines(t *testing.T) {
	client, httpCtx := newTestClient(jsonResponse(http.StatusOK, `[{"ppl_id":"p1"},{"ppl_id":"p2"}]`))

	pipelines, err := client.ListPipelines("proj-1")
	require.NoError(t, err)

	assert.Equal(t, testOrgURL+"/api/v1alpha/pipelines?project_id=proj-1", httpCtx.Requests[0].URL.String())
	require.Len(t, pipelines, 2)
	assert.Equal(t, "p2", pipelines[1].(map[string]any)["ppl_id"])
}

func Test__Semaphore__Common__RunWorkflow(t *testing.T) {
	client, httpCtx := newTestClient(jsonResponse(http.StatusOK, `{"workflow_id":"wf-9","pipeline_id":"ppl-9"}`))

	response, err := client.RunWorkflow(map[string]any{
		"project_id":    "proj-1",
		"reference":     "refs/heads/main",
		"pipeline_file": ".semaphore/deploy.yml",
		"commit_sha":    "abc123",
		"parameters":    map[string]any{"ENV": "prod"},
	})
	require.NoError(t, err)

	require.Len(t, httpCtx.Requests, 1)
	req := httpCtx.Requests[0]
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, testOrgURL+"/api/v1alpha/plumber-workflows", req.URL.String())

	body := requestBody(t, req)
	assert.Equal(t, "proj-1", body["project_id"])
	assert.Equal(t, "refs/heads/main", body["reference"])
	assert.Equal(t, ".semaphore/deploy.yml", body["pipeline_file"])
	assert.Equal(t, map[string]any{"ENV": "prod"}, body["parameters"])

	assert.Equal(t, "wf-9", response.WorkflowID)
	assert.Equal(t, "ppl-9", response.PipelineID)
}

func Test__Semaphore__Common__Notifications(t *testing.T) {
	t.Run("GetNotification parses rules", func(t *testing.T) {
		client, httpCtx := newTestClient(jsonResponse(http.StatusOK, `{
			"apiVersion":"v1alpha","kind":"Notification",
			"metadata":{"id":"n-1","name":"superplane-hook"},
			"spec":{"rules":[{
				"name":"all",
				"filter":{"branches":["main"],"pipelines":["deploy"],"projects":["proj"],"results":["passed","failed"]},
				"notify":{"webhook":{"endpoint":"https://hooks.example.com/x","secret":"WEBHOOK_SECRET"}}
			}]}
		}`))

		notification, err := client.GetNotification("n-1")
		require.NoError(t, err)

		assert.Equal(t, testOrgURL+"/api/v1alpha/notifications/n-1", httpCtx.Requests[0].URL.String())
		assert.Equal(t, "n-1", notification.Metadata.ID)
		assert.Equal(t, "superplane-hook", notification.Metadata.Name)
		require.Len(t, notification.Spec.Rules, 1)
		rule := notification.Spec.Rules[0]
		assert.Equal(t, "all", rule.Name)
		assert.Equal(t, []string{"main"}, rule.Filter.Branches)
		assert.Equal(t, []string{"passed", "failed"}, rule.Filter.Results)
		assert.Equal(t, "https://hooks.example.com/x", rule.Notify.Webhook.Endpoint)
		assert.Equal(t, "WEBHOOK_SECRET", rule.Notify.Webhook.Secret)
	})

	t.Run("CreateNotification stamps API version and kind", func(t *testing.T) {
		client, httpCtx := newTestClient(jsonResponse(http.StatusOK, `{"metadata":{"id":"n-2","name":"hook"}}`))

		created, err := client.CreateNotification(&Notification{
			Metadata: NotificationMetadata{Name: "hook"},
			Spec: NotificationSpec{Rules: []NotificationRule{{
				Name:   "rule",
				Filter: NotificationRuleFilter{Projects: []string{"proj"}},
				Notify: NotificationRuleNotify{Webhook: NotificationNotifyWebhook{Endpoint: "https://hooks.example.com/x", Secret: "s"}},
			}}},
		})
		require.NoError(t, err)

		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, testOrgURL+"/api/v1alpha/notifications", req.URL.String())

		body := requestBody(t, req)
		assert.Equal(t, "v1alpha", body["apiVersion"])
		assert.Equal(t, "Notification", body["kind"])
		metadata := body["metadata"].(map[string]any)
		assert.Equal(t, "hook", metadata["name"])
		rules := body["spec"].(map[string]any)["rules"].([]any)
		require.Len(t, rules, 1)
		webhook := rules[0].(map[string]any)["notify"].(map[string]any)["webhook"].(map[string]any)
		assert.Equal(t, "https://hooks.example.com/x", webhook["endpoint"])

		assert.Equal(t, "n-2", created.Metadata.ID)
	})

	t.Run("CreateNotification rejects other parameter types", func(t *testing.T) {
		client, httpCtx := newTestClient()

		_, err := client.CreateNotification(map[string]any{"name": "hook"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid params type")
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("DeleteNotification sends DELETE and wraps errors", func(t *testing.T) {
		client, httpCtx := newTestClient(
			jsonResponse(http.StatusOK, ""),
			jsonResponse(http.StatusNotFound, "not found"),
		)

		require.NoError(t, client.DeleteNotification("n-1"))
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodDelete, req.Method)
		assert.Equal(t, testOrgURL+"/api/v1alpha/notifications/n-1", req.URL.String())

		err := client.DeleteNotification("n-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error deleting notification")
		assert.True(t, IsNotFoundError(err))
	})
}

func Test__Semaphore__Common__Secrets(t *testing.T) {
	t.Run("GetSecret parses env vars", func(t *testing.T) {
		client, httpCtx := newTestClient(jsonResponse(http.StatusOK, `{
			"apiVersion":"v1beta","kind":"Secret",
			"metadata":{"id":"s-1","name":"superplane-secret"},
			"data":{"env_vars":[{"name":"WEBHOOK_SECRET","value":"abc"}]}
		}`))

		secret, err := client.GetSecret("s-1")
		require.NoError(t, err)

		assert.Equal(t, testOrgURL+"/api/v1beta/secrets/s-1", httpCtx.Requests[0].URL.String())
		assert.Equal(t, "superplane-secret", secret.Metadata.Name)
		require.Len(t, secret.Data.EnvVars, 1)
		assert.Equal(t, "WEBHOOK_SECRET", secret.Data.EnvVars[0].Name)
		assert.Equal(t, "abc", secret.Data.EnvVars[0].Value)
	})

	t.Run("CreateWebhookSecret sends a v1beta secret with WEBHOOK_SECRET", func(t *testing.T) {
		client, httpCtx := newTestClient(jsonResponse(http.StatusOK, `{"metadata":{"id":"s-2","name":"hook-secret"}}`))

		secret, err := client.CreateWebhookSecret("hook-secret", "shared-key")
		require.NoError(t, err)

		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, testOrgURL+"/api/v1beta/secrets", req.URL.String())

		body := requestBody(t, req)
		assert.Equal(t, "v1beta", body["apiVersion"])
		assert.Equal(t, "Secret", body["kind"])
		assert.Equal(t, "hook-secret", body["metadata"].(map[string]any)["name"])
		envVars := body["data"].(map[string]any)["env_vars"].([]any)
		require.Len(t, envVars, 1)
		assert.Equal(t, map[string]any{"name": "WEBHOOK_SECRET", "value": "shared-key"}, envVars[0])

		assert.Equal(t, "s-2", secret.Metadata.ID)
	})

	t.Run("DeleteSecret sends DELETE and wraps errors", func(t *testing.T) {
		client, httpCtx := newTestClient(
			jsonResponse(http.StatusNoContent, ""),
			jsonResponse(http.StatusInternalServerError, "boom"),
		)

		require.NoError(t, client.DeleteSecret("hook-secret"))
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodDelete, req.Method)
		assert.Equal(t, testOrgURL+"/api/v1beta/secrets/hook-secret", req.URL.String())

		err := client.DeleteSecret("hook-secret")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error deleting secret")
		assert.False(t, IsNotFoundError(err))
	})
}
