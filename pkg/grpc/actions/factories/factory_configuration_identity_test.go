package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	canvasyaml "github.com/superplanehq/superplane/pkg/yaml"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

func TestFactoryConfigurationExportAdoptsIDsOnce(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	first := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*canvasyaml.Canvas) {})
	second := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*canvasyaml.Canvas) {})
	require.NoError(t, db.Model(first).Update("name", "Review Task").Error)
	require.NoError(t, db.Model(second).Update("name", "Review-Task").Error)
	// Simulate an existing workspace, including a line whose stages are derived from steps.
	require.NoError(t, db.Model(&models.Canvas{}).Where("factory_id = ?", f.factory.ID).Update("configuration_id", "").Error)
	require.NoError(t, db.Model(f.line).Update("stages", datatypes.JSONSlice[models.FactoryLineStage]{}).Error)
	owner := models.FactoryConfigurationAutomation{
		OrganizationID: f.r.Organization.ID, FactoryID: f.factory.ID, LineID: f.line.ID,
		StageKey: "verify", ManifestID: "review-task-2", CanvasID: first.ID,
		Binding: datatypes.JSONMap{"resource_id": "acme/app"},
	}
	require.NoError(t, owner.Save(db))
	secondOwner := owner
	secondOwner.ManifestID = "review-task"
	secondOwner.CanvasID = second.ID
	secondOwner.Binding = datatypes.JSONMap{"resource_id": "acme/backlog"}
	require.NoError(t, secondOwner.Save(db))
	duplicate := owner
	duplicate.StageKey = "zzz"
	require.NoError(t, db.Create(&duplicate).Error)
	other, err := f.factory.CreateLine(db, "Unexported line", nil)
	require.NoError(t, err)
	beforeVersions := configurationRevisionCount(t, f.canvas)
	invalidStages := datatypes.JSONSlice[models.FactoryLineStage]{{ID: uuid.New(), Key: "backlog", Name: "Backlog", Kind: models.FactoryLineStageBacklog}}
	require.NoError(t, db.Model(f.line).Update("stages", invalidStages).Error)
	_, err = ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.Key, LineId: f.line.ID.String()})
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	stored, err := models.FindCanvasInTransaction(db, f.r.Organization.ID, first.ID)
	require.NoError(t, err)
	require.Empty(t, stored.ConfigurationID, "failed export rolls back ID assignment")
	unchanged, err := f.factory.FindLine(db, f.line.ID)
	require.NoError(t, err)
	require.Equal(t, invalidStages, unchanged.Stages)
	owners, err := models.ListFactoryConfigurationAutomations(db, f.r.Organization.ID, f.factory.ID)
	require.NoError(t, err)
	require.Len(t, owners, 3, "failed export retains legacy bindings")
	require.NoError(t, db.Model(f.line).Update("stages", datatypes.JSONSlice[models.FactoryLineStage]{}).Error)
	export := func() *pb.FactoryConfigurationBundle {
		t.Helper()
		response, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.Key, LineId: f.line.ID.String()})
		require.NoError(t, err)
		return response.Bundle
	}
	export()
	ids := map[uuid.UUID]string{}
	for _, canvas := range []*models.Canvas{f.canvas, first, second} {
		stored, err := models.FindCanvasInTransaction(db, f.r.Organization.ID, canvas.ID)
		require.NoError(t, err)
		require.NotEmpty(t, stored.ConfigurationID)
		ids[canvas.ID] = stored.ConfigurationID
	}
	require.ElementsMatch(t, []string{"review-task", "review-task-2"}, []string{ids[first.ID], ids[second.ID]})
	line, err := f.factory.FindLine(db, f.line.ID)
	require.NoError(t, err)
	require.Len(t, line.Stages, 4)
	require.Equal(t, f.line.Steps, line.Steps)
	for _, stage := range line.Stages {
		require.NotEmpty(t, stage.ConfigurationID)
	}
	untouched, err := f.factory.FindLine(db, other.ID)
	require.NoError(t, err)
	require.Equal(t, other.Stages, untouched.Stages)
	require.Equal(t, beforeVersions, configurationRevisionCount(t, f.canvas))
	storedOwner, err := models.FindFactoryConfigurationAutomationByCanvas(db, first.ID)
	require.NoError(t, err)
	require.Equal(t, owner.Binding, storedOwner.Binding)
	storedOwner, err = models.FindFactoryConfigurationAutomationByCanvas(db, second.ID)
	require.NoError(t, err)
	require.Equal(t, secondOwner.Binding, storedOwner.Binding)
	owners, err = models.ListFactoryConfigurationAutomations(db, f.r.Organization.ID, f.factory.ID)
	require.NoError(t, err)
	require.Len(t, owners, 2)

	require.NoError(t, db.Model(first).Update("name", "Renamed review").Error)
	stages := line.StagesValue()
	stages[1].Name = "Renamed phase"
	stages[1].ConfigurationID = ""
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
	export()
	for id, expected := range ids {
		stored, err := models.FindCanvasInTransaction(db, f.r.Organization.ID, id)
		require.NoError(t, err)
		require.Equal(t, expected, stored.ConfigurationID)
	}
	reloaded, err := f.factory.FindLine(db, line.ID)
	require.NoError(t, err)
	require.Equal(t, line.Stages, reloaded.Stages)
	require.Equal(t, export(), export(), "later exports reuse saved identities")
}
