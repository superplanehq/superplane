package factories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func MaterializeFactoryAppTemplate(
	ctx context.Context,
	organizationID string,
	req *pb.MaterializeFactoryAppTemplateRequest,
) (*pb.MaterializeFactoryAppTemplateResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app template")
	}
	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app template")
	}
	appID, err := parseFactoryAppID(req.GetAppId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app template")
	}

	db := database.DB(ctx)
	if _, err := models.FindFactory(db, orgID, factoryID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app template")
	}
	canvas, _, err := findFactoryAppForDefaults(db, orgID, factoryID, appID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app template")
	}

	input := factoryTemplateInputFromRequest(req)
	input.appID = canvas.ID.String()
	input.appName = canvas.Name
	result, err := materializeFactoryTemplate(req.GetTemplateId(), input)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app template")
	}

	return &pb.MaterializeFactoryAppTemplateResponse{
		TemplateId:  result.templateID,
		CanvasYaml:  result.canvasYAML,
		ConsoleYaml: result.consoleYAML,
	}, nil
}

func MaterializeFactoryAppDefaults(
	ctx context.Context,
	organizationID string,
	req *pb.MaterializeFactoryAppDefaultsRequest,
) (*pb.MaterializeFactoryAppDefaultsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app defaults")
	}
	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app defaults")
	}
	appID, err := parseFactoryAppID(req.GetAppId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app defaults")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app defaults")
	}
	canvas, version, err := findFactoryAppForDefaults(db, orgID, factoryID, appID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app defaults")
	}

	var result *materializedFactoryTemplate
	intake, intakeErr := models.FindFactoryIntakeByCanvasID(db, appID)
	switch {
	case intakeErr == nil && intake.FactoryID == factoryID:
		result, err = materializeIntakeDefaults(db, canvas, version, intake)
	case intakeErr != nil && !errors.Is(intakeErr, models.ErrFactoryIntakeNotFound):
		err = intakeErr
	default:
		result, err = materializeNonIntakeFactoryAppDefaults(db, factory, canvas, version)
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app defaults")
	}

	return &pb.MaterializeFactoryAppDefaultsResponse{
		TemplateId:  result.templateID,
		CanvasYaml:  result.canvasYAML,
		ConsoleYaml: result.consoleYAML,
	}, nil
}

func parseFactoryAppID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid factory app ID")
	}
	return id, nil
}
