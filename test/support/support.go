package support

import (
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/delivery"
	"github.com/superplanehq/superplane/test/semaphore"
)

type ResourceRegistry struct {
	Org              uuid.UUID
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
		Org:  uuid.New(),
		User: uuid.New(),
	}

	var err error
	r.Canvas, err = models.CreateCanvas(r.Org, r.User, "test")
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

		err = r.Canvas.CreateStage("stage-1", r.User.String(), conditions, RunTemplate(), []models.StageConnection{}, TagUsageDef(r.Source.Name))
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
	return CreateStageEventWithData(t, source, stage, []byte(`{"ref":"v1"}`), []byte(`{"ref":"v1"}`))
}

func CreateStageEventWithData(t *testing.T, source *models.EventSource, stage *models.Stage, data []byte, headers []byte) *models.StageEvent {
	event, err := models.CreateEvent(source.ID, source.Name, models.SourceTypeEventSource, data, headers)
	require.NoError(t, err)
	stageEvent, err := models.CreateStageEvent(stage.ID, event, models.StageEventStatePending, "")
	require.NoError(t, err)

	tags := map[string]string{}
	for _, tag := range stage.Use.Data().Tags {
		v, err := event.EvaluateStringExpression(tag.ValueFrom)
		require.NoError(t, err)
		tags[tag.Name] = v
	}

	require.NoError(t,
		models.UpdateStageEventTagStateInBulk(
			database.Conn(),
			stageEvent.ID,
			models.TagStateUnknown,
			tags,
		),
	)

	return stageEvent
}

func CreateExecution(t *testing.T, source *models.EventSource, stage *models.Stage) *models.StageExecution {
	return CreateExecutionWithData(t, source, stage, []byte(`{"ref":"v1"}`), []byte(`{"ref":"v1"}`))
}

func CreateExecutionWithData(t *testing.T, source *models.EventSource, stage *models.Stage, data []byte, headers []byte) *models.StageExecution {
	event := CreateStageEventWithData(t, source, stage, data, headers)
	execution, err := models.CreateStageExecution(stage.ID, event.ID)
	require.NoError(t, err)
	return execution
}

func TagUsageDef(sourceName string) models.StageTagUsageDefinition {
	return models.StageTagUsageDefinition{
		From: []string{sourceName},
		Tags: []models.StageTagDefinition{
			{Name: "VERSION", ValueFrom: "ref"},
		},
	}
}

func RunTemplate() models.RunTemplate {
	return RunTemplateWithURL("http://localhost:8000")
}

func RunTemplateWithURL(URL string) models.RunTemplate {
	return models.RunTemplate{
		Type: models.RunTemplateTypeSemaphore,
		Semaphore: &models.SemaphoreRunTemplate{
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

func ProtoRunTemplate() *protos.RunTemplate {
	return &protos.RunTemplate{
		Type: protos.RunTemplate_TYPE_SEMAPHORE,
		Semaphore: &protos.SemaphoreRunTemplate{
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
