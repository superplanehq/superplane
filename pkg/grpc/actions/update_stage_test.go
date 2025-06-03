package actions

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/encryptor"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__UpdateStage(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Source: true})

	encryptor := &encryptor.NoOpEncryptor{}

	// Create a stage first that we'll update in tests
	executor := support.ProtoExecutor()
	stage, err := CreateStage(context.Background(), encryptor, &protos.CreateStageRequest{
		CanvasIdOrName: r.Canvas.ID.String(),
		Name:           "test-update-stage",
		Executor:       executor,
		RequesterId:    r.User.String(),
		Conditions: []*protos.Condition{
			{
				Type:     protos.Condition_CONDITION_TYPE_APPROVAL,
				Approval: &protos.ConditionApproval{Count: 1},
			},
			{
				Type: protos.Condition_CONDITION_TYPE_TIME_WINDOW,
				TimeWindow: &protos.ConditionTimeWindow{
					Start:    "08:00",
					End:      "17:00",
					WeekDays: []string{"Monday", "Tuesday"},
				},
			},
		},
		Connections: []*protos.Connection{
			{
				Name: r.Source.Name,
				Type: protos.Connection_TYPE_EVENT_SOURCE,
				Filters: []*protos.Connection_Filter{
					{
						Type: protos.Connection_FILTER_TYPE_DATA,
						Data: &protos.Connection_DataFilter{
							Expression: "test == 1",
						},
					},
				},
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, stage)
	stageID := stage.Stage.Id

	t.Run("invalid stage ID -> error", func(t *testing.T) {
		_, err := UpdateStage(context.Background(), encryptor, &protos.UpdateStageRequest{
			IdOrName:       "invalid-uuid",
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    r.User.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "canvas not found")
	})

	t.Run("stage does not exist -> error", func(t *testing.T) {
		_, err := UpdateStage(context.Background(), encryptor, &protos.UpdateStageRequest{
			IdOrName:       uuid.NewString(),
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    r.User.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "stage not found")
	})

	t.Run("missing requester ID -> error", func(t *testing.T) {
		_, err := UpdateStage(context.Background(), encryptor, &protos.UpdateStageRequest{
			IdOrName:       stageID,
			CanvasIdOrName: r.Canvas.ID.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "requester ID is invalid")
	})

	t.Run("connection for source that does not exist -> error", func(t *testing.T) {
		_, err := UpdateStage(context.Background(), encryptor, &protos.UpdateStageRequest{
			IdOrName:       stageID,
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    r.User.String(),
			Executor:       support.ProtoExecutor(),
			Connections: []*protos.Connection{
				{
					Name: "source-does-not-exist",
					Type: protos.Connection_TYPE_EVENT_SOURCE,
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid connection: event source source-does-not-exist not found")
	})

	t.Run("invalid filter -> error", func(t *testing.T) {
		_, err := UpdateStage(context.Background(), encryptor, &protos.UpdateStageRequest{
			IdOrName:       stageID,
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    r.User.String(),
			Executor:       support.ProtoExecutor(),
			Connections: []*protos.Connection{
				{
					Name: r.Source.Name,
					Type: protos.Connection_TYPE_EVENT_SOURCE,
					Filters: []*protos.Connection_Filter{
						{
							Type: protos.Connection_FILTER_TYPE_DATA,
							Data: &protos.Connection_DataFilter{
								Expression: "",
							},
						},
					},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid filter [0]: expression is empty")
	})

	t.Run("invalid approval condition -> error", func(t *testing.T) {
		_, err := UpdateStage(context.Background(), encryptor, &protos.UpdateStageRequest{
			IdOrName:       stageID,
			CanvasIdOrName: r.Canvas.ID.String(),
			Executor:       support.ProtoExecutor(),
			RequesterId:    r.User.String(),
			Connections: []*protos.Connection{
				{
					Name: r.Source.Name,
					Type: protos.Connection_TYPE_EVENT_SOURCE,
				},
			},
			Conditions: []*protos.Condition{
				{Type: protos.Condition_CONDITION_TYPE_APPROVAL, Approval: &protos.ConditionApproval{}},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid condition: invalid approval condition")
	})

	t.Run("stage is updated", func(t *testing.T) {
		res, err := UpdateStage(context.Background(), encryptor, &protos.UpdateStageRequest{
			IdOrName:       stageID,
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    r.User.String(),
			Executor: &protos.ExecutorSpec{
				Type: protos.ExecutorSpec_TYPE_SEMAPHORE,
				Semaphore: &protos.ExecutorSpec_Semaphore{
					OrganizationUrl: "http://localhost:8000",
					ApiToken:        "test",
					ProjectId:       "test-2",
					TaskId:          "task-2",
					Branch:          "other",
					PipelineFile:    ".semaphore/other.yml",
					Parameters:      map[string]string{},
				},
			},
			Conditions: []*protos.Condition{},
			Connections: []*protos.Connection{
				{
					Name:           r.Source.Name,
					Type:           protos.Connection_TYPE_EVENT_SOURCE,
					FilterOperator: protos.Connection_FILTER_OPERATOR_OR,
					Filters: []*protos.Connection_Filter{
						{
							Type: protos.Connection_FILTER_TYPE_DATA,
							Data: &protos.Connection_DataFilter{
								Expression: "test == 42",
							},
						},
						{
							Type: protos.Connection_FILTER_TYPE_DATA,
							Data: &protos.Connection_DataFilter{
								Expression: "status == 'active'",
							},
						},
					},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, stageID, res.Stage.Id)
		assert.Equal(t, r.Canvas.ID.String(), res.Stage.CanvasId)
		assert.Equal(t, "test-update-stage", res.Stage.Name)

		// Connections are updated
		require.Len(t, res.Stage.Connections, 1)
		assert.Equal(t, r.Source.Name, res.Stage.Connections[0].Name)
		assert.Equal(t, protos.Connection_TYPE_EVENT_SOURCE, res.Stage.Connections[0].Type)
		assert.Equal(t, protos.Connection_FILTER_OPERATOR_OR, res.Stage.Connections[0].FilterOperator)
		require.Len(t, res.Stage.Connections[0].Filters, 2)
		assert.Equal(t, "test == 42", res.Stage.Connections[0].Filters[0].Data.Expression)
		assert.Equal(t, "status == 'active'", res.Stage.Connections[0].Filters[1].Data.Expression)

		// Executor spec is updated
		assert.Equal(t, protos.ExecutorSpec_TYPE_SEMAPHORE, res.Stage.Executor.Type)
		assert.Equal(t, "task-2", res.Stage.Executor.Semaphore.TaskId)
		assert.Equal(t, "test-2", res.Stage.Executor.Semaphore.ProjectId)
		assert.Equal(t, "other", res.Stage.Executor.Semaphore.Branch)
		assert.Equal(t, ".semaphore/other.yml", res.Stage.Executor.Semaphore.PipelineFile)
		assert.Equal(t, "http://localhost:8000", res.Stage.Executor.Semaphore.OrganizationUrl)

		// Conditions are updated
		require.Empty(t, res.Stage.Conditions)
	})
}
