package factories

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func TestFactoryConfigurationPreservesSharedStageAttachments(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	id := uuid.New()
	shared := models.FactoryLineStage{ID: id, Key: "custom-" + id.String(), Name: "Recheck", Kind: models.FactoryLineStagePhase}
	stages := f.line.StagesValue()
	done := stages[len(stages)-1]
	stages = append(stages[:len(stages)-1], shared, done)
	require.NoError(t, f.line.UpdateConfiguration(db, nil, nil, nil, stages))
	require.NoError(t, f.canvas.SetColumnKey(db, shared.Key))
	beforeSteps, err := f.line.ExecutionPlan(db)
	require.NoError(t, err)
	require.Len(t, beforeSteps, 2)
	beforeRevisionCount := configurationRevisionCount(t, f.canvas)
	exported, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.Key})
	require.NoError(t, err)
	require.Equal(t, 2, strings.Count(exported.Bundle.Files[0].Content, "@superplane/implement"), "Export both stage attachments of one automation")
	plan := f.plan(t, exported.Bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	exported.Bundle.Files[0].Content = strings.ReplaceAll(exported.Bundle.Files[0].Content, "Original instructions", "Keep both stage bindings.")
	plan = f.plan(t, exported.Bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 1, "Plan shared settings once")
	applied, err := f.apply(exported.Bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, applied.Changed)
	require.Equal(t, beforeRevisionCount+1, configurationRevisionCount(t, f.canvas), "Publish one revision for the shared automation")
	line, err := f.factory.FindLine(db, f.line.ID)
	require.NoError(t, err)
	afterSteps, err := line.ExecutionPlan(db)
	require.NoError(t, err)
	require.Equal(t, beforeSteps, afterSteps)
	canvas, err := models.FindCanvasInTransaction(db, f.factory.OrganizationID, f.canvas.ID)
	require.NoError(t, err)
	require.Equal(t, shared.Key, *canvas.ColumnKey)
}
