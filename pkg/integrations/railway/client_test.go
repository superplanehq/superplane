package railway

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Railway__Client__Verify(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"apiToken":{"workspaces":[]}}}`)),
				},
			},
		}

		client := NewClientWithAPIToken(httpCtx, "test-token")
		err := client.Verify()
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "Bearer test-token", httpCtx.Requests[0].Header.Get("Authorization"))
	})

	t.Run("error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"message":"Not Authorized"}]}`)),
				},
			},
		}

		client := NewClientWithAPIToken(httpCtx, "test-token")
		err := client.Verify()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GraphQL error: Not Authorized")
	})
}

func Test__Railway__Client__ListWorkspaces(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"apiToken":{"workspaces":[{"id":"w-1","name":"WS 1"}]}}}`)),
			},
		},
	}

	client := NewClientWithAPIToken(httpCtx, "test-token")
	workspaces, err := client.ListWorkspaces()
	require.NoError(t, err)
	require.Len(t, workspaces, 1)
	assert.Equal(t, "w-1", workspaces[0].ID)
	assert.Equal(t, "WS 1", workspaces[0].Name)
}

func Test__Railway__Client__ListProjects(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"projects":{"edges":[{"node":{"id":"p-1","name":"Project 1"}}]}}}`)),
			},
		},
	}

	client := NewClientWithAPIToken(httpCtx, "test-token")
	projects, err := client.ListProjects("w-1")
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, "p-1", projects[0].ID)
	assert.Equal(t, "Project 1", projects[0].Name)
}

func Test__Railway__Client__GetProjectDetails(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"project":{"id":"p-1","name":"Project 1","workspaceId":"w-2","services":{"edges":[{"node":{"id":"s-1","name":"Service 1"}}]},"environments":{"edges":[{"node":{"id":"e-1","name":"Env 1"}}]}}}}`)),
			},
		},
	}

	client := NewClientWithAPIToken(httpCtx, "test-token")
	project, err := client.GetProjectDetails("p-1")
	require.NoError(t, err)
	assert.Equal(t, "p-1", project.ID)
	assert.Equal(t, "Project 1", project.Name)
	assert.Equal(t, "w-2", project.WorkspaceID)
	require.Len(t, project.Services.Edges, 1)
	assert.Equal(t, "s-1", project.Services.Edges[0].Node.ID)
	require.Len(t, project.Environments.Edges, 1)
	assert.Equal(t, "e-1", project.Environments.Edges[0].Node.ID)
}

func Test__Railway__Client__TriggerDeploy(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"serviceInstanceDeployV2":"deploy-123"}}`)),
			},
		},
	}

	client := NewClientWithAPIToken(httpCtx, "test-token")
	deployID, err := client.TriggerDeploy("e-1", "s-1")
	require.NoError(t, err)
	assert.Equal(t, "deploy-123", deployID)
}

func Test__Railway__Client__GetDeployment(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"deployment":{"id":"deploy-123","status":"SUCCESS","createdAt":"2026-05-30T00:00:00Z","updatedAt":"2026-05-30T00:01:00Z"}}}`)),
			},
		},
	}

	client := NewClientWithAPIToken(httpCtx, "test-token")
	deploy, err := client.GetDeployment("deploy-123")
	require.NoError(t, err)
	assert.Equal(t, "deploy-123", deploy.ID)
	assert.Equal(t, "SUCCESS", deploy.Status)
}

func Test__Railway__Client__RollbackDeployment(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"deploymentRollback":true}}`)),
			},
		},
	}

	client := NewClientWithAPIToken(httpCtx, "test-token")
	err := client.RollbackDeployment("deploy-123")
	require.NoError(t, err)
}

func Test__Railway__Client__CreateNotificationRule(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"notificationRuleCreate":{"id":"rule-123","projectId":"p-1","eventTypes":["Deployment.deployed"],"severities":[],"ephemeralEnvironments":false,"createdAt":"","updatedAt":"","channels":[]}}}`)),
			},
		},
	}

	client := NewClientWithAPIToken(httpCtx, "test-token")
	rule, err := client.CreateNotificationRule("w-1", "p-1", []string{"Deployment.deployed"}, "https://hook.superplane.dev")
	require.NoError(t, err)
	assert.Equal(t, "rule-123", rule.ID)
}

func Test__Railway__Client__DeleteNotificationRule(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"notificationRuleDelete":true}}`)),
			},
		},
	}

	client := NewClientWithAPIToken(httpCtx, "test-token")
	err := client.DeleteNotificationRule("rule-123")
	require.NoError(t, err)
}
