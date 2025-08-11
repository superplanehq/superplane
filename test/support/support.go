package support

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations"
	semaphoreIntegration "github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
	authpb "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	integrationPb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/semaphore"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
)

type ResourceRegistry struct {
	User             uuid.UUID
	Canvas           *models.Canvas
	Source           *models.EventSource
	Stage            *models.Stage
	Organization     *models.Organization
	Integration      *models.Integration
	Encryptor        crypto.Encryptor
	Registry         *registry.Registry
	SemaphoreAPIMock *semaphore.SemaphoreAPIMock
}

func (r *ResourceRegistry) Close() {
	if r.SemaphoreAPIMock != nil {
		r.SemaphoreAPIMock.Close()
	}
}

type SetupOptions struct {
	Source      bool
	Stage       bool
	Approvals   int
	Integration bool
}

func Setup(t *testing.T) *ResourceRegistry {
	return SetupWithOptions(t, SetupOptions{
		Source:      true,
		Stage:       true,
		Integration: true,
		Approvals:   1,
	})
}

func SetupWithOptions(t *testing.T, options SetupOptions) *ResourceRegistry {
	require.NoError(t, database.TruncateTables())

	user := &models.User{
		Name:     "Test User",
		IsActive: true,
	}
	require.NoError(t, user.Create())

	accountProvider := &models.AccountProvider{
		UserID:     user.ID,
		Email:      "test@test.com",
		Username:   "Test",
		Provider:   "github",
		ProviderID: "123",
	}

	require.NoError(t, accountProvider.Create())

	r := ResourceRegistry{
		User:      user.ID,
		Encryptor: crypto.NewNoOpEncryptor(),
	}

	r.Registry = registry.NewRegistry(r.Encryptor)
	r.SemaphoreAPIMock = semaphore.NewSemaphoreAPIMock()
	require.NoError(t, r.SemaphoreAPIMock.Init())
	log.Infof("Semaphore API mock started at %s", r.SemaphoreAPIMock.Server.URL)

	var err error
	r.Organization, err = models.CreateOrganization(r.User, uuid.New().String(), "test", "")
	require.NoError(t, err)

	r.Canvas, err = models.CreateCanvas(r.User, r.Organization.ID, "test", "Test Canvas")
	require.NoError(t, err)

	//
	// Create integration
	//
	if options.Integration {
		secret, err := CreateCanvasSecret(t, &r, map[string]string{"key": "test"})
		require.NoError(t, err)
		integration, err := models.CreateIntegration(&models.Integration{
			Name:       RandomName("integration"),
			CreatedBy:  r.User,
			Type:       models.IntegrationTypeSemaphore,
			DomainType: models.DomainTypeCanvas,
			DomainID:   r.Canvas.ID,
			URL:        r.SemaphoreAPIMock.Server.URL,
			AuthType:   models.IntegrationAuthTypeToken,
			Auth: datatypes.NewJSONType(models.IntegrationAuth{
				Token: &models.IntegrationAuthToken{
					ValueFrom: models.ValueDefinitionFrom{
						Secret: &models.ValueDefinitionFromSecret{
							DomainType: models.DomainTypeCanvas,
							Name:       secret.Name,
							Key:        "key",
						},
					},
				},
			}),
		})

		require.NoError(t, err)
		r.Integration = integration
	}

	//
	// Create source
	//
	if options.Source {
		r.Source = &models.EventSource{
			CanvasID:   r.Canvas.ID,
			Name:       "gh",
			Key:        []byte(`my-key`),
			Scope:      models.EventSourceScopeExternal,
			EventTypes: datatypes.NewJSONSlice([]models.EventType{}),
		}

		err = r.Source.Create()
		require.NoError(t, err)
	}

	if options.Stage {
		conditions := []models.StageCondition{
			{
				Type:     models.StageConditionTypeApproval,
				Approval: &models.ApprovalCondition{Count: options.Approvals},
			},
		}

		executorType, executorSpec, resource := Executor(t, &r)
		stage, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas.ID).
			WithName("stage-1").
			WithRequester(r.User).
			WithConditions(conditions).
			WithConnections([]models.Connection{
				{
					SourceType: models.SourceTypeEventSource,
					SourceID:   r.Source.ID,
					SourceName: r.Source.Name,
				},
			}).
			WithInputs([]models.InputDefinition{
				{Name: "VERSION"},
			}).
			WithInputMappings([]models.InputMapping{
				{
					Values: []models.ValueDefinition{
						{Name: "VERSION", ValueFrom: &models.ValueDefinitionFrom{
							EventData: &models.ValueDefinitionFromEventData{
								Connection: r.Source.Name,
								Expression: "ref",
							},
						}},
					},
				},
			}).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForIntegration(r.Integration).
			ForResource(resource).
			Create()

		require.NoError(t, err)
		r.Stage = stage
	}

	return &r
}

