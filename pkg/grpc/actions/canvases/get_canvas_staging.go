package canvases

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

func GetCanvasStaging(ctx context.Context, organizationID string, canvasID string) (*pb.Staging, error) {
	db := database.DB(ctx)

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	canvas, err := models.FindCanvasInTransaction(db, organizationUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "canvas not found")
		}

		return nil, grpcerrors.Internal(err, "failed to load canvas")
	}

	userUUID := uuid.MustParse(userID)
	rows, err := models.ListStagedFilesForUser(db, canvas.ID, userUUID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load staging")
	}

	return buildStaging(ctx, canvas, rows)
}

func buildStaging(ctx context.Context, canvas *models.Canvas, rows []models.WorkflowStagedFile) (*pb.Staging, error) {
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.DB(ctx), canvas.ID)
	if err != nil {
		return nil, err
	}

	state := &pb.Staging{}

	//
	// If nothing is staged, just use what is in the live version
	//
	if len(rows) == 0 {
		spec, err := SerializeCanvasSpecFromVersion(liveVersion)
		if err != nil {
			return nil, err
		}
		state.Spec = spec
		return state, nil
	}

	//
	// Otherwise, combine the live version with the staged changes
	//
	spec, err := effectiveCanvasSpec(canvas, liveVersion, rows)
	if err != nil {
		return nil, err
	}

	state.Spec = spec

	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		paths = append(paths, row.Path)
	}

	base := findStagingBaseVersionID(rows)
	state.HasStaging = true
	state.StagedPaths = paths
	state.BaseVersionId = base.String()
	state.Stale = canvas.LiveVersionID.String() != base.String()

	return state, nil
}

func findStagingBaseVersionID(rows []models.WorkflowStagedFile) uuid.UUID {
	if len(rows) == 0 {
		return uuid.Nil
	}
	return rows[0].BaseVersionID
}

func effectiveCanvasSpec(canvas *models.Canvas, version *models.CanvasVersion, rows []models.WorkflowStagedFile) (*pb.Canvas_Spec, error) {
	spec, err := SerializeCanvasSpecFromVersion(version)
	if err != nil {
		return nil, err
	}

	canvasYAML, err := effectiveSpecYAML(canvas, version, rows, CanvasYAMLRepositoryPath)
	if err != nil {
		return nil, err
	}

	if err := applyCanvasYAMLToSpec(spec, canvasYAML); err != nil {
		return nil, err
	}

	consoleYAML, err := effectiveSpecYAML(canvas, version, rows, ConsoleYAMLRepositoryPath)
	if err != nil {
		return nil, err
	}
	if err := applyConsoleYAMLToSpec(spec, consoleYAML); err != nil {
		return nil, err
	}

	return spec, nil
}

func applyCanvasYAMLToSpec(spec *pb.Canvas_Spec, yamlText string) error {
	if strings.TrimSpace(yamlText) == "" {
		spec.Nodes = nil
		spec.Edges = nil
		return nil
	}

	canvas, err := yaml.CanvasFromYAML([]byte(yamlText))
	if err != nil {
		return err
	}

	if canvas == nil || canvas.Spec == nil {
		return nil
	}

	spec.Nodes = actions.NodesToProto(canvas.Nodes())
	spec.Edges = actions.EdgesToProto(canvas.Edges())
	return nil
}

func applyConsoleYAMLToSpec(spec *pb.Canvas_Spec, yamlText string) error {
	if strings.TrimSpace(yamlText) == "" {
		spec.Panels = nil
		spec.Layout = nil
		return nil
	}

	console, err := yaml.ConsoleFromYML([]byte(yamlText))
	if err != nil {
		return err
	}

	protoPanels, err := ConsolePanelsToProto(console.Panels())
	if err != nil {
		return err
	}

	spec.Panels = protoPanels
	spec.Layout = ConsoleLayoutToProto(console.Layout())
	return nil
}
