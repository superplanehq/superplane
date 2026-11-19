package factories

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

const factoryLineStageMaxCount = 64
const factoryLineStageNameMaxLength = 64

func parseFactoryLineStages(
	tx *gorm.DB,
	line *models.FactoryLine,
	stages []*pb.FactoryLine_Stage,
) ([]models.FactoryLineStage, error) {
	if len(stages) > factoryLineStageMaxCount {
		return nil, invalidArgument("a line cannot have more than 64 stages")
	}

	existingByID := make(map[uuid.UUID]models.FactoryLineStage, len(line.Stages))
	for _, stage := range line.StagesValue() {
		existingByID[stage.ID] = stage
	}

	result := make([]models.FactoryLineStage, 0, len(stages))
	seenIDs := make(map[uuid.UUID]bool, len(stages))
	seenNames := make(map[string]bool, len(stages))
	for _, stage := range stages {
		parsed, err := parseFactoryLineStage(stage, existingByID)
		if err != nil {
			return nil, err
		}
		nameKey := strings.ToLower(parsed.Name)
		if seenNames[nameKey] {
			return nil, invalidArgument(fmt.Sprintf("stage name %q is duplicated", parsed.Name))
		}
		if seenIDs[parsed.ID] {
			return nil, invalidArgument("stage id is duplicated")
		}
		seenNames[nameKey] = true
		seenIDs[parsed.ID] = true
		result = append(result, parsed)
	}

	if err := validateFactoryLineSystemStages(line.StagesValue(), result); err != nil {
		return nil, err
	}
	if err := rejectReferencedStageRemovals(tx, line, result); err != nil {
		return nil, err
	}
	return result, nil
}

func parseFactoryLineStage(
	stage *pb.FactoryLine_Stage,
	existingByID map[uuid.UUID]models.FactoryLineStage,
) (models.FactoryLineStage, error) {
	if stage == nil {
		return models.FactoryLineStage{}, invalidArgument("stage is required")
	}
	name := strings.TrimSpace(stage.GetName())
	if name == "" {
		return models.FactoryLineStage{}, invalidArgument("stage name cannot be empty")
	}
	if utf8.RuneCountInString(name) > factoryLineStageNameMaxLength {
		return models.FactoryLineStage{}, invalidArgument("stage name cannot be longer than 64 characters")
	}

	if stage.GetId() == "" {
		if len(stage.GetKey()) > factoryLineColumnColorKeyMaxLength {
			return models.FactoryLineStage{}, invalidArgument("temporary stage key is invalid")
		}
		if (stage.GetKey() != "" && !strings.HasPrefix(stage.GetKey(), "new-")) || (stage.GetKind() != pb.FactoryLine_STAGE_KIND_UNSPECIFIED && stage.GetKind() != pb.FactoryLine_STAGE_KIND_CUSTOM) || stage.StepIndex != nil {
			return models.FactoryLineStage{}, invalidArgument("new stages must be custom stages without an id or step index")
		}
		id := uuid.New()
		return models.FactoryLineStage{
			ID:   id,
			Key:  "custom-" + id.String(),
			Name: name,
			Kind: models.FactoryLineStageCustom,
		}, nil
	}

	id, err := uuid.Parse(stage.GetId())
	if err != nil {
		return models.FactoryLineStage{}, invalidArgument("stage id is invalid")
	}
	existing, ok := existingByID[id]
	if !ok {
		return models.FactoryLineStage{}, invalidArgument("stage does not belong to this line")
	}
	if stage.GetKey() != existing.Key || factoryLineStageKind(stage.GetKind()) != existing.Kind || !sameStepIndex(stage.StepIndex, existing.StepIndex) {
		return models.FactoryLineStage{}, invalidArgument("stage id, key, kind, and step index cannot change")
	}
	existing.Name = name
	return existing, nil
}

func remapNewFactoryLineStageColors(
	request []*pb.FactoryLine_Stage,
	stages []models.FactoryLineStage,
	colors map[string]string,
) map[string]string {
	result := make(map[string]string, len(colors))
	for key, color := range colors {
		result[key] = color
	}
	for index, stage := range request {
		if stage.GetId() != "" || stage.GetKey() == "" || index >= len(stages) {
			continue
		}
		if color, ok := result[stage.GetKey()]; ok {
			delete(result, stage.GetKey())
			result[stages[index].Key] = color
		}
	}
	return result
}

func factoryLineStageKind(kind pb.FactoryLine_StageKind) string {
	switch kind {
	case pb.FactoryLine_STAGE_KIND_BACKLOG:
		return models.FactoryLineStageBacklog
	case pb.FactoryLine_STAGE_KIND_PHASE:
		return models.FactoryLineStagePhase
	case pb.FactoryLine_STAGE_KIND_CUSTOM:
		return models.FactoryLineStageCustom
	case pb.FactoryLine_STAGE_KIND_VERIFY:
		return models.FactoryLineStageVerify
	case pb.FactoryLine_STAGE_KIND_DONE:
		return models.FactoryLineStageDone
	default:
		return ""
	}
}

func sameStepIndex(value *int32, expected *int) bool {
	if value == nil || expected == nil {
		return value == nil && expected == nil
	}
	return int(*value) == *expected
}

func validateFactoryLineSystemStages(existing, next []models.FactoryLineStage) error {
	if len(next) == 0 || next[0].Kind != models.FactoryLineStageBacklog {
		return invalidArgument("Backlog must remain the first stage")
	}
	if next[len(next)-1].Kind != models.FactoryLineStageDone {
		return invalidArgument("Done must remain the last stage")
	}

	expected := systemStageIDs(existing)
	actual := systemStageIDs(next)
	if !slices.Equal(expected, actual) {
		return invalidArgument("system stages must remain present and in order")
	}
	return nil
}

func systemStageIDs(stages []models.FactoryLineStage) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(stages))
	for _, stage := range stages {
		if stage.Kind != models.FactoryLineStageCustom {
			result = append(result, stage.ID)
		}
	}
	return result
}

func rejectReferencedStageRemovals(tx *gorm.DB, line *models.FactoryLine, next []models.FactoryLineStage) error {
	retained := make(map[uuid.UUID]bool, len(next))
	for _, stage := range next {
		retained[stage.ID] = true
	}
	for _, stage := range line.StagesValue() {
		if stage.Kind != models.FactoryLineStageCustom || retained[stage.ID] {
			continue
		}
		referenced, err := line.IsStageReferenced(tx, stage.Key)
		if err != nil {
			return err
		}
		if referenced {
			return fmt.Errorf("%w: %s", errFactoryStageInUse, stage.Name)
		}
	}
	return nil
}
