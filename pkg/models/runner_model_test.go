package models_test

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestResolveBYOKRunnerModelPreservesConfiguredModelWithoutPolicy(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	configuration := map[string]any{
		"model":       "claude-sonnet-4-6",
		"credentials": map[string]any{"source": "secret"},
	}

	resolved, err := models.ResolveBYOKRunnerModel(db, r.Organization.ID, nil, "runnerClaudeCode", configuration)
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet-4-6", resolved)

	resolved, err = models.ResolveBYOKRunnerModel(db, r.Organization.ID, &factory.ID, "runnerClaudeCode", configuration)
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet-4-6", resolved)
}

func TestQueuedAPIKeyTasksPreserveSnapshotsWithoutCatalogSetup(t *testing.T) {
	for _, runnerModel := range []struct{ component, provider, model string }{
		{"runnerCodex", models.UsageProviderOpenAI, "gpt-5"},
		{"runnerClaudeCode", models.UsageProviderAnthropic, "claude-sonnet-4-6"},
		{"runnerOpenRouter", models.UsageProviderOpenRouter, "openai/gpt-5"},
	} {
		for _, source := range []string{"integration", "secret"} {
			for _, scenario := range []struct {
				name                            string
				inherit                         *bool
				restricted, blank, commandsOnly bool
			}{
				{name: "missing policy"}, {name: "native default", blank: true},
				{name: "inherited policy", inherit: boolPointer(true)},
				{name: "explicit restriction", inherit: boolPointer(false), restricted: true},
				{name: "commands only", inherit: boolPointer(false), restricted: true, commandsOnly: true},
			} {
				t.Run(runnerModel.component+"/"+source+"/"+scenario.name, func(t *testing.T) {
					r := support.Setup(t)
					require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
					db := database.DB(t.Context())
					factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
					require.NoError(t, err)
					if scenario.inherit != nil {
						_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, runnerModel.provider, models.UsageFundingSourceBYOK, nil, scenario.inherit)
						require.NoError(t, err)
					}
					canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "agent", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: runnerModel.component}})}}, nil)
					require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
					run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "agent", models.CanvasRunStateStarted, "")
					require.NoError(t, err)
					event := support.EmitCanvasEventForNode(t, canvas.ID, "agent", "default", nil)
					execution := support.CreateCanvasNodeExecution(t, canvas.ID, "agent", event.ID, event.ID)
					model := runnerModel.model
					if scenario.blank {
						model = ""
					}
					configuration := map[string]any{"model": model, "credentials": map[string]any{"source": source}}
					if scenario.commandsOnly {
						configuration["steps"] = []any{map[string]any{"type": "bash", "command": "echo ready"}}
					}
					require.NoError(t, db.Model(execution).Updates(map[string]any{"run_id": run.ID, "configuration": datatypes.NewJSONType(configuration), "metadata": datatypes.NewJSONType(map[string]any{models.CanvasNodeExecutionFrozenComponentKey: runnerModel.component})}).Error)
					fleet := newTestRunnerFleet()
					require.NoError(t, fleet.Create(db))
					task := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, PayloadCiphertext: []byte("unchanged payload"), QueuedAt: time.Now()}
					require.NoError(t, db.Create(task).Error)
					require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, canvas.ID, "agent", execution.ID, "task_id", task.ID.String()))
					runner := &models.Runner{ID: uuid.New(), FleetID: fleet.ID, State: models.RunnerStateIdle, RunnerVersion: fleet.RunnerVersion}
					require.NoError(t, db.Create(runner).Error)
					reservation, err := runner.ReserveNextTask(db)
					require.NoError(t, err)
					if scenario.restricted && !scenario.commandsOnly {
						assert.Nil(t, reservation.Task)
						require.Len(t, reservation.FailedTasks, 1)
						assert.Equal(t, task.ID, reservation.FailedTasks[0].ID)
					} else {
						require.NotNil(t, reservation.Task)
						assert.Equal(t, task.ID, reservation.Task.ID)
						assert.Equal(t, []byte("unchanged payload"), reservation.Task.PayloadCiphertext)
						assert.Empty(t, reservation.FailedTasks)
					}
					saved, err := task.FindExecution(db)
					require.NoError(t, err)
					assert.Equal(t, configuration, saved.Configuration.Data())
				})
			}
		}
	}
}
