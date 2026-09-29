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

	result := make([]models.FactoryLineStep, 0, len(steps))
	for i, step := range steps {
		parsed, err := parseLineStep(tx, organizationID, factoryID, step, i+1)
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
) (models.FactoryLineStep, error) {
	stepType := strings.TrimSpace(step.GetType())
	if stepType != models.FactoryLineStepTypeRunApp {
		return models.FactoryLineStep{}, invalidArgument(
			fmt.Sprintf("step %d: unsupported type %q", stepNumber, stepType),
		)
	}

	appStep := step.GetApp()
	if appStep == nil {
		return models.FactoryLineStep{}, invalidArgument(fmt.Sprintf("step %d: app is required", stepNumber))
	}

	appRef := strings.TrimSpace(appStep.GetApp())
	entrypoint := strings.TrimSpace(appStep.GetEntrypoint())
	if appRef == "" {
		return models.FactoryLineStep{}, invalidArgument(fmt.Sprintf("step %d: app is required", stepNumber))
	}
	if entrypoint == "" {
		return models.FactoryLineStep{}, invalidArgument(fmt.Sprintf("step %d: entrypoint is required", stepNumber))
	}

	canvas, err := resolveFactoryOwnedApp(tx, organizationID, factoryID, appRef)
	if err != nil {
		return models.FactoryLineStep{}, err
	}

	node, err := models.FindCanvasNode(tx, canvas.ID, entrypoint)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.FactoryLineStep{}, invalidArgument(
				fmt.Sprintf("step %d: entrypoint %q not found", stepNumber, entrypoint),
			)
		}
		return models.FactoryLineStep{}, err
	}

	if node.Type != models.NodeTypeTrigger {
		return models.FactoryLineStep{}, invalidArgument(
			fmt.Sprintf("step %d: entrypoint %q is not a trigger", stepNumber, entrypoint),
		)
	}

	var maxParallelism *int
	if step.MaxParallelism != nil {
		if *step.MaxParallelism < 1 {
			return models.FactoryLineStep{}, invalidArgument(
				fmt.Sprintf("step %d: max_parallelism must be at least 1", stepNumber),
			)
		}
		value := int(*step.MaxParallelism)
		maxParallelism = &value
	}

	return models.FactoryLineStep{
		Type:           models.FactoryLineStepTypeRunApp,
		AppID:          canvas.ID,
		Entrypoint:     entrypoint,
		MaxParallelism: maxParallelism,
	}, nil
}

// factoryLineColumnColorIDs are the allowed board column color ids. Keep in
// sync with LINE_BOARD_COLUMN_COLORS in
// web_src/src/pages/factories/pages/lineBoardColumnColors.ts.
var factoryLineColumnColorIDs = map[string]bool{
	"lime":   true,
	"yellow": true,
	"teal":   true,
	"sky":    true,
	"purple": true,
	"slate":  true,
}

const (
	factoryLineColumnColorKeyMaxLength = 64
	factoryLineColumnColorMaxEntries   = 64
)

// parseLineColumnColors validates a full replacement map of board column
// colors and returns a normalized, non-nil copy (an empty input map is a
// valid "clear all colors" request). Unknown color ids, and empty or
// oversized keys, are rejected.
func parseLineColumnColors(colors map[string]string) (map[string]string, error) {
	if len(colors) > factoryLineColumnColorMaxEntries {
		return nil, invalidArgument("too many column colors")
	}

	result := make(map[string]string, len(colors))
	for key, colorID := range colors {
		if key == "" || len(key) > factoryLineColumnColorKeyMaxLength {
			return nil, invalidArgument(fmt.Sprintf("column color key %q is invalid", key))
		}
		if !factoryLineColumnColorIDs[colorID] {
			return nil, invalidArgument(fmt.Sprintf("column color %q is not supported", colorID))
		}
		result[key] = colorID
	}

	return result, nil
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
		canvas, err = models.FindCanvasByName(tx, organizationID, &factoryID, appRef)
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
