package support

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	authpb "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/semaphore"
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
	SemaphoreAPIMock *semaphore.SemaphoreAPIMock
}

func (r *ResourceRegistry) Close() {
	if r.SemaphoreAPIMock != nil {
		r.SemaphoreAPIMock.Close()
	}
}

type SetupOptions struct {
	Source    bool
	Stage     bool
	Approvals int
}

func Setup(t *testing.T) *ResourceRegistry {
	return SetupWithOptions(t, SetupOptions{
		Source:    true,
		Stage:     true,
		Approvals: 1,
	})
}

func SetupWithOptions(t *testing.T, options SetupOptions) *ResourceRegistry {
	require.NoError(t, database.TruncateTables())

	r := ResourceRegistry{
		User:      uuid.New(),
		Encryptor: crypto.NewNoOpEncryptor(),
	}

	var err error
	r.Organization, err = models.CreateOrganization(r.User, uuid.New().String(), "test")
	require.NoError(t, err)

	r.Canvas, err = models.CreateCanvas(r.User, r.Organization.ID, "test")
	require.NoError(t, err)

	if options.Source {
		r.Source, err = r.Canvas.CreateEventSource("gh", []byte("my-key"), nil)
		require.NoError(t, err)
	}

	r.SemaphoreAPIMock = semaphore.NewSemaphoreAPIMock()
	r.SemaphoreAPIMock.Init()
	log.Infof("Semaphore API mock started at %s", r.SemaphoreAPIMock.Server.URL)

	//
	// Create Semaphore integration
	//
	secretData := map[string]string{"key": "test"}
	data, err := json.Marshal(secretData)
	require.NoError(t, err)

	secret, err := models.CreateSecret(randomName("secret"), secrets.ProviderLocal, r.User.String(), r.Canvas.ID, data)
	require.NoError(t, err)

	integration, err := models.CreateIntegration(&models.Integration{
		Name:       randomName("integration"),
		CreatedBy:  r.User,
		Type:       models.IntegrationTypeSemaphore,
		DomainType: "canvas",
		DomainID:   r.Canvas.ID,
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

	require.NoError(t, err)
	r.Integration = integration

	if options.Stage {
		conditions := []models.StageCondition{
			{
				Type:     models.StageConditionTypeApproval,
				Approval: &models.ApprovalCondition{Count: options.Approvals},
			},
		}

		executor, resource := Executor(&r)
		stage, err := r.Canvas.CreateStage(r.Encryptor, "stage-1",
			r.User.String(),
			conditions,
			executor,
			&resource,
			[]models.Connection{
				{
					SourceType: models.SourceTypeEventSource,
					SourceID:   r.Source.ID,
					SourceName: r.Source.Name,
				},
			},
			[]models.InputDefinition{
				{Name: "VERSION"},
			},
			[]models.InputMapping{
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
			},
			[]models.OutputDefinition{},
			[]models.ValueDefinition{},
		)

		require.NoError(t, err)
		r.Stage = stage
	}

	return &r
}

func CreateConnectionGroup(t *testing.T, name string, canvas *models.Canvas, source *models.EventSource, timeout uint32, timeoutBehavior string) *models.ConnectionGroup {
	connectionGroup, err := canvas.CreateConnectionGroup(
		name,
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

	event, err := models.CreateEvent(source.ID, source.Name, models.SourceTypeEventSource, []byte(`{}`), []byte(`{}`))
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
	event, err := models.CreateEvent(source.ID, source.Name, models.SourceTypeEventSource, data, headers)
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
	execution, err := models.CreateStageExecution(stage.ID, event.ID)
	require.NoError(t, err)
	return execution
}

func Executor(r *ResourceRegistry) (models.StageExecutor, models.Resource) {
	return models.StageExecutor{
			Type: models.ExecutorSpecTypeSemaphore,
			Spec: datatypes.NewJSONType(models.ExecutorSpec{
				Semaphore: &models.SemaphoreExecutorSpec{
					ProjectID:    "demo-project",
					Branch:       "main",
					PipelineFile: ".semaphore/run.yml",
					Parameters: map[string]string{
						"PARAM_1": "VALUE_1",
						"PARAM_2": "VALUE_2",
					},
				},
			}),
		}, models.Resource{
			Type:          integrations.ResourceTypeProject,
			ExternalID:    uuid.NewString(),
			IntegrationID: r.Integration.ID,
			Name:          "demo-project",
		}
}

func ProtoExecutor(r *ResourceRegistry) *pb.ExecutorSpec {
	return &pb.ExecutorSpec{
		Type: pb.ExecutorSpec_TYPE_SEMAPHORE,
		Integration: &pb.IntegrationRef{
			DomainType: authpb.DomainType_DOMAIN_TYPE_CANVAS,
			Name:       r.Integration.Name,
		},
		Semaphore: &pb.ExecutorSpec_Semaphore{
			ProjectId:    "demo-project",
			TaskId:       "task",
			Branch:       "main",
			PipelineFile: ".semaphore/semaphore.yml",
			Parameters:   map[string]string{},
		},
	}
}

func randomName(prefix string) string {
	return prefix + "-" + uuid.New().String()
}