func CreateConnectionGroup(t *testing.T, name string, canvas *models.Canvas, source *models.EventSource, timeout uint32, timeoutBehavior string) *models.ConnectionGroup {
	connectionGroup, err := models.CreateConnectionGroup(
		canvas.ID,
		name,
		"description",
		uuid.NewString(),
		[]models.Connection{
			{SourceID: source.ID, SourceName: source.Name, SourceType: models.SourceTypeEventSource},
		},
		models.ConnectionGroupSpec{
			Timeout:         timeout,
			TimeoutBehavior: timeoutBehavior,
			GroupBy: &models.ConnectionGroupBySpec{
				Fields: []models.ConnectionGroupByField{
					{Name: "test", Expression: "test"},
				},
			},
		},
	)

	require.NoError(t, err)
	return connectionGroup
}

func CreateFieldSet(t *testing.T, fields map[string]string, connectionGroup *models.ConnectionGroup, source *models.EventSource) *models.ConnectionGroupFieldSet {
	hash, err := crypto.SHA256ForMap(fields)
	require.NoError(t, err)

	fieldSet, err := connectionGroup.CreateFieldSet(database.Conn(), fields, hash)
	require.NoError(t, err)

	event, err := models.CreateEvent(source.ID, source.Name, models.SourceTypeEventSource, "push", []byte(`{}`), []byte(`{}`))
	require.NoError(t, err)
	fieldSet.AttachEvent(database.Conn(), event)
	return fieldSet
}

func CreateStageEvent(t *testing.T, source *models.EventSource, stage *models.Stage) *models.StageEvent {
	return CreateStageEventWithData(t, source, stage, []byte(`{"ref":"v1"}`), []byte(`{"ref":"v1"}`), map[string]any{})
}

func CreateStageEventWithData(t *testing.T,
	source *models.EventSource,
	stage *models.Stage,
	data []byte,
	headers []byte,
	inputs map[string]any,
) *models.StageEvent {
	event, err := models.CreateEvent(source.ID, source.Name, models.SourceTypeEventSource, "push", data, headers)
	require.NoError(t, err)
	stageEvent, err := models.CreateStageEvent(stage.ID, event, models.StageEventStatePending, "", inputs)
	require.NoError(t, err)
	return stageEvent
}

func CreateExecution(t *testing.T, source *models.EventSource, stage *models.Stage) *models.StageExecution {
	return CreateExecutionWithData(t, source, stage, []byte(`{"ref":"v1"}`), []byte(`{"ref":"v1"}`), map[string]any{})
}

func CreateExecutionWithData(t *testing.T,
	source *models.EventSource,
	stage *models.Stage,
	data []byte,
	headers []byte,
	inputs map[string]any,
) *models.StageExecution {
	event := CreateStageEventWithData(t, source, stage, data, headers, inputs)
	execution, err := models.CreateStageExecution(stage.CanvasID, stage.ID, event.ID)
	require.NoError(t, err)
	return execution
}

func Executor(t *testing.T, r *ResourceRegistry) (string, []byte, integrations.Resource) {
	spec, err := json.Marshal(map[string]any{
		"branch":       "main",
		"pipelineFile": ".semaphore/run.yml",
		"parameters": map[string]string{
			"PARAM_1": "VALUE_1",
			"PARAM_2": "VALUE_2",
		},
	})

	require.NoError(t, err)

	return models.IntegrationTypeSemaphore, spec, &models.Resource{
		ResourceType:  semaphoreIntegration.ResourceTypeProject,
		ExternalID:    uuid.NewString(),
		IntegrationID: r.Integration.ID,
		ResourceName:  "demo-project",
	}
}

func ProtoExecutor(t *testing.T, r *ResourceRegistry) *pb.Executor {
	spec, err := structpb.NewStruct(map[string]any{
		"branch":       "main",
		"pipelineFile": ".semaphore/run.yml",
		"parameters":   map[string]any{},
	})

	require.NoError(t, err)

	return &pb.Executor{
		Type: models.IntegrationTypeSemaphore,
		Spec: spec,
		Integration: &integrationPb.IntegrationRef{
			DomainType: authpb.DomainType_DOMAIN_TYPE_CANVAS,
			Name:       r.Integration.Name,
		},
		Resource: &integrationPb.ResourceRef{
			Type: semaphoreIntegration.ResourceTypeProject,
			Name: "demo-project",
		},
	}
}

func CreateCanvasSecret(t *testing.T, r *ResourceRegistry, secretData map[string]string) (*models.Secret, error) {
	data, err := json.Marshal(secretData)
	require.NoError(t, err)
	secret, err := models.CreateSecret(RandomName("secret"), secrets.ProviderLocal, r.User.String(), models.DomainTypeCanvas, r.Canvas.ID, data)
	require.NoError(t, err)
	return secret, nil
}

func CreateOrganizationSecret(t *testing.T, r *ResourceRegistry, secretData map[string]string) (*models.Secret, error) {
	data, err := json.Marshal(secretData)
	require.NoError(t, err)
	secret, err := models.CreateSecret(RandomName("secret"), secrets.ProviderLocal, r.User.String(), models.DomainTypeOrganization, r.Organization.ID, data)
	require.NoError(t, err)
	return secret, nil
}

func RandomName(prefix string) string {
	return prefix + "-" + uuid.New().String()
}
