package jenkins

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__TriggerBuild__ComponentInfo(t *testing.T) {
	component := TriggerBuild{}

	assert.Equal(t, "jenkins.triggerBuild", component.Name())
	assert.Equal(t, "Trigger Build", component.Label())
	assert.Equal(t, "Trigger a Jenkins build and wait for completion", component.Description())
	assert.Equal(t, "jenkins", component.Icon())
	assert.Equal(t, "gray", component.Color())
	assert.NotEmpty(t, component.Documentation())
}

func Test__TriggerBuild__Configuration(t *testing.T) {
	component := TriggerBuild{}
	config := component.Configuration()

	assert.Len(t, config, 2)

	fieldNames := make([]string, len(config))
	for i, f := range config {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "job")
	assert.Contains(t, fieldNames, "parameters")

	for _, f := range config {
		if f.Name == "job" {
			assert.True(t, f.Required)
		}
		if f.Name == "parameters" {
			assert.False(t, f.Required)
		}
	}
}

func Test__TriggerBuild__OutputChannels(t *testing.T) {
	component := TriggerBuild{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 2)
	assert.Equal(t, PassedOutputChannel, channels[0].Name)
	assert.Equal(t, FailedOutputChannel, channels[1].Name)
}

func Test__TriggerBuild__Setup(t *testing.T) {
	component := TriggerBuild{}

	t.Run("missing job -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		err := component.Setup(core.SetupContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   appCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "job is required")
	})

	t.Run("job not found -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`Not found`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		err := component.Setup(core.SetupContext{
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"job": "nonexistent-job",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error finding job")
	})

	t.Run("valid setup", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"name":"my-job","fullName":"my-job","url":"https://jenkins.example.com/job/my-job/","color":"blue"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata:    metadataCtx,
			Webhook:     &contexts.WebhookContext{},
			Configuration: map[string]any{
				"job": "my-job",
			},
		})

		require.NoError(t, err)

		metadata, ok := metadataCtx.Metadata.(TriggerBuildNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "my-job", metadata.Job.Name)
		require.Len(t, appCtx.WebhookRequests, 1)
	})

	t.Run("already set up for same job -> no-op", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: TriggerBuildNodeMetadata{
				Job: &JobInfo{Name: "my-job", URL: "https://jenkins.example.com/job/my-job/"},
			},
		}

		err := component.Setup(core.SetupContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: appCtx,
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"job": "my-job",
			},
		})

		require.NoError(t, err)
	})
}

func Test__TriggerBuild__Execute(t *testing.T) {
	component := TriggerBuild{}

	t.Run("successful trigger", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Header:     http.Header{"Location": []string{"https://jenkins.example.com/queue/item/42/"}},
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		execMetadata := &contexts.MetadataContext{}
		nodeMetadata := &contexts.MetadataContext{
			Metadata: TriggerBuildNodeMetadata{
				Job: &JobInfo{Name: "my-job", URL: "https://jenkins.example.com/job/my-job/"},
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		reqCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"job": "my-job",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			Metadata:       execMetadata,
			NodeMetadata:   nodeMetadata,
			ExecutionState: execState,
			Requests:       reqCtx,
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)

		metadata, ok := execMetadata.Metadata.(TriggerBuildExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, int64(42), metadata.QueueItemID)
		assert.Equal(t, "42", execState.KVs["queueItem"])
		assert.Equal(t, "my-job", execState.KVs["buildJob"])
		assert.Equal(t, "poll", reqCtx.Action)
	})

	t.Run("trigger failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`Not found`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		nodeMetadata := &contexts.MetadataContext{
			Metadata: TriggerBuildNodeMetadata{
				Job: &JobInfo{Name: "my-job", URL: "https://jenkins.example.com/job/my-job/"},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"job": "my-job",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   nodeMetadata,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Requests:       &contexts.RequestContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error triggering build")
	})
}

