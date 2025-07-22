package eventsources

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"

	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	integrationPb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const EventSourceCreatedRoutingKey = "event-source-created"

func Test__CreateEventSource(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Integration: true})
	encryptor := &crypto.NoOpEncryptor{}

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		eventSource := &protos.EventSource{
			Metadata: &protos.EventSource_Metadata{
				Name: "test",
			},
		}

		req := &protos.CreateEventSourceRequest{
			CanvasIdOrName: uuid.New().String(),
			EventSource:    eventSource,
		}

		_, err := CreateEventSource(context.Background(), encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("event source without integration is created", func(t *testing.T) {
		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, EventSourceCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		name := support.RandomName("source")
		eventSource := &protos.EventSource{
			Metadata: &protos.EventSource_Metadata{
				Name: name,
			},
		}

		response, err := CreateEventSource(context.Background(), encryptor, &protos.CreateEventSourceRequest{
			CanvasIdOrName: r.Canvas.Name,
			EventSource:    eventSource,
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.EventSource)
		assert.NotEmpty(t, response.EventSource.Metadata.Id)
		assert.NotEmpty(t, response.EventSource.Metadata.CreatedAt)
		assert.NotEmpty(t, response.Key)
		assert.Equal(t, name, response.EventSource.Metadata.Name)
		assert.Equal(t, r.Canvas.ID.String(), response.EventSource.Metadata.CanvasId)
		assert.Nil(t, response.EventSource.Spec.Integration)
		assert.Nil(t, response.EventSource.Spec.Semaphore)
		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("name already used -> error", func(t *testing.T) {
		name := support.RandomName("source")
		eventSource := &protos.EventSource{
			Metadata: &protos.EventSource_Metadata{
				Name: name,
			},
		}

		//
		// First one is created.
		//
		_, err := CreateEventSource(context.Background(), encryptor, &protos.CreateEventSourceRequest{
			CanvasIdOrName: r.Canvas.Name,
			EventSource:    eventSource,
		})

		require.NoError(t, err)

		//
		// Second one fails.
		//
		_, err = CreateEventSource(context.Background(), encryptor, &protos.CreateEventSourceRequest{
			CanvasIdOrName: r.Canvas.Name,
			EventSource:    eventSource,
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())
	})

	t.Run("event source for integration resource is created", func(t *testing.T) {
		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, EventSourceCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		name := support.RandomName("source")
		eventSource := &protos.EventSource{
			Metadata: &protos.EventSource_Metadata{
				Name: name,
			},
			Spec: &protos.EventSource_Spec{
				Integration: &integrationPb.IntegrationRef{
					Name: r.Integration.Name,
				},
				Semaphore: &protos.EventSource_Spec_Semaphore{
					Project: "demo-project",
				},
			},
		}

		response, err := CreateEventSource(context.Background(), encryptor, &protos.CreateEventSourceRequest{
			CanvasIdOrName: r.Canvas.Name,
			EventSource:    eventSource,
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.EventSource)
		assert.NotEmpty(t, response.EventSource.Metadata.Id)
		assert.NotEmpty(t, response.EventSource.Metadata.CreatedAt)
		assert.NotEmpty(t, response.Key)
		assert.Equal(t, name, response.EventSource.Metadata.Name)
		assert.Equal(t, r.Canvas.ID.String(), response.EventSource.Metadata.CanvasId)
		assert.Equal(t, r.Integration.Name, response.EventSource.Spec.Integration.Name)
		assert.Equal(t, "demo-project", response.EventSource.Spec.Semaphore.Project)
		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("event source for the same integration resource -> error", func(t *testing.T) {
		name := support.RandomName("source")
		eventSource := &protos.EventSource{
			Metadata: &protos.EventSource_Metadata{
				Name: name,
			},
			Spec: &protos.EventSource_Spec{
				Integration: &integrationPb.IntegrationRef{
					Name: r.Integration.Name,
				},
				Semaphore: &protos.EventSource_Spec_Semaphore{
					Project: "demo-project",
				},
			},
		}

		_, err := CreateEventSource(context.Background(), encryptor, &protos.CreateEventSourceRequest{
			CanvasIdOrName: r.Canvas.Name,
			EventSource:    eventSource,
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "event source for project demo-project already exists", s.Message())
	})

	t.Run("event source when internal one exists for the same integration resource -> becomes external", func(t *testing.T) {
		//
		// Create internal source for integration resource
		//
		internalName := support.RandomName("internal")
		internalSource, _, err := builders.NewEventSourceBuilder(r.Encryptor).
			InCanvas(r.Canvas).
			WithName(internalName).
			WithScope(models.EventSourceScopeInternal).
			ForIntegration(r.Integration).
			ForResource(&models.Resource{
				ResourceName: "demo-project-2",
				ResourceType: "project",
			}).
			Create()

		require.NoError(t, err)

		//
		// Create external source for the same integration resource
		//
		externalName := support.RandomName("external")
		eventSource := &protos.EventSource{
			Metadata: &protos.EventSource_Metadata{
				Name: externalName,
			},
			Spec: &protos.EventSource_Spec{
				Integration: &integrationPb.IntegrationRef{
					Name: r.Integration.Name,
				},
				Semaphore: &protos.EventSource_Spec_Semaphore{
					Project: "demo-project-2",
				},
			},
		}

		response, err := CreateEventSource(context.Background(), encryptor, &protos.CreateEventSourceRequest{
			CanvasIdOrName: r.Canvas.Name,
			EventSource:    eventSource,
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.EventSource)
		assert.NotEmpty(t, response.EventSource.Metadata.Id)
		assert.NotEmpty(t, response.EventSource.Metadata.CreatedAt)
		assert.NotEmpty(t, response.Key)
		assert.Equal(t, externalName, response.EventSource.Metadata.Name)
		assert.Equal(t, r.Canvas.ID.String(), response.EventSource.Metadata.CanvasId)
		assert.Equal(t, r.Integration.Name, response.EventSource.Spec.Integration.Name)
		assert.Equal(t, "demo-project-2", response.EventSource.Spec.Semaphore.Project)

		//
		// Verify that internal source was updated to be external
		//
		_, err = models.FindEventSourceByName(internalSource.Name)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		source, err := models.FindEventSource(internalSource.ID)
		require.NoError(t, err)
		assert.Equal(t, models.EventSourceScopeExternal, source.Scope)
		assert.Equal(t, externalName, source.Name)
	})
}
