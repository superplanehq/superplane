package eventsources

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	authpb "github.com/superplanehq/superplane/pkg/protos/authorization"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	integrationPb "github.com/superplanehq/superplane/pkg/protos/integrations"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const EventSourceCreatedRoutingKey = "event-source-created"

func Test__CreateEventSource(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Integration: true})

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		req := &protos.CreateEventSourceRequest{
			EventSource: &protos.EventSource{
				Metadata: &protos.EventSource_Metadata{
					Name: "test",
				},
			},
		}

		_, err := CreateEventSource(context.Background(), r.Encryptor, r.Registry, uuid.NewString(), req)
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
		req := &protos.CreateEventSourceRequest{
			EventSource: &protos.EventSource{
				Metadata: &protos.EventSource_Metadata{
					Name: name,
				},
			},
		}

		response, err := CreateEventSource(context.Background(), r.Encryptor, r.Registry, r.Canvas.ID.String(), req)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.EventSource)
		assert.NotEmpty(t, response.EventSource.Metadata.Id)
		assert.NotEmpty(t, response.EventSource.Metadata.CreatedAt)
		assert.NotEmpty(t, response.Key)
		assert.Equal(t, name, response.EventSource.Metadata.Name)
		assert.Equal(t, r.Canvas.ID.String(), response.EventSource.Metadata.CanvasId)
		assert.Nil(t, response.EventSource.Spec.Integration)
		assert.Nil(t, response.EventSource.Spec.Resource)
		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("name already used -> error", func(t *testing.T) {
		name := support.RandomName("source")
		req := &protos.CreateEventSourceRequest{
			EventSource: &protos.EventSource{
				Metadata: &protos.EventSource_Metadata{
					Name: name,
				},
			},
		}

		//
		// First one is created.
		//
		_, err := CreateEventSource(context.Background(), r.Encryptor, r.Registry, r.Canvas.ID.String(), req)
		require.NoError(t, err)

		//
		// Second one fails.
		//
		_, err = CreateEventSource(context.Background(), r.Encryptor, r.Registry, r.Canvas.ID.String(), req)
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
		req := &protos.CreateEventSourceRequest{
			EventSource: &protos.EventSource{
				Metadata: &protos.EventSource_Metadata{
					Name: name,
				},
				Spec: &protos.EventSource_Spec{
					Integration: &integrationPb.IntegrationRef{
						Name: r.Integration.Name,
					},
					Resource: &integrationPb.ResourceRef{
						Type: "project",
						Name: "demo-project",
					},
				},
			},
		}

		response, err := CreateEventSource(context.Background(), r.Encryptor, r.Registry, r.Canvas.ID.String(), req)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.EventSource)
		assert.NotEmpty(t, response.EventSource.Metadata.Id)
		assert.NotEmpty(t, response.EventSource.Metadata.CreatedAt)
		assert.NotEmpty(t, response.Key)
		assert.Equal(t, name, response.EventSource.Metadata.Name)
		assert.Equal(t, r.Canvas.ID.String(), response.EventSource.Metadata.CanvasId)
		assert.Equal(t, r.Integration.Name, response.EventSource.Spec.Integration.Name)
		assert.Equal(t, "demo-project", response.EventSource.Spec.Resource.Name)
		assert.Equal(t, "project", response.EventSource.Spec.Resource.Type)
		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("event source for integration that does not exist -> error", func(t *testing.T) {
		name := support.RandomName("source")
		req := &protos.CreateEventSourceRequest{
			EventSource: &protos.EventSource{
				Metadata: &protos.EventSource_Metadata{
					Name: name,
				},
				Spec: &protos.EventSource_Spec{
					Integration: &integrationPb.IntegrationRef{
						Name: "does-not-exist",
					},
					Resource: &integrationPb.ResourceRef{
						Type: "project",
						Name: "demo-project",
					},
				},
			},
		}

		_, err := CreateEventSource(context.Background(), r.Encryptor, r.Registry, r.Canvas.ID.String(), req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "integration does-not-exist not found", s.Message())
	})

	t.Run("event source with organization-level integration is created", func(t *testing.T) {
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
							DomainType: models.DomainTypeOrganization,
							Name:       secret.Name,
							Key:        "key",
						},
					},
				},
			}),
		})

		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, EventSourceCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		name := support.RandomName("source")
		req := &protos.CreateEventSourceRequest{
			EventSource: &protos.EventSource{
				Metadata: &protos.EventSource_Metadata{
					Name: name,
				},
				Spec: &protos.EventSource_Spec{
					Integration: &integrationPb.IntegrationRef{
						DomainType: authpb.DomainType_DOMAIN_TYPE_ORGANIZATION,
						Name:       integration.Name,
					},
					Resource: &integrationPb.ResourceRef{
						Type: "project",
						Name: "demo-project",
					},
				},
			},
		}

		response, err := CreateEventSource(context.Background(), r.Encryptor, r.Registry, r.Canvas.ID.String(), req)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.EventSource)
		assert.NotEmpty(t, response.EventSource.Metadata.Id)
		assert.NotEmpty(t, response.EventSource.Metadata.CreatedAt)
		assert.NotEmpty(t, response.Key)
		assert.Equal(t, name, response.EventSource.Metadata.Name)
		assert.Equal(t, r.Canvas.ID.String(), response.EventSource.Metadata.CanvasId)
		assert.Equal(t, integration.Name, response.EventSource.Spec.Integration.Name)
		assert.Equal(t, authpb.DomainType_DOMAIN_TYPE_ORGANIZATION, response.EventSource.Spec.Integration.DomainType)
		assert.Equal(t, "demo-project", response.EventSource.Spec.Resource.Name)
		assert.Equal(t, "project", response.EventSource.Spec.Resource.Type)
		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("event source for the same integration resource -> error", func(t *testing.T) {
		name := support.RandomName("source")
		req := &protos.CreateEventSourceRequest{
			EventSource: &protos.EventSource{
				Metadata: &protos.EventSource_Metadata{
					Name: name,
				},
				Spec: &protos.EventSource_Spec{
					Integration: &integrationPb.IntegrationRef{
						Name: r.Integration.Name,
					},
					Resource: &integrationPb.ResourceRef{
						Type: "project",
						Name: "demo-project",
					},
				},
			},
		}

		_, err := CreateEventSource(context.Background(), r.Encryptor, r.Registry, r.Canvas.ID.String(), req)
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
		req := &protos.CreateEventSourceRequest{
			EventSource: &protos.EventSource{
				Metadata: &protos.EventSource_Metadata{
					Name: externalName,
				},
				Spec: &protos.EventSource_Spec{
					Integration: &integrationPb.IntegrationRef{
						Name: r.Integration.Name,
					},
					Resource: &integrationPb.ResourceRef{
						Type: "project",
						Name: "demo-project-2",
					},
				},
			},
		}

		response, err := CreateEventSource(context.Background(), r.Encryptor, r.Registry, r.Canvas.ID.String(), req)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.EventSource)
		assert.NotEmpty(t, response.EventSource.Metadata.Id)
		assert.NotEmpty(t, response.EventSource.Metadata.CreatedAt)
		assert.NotEmpty(t, response.Key)
		assert.Equal(t, externalName, response.EventSource.Metadata.Name)
		assert.Equal(t, r.Canvas.ID.String(), response.EventSource.Metadata.CanvasId)
		assert.Equal(t, r.Integration.Name, response.EventSource.Spec.Integration.Name)
		assert.Equal(t, "demo-project-2", response.EventSource.Spec.Resource.Name)
		assert.Equal(t, "project", response.EventSource.Spec.Resource.Type)

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