func Test__TriggerBuild__Poll(t *testing.T) {
	component := TriggerBuild{}

	t.Run("already finished -> noop", func(t *testing.T) {
		err := component.poll(core.ActionContext{
			ExecutionState: &contexts.ExecutionStateContext{Finished: true},
		})

		require.NoError(t, err)
	})

	t.Run("build still in queue -> reschedule", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":42,"blocked":false}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		reqCtx := &contexts.RequestContext{}
		err := component.poll(core.ActionContext{
			Configuration: map[string]any{
				"job": "my-job",
			},
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata: &contexts.MetadataContext{
				Metadata: TriggerBuildExecutionMetadata{
					Job:         &JobInfo{Name: "my-job"},
					QueueItemID: 42,
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Requests:       reqCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", reqCtx.Action)
		assert.Equal(t, QueuePollInterval, reqCtx.Duration)
	})

	t.Run("build running -> reschedule", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"number":1,"url":"https://jenkins.example.com/job/my-job/1/","result":null,"building":true}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		reqCtx := &contexts.RequestContext{}
		err := component.poll(core.ActionContext{
			Configuration: map[string]any{
				"job": "my-job",
			},
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata: &contexts.MetadataContext{
				Metadata: TriggerBuildExecutionMetadata{
					Job:         &JobInfo{Name: "my-job"},
					QueueItemID: 42,
					Build:       &BuildInfo{Number: 1, Building: true},
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Requests:       reqCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", reqCtx.Action)
		assert.Equal(t, PollInterval, reqCtx.Duration)
	})

	t.Run("build succeeded -> emit passed", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"number":1,"url":"https://jenkins.example.com/job/my-job/1/","result":"SUCCESS","building":false,"duration":120000}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.poll(core.ActionContext{
			Configuration: map[string]any{
				"job": "my-job",
			},
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata: &contexts.MetadataContext{
				Metadata: TriggerBuildExecutionMetadata{
					Job:         &JobInfo{Name: "my-job", URL: "https://jenkins.example.com/job/my-job/"},
					QueueItemID: 42,
					Build:       &BuildInfo{Number: 1, Building: true},
				},
			},
			ExecutionState: execState,
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, PassedOutputChannel, execState.Channel)
		assert.Equal(t, PayloadType, execState.Type)
	})

	t.Run("build failed -> emit failed", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"number":1,"url":"https://jenkins.example.com/job/my-job/1/","result":"FAILURE","building":false,"duration":60000}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.poll(core.ActionContext{
			Configuration: map[string]any{
				"job": "my-job",
			},
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata: &contexts.MetadataContext{
				Metadata: TriggerBuildExecutionMetadata{
					Job:         &JobInfo{Name: "my-job", URL: "https://jenkins.example.com/job/my-job/"},
					QueueItemID: 42,
					Build:       &BuildInfo{Number: 1, Building: true},
				},
			},
			ExecutionState: execState,
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, FailedOutputChannel, execState.Channel)
	})

	t.Run("build unstable -> emit failed", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"number":1,"url":"https://jenkins.example.com/job/my-job/1/","result":"UNSTABLE","building":false,"duration":90000}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.poll(core.ActionContext{
			Configuration: map[string]any{
				"job": "my-job",
			},
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata: &contexts.MetadataContext{
				Metadata: TriggerBuildExecutionMetadata{
					Job:         &JobInfo{Name: "my-job", URL: "https://jenkins.example.com/job/my-job/"},
					QueueItemID: 42,
					Build:       &BuildInfo{Number: 1, Building: true},
				},
			},
			ExecutionState: execState,
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, FailedOutputChannel, execState.Channel)
	})
}

func Test__TriggerBuild__HandleWebhook(t *testing.T) {
	component := TriggerBuild{}

	buildWebhookBody := func(name, phase, status string, number int64) []byte {
		payload := map[string]any{
			"name": name,
			"url":  "job/" + name + "/",
			"build": map[string]any{
				"full_url": "http://jenkins.example.com/job/" + name + "/1/",
				"number":   number,
				"phase":    phase,
				"status":   status,
				"url":      "job/" + name + "/1/",
			},
		}
		body, _ := json.Marshal(payload)
		return body
	}

	t.Run("invalid json -> bad request", func(t *testing.T) {
		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: []byte("not json"),
		})

		assert.Equal(t, http.StatusBadRequest, status)
		require.Error(t, err)
	})

	t.Run("no build field -> ok ignored", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"name": "my-job"})
		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: body,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)
	})

	t.Run("non-terminal phase -> ok ignored", func(t *testing.T) {
		body := buildWebhookBody("my-job", "STARTED", "", 1)
		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: body,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)
	})

	t.Run("no matching execution -> ok ignored", func(t *testing.T) {
		body := buildWebhookBody("my-job", "COMPLETED", "SUCCESS", 1)
		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return nil, nil
			},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)
	})

	t.Run("already finished execution -> ok ignored", func(t *testing.T) {
		body := buildWebhookBody("my-job", "COMPLETED", "SUCCESS", 1)
		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					ExecutionState: &contexts.ExecutionStateContext{Finished: true},
				}, nil
			},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)
	})

	t.Run("build succeeded -> emit passed", func(t *testing.T) {
		body := buildWebhookBody("my-job", "COMPLETED", "SUCCESS", 5)
		metadataCtx := &contexts.MetadataContext{
			Metadata: TriggerBuildExecutionMetadata{
				Job:         &JobInfo{Name: "my-job", URL: "https://jenkins.example.com/job/my-job/"},
				QueueItemID: 42,
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				assert.Equal(t, buildJobKey, key)
				assert.Equal(t, "my-job", value)
				return &core.ExecutionContext{
					Metadata:       metadataCtx,
					ExecutionState: execState,
				}, nil
			},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, PassedOutputChannel, execState.Channel)
		assert.Equal(t, PayloadType, execState.Type)

		updatedMetadata, ok := metadataCtx.Metadata.(TriggerBuildExecutionMetadata)
		require.True(t, ok)
		require.NotNil(t, updatedMetadata.Build)
		assert.Equal(t, int64(5), updatedMetadata.Build.Number)
		assert.Equal(t, "SUCCESS", updatedMetadata.Build.Result)
		assert.False(t, updatedMetadata.Build.Building)
	})

	t.Run("build failed -> emit failed", func(t *testing.T) {
		body := buildWebhookBody("my-job", "COMPLETED", "FAILURE", 6)
		metadataCtx := &contexts.MetadataContext{
			Metadata: TriggerBuildExecutionMetadata{
				Job:         &JobInfo{Name: "my-job", URL: "https://jenkins.example.com/job/my-job/"},
				QueueItemID: 42,
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata:       metadataCtx,
					ExecutionState: execState,
				}, nil
			},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, FailedOutputChannel, execState.Channel)
	})

	t.Run("build unstable -> emit failed", func(t *testing.T) {
		body := buildWebhookBody("my-job", "FINALIZED", "UNSTABLE", 7)
		metadataCtx := &contexts.MetadataContext{
			Metadata: TriggerBuildExecutionMetadata{
				Job:         &JobInfo{Name: "my-job", URL: "https://jenkins.example.com/job/my-job/"},
				QueueItemID: 42,
			},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		status, err := component.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata:       metadataCtx,
					ExecutionState: execState,
				}, nil
			},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, FailedOutputChannel, execState.Channel)
	})
}
