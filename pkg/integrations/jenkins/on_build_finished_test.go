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

func Test__OnBuildFinished__TriggerInfo(t *testing.T) {
	trigger := OnBuildFinished{}

	assert.Equal(t, "jenkins.onBuildFinished", trigger.Name())
	assert.Equal(t, "On Build Finished", trigger.Label())
	assert.Equal(t, "Listen to Jenkins build completion events", trigger.Description())
	assert.Equal(t, "jenkins", trigger.Icon())
	assert.Equal(t, "gray", trigger.Color())
	assert.NotEmpty(t, trigger.Documentation())
}

func Test__OnBuildFinished__Configuration(t *testing.T) {
	trigger := OnBuildFinished{}
	config := trigger.Configuration()

	require.Len(t, config, 1)
	assert.Equal(t, "job", config[0].Name)
	assert.True(t, config[0].Required)
}

func Test__OnBuildFinished__ExampleData(t *testing.T) {
	trigger := OnBuildFinished{}
	data := trigger.ExampleData()

	require.NotNil(t, data)
	assert.NotEmpty(t, data)
}

func Test__OnBuildFinished__Setup(t *testing.T) {
	trigger := OnBuildFinished{}

	t.Run("missing job -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		err := trigger.Setup(core.TriggerContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   appCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
			Logger:        logrus.NewEntry(logrus.New()),
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

		err := trigger.Setup(core.TriggerContext{
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"job": "nonexistent-job",
			},
			Logger: logrus.NewEntry(logrus.New()),
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
		err := trigger.Setup(core.TriggerContext{
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata:    metadataCtx,
			Webhook:     &contexts.WebhookContext{},
			Configuration: map[string]any{
				"job": "my-job",
			},
			Logger: logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)

		metadata, ok := metadataCtx.Metadata.(OnBuildFinishedMetadata)
		require.True(t, ok)
		assert.Equal(t, "my-job", metadata.Job.Name)
		assert.Equal(t, "https://jenkins.example.com/job/my-job/", metadata.Job.URL)
		require.Len(t, appCtx.WebhookRequests, 1)
	})

	t.Run("already set up for same job -> skip client call", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: OnBuildFinishedMetadata{
				Job: &JobInfo{Name: "my-job", URL: "https://jenkins.example.com/job/my-job/"},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: appCtx,
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"job": "my-job",
			},
			Logger: logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		require.Len(t, appCtx.WebhookRequests, 1)
	})
}

func Test__OnBuildFinished__HandleWebhook(t *testing.T) {
	trigger := OnBuildFinished{}

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
		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte("not json"),
			Configuration: map[string]any{"job": "my-job"},
		})

		assert.Equal(t, http.StatusBadRequest, status)
		require.Error(t, err)
	})

	t.Run("no build field -> ok ignored", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"name": "my-job"})
		eventCtx := &contexts.EventContext{}

		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Events:        eventCtx,
			Configuration: map[string]any{"job": "my-job"},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)
		assert.Equal(t, 0, eventCtx.Count())
	})

	t.Run("non-terminal phase -> ok ignored", func(t *testing.T) {
		body := buildWebhookBody("my-job", "STARTED", "", 1)
		eventCtx := &contexts.EventContext{}

		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Events:        eventCtx,
			Configuration: map[string]any{"job": "my-job"},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)
		assert.Equal(t, 0, eventCtx.Count())
	})

	t.Run("wrong job name -> ok ignored", func(t *testing.T) {
		body := buildWebhookBody("other-job", "COMPLETED", "SUCCESS", 1)
		eventCtx := &contexts.EventContext{}

		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Events:        eventCtx,
			Configuration: map[string]any{"job": "my-job"},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)
		assert.Equal(t, 0, eventCtx.Count())
	})

	t.Run("build completed -> emit event", func(t *testing.T) {
		body := buildWebhookBody("my-job", "COMPLETED", "SUCCESS", 42)
		eventCtx := &contexts.EventContext{}

		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Events:        eventCtx,
			Configuration: map[string]any{"job": "my-job"},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, PayloadType, eventCtx.Payloads[0].Type)

		data, ok := eventCtx.Payloads[0].Data.(map[string]any)
		require.True(t, ok)

		job, ok := data["job"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "my-job", job["name"])

		build, ok := data["build"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, int64(42), build["number"])
		assert.Equal(t, "SUCCESS", build["result"])
	})

	t.Run("finalized phase -> ok ignored", func(t *testing.T) {
		body := buildWebhookBody("my-job", "FINALIZED", "FAILURE", 43)
		eventCtx := &contexts.EventContext{}

		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Events:        eventCtx,
			Configuration: map[string]any{"job": "my-job"},
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)
		assert.Equal(t, 0, eventCtx.Count())
	})
}
