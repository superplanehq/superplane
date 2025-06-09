package inputs

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__InputBuilder(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})

	docsSource, err := r.Canvas.CreateEventSource("docs", []byte("docs-key"))
	require.NoError(t, err)
	require.NotNil(t, docsSource)
	tfSource, err := r.Canvas.CreateEventSource("tf", []byte("tf-key"))
	require.NoError(t, err)

	t.Run("no inputs", func(t *testing.T) {
		builder := NewBuilder(models.Stage{})
		inputs, err := builder.Build(nil, nil)
		require.NoError(t, err)
		require.Empty(t, inputs)
	})

	t.Run("value is defined statically", func(t *testing.T) {
		stage := models.Stage{
			Inputs: []models.InputDefinition{{Name: "VERSION"}},
			InputMappings: []models.InputMapping{
				{
					Values: []models.ValueDefinition{
						{
							Name:  "VERSION",
							Value: strAsPointer("static"),
						},
					},
				},
			},
		}

		builder := NewBuilder(stage)
		inputs, err := builder.Build(database.Conn(), &models.Event{})
		require.NoError(t, err)
		require.Equal(t, map[string]any{"VERSION": "static"}, inputs)
	})

	t.Run("value is defined from event data", func(t *testing.T) {
		stage := models.Stage{
			Inputs: []models.InputDefinition{{Name: "VERSION"}},
			InputMappings: []models.InputMapping{
				{
					Values: []models.ValueDefinition{
						{
							Name: "VERSION",
							ValueFrom: &models.ValueDefinitionFrom{
								EventData: &models.ValueDefinitionFromEventData{
									Connection: "github",
									Expression: "ref",
								},
							},
						},
					},
				},
			},
		}

		builder := NewBuilder(stage)
		inputs, err := builder.Build(database.Conn(), &models.Event{
			SourceName: "github",
			Raw:        []byte(`{"ref":"from-event"}`),
		})

		require.NoError(t, err)
		require.Equal(t, map[string]any{"VERSION": "from-event"}, inputs)
	})

	t.Run("one value defined from event data, another from last execution", func(t *testing.T) {

		//
		// Create stage, connected to our two sources
		//
		err := r.Canvas.CreateStage("stage-1", r.User.String(), []models.StageCondition{}, support.ExecutorSpec(), []models.StageConnection{
			{
				SourceID:   docsSource.ID,
				SourceName: docsSource.Name,
				SourceType: models.SourceTypeEventSource,
			},
			{
				SourceID:   tfSource.ID,
				SourceName: tfSource.Name,
				SourceType: models.SourceTypeEventSource,
			},
		}, []models.InputDefinition{
			{
				Name: "DOCS_VERSION",
			},
			{
				Name: "TF_VERSION",
			},
		}, []models.InputMapping{
			{
				When: &models.InputMappingWhen{
					TriggeredBy: &models.WhenTriggeredBy{
						Connection: docsSource.Name,
					},
				},
				Values: []models.ValueDefinition{
					{
						Name: "DOCS_VERSION",
						ValueFrom: &models.ValueDefinitionFrom{
							EventData: &models.ValueDefinitionFromEventData{
								Connection: "docs",
								Expression: "ref",
							},
						},
					},
					{
						Name: "TF_VERSION",
						ValueFrom: &models.ValueDefinitionFrom{
							LastExecution: &models.ValueDefinitionFromLastExecution{
								Results: []string{"passed"},
							},
						},
					},
				},
			},
			{
				When: &models.InputMappingWhen{
					TriggeredBy: &models.WhenTriggeredBy{
						Connection: tfSource.Name,
					},
				},
				Values: []models.ValueDefinition{
					{
						Name: "DOCS_VERSION",
						ValueFrom: &models.ValueDefinitionFrom{
							LastExecution: &models.ValueDefinitionFromLastExecution{
								Results: []string{"passed"},
							},
						},
					},
					{
						Name: "TF_VERSION",
						ValueFrom: &models.ValueDefinitionFrom{
							EventData: &models.ValueDefinitionFromEventData{
								Connection: "docs",
								Expression: "ref",
							},
						},
					},
				},
			},
		}, []models.OutputDefinition{}, []models.ValueDefinition{})

		require.NoError(t, err)
		stage, err := r.Canvas.FindStageByName("stage-1")
		require.NoError(t, err)

		//
		// Mock a completed previous execution of the stage
		//
		execution := support.CreateExecutionWithData(t, docsSource, stage, []byte(`{"ref":"docs.v1"}`), []byte(`{}`), map[string]any{"DOCS_VERSION": "docs.v1", "TF_VERSION": "terraform.v1"})
		execution.Finish(stage, models.StageExecutionResultPassed)

		//
		// Build inputs from docs source event
		//
		builder := NewBuilder(*stage)
		inputs, err := builder.Build(database.Conn(), &models.Event{
			SourceName: docsSource.Name,
			Raw:        []byte(`{"ref":"docs.v2"}`),
		})

		require.NoError(t, err)
		require.Equal(t, map[string]any{"DOCS_VERSION": "docs.v2", "TF_VERSION": "terraform.v1"}, inputs)

		//
		// Build inputs from tf source event
		//
		builder = NewBuilder(*stage)
		inputs, err = builder.Build(database.Conn(), &models.Event{
			SourceName: tfSource.Name,
			Raw:        []byte(`{"ref":"terraform.v2"}`),
		})

		require.NoError(t, err)
		require.Equal(t, map[string]any{"DOCS_VERSION": "docs.v1", "TF_VERSION": "terraform.v2"}, inputs)
	})
}

func strAsPointer(s string) *string {
	return &s
}
