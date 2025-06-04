package support

import (
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/semaphore"
)

type ResourceRegistry struct {
	User             uuid.UUID
	Canvas           *models.Canvas
	Source           *models.EventSource
	Stage            *models.Stage
	SemaphoreAPIMock *semaphore.SemaphoreAPIMock
}

func (r *ResourceRegistry) Close() {
	if r.SemaphoreAPIMock != nil {
		r.SemaphoreAPIMock.Close()
	}
}

type SetupOptions struct {
	Source       bool
	Stage        bool
	SemaphoreAPI bool
	Approvals    int
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
		User: uuid.New(),
	}

	var err error
	r.Canvas, err = models.CreateCanvas(r.User, "test")
	require.NoError(t, err)

	if options.Source {
		r.Source, err = r.Canvas.CreateEventSource("gh", []byte("my-key"))
		require.NoError(t, err)
	}

	if options.Stage {
		conditions := []models.StageCondition{
			{
				Type:     models.StageConditionTypeApproval,
				Approval: &models.ApprovalCondition{Count: options.Approvals},
			},
		}

		err = r.Canvas.CreateStage("stage-1",
			r.User.String(),
			conditions,
			ExecutorSpec(),
			[]models.StageConnection{
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
		r.Stage, err = r.Canvas.FindStageByName("stage-1")
		require.NoError(t, err)
	}

	if options.SemaphoreAPI {
		r.SemaphoreAPIMock = semaphore.NewSemaphoreAPIMock()
		r.SemaphoreAPIMock.Init()
		log.Infof("Semaphore API mock started at %s", r.SemaphoreAPIMock.Server.URL)
	}

	return &r
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

func ExecutorSpec() models.ExecutorSpec {
	return ExecutorSpecWithURL("http://localhost:8000")
}

func ExecutorSpecWithURL(URL string) models.ExecutorSpec {
	return models.ExecutorSpec{
		Type: models.ExecutorSpecTypeSemaphore,
		Semaphore: &models.SemaphoreExecutorSpec{
			OrganizationURL: URL,
			APIToken:        base64.StdEncoding.EncodeToString([]byte("token")),
			ProjectID:       "demo-project",
			TaskID:          "demo-task",
			Branch:          "main",
			PipelineFile:    ".semaphore/run.yml",
			Parameters: map[string]string{
				"PARAM_1": "VALUE_1",
				"PARAM_2": "VALUE_2",
			},
		},
	}
}

func ProtoExecutor() *protos.ExecutorSpec {
	return &protos.ExecutorSpec{
		Type: protos.ExecutorSpec_TYPE_SEMAPHORE,
		Semaphore: &protos.ExecutorSpec_Semaphore{
			OrganizationUrl: "http://localhost:8000",
			ApiToken:        "test",
			ProjectId:       "test",
			TaskId:          "task",
			Branch:          "main",
			PipelineFile:    ".semaphore/semaphore.yml",
			Parameters:      map[string]string{},
		},
	}
}
