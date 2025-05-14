package models

import (
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func Test__StageExecution(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := uuid.New()
	user := uuid.New()

	canvas, err := CreateCanvas(org, user, "test")
	require.NoError(t, err)
	source, err := canvas.CreateEventSource("gh", []byte("my-key"))
	require.NoError(t, err)
	require.NoError(t, canvas.CreateStage("stg-1", user.String(), []StageCondition{}, RunTemplate{}, []StageConnection{}, StageTagUsageDefinition{}))
	stage, err := canvas.FindStageByName("stg-1")
	require.NoError(t, err)

	data := `{"hello": "world"}`
	event, err := CreateEvent(source.ID, source.Name, SourceTypeEventSource, []byte(data), []byte(data))
	require.NoError(t, err)
	stageEvent, err := CreateStageEvent(stage.ID, event, StageEventStatePending, "")
	require.NoError(t, err)

	t.Run("can get event data for execution", func(t *testing.T) {
		stageExecution, err := CreateStageExecution(stage.ID, stageEvent.ID)
		require.NoError(t, err)
		raw, err := stageExecution.GetEventData()
		require.NoError(t, err)
		require.Equal(t, map[string]any{"hello": "world"}, raw)
	})
}
