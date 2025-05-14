package actions

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	protos "github.com/superplanehq/superplane/pkg/protos/delivery"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const StageCreatedRoutingKey = "stage-created"

func Test__CreateStage(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Source: true})

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       uuid.New().String(),
			Name:           "test",
			RequesterId:    r.User.String(),
			RunTemplate:    support.ProtoRunTemplate(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("missing requester ID -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RunTemplate:    support.ProtoRunTemplate(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid UUID")
	})

	t.Run("connection for source that does not exist -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RunTemplate:    support.ProtoRunTemplate(),
			RequesterId:    r.User.String(),
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
		assert.Equal(t, "invalid connection: event source source-does-not-exist not found", s.Message())
	})

	t.Run("invalid approval condition -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RunTemplate:    support.ProtoRunTemplate(),
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
		assert.Equal(t, "invalid condition: invalid approval condition: count must be greater than 0", s.Message())
	})

	t.Run("time window condition with no start -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RunTemplate:    support.ProtoRunTemplate(),
			RequesterId:    r.User.String(),
			Connections: []*protos.Connection{
				{
					Name: r.Source.Name,
					Type: protos.Connection_TYPE_EVENT_SOURCE,
				},
			},
			Conditions: []*protos.Condition{
				{
					Type:       protos.Condition_CONDITION_TYPE_TIME_WINDOW,
					TimeWindow: &protos.ConditionTimeWindow{},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid condition: invalid time window condition: invalid start", s.Message())
	})

	t.Run("time window condition with no end -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RunTemplate:    support.ProtoRunTemplate(),
			RequesterId:    r.User.String(),
			Connections: []*protos.Connection{
				{
					Name: r.Source.Name,
					Type: protos.Connection_TYPE_EVENT_SOURCE,
				},
			},
			Conditions: []*protos.Condition{
				{
					Type: protos.Condition_CONDITION_TYPE_TIME_WINDOW,
					TimeWindow: &protos.ConditionTimeWindow{
						Start: "08:00",
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
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RunTemplate:    support.ProtoRunTemplate(),
			RequesterId:    r.User.String(),
			Connections: []*protos.Connection{
				{
					Name: r.Source.Name,
					Type: protos.Connection_TYPE_EVENT_SOURCE,
				},
			},
			Conditions: []*protos.Condition{
				{
					Type: protos.Condition_CONDITION_TYPE_TIME_WINDOW,
					TimeWindow: &protos.ConditionTimeWindow{
						Start: "52:00",
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
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RunTemplate:    support.ProtoRunTemplate(),
			RequesterId:    r.User.String(),
			Connections: []*protos.Connection{
				{
					Name: r.Source.Name,
					Type: protos.Connection_TYPE_EVENT_SOURCE,
				},
			},
			Conditions: []*protos.Condition{
				{
					Type: protos.Condition_CONDITION_TYPE_TIME_WINDOW,
					TimeWindow: &protos.ConditionTimeWindow{
						Start: "08:00",
						End:   "17:00",
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
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RunTemplate:    support.ProtoRunTemplate(),
			RequesterId:    r.User.String(),
			Connections: []*protos.Connection{
				{
					Name: r.Source.Name,
					Type: protos.Connection_TYPE_EVENT_SOURCE,
				},
			},
			Conditions: []*protos.Condition{
				{
					Type: protos.Condition_CONDITION_TYPE_TIME_WINDOW,
					TimeWindow: &protos.ConditionTimeWindow{
						Start:    "08:00",
						End:      "17:00",
						WeekDays: []string{"Monday", "DoesNotExist"},
					},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid condition: invalid time window condition: invalid day DoesNotExist", s.Message())
	})

	t.Run("no tag usage definition -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RunTemplate:    support.ProtoRunTemplate(),
			RequesterId:    r.User.String(),
			Conditions:     []*protos.Condition{},
			Connections: []*protos.Connection{
				{
					Name: r.Source.Name,
					Type: protos.Connection_TYPE_EVENT_SOURCE,
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "missing tag usage definition", s.Message())
	})

	t.Run("tag usage definition with invalid from -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RunTemplate:    support.ProtoRunTemplate(),
			RequesterId:    r.User.String(),
			Conditions:     []*protos.Condition{},
			Use: &protos.TagUsageDefinition{
				From: []string{"does-not-exist"},
				Tags: []*protos.TagDefinition{{Name: "version", ValueFrom: "ref"}},
			},
			Connections: []*protos.Connection{
				{
					Name: r.Source.Name,
					Type: protos.Connection_TYPE_EVENT_SOURCE,
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid tag: invalid from does-not-exist", s.Message())
	})

	t.Run("no tags in tag usage definition -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RunTemplate:    support.ProtoRunTemplate(),
			RequesterId:    r.User.String(),
			Conditions:     []*protos.Condition{},
			Use: &protos.TagUsageDefinition{
				From: []string{r.Source.Name},
				Tags: []*protos.TagDefinition{},
			},
			Connections: []*protos.Connection{
				{
					Name: r.Source.Name,
					Type: protos.Connection_TYPE_EVENT_SOURCE,
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "tags must not be empty", s.Message())
	})

	t.Run("tag with empty name -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RunTemplate:    support.ProtoRunTemplate(),
			RequesterId:    r.User.String(),
			Conditions:     []*protos.Condition{},
			Use: &protos.TagUsageDefinition{
				From: []string{r.Source.Name},
				Tags: []*protos.TagDefinition{{Name: ""}},
			},
			Connections: []*protos.Connection{
				{
					Name: r.Source.Name,
					Type: protos.Connection_TYPE_EVENT_SOURCE,
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid tag: no name or value defined", s.Message())
	})

	t.Run("stage is created", func(t *testing.T) {
		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, StageCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		runTemplate := support.ProtoRunTemplate()
		res, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RunTemplate:    runTemplate,
			RequesterId:    r.User.String(),
			Use: &protos.TagUsageDefinition{
				From: []string{r.Source.Name},
				Tags: []*protos.TagDefinition{
					{
						Name:      "version",
						ValueFrom: "ref",
					},
				},
			},
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
								Expression: "test == 12",
							},
						},
						{
							Type: protos.Connection_FILTER_TYPE_HEADER,
							Header: &protos.Connection_HeaderFilter{
								Expression: "test == 12",
							},
						},
					},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotNil(t, res.Stage.Id)
		assert.NotNil(t, res.Stage.CreatedAt)
		assert.Equal(t, r.Org.String(), res.Stage.OrganizationId)
		assert.Equal(t, r.Canvas.ID.String(), res.Stage.CanvasId)
		assert.Equal(t, "test", res.Stage.Name)
		assert.Equal(t, runTemplate, res.Stage.RunTemplate)

		// Assert connections are correct
		require.Len(t, res.Stage.Connections, 1)
		assert.Len(t, res.Stage.Connections[0].Filters, 2)
		assert.Equal(t, protos.Connection_FILTER_OPERATOR_AND, res.Stage.Connections[0].FilterOperator)

		// Assert tag usage definition is correct
		require.NotNil(t, res.Stage.Use)
		assert.Equal(t, []string{r.Source.Name}, res.Stage.Use.From)
		assert.Equal(t, "version", res.Stage.Use.Tags[0].Name)
		assert.Equal(t, "ref", res.Stage.Use.Tags[0].ValueFrom)

		// Assert conditions are correct
		require.Len(t, res.Stage.Conditions, 2)
		assert.Equal(t, protos.Condition_CONDITION_TYPE_APPROVAL, res.Stage.Conditions[0].Type)
		assert.Equal(t, uint32(1), res.Stage.Conditions[0].Approval.Count)
		assert.Equal(t, protos.Condition_CONDITION_TYPE_TIME_WINDOW, res.Stage.Conditions[1].Type)
		assert.Equal(t, "08:00", res.Stage.Conditions[1].TimeWindow.Start)
		assert.Equal(t, "17:00", res.Stage.Conditions[1].TimeWindow.End)
		assert.Equal(t, []string{"Monday", "Tuesday"}, res.Stage.Conditions[1].TimeWindow.WeekDays)
		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("stage name already used -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), &protos.CreateStageRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
			Name:           "test",
			RequesterId:    r.User.String(),
			RunTemplate:    support.ProtoRunTemplate(),
			Use: &protos.TagUsageDefinition{
				From: []string{r.Source.Name},
				Tags: []*protos.TagDefinition{
					{
						Name:      "version",
						ValueFrom: "ref",
					},
				},
			},
			Connections: []*protos.Connection{
				{
					Name: r.Source.Name,
					Type: protos.Connection_TYPE_EVENT_SOURCE,
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())
	})
}
