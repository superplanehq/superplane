package factories

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func parseLineID(lineID string) (uuid.UUID, error) {
	id, err := uuid.Parse(lineID)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid line id")
	}

	return id, nil
}

func parseLineSteps(
	tx *gorm.DB,
	organizationID, factoryID uuid.UUID,
	steps []*pb.FactoryLine_Step,
) ([]models.FactoryLineStep, error) {
	if len(steps) == 0 {
		return nil, invalidArgument("at least one step is required")
	}

	seenNames := make(map[string]struct{}, len(steps))
	result := make([]models.FactoryLineStep, 0, len(steps))

	for i, step := range steps {
		parsed, err := parseLineStep(tx, organizationID, factoryID, step, i+1, seenNames)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}

	return result, nil
}

func parseLineStep(
	tx *gorm.DB,
	organizationID, factoryID uuid.UUID,
	step *pb.FactoryLine_Step,
	stepNumber int,
	seenNames map[string]struct{},
) (models.FactoryLineStep, error) {
	name := strings.TrimSpace(step.GetName())
	if name == "" {
		return models.FactoryLineStep{}, invalidArgument(fmt.Sprintf("step %d: name is required", stepNumber))
	}

	if _, exists := seenNames[name]; exists {
		return models.FactoryLineStep{}, invalidArgument(fmt.Sprintf("duplicate step name %q", name))
	}
	seenNames[name] = struct{}{}

	stepType := strings.TrimSpace(step.GetType())
	if stepType != models.FactoryLineStepTypeRunApp {
		return models.FactoryLineStep{}, invalidArgument(
			fmt.Sprintf("step %q: unsupported type %q", name, stepType),
		)
	}

	appStep := step.GetApp()
	if appStep == nil {
		return models.FactoryLineStep{}, invalidArgument(fmt.Sprintf("step %q: app is required", name))
	}

	appRef := strings.TrimSpace(appStep.GetApp())
	entrypoint := strings.TrimSpace(appStep.GetEntrypoint())
	if appRef == "" {
		return models.FactoryLineStep{}, invalidArgument(fmt.Sprintf("step %q: app is required", name))
	}
	if entrypoint == "" {
		return models.FactoryLineStep{}, invalidArgument(fmt.Sprintf("step %q: entrypoint is required", name))
	}

	canvas, err := resolveFactoryOwnedApp(tx, organizationID, factoryID, appRef)
	if err != nil {
		return models.FactoryLineStep{}, err
	}

	node, err := models.FindCanvasNode(tx, canvas.ID, entrypoint)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.FactoryLineStep{}, invalidArgument(
				fmt.Sprintf("step %q: entrypoint %q not found", name, entrypoint),
			)
		}
		return models.FactoryLineStep{}, err
	}

	if node.Type != models.NodeTypeTrigger {
		return models.FactoryLineStep{}, invalidArgument(
			fmt.Sprintf("step %q: entrypoint %q is not a trigger", name, entrypoint),
		)
	}

	var maxParallelism *int
	if step.MaxParallelism != nil {
		if *step.MaxParallelism < 0 {
			return models.FactoryLineStep{}, invalidArgument(
				fmt.Sprintf("step %q: max_parallelism must be non-negative (0 means unlimited)", name),
			)
		}
		value := int(*step.MaxParallelism)
		maxParallelism = &value
	}

	return models.FactoryLineStep{
		Name:           name,
		Type:           models.FactoryLineStepTypeRunApp,
		AppID:          canvas.ID,
		Entrypoint:     entrypoint,
		MaxParallelism: maxParallelism,
	}, nil
}

func resolveFactoryOwnedApp(
	tx *gorm.DB,
	organizationID, factoryID uuid.UUID,
	appRef string,
) (*models.Canvas, error) {
	var canvas *models.Canvas
	var err error

	if appID, parseErr := uuid.Parse(appRef); parseErr == nil {
		canvas, err = models.FindCanvasInTransaction(tx, organizationID, appID)
	} else {
		canvas, err = models.FindCanvasByNameInTransaction(tx, appRef, organizationID)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, invalidArgument(fmt.Sprintf("app %q not found", appRef))
		}
		return nil, err
	}

	if canvas.FactoryID == nil || *canvas.FactoryID != factoryID {
		return nil, invalidArgument(fmt.Sprintf("app %q is not owned by this factory", appRef))
	}

	return canvas, nil
}
