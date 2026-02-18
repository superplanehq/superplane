package harness

import (
	"io"
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__RunPipeline__Setup(t *testing.T) {
	component := &RunPipeline{}

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: RunPipelineConfiguration{
				OrgIdentifier:      "default",
				ProjectIdentifier:  "my_project",
				PipelineIdentifier: "my_pipeline",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{},
		})

		require.NoError(t, err)
	})

	t.Run("missing organization", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: RunPipelineConfiguration{
				ProjectIdentifier:  "my_project",
				PipelineIdentifier: "my_pipeline",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{},
		})

		require.ErrorContains(t, err, "organization is required")
	})

	t.Run("missing project", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: RunPipelineConfiguration{
				OrgIdentifier:      "default",
				PipelineIdentifier: "my_pipeline",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("missing pipeline", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: RunPipelineConfiguration{
				OrgIdentifier:     "default",
				ProjectIdentifier: "my_project",
			},
			Metadata:    &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{},
		})

		require.ErrorContains(t, err, "pipeline is required")
	})

	t.Run("invalid configuration type", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
			Metadata:      &contexts.MetadataContext{},
			Integration:   &contexts.IntegrationContext{},
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})
}

func Test__RunPipeline__Execute(t *testing.T) {
	component := &RunPipeline{}

	t.Run("successful pipeline execution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"status": "SUCCESS",
						"data": {
							"planExecution": {
								"uuid": "exec-abc-123",
								"status": "Running"
							}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId": "account-123",
				"apiToken":  "token-xyz",
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		executionStateCtx := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}
		requestsCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: RunPipelineConfiguration{
				OrgIdentifier:      "default",
				ProjectIdentifier:  "my_project",
				PipelineIdentifier: "my_pipeline",
				Module:             "CI",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			ExecutionState: executionStateCtx,
			Requests:       requestsCtx,
			Logger:         log.NewEntry(log.New()),
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/pipeline/api/pipeline/execute/my_pipeline")
		assert.Contains(t, httpContext.Requests[0].URL.String(), "accountIdentifier=account-123")
		assert.Equal(t, "exec-abc-123", executionStateCtx.KVs["planExecutionId"])
		assert.Equal(t, "poll", requestsCtx.Action)
	})
}
