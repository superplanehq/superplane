package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
)

const ExecutionFinishedRoutingKey = "execution-finished"

func Test__ExecutionResourcePoller(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
	})

	defer r.Close()

	connections := []models.Connection{
		{
			SourceID:   r.Source.ID,
			SourceType: models.SourceTypeEventSource,
		},
	}

	executorType, executorSpec, resource := support.Executor(r)
	stage, err := builders.NewStageBuilder().
		WithEncryptor(r.Encryptor).
		InCanvas(r.Canvas).
		WithName("stage-1").
		WithRequester(r.User).
		WithConnections(connections).
		WithExecutorType(executorType).
		WithExecutorSpec(executorSpec).
		ForResource(resource).
		ForIntegration(r.Integration).
		Create()

	require.NoError(t, err)

	amqpURL := "amqp://guest:guest@rabbitmq:5672"
	w := NewExecutionResourcePoller(r.Encryptor)

	t.Run("failed pipeline -> execution resource fails", func(t *testing.T) {
		require.NoError(t, database.Conn().Exec(`truncate table events`).Error)

		//
		// Create execution
		//
		workflowID := uuid.New().String()
		execution := support.CreateExecutionWithData(t, r.Source, stage,
			[]byte(`{"ref":"v1"}`),
			[]byte(`{"ref":"v1"}`),
			map[string]any{},
		)

		resource, err := models.FindResource(r.Integration.ID, resource.Type(), resource.Name())
		require.NoError(t, err)
		_, err = execution.AddResource(workflowID, resource.ID)
		require.NoError(t, err)

		testconsumer := testconsumer.New(amqpURL, ExecutionFinishedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Mock failed result and tick worker
		//
		pipelineID := uuid.New().String()
		r.SemaphoreAPIMock.AddPipeline(pipelineID, workflowID, integrations.SemaphorePipelineResultFailed)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Verify resource eventually goes to the finished state, with result = failed.
		//
		require.Eventually(t, func() bool {
			e, err := models.FindExecutionResource(workflowID, resource.ID)
			if err != nil {
				return false
			}

			return e.State == models.ExecutionFinished && e.Result == models.ResultFailed
		}, 5*time.Second, 200*time.Millisecond)
	})

	t.Run("passed pipeline -> execution resource passes", func(t *testing.T) {
		require.NoError(t, database.Conn().Exec(`truncate table events`).Error)

		//
		// Create execution
		//
		workflowID := uuid.New().String()
		execution := support.CreateExecutionWithData(t, r.Source, stage,
			[]byte(`{"ref":"v1"}`),
			[]byte(`{"ref":"v1"}`),
			map[string]any{},
		)

		resource, err := models.FindResource(r.Integration.ID, resource.Type(), resource.Name())
		require.NoError(t, err)
		_, err = execution.AddResource(workflowID, resource.ID)
		require.NoError(t, err)

		testconsumer := testconsumer.New(amqpURL, ExecutionFinishedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Mock passed result and tick worker
		//
		pipelineID := uuid.New().String()
		r.SemaphoreAPIMock.AddPipeline(pipelineID, workflowID, integrations.SemaphorePipelineResultPassed)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Verify execution eventually goes to the finished state, with result = failed.
		//
		require.Eventually(t, func() bool {
			e, err := models.FindExecutionResource(workflowID, resource.ID)
			if err != nil {
				return false
			}

			return e.State == models.ExecutionFinished && e.Result == models.ResultPassed
		}, 5*time.Second, 200*time.Millisecond)
	})
}
