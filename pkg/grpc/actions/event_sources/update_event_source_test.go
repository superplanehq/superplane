package eventsources

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	integrationPb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__UpdateEventSource(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
	})

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	eventSource, err := CreateEventSource(ctx, r.Encryptor, r.Registry, r.Organization.ID.String(), r.Canvas.ID.String(), &protos.EventSource{
		Metadata: &protos.EventSource_Metadata{
			Name: "test-update-event-source",
		},
		Spec: &protos.EventSource_Spec{
			Type: "github",
			Integration: &integrationPb.IntegrationRef{
				Name: r.Integration.Name,
			},
			Resource: &integrationPb.ResourceRef{
				Type: "project",
				Name: "demo-project",
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, eventSource)
	eventSourceID := eventSource.EventSource.Metadata.Id

	t.Run("event source does not exist -> error", func(t *testing.T) {
		_, err := UpdateEventSource(ctx, r.Encryptor, r.Registry, r.Organization.ID.String(), r.Canvas.ID.String(), uuid.NewString(), &protos.EventSource{})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "event source not found")
	})

	t.Run("unauthenticated user -> error", func(t *testing.T) {
		_, err := UpdateEventSource(context.Background(), r.Encryptor, r.Registry, r.Organization.ID.String(), r.Canvas.ID.String(), eventSourceID, &protos.EventSource{})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		assert.Contains(t, s.Message(), "user not authenticated")
	})

	t.Run("event source name already in use -> error", func(t *testing.T) {
		_, err := CreateEventSource(ctx, r.Encryptor, r.Registry, r.Organization.ID.String(), r.Canvas.ID.String(), &protos.EventSource{
			Metadata: &protos.EventSource_Metadata{
				Name: "existing-source",
			},
			Spec: &protos.EventSource_Spec{
				Type: models.EventSourceTypeManual,
			},
		})
		require.NoError(t, err)

		_, err = UpdateEventSource(ctx, r.Encryptor, r.Registry, r.Organization.ID.String(), r.Canvas.ID.String(), eventSourceID, &protos.EventSource{
			Metadata: &protos.EventSource_Metadata{
				Name: "existing-source",
			},
			Spec: &protos.EventSource_Spec{
				Type: "github",
				Integration: &integrationPb.IntegrationRef{
					Name: r.Integration.Name,
				},
				Resource: &integrationPb.ResourceRef{
					Type: "project",
					Name: "demo-project",
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "event source name already in use")
	})

	t.Run("event source is updated", func(t *testing.T) {
		res, err := UpdateEventSource(ctx, r.Encryptor, r.Registry, r.Organization.ID.String(), r.Canvas.ID.String(), eventSourceID, &protos.EventSource{
			Metadata: &protos.EventSource_Metadata{
				Name:        "new-event-source-name",
				Description: "new-event-source-description",
			},
			Spec: &protos.EventSource_Spec{
				Type: "github",
				Integration: &integrationPb.IntegrationRef{
					Name: r.Integration.Name,
				},
				Resource: &integrationPb.ResourceRef{
					Type: "project",
					Name: "demo-project",
				},
				Events: []*protos.EventSource_EventType{
					{
						Type: "pipeline_done",
					},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, eventSourceID, res.EventSource.Metadata.Id)
		assert.Equal(t, r.Canvas.ID.String(), res.EventSource.Metadata.CanvasId)
		assert.Equal(t, "new-event-source-name", res.EventSource.Metadata.Name)
		assert.Equal(t, "new-event-source-description", res.EventSource.Metadata.Description)
		assert.Equal(t, r.Integration.Name, res.EventSource.Spec.Integration.Name)
		assert.Equal(t, "demo-project", res.EventSource.Spec.Resource.Name)
		assert.Equal(t, "project", res.EventSource.Spec.Resource.Type)
		require.Len(t, res.EventSource.Spec.Events, 1)
		assert.Equal(t, "pipeline_done", res.EventSource.Spec.Events[0].Type)
		assert.Empty(t, res.Key)
	})

	t.Run("event source updated to remove integration", func(t *testing.T) {
		res, err := UpdateEventSource(ctx, r.Encryptor, r.Registry, r.Organization.ID.String(), r.Canvas.ID.String(), eventSourceID, &protos.EventSource{
			Metadata: &protos.EventSource_Metadata{
				Name:        "standalone-event-source",
				Description: "no integration",
			},
			Spec: &protos.EventSource_Spec{
				Type: models.EventSourceTypeManual,
			},
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, eventSourceID, res.EventSource.Metadata.Id)
		assert.Equal(t, "standalone-event-source", res.EventSource.Metadata.Name)
		assert.Equal(t, "no integration", res.EventSource.Metadata.Description)
		assert.Nil(t, res.EventSource.Spec.Integration)
		assert.Nil(t, res.EventSource.Spec.Resource)
		assert.Empty(t, res.EventSource.Spec.Events)
		assert.Empty(t, res.Key)
	})

	t.Run("connection source names are updated when event source name changes", func(t *testing.T) {
		executorType, executorSpec, resource := support.Executor(t, r)

		connectedStage, err := builders.NewStageBuilder(r.Registry).
			WithContext(ctx).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas.ID).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(resource).
			ForIntegration(r.Integration).
			WithConnections([]models.Connection{
				{
					SourceName: "standalone-event-source",
					SourceType: models.SourceTypeEventSource,
					SourceID:   uuid.MustParse(eventSource.EventSource.Metadata.Id),
				},
			}).
			Create()

		require.NoError(t, err)
		connectedStageUUID, err := uuid.Parse(connectedStage.ID.String())
		require.NoError(t, err)
		dbConnections, err := models.ListConnections(connectedStageUUID, models.ConnectionTargetTypeStage)
		require.NoError(t, err)
		require.Len(t, dbConnections, 1)
		assert.Equal(t, "standalone-event-source", dbConnections[0].SourceName)

		_, err = UpdateEventSource(ctx, r.Encryptor, r.Registry, r.Organization.ID.String(), r.Canvas.ID.String(), eventSourceID, &protos.EventSource{
			Metadata: &protos.EventSource_Metadata{
				Name: "updated-event-source-name",
			},
			Spec: &protos.EventSource_Spec{
				Type: "github",
				Integration: &integrationPb.IntegrationRef{
					Name: r.Integration.Name,
				},
				Resource: &integrationPb.ResourceRef{
					Type: "project",
					Name: "demo-project",
				},
			},
		})
		require.NoError(t, err)

		dbConnections, err = models.ListConnections(connectedStageUUID, models.ConnectionTargetTypeStage)
		require.NoError(t, err)
		require.Len(t, dbConnections, 1)
		assert.Equal(t, "updated-event-source-name", dbConnections[0].SourceName)
	})

	t.Run("event source updated to add schedule", func(t *testing.T) {
		res, err := UpdateEventSource(ctx, r.Encryptor, r.Registry, r.Organization.ID.String(), r.Canvas.ID.String(), eventSourceID, &protos.EventSource{
			Metadata: &protos.EventSource_Metadata{
				Name:        "scheduled-event-source",
				Description: "with schedule",
			},
			Spec: &protos.EventSource_Spec{
				Type: models.EventSourceTypeScheduled,
				Schedule: &protos.EventSource_Schedule{
					Type: protos.EventSource_Schedule_TYPE_DAILY,
					Daily: &protos.EventSource_DailySchedule{
						Time: "14:30",
					},
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, eventSourceID, res.EventSource.Metadata.Id)
		assert.Equal(t, "scheduled-event-source", res.EventSource.Metadata.Name)
		assert.Equal(t, "with schedule", res.EventSource.Metadata.Description)
		assert.Nil(t, res.EventSource.Spec.Integration)
		assert.Nil(t, res.EventSource.Spec.Resource)

		require.NotNil(t, res.EventSource.Spec.Schedule)
		assert.Equal(t, protos.EventSource_Schedule_TYPE_DAILY, res.EventSource.Spec.Schedule.Type)
		require.NotNil(t, res.EventSource.Spec.Schedule.Daily)
		assert.Equal(t, "14:30", res.EventSource.Spec.Schedule.Daily.Time)
		assert.Empty(t, res.Key)
	})

	t.Run("event source updated to remove schedule", func(t *testing.T) {
		res, err := UpdateEventSource(ctx, r.Encryptor, r.Registry, r.Organization.ID.String(), r.Canvas.ID.String(), eventSourceID, &protos.EventSource{
			Metadata: &protos.EventSource_Metadata{
				Name:        "no-schedule-event-source",
				Description: "no schedule",
			},
			Spec: &protos.EventSource_Spec{
				Type: models.EventSourceTypeManual,
			},
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, eventSourceID, res.EventSource.Metadata.Id)
		assert.Equal(t, "no-schedule-event-source", res.EventSource.Metadata.Name)
		assert.Equal(t, "no schedule", res.EventSource.Metadata.Description)
		assert.Nil(t, res.EventSource.Spec.Integration)
		assert.Nil(t, res.EventSource.Spec.Resource)
		assert.Nil(t, res.EventSource.Spec.Schedule)
		assert.Empty(t, res.Key)
	})
}
