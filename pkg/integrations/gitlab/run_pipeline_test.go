package gitlab

import (
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__RunPipeline__Execute(t *testing.T) {
	component := &RunPipeline{}
	metadataCtx := &contexts.MetadataContext{}
	requestsCtx := &contexts.RequestContext{}
	executionState := &contexts.ExecutionStateContext{
		KVs: map[string]string{},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"project": "123",
			"ref":     "refs/heads/main",
			"variables": []map[string]any{
				{"name": "DEPLOY_ENV", "value": "dev", "variableType": "env_var"},
			},
		},
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":    AuthTypePersonalAccessToken,
				"groupId":     "123",
				"accessToken": "pat",
				"baseUrl":     "https://gitlab.com",
			},
		},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{
					"id": 1001,
					"iid": 73,
					"project_id": 123,
					"status": "pending",
					"ref": "main",
					"sha": "abc123",
					"web_url": "https://gitlab.com/group/project/-/pipelines/1001"
				}`),
			},
		},
		Metadata:       metadataCtx,
		NodeMetadata:   &contexts.MetadataContext{},
		ExecutionState: executionState,
		Requests:       requestsCtx,
		Logger:         log.NewEntry(log.New()),
	})

	require.NoError(t, err)

	metadata, ok := metadataCtx.Metadata.(RunPipelineExecutionMetadata)
	require.True(t, ok)
	require.NotNil(t, metadata.Pipeline)
	assert.Equal(t, 1001, metadata.Pipeline.ID)
	assert.Equal(t, "pending", metadata.Pipeline.Status)
	assert.Equal(t, "main", metadata.Pipeline.Ref)

	assert.Equal(t, "1001", executionState.KVs[GitLabRunPipelineKVPipelineID])
	assert.Equal(t, "123:1001", executionState.KVs[GitLabRunPipelineKVProjectPipeline])

	assert.Equal(t, GitLabRunPipelinePollAction, requestsCtx.Action)
	assert.Equal(t, GitLabRunPipelinePollInterval, requestsCtx.Duration)
}

func Test__RunPipeline__HandleWebhook__FinishedPipeline(t *testing.T) {
	component := &RunPipeline{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: RunPipelineExecutionMetadata{
			Pipeline: &RunPipelineMetadata{
				ID:     1001,
				Status: "running",
			},
		},
	}
	executionState := &contexts.ExecutionStateContext{
		KVs: map[string]string{},
	}

	code, err := component.HandleWebhook(core.WebhookRequestContext{
		Headers: gitlabHeaders("Pipeline Hook", "token"),
		Body: []byte(`{
			"object_kind": "pipeline",
			"project": {"id": 123},
			"object_attributes": {
				"id": 1001,
				"iid": 73,
				"status": "success",
				"ref": "main",
				"url": "https://gitlab.com/group/project/-/pipelines/1001"
			}
		}`),
		Webhook: &contexts.WebhookContext{
			Secret: "token",
		},
		FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
			if key == GitLabRunPipelineKVProjectPipeline && value == "123:1001" {
				return &core.ExecutionContext{
					Metadata:       metadataCtx,
					ExecutionState: executionState,
				}, nil
			}
			return nil, assert.AnError
		},
	})

	assert.Equal(t, http.StatusOK, code)
	require.NoError(t, err)
	assert.Equal(t, GitLabPipelinePassedOutputChannel, executionState.Channel)
	assert.Equal(t, GitLabPipelinePayloadType, executionState.Type)

	metadata, ok := metadataCtx.Metadata.(RunPipelineExecutionMetadata)
	require.True(t, ok)
	require.NotNil(t, metadata.Pipeline)
	assert.Equal(t, "success", metadata.Pipeline.Status)
}

func Test__RunPipeline__Poll__SchedulesNextWhenRunning(t *testing.T) {
	component := &RunPipeline{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: RunPipelineExecutionMetadata{
			Pipeline: &RunPipelineMetadata{
				ID:     1001,
				Status: "running",
				Ref:    "main",
			},
		},
	}
	requestsCtx := &contexts.RequestContext{}
	executionState := &contexts.ExecutionStateContext{
		KVs: map[string]string{},
	}

	err := component.HandleAction(core.ActionContext{
		Name: GitLabRunPipelinePollAction,
		Configuration: map[string]any{
			"project": "123",
			"ref":     "main",
		},
		Metadata: metadataCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":    AuthTypePersonalAccessToken,
				"groupId":     "123",
				"accessToken": "pat",
				"baseUrl":     "https://gitlab.com",
			},
		},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"id": 1001,
					"iid": 73,
					"project_id": 123,
					"status": "running",
					"ref": "main"
				}`),
			},
		},
		Requests:       requestsCtx,
		ExecutionState: executionState,
		Logger:         log.NewEntry(log.New()),
	})

	require.NoError(t, err)
	assert.Equal(t, GitLabRunPipelinePollAction, requestsCtx.Action)
	assert.Equal(t, GitLabRunPipelinePollInterval, requestsCtx.Duration)
	assert.Empty(t, executionState.Channel)
}
