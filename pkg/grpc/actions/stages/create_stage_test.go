package stages

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/executors"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const StageCreatedRoutingKey = "stage-created"

func Test__CreateStage(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Source: true})
	specValidator := executors.SpecValidator{}

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: uuid.New().String(),
			RequesterId:    r.User.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: support.ProtoExecutor(),
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("missing requester ID -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: support.ProtoExecutor(),
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("connection for source that does not exist -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.Name,
			RequesterId:    r.User.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: support.ProtoExecutor(),
					Connections: []*pb.Connection{
						{
							Name: "source-does-not-exist",
							Type: pb.Connection_TYPE_EVENT_SOURCE,
						},
					},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid connection: event source source-does-not-exist not found", s.Message())
	})

	t.Run("invalid approval condition -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    r.User.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: support.ProtoExecutor(),
					Connections: []*pb.Connection{
						{
							Name: r.Source.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
						},
					},
					Conditions: []*pb.Condition{
						{Type: pb.Condition_CONDITION_TYPE_APPROVAL, Approval: &pb.ConditionApproval{}},
					},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid condition: invalid approval condition: count must be greater than 0", s.Message())
	})

	t.Run("time window condition with no start -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    r.User.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: support.ProtoExecutor(),
					Connections: []*pb.Connection{
						{
							Name: r.Source.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
						},
					},
					Conditions: []*pb.Condition{
						{
							Type:       pb.Condition_CONDITION_TYPE_TIME_WINDOW,
							TimeWindow: &pb.ConditionTimeWindow{},
						},
					},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid condition: invalid time window condition: invalid start", s.Message())
	})

	t.Run("time window condition with no end -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    r.User.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: support.ProtoExecutor(),
					Connections: []*pb.Connection{
						{
							Name: r.Source.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
						},
					},
					Conditions: []*pb.Condition{
						{
							Type: pb.Condition_CONDITION_TYPE_TIME_WINDOW,
							TimeWindow: &pb.ConditionTimeWindow{
								Start: "08:00",
							},
						},
					},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid condition: invalid time window condition: invalid end", s.Message())
	})

	t.Run("time window condition with invalid start -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    r.User.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: support.ProtoExecutor(),
					Connections: []*pb.Connection{
						{
							Name: r.Source.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
						},
					},
					Conditions: []*pb.Condition{
						{
							Type: pb.Condition_CONDITION_TYPE_TIME_WINDOW,
							TimeWindow: &pb.ConditionTimeWindow{
								Start: "52:00",
							},
						},
					},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid condition: invalid time window condition: invalid start", s.Message())
	})

	t.Run("time window condition with no week days list -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    r.User.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: support.ProtoExecutor(),
					Connections: []*pb.Connection{
						{
							Name: r.Source.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
						},
					},
					Conditions: []*pb.Condition{
						{
							Type: pb.Condition_CONDITION_TYPE_TIME_WINDOW,
							TimeWindow: &pb.ConditionTimeWindow{
								Start: "08:00",
								End:   "17:00",
							},
						},
					},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid condition: invalid time window condition: missing week day list", s.Message())
	})

	t.Run("time window condition with invalid day -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    r.User.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: support.ProtoExecutor(),
					Connections: []*pb.Connection{
						{
							Name: r.Source.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
						},
					},
					Conditions: []*pb.Condition{
						{
							Type: pb.Condition_CONDITION_TYPE_TIME_WINDOW,
							TimeWindow: &pb.ConditionTimeWindow{
								Start:    "08:00",
								End:      "17:00",
								WeekDays: []string{"Monday", "DoesNotExist"},
							},
						},
					},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid condition: invalid time window condition: invalid day DoesNotExist", s.Message())
	})

	t.Run("stage is created", func(t *testing.T) {
		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, StageCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		executor := support.ProtoExecutor()
		res, err := CreateStage(context.Background(), specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    r.User.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: executor,
					Conditions: []*pb.Condition{
						{
							Type:     pb.Condition_CONDITION_TYPE_APPROVAL,
							Approval: &pb.ConditionApproval{Count: 1},
						},
						{
							Type: pb.Condition_CONDITION_TYPE_TIME_WINDOW,
							TimeWindow: &pb.ConditionTimeWindow{
								Start:    "08:00",
								End:      "17:00",
								WeekDays: []string{"Monday", "Tuesday"},
							},
						},
					},
					Connections: []*pb.Connection{
						{
							Name: r.Source.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
							Filters: []*pb.Connection_Filter{
								{
									Type: pb.Connection_FILTER_TYPE_DATA,
									Data: &pb.Connection_DataFilter{
										Expression: "test == 12",
									},
								},
								{
									Type: pb.Connection_FILTER_TYPE_HEADER,
									Header: &pb.Connection_HeaderFilter{
										Expression: "test == 12",
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
		require.NotNil(t, res.Stage.Metadata)
		assert.NotNil(t, res.Stage.Metadata.Id)
		assert.NotNil(t, res.Stage.Metadata.CreatedAt)
		assert.Equal(t, r.Canvas.ID.String(), res.Stage.Metadata.CanvasId)
		assert.Equal(t, "test", res.Stage.Metadata.Name)
		// Assert executor is correct
		require.NotNil(t, res.Stage.Spec)
		assert.Equal(t, executor.Type, res.Stage.Spec.Executor.Type)
		assert.Equal(t, executor.Semaphore.Branch, res.Stage.Spec.Executor.Semaphore.Branch)
		assert.Equal(t, executor.Semaphore.PipelineFile, res.Stage.Spec.Executor.Semaphore.PipelineFile)
		assert.Equal(t, executor.Semaphore.OrganizationUrl, res.Stage.Spec.Executor.Semaphore.OrganizationUrl)
		assert.Equal(t, executor.Semaphore.Parameters, res.Stage.Spec.Executor.Semaphore.Parameters)

		// Check that we have a connection to the source
		require.Len(t, res.Stage.Spec.Connections, 1)
		assert.Len(t, res.Stage.Spec.Connections[0].Filters, 2)
		assert.Equal(t, pb.Connection_FILTER_OPERATOR_AND, res.Stage.Spec.Connections[0].FilterOperator)

		// Assert metadata and conditions are correct
		require.NotNil(t, res.Stage.Metadata)
		require.NotNil(t, res.Stage.Spec)
		require.Len(t, res.Stage.Spec.Conditions, 2)
		assert.Equal(t, pb.Condition_CONDITION_TYPE_APPROVAL, res.Stage.Spec.Conditions[0].Type)
		assert.Equal(t, uint32(1), res.Stage.Spec.Conditions[0].Approval.Count)
		assert.Equal(t, pb.Condition_CONDITION_TYPE_TIME_WINDOW, res.Stage.Spec.Conditions[1].Type)
		assert.Equal(t, "08:00", res.Stage.Spec.Conditions[1].TimeWindow.Start)
		assert.Equal(t, "17:00", res.Stage.Spec.Conditions[1].TimeWindow.End)
		assert.Equal(t, []string{"Monday", "Tuesday"}, res.Stage.Spec.Conditions[1].TimeWindow.WeekDays)
		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("stage name already used -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    r.User.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: support.ProtoExecutor(),
					Connections: []*pb.Connection{
						{
							Name: r.Source.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
						},
					},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())
	})
}
