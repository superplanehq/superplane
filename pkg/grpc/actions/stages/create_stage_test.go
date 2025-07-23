package stages

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	integrationpb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

const StageCreatedRoutingKey = "stage-created"

func Test__CreateStage(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
	})

	specValidator := executors.SpecValidator{
		Encryptor: r.Encryptor,
	}

	executor := support.ProtoExecutor(r)

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: uuid.New().String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: executor,
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("unauthenticated user -> error", func(t *testing.T) {
		_, err := CreateStage(context.Background(), r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: executor,
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		assert.Equal(t, "user not authenticated", s.Message())
	})

	t.Run("connection for source that does not exist -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.Name,
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: executor,
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

	t.Run("connection for internal event source -> error", func(t *testing.T) {
		internalSource, err := r.Canvas.CreateEventSource("internal", []byte(`key`), models.EventSourceScopeInternal, nil)
		require.NoError(t, err)

		ctx := authentication.SetUserIdInMetadata(context.Background(), uuid.NewString())
		_, err = CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.Name,
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: executor,
					Connections: []*pb.Connection{
						{
							Name: internalSource.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
						},
					},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid connection: event source internal not found", s.Message())
	})

	t.Run("invalid approval condition -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: executor,
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
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: executor,
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
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: executor,
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
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: executor,
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
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: executor,
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
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: "test",
				},
				Spec: &pb.Stage_Spec{
					Executor: executor,
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

	t.Run("stage with integration that does not exist -> error", func(t *testing.T) {
		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, StageCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		name := support.RandomName("test")
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{Name: name},
				Spec: &pb.Stage_Spec{
					Executor: &pb.ExecutorSpec{
						Type:        executor.Type,
						Integration: &integrationpb.IntegrationRef{Name: "does-not-exist"},
						Semaphore:   executor.Semaphore,
					},
					Connections: []*pb.Connection{
						{
							Name: r.Source.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
						},
					},
				},
			},
		})

		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "integration does-not-exist not found", s.Message())
	})

	t.Run("stage with integration", func(t *testing.T) {
		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, StageCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		name := support.RandomName("test")
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		res, err := CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: name,
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
							Filters: []*pb.Filter{
								{
									Type: pb.FilterType_FILTER_TYPE_DATA,
									Data: &pb.DataFilter{
										Expression: "test == 12",
									},
								},
								{
									Type: pb.FilterType_FILTER_TYPE_HEADER,
									Header: &pb.HeaderFilter{
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
		assert.Equal(t, name, res.Stage.Metadata.Name)

		// Assert executor is correct
		require.NotNil(t, res.Stage.Spec)
		assert.Equal(t, executor.Type, res.Stage.Spec.Executor.Type)
		assert.Equal(t, executor.Integration.Name, res.Stage.Spec.Executor.Integration.Name)
		assert.Equal(t, executor.Semaphore.Project, res.Stage.Spec.Executor.Semaphore.Project)
		assert.Equal(t, executor.Semaphore.Branch, res.Stage.Spec.Executor.Semaphore.Branch)
		assert.Equal(t, executor.Semaphore.PipelineFile, res.Stage.Spec.Executor.Semaphore.PipelineFile)
		assert.Equal(t, executor.Semaphore.Parameters, res.Stage.Spec.Executor.Semaphore.Parameters)

		// Check that we have a connection to the source
		require.Len(t, res.Stage.Spec.Connections, 1)
		assert.Len(t, res.Stage.Spec.Connections[0].Filters, 2)
		assert.Equal(t, pb.FilterOperator_FILTER_OPERATOR_AND, res.Stage.Spec.Connections[0].FilterOperator)

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

		// Assert internally scoped event source was created
		resource, err := models.FindResource(r.Integration.ID, integrations.ResourceTypeProject, executor.Semaphore.Project)
		require.NoError(t, err)
		require.NotNil(t, resource)
		eventSource, err := resource.FindEventSource()
		require.NoError(t, err)
		require.NotNil(t, eventSource)
		require.Equal(t, eventSource.Name, executor.Integration.Name+"-"+executor.Semaphore.Project)
		require.Equal(t, eventSource.Scope, models.EventSourceScopeInternal)
	})

	t.Run("stage with org-level integration", func(t *testing.T) {
		secret, err := support.CreateOrganizationSecret(t, r, map[string]string{"key": "test"})
		require.NoError(t, err)
		integration, err := models.CreateIntegration(&models.Integration{
			Name:       support.RandomName("integration"),
			CreatedBy:  r.User,
			Type:       models.IntegrationTypeSemaphore,
			DomainType: models.DomainTypeOrganization,
			DomainID:   r.Organization.ID,
			URL:        r.SemaphoreAPIMock.Server.URL,
			AuthType:   models.IntegrationAuthTypeToken,
			Auth: datatypes.NewJSONType(models.IntegrationAuth{
				Token: &models.IntegrationAuthToken{
					ValueFrom: models.ValueDefinitionFrom{
						Secret: &models.ValueDefinitionFromSecret{
							Name: secret.Name,
							Key:  "key",
						},
					},
				},
			}),
		})

		name := support.RandomName("test")
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		res, err := CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{Name: name},
				Spec: &pb.Stage_Spec{
					Executor: &pb.ExecutorSpec{
						Type: executor.Type,
						Integration: &integrationpb.IntegrationRef{
							Name:       integration.Name,
							DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
						},
						Semaphore: executor.Semaphore,
					},
					Conditions: []*pb.Condition{},
					Connections: []*pb.Connection{
						{
							Name: r.Source.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
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
		assert.Equal(t, name, res.Stage.Metadata.Name)

		// Assert executor is correct
		require.NotNil(t, res.Stage.Spec)
		assert.Equal(t, executor.Type, res.Stage.Spec.Executor.Type)
		assert.Equal(t, integration.Name, res.Stage.Spec.Executor.Integration.Name)
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, res.Stage.Spec.Executor.Integration.DomainType)
		assert.Equal(t, executor.Semaphore.Project, res.Stage.Spec.Executor.Semaphore.Project)
		assert.Equal(t, executor.Semaphore.Branch, res.Stage.Spec.Executor.Semaphore.Branch)
		assert.Equal(t, executor.Semaphore.PipelineFile, res.Stage.Spec.Executor.Semaphore.PipelineFile)
		assert.Equal(t, executor.Semaphore.Parameters, res.Stage.Spec.Executor.Semaphore.Parameters)

		// Check that we have a connection to the source
		require.Len(t, res.Stage.Spec.Connections, 1)

		// Assert internally scoped event source was created
		resource, err := models.FindResource(integration.ID, integrations.ResourceTypeProject, executor.Semaphore.Project)
		require.NoError(t, err)
		require.NotNil(t, resource)
		eventSource, err := resource.FindEventSource()
		require.NoError(t, err)
		require.NotNil(t, eventSource)
		require.Equal(t, eventSource.Name, integration.Name+"-"+executor.Semaphore.Project)
		require.Equal(t, eventSource.Scope, models.EventSourceScopeInternal)
	})

	t.Run("stage with same integration resource re-uses internally scoped event source", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

		//
		// Create first stage using the demo-project Semaphore project integration resource.
		//
		res, err := CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: support.RandomName("test"),
				},
				Spec: &pb.Stage_Spec{
					Executor:   executor,
					Conditions: []*pb.Condition{},
					Connections: []*pb.Connection{
						{
							Name: r.Source.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
						},
					},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, res.Stage)

		//
		// Create second stage using the demo-project Semaphore project integration resource.
		//
		res, err = CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: support.RandomName("test"),
				},
				Spec: &pb.Stage_Spec{
					Executor:   executor,
					Conditions: []*pb.Condition{},
					Connections: []*pb.Connection{
						{
							Name: r.Source.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
						},
					},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, res.Stage)

		// Assert the same integration resource record and
		// internally scoped event source are re-used by both stages.
		resources, err := r.Integration.ListResources(integrations.ResourceTypeProject)
		require.NoError(t, err)
		require.Len(t, resources, 1)
		sources, err := resources[0].ListEventSources()
		require.NoError(t, err)
		assert.Len(t, sources, 1)
		assert.Equal(t, sources[0].Name, r.Integration.Name+"-"+executor.Semaphore.Project)
		assert.Equal(t, sources[0].Scope, models.EventSourceScopeInternal)
	})

	t.Run("stage name already used -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

		//
		// First stage works
		//
		name := support.RandomName("test")
		res, err := CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: name,
				},
				Spec: &pb.Stage_Spec{
					Executor: executor,
					Connections: []*pb.Connection{
						{
							Name: r.Source.Name,
							Type: pb.Connection_TYPE_EVENT_SOURCE,
						},
					},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, res.Stage)

		//
		// Second stage with the same name fails
		//
		_, err = CreateStage(ctx, r.Encryptor, specValidator, &pb.CreateStageRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Stage: &pb.Stage{
				Metadata: &pb.Stage_Metadata{
					Name: name,
				},
				Spec: &pb.Stage_Spec{
					Executor: executor,
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
