package models

import (
	"regexp"
	"slices"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListCanvasesMentioningSkill checks saved prompt fields, including agent steps.
// Other configuration strings, such as shell commands, do not count as mentions.
func (f *Factory) ListCanvasesMentioningSkill(tx *gorm.DB, command string) ([]Canvas, error) {
	canvases, err := f.ListCanvases(tx)
	if err != nil || len(canvases) == 0 {
		return canvases, err
	}
	ids := make([]uuid.UUID, len(canvases))
	for i := range canvases {
		ids[i] = canvases[i].ID
	}
	var nodes []CanvasNode
	if err := tx.Select("workflow_id", "configuration").Where("workflow_id IN ?", ids).Find(&nodes).Error; err != nil {
		return nil, err
	}
	// Keep the same boundaries as skill tokens in the prompt editor.
	mention := regexp.MustCompile(`(^|[\s(])/` + regexp.QuoteMeta(command) + `($|[\s.,!?;:)])`)
	used := make(map[uuid.UUID]bool)
	for _, node := range nodes {
		if configurationMentionsSkill(node.Configuration.Data(), mention) {
			used[node.WorkflowID] = true
		}
	}
	return slices.DeleteFunc(canvases, func(canvas Canvas) bool { return !used[canvas.ID] }), nil
}

func configurationMentionsSkill(value any, mention *regexp.Regexp) bool {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if key == "prompt" {
				if prompt, ok := child.(string); ok && mention.MatchString(prompt) {
					return true
				}
			}
			if configurationMentionsSkill(child, mention) {
				return true
			}
		}
	case []any:
		for _, child := range value {
			if configurationMentionsSkill(child, mention) {
				return true
			}
		}
	}
	return false
}
