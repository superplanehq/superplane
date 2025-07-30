package stages

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test__UpdateStage(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
	})

	// Create a stage first that we'll update in tests
	executor := support.ProtoExecutor(t, r)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	stage, err := CreateStage(ctx, r.Encryptor, r.Registry, &protos.CreateStageRequest{
		CanvasIdOrName: r.Canvas.ID.String(),
		Stage: &protos.Stage{
			Metadata: &protos.Stage_Metadata{
				Name: "test-update-stage",
			},
			Spec: &protos.Stage_Spec{
				Executor: executor,
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
						Filters: []*protos.Filter{
							{
								Type: protos.FilterType_FILTER_TYPE_DATA,
								Data: &protos.DataFilter{
									Expression: "test == 1",
								},
							},
						},
					},
				},
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, stage)
	stageID := stage.Stage.Metadata.Id

	t.Run("invalid stage ID -> error", func(t *testing.T) {
		_, err := UpdateStage(ctx, r.Encryptor, r.Registry, &protos.UpdateStageRequest{
			IdOrName:       "invalid-uuid",
			CanvasIdOrName: r.Canvas.ID.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "canvas not found")
	})

	t.Run("stage does not exist -> error", func(t *testing.T) {
		_, err := UpdateStage(ctx, r.Encryptor, r.Registry, &protos.UpdateStageRequest{
			IdOrName:       uuid.NewString(),
			CanvasIdOrName: r.Canvas.ID.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "stage not found")
	})

	t.Run("unauthenticated user -> error", func(t *testing.T) {
		_, err := UpdateStage(context.Background(), r.Encryptor, r.Registry, &protos.UpdateStageRequest{
			IdOrName:       stageID,
			CanvasIdOrName: r.Canvas.ID.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		assert.Contains(t, s.Message(), "user not authenticated")
	})

	t.Run("connection for source that does not exist -> error", func(t *testing.T) {
		_, err := UpdateStage(ctx, r.Encryptor, r.Registry, &protos.UpdateStageRequest{
			IdOrName:       stageID,
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &protos.Stage{
				Spec: &protos.Stage_Spec{
					Executor: executor,
					Connections: []*protos.Connection{
						{
							Name: "source-does-not-exist",
							Type: protos.Connection_TYPE_EVENT_SOURCE,
						},
					},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid connection: event source source-does-not-exist not found")
	})

	t.Run("invalid filter -> error", func(t *testing.T) {
		_, err := UpdateStage(ctx, r.Encryptor, r.Registry, &protos.UpdateStageRequest{
			IdOrName:       stageID,
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &protos.Stage{
				Spec: &protos.Stage_Spec{
					Executor: executor,
					Connections: []*protos.Connection{
						{
							Name: r.Source.Name,
							Type: protos.Connection_TYPE_EVENT_SOURCE,
							Filters: []*protos.Filter{
								{
									Type: protos.FilterType_FILTER_TYPE_DATA,
									Data: &protos.DataFilter{
										Expression: "",
									},
								},
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
		_, err := UpdateStage(ctx, r.Encryptor, r.Registry, &protos.UpdateStageRequest{
			IdOrName:       stageID,
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &protos.Stage{
				Spec: &protos.Stage_Spec{
					Executor: executor,
					Connections: []*protos.Connection{
						{
							Name: r.Source.Name,
							Type: protos.Connection_TYPE_EVENT_SOURCE,
						},
					},
					Conditions: []*protos.Condition{
						{Type: protos.Condition_CONDITION_TYPE_APPROVAL, Approval: &protos.ConditionApproval{}},
					},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid condition: invalid approval condition")
	})

	t.Run("stage is updated", func(t *testing.T) {
		newSpec, err := structpb.NewStruct(map[string]any{
			"branch":       "other",
			"pipelineFile": ".semaphore/other.yml",
			"parameters":   map[string]any{},
		})

		require.NoError(t, err)

		res, err := UpdateStage(ctx, r.Encryptor, r.Registry, &protos.UpdateStageRequest{
			IdOrName:       stageID,
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &protos.Stage{
				Spec: &protos.Stage_Spec{
					Executor: &protos.Executor{
						Type:        executor.Type,
						Integration: executor.Integration,
						Resource:    executor.Resource,
						Spec:        newSpec,
					},
					Conditions: []*protos.Condition{},
					Connections: []*protos.Connection{
						{
							Name:           r.Source.Name,
							Type:           protos.Connection_TYPE_EVENT_SOURCE,
							FilterOperator: protos.FilterOperator_FILTER_OPERATOR_OR,
							Filters: []*protos.Filter{
								{
									Type: protos.FilterType_FILTER_TYPE_DATA,
									Data: &protos.DataFilter{
										Expression: "test == 42",
									},
								},
								{
									Type: protos.FilterType_FILTER_TYPE_DATA,
									Data: &protos.DataFilter{
										Expression: "status == 'active'",
									},
								},
							},
						},
					},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, stageID, res.Stage.Metadata.Id)
		assert.Equal(t, r.Canvas.ID.String(), res.Stage.Metadata.CanvasId)
		assert.Equal(t, "test-update-stage", res.Stage.Metadata.Name)

		// Connections are updated
		require.Len(t, res.Stage.Spec.Connections, 1)
		assert.Equal(t, r.Source.Name, res.Stage.Spec.Connections[0].Name)
		assert.Equal(t, protos.Connection_TYPE_EVENT_SOURCE, res.Stage.Spec.Connections[0].Type)
		assert.Equal(t, protos.FilterOperator_FILTER_OPERATOR_OR, res.Stage.Spec.Connections[0].FilterOperator)
		require.Len(t, res.Stage.Spec.Connections[0].Filters, 2)
		assert.Equal(t, "test == 42", res.Stage.Spec.Connections[0].Filters[0].Data.Expression)
		assert.Equal(t, "status == 'active'", res.Stage.Spec.Connections[0].Filters[1].Data.Expression)

		// Executor spec is updated
		assert.Equal(t, models.IntegrationTypeSemaphore, res.Stage.Spec.Executor.Type)
		assert.Equal(t, "other", res.Stage.Spec.Executor.Spec.GetFields()["branch"].GetStringValue())
		assert.Equal(t, ".semaphore/other.yml", res.Stage.Spec.Executor.Spec.GetFields()["pipelineFile"].GetStringValue())
		assert.Equal(t, map[string]any{}, res.Stage.Spec.Executor.Spec.GetFields()["parameters"].GetStructValue().AsMap())

		// Conditions are updated
		require.Empty(t, res.Stage.Spec.Conditions)
	})
}
