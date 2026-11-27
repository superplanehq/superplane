package models

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (c *Canvas) SetConfigurationID(tx *gorm.DB, id string) error {
	if err := validateConfigurationID(id); err != nil {
		return err
	}
	if c.FactoryID == nil {
		return fmt.Errorf("a public automation ID requires a workspace")
	}
	if err := tx.Model(c).Update("configuration_id", id).Error; err != nil {
		return err
	}
	c.ConfigurationID = id
	return nil
}

// AssignConfigurationIDs persists missing IDs for this line during YAML export.
// The caller holds the workspace lock and commits the complete export atomically.
func (l *FactoryLine) AssignConfigurationIDs(tx *gorm.DB) error {
	stages, err := l.executionStages(tx)
	if err != nil {
		return err
	}
	if err := l.retainConfigurationStageIDs(tx, stages); err != nil {
		return err
	}
	var ids []string
	if err := tx.Raw(`SELECT stage->>'configuration_id'
		FROM factory_lines line, jsonb_array_elements(line.stages) stage
		WHERE line.factory_id = ? AND stage->>'configuration_id' IS NOT NULL`, l.FactoryID).Scan(&ids).Error; err != nil {
		return err
	}
	taken := make(map[string]bool, len(ids)+len(stages))
	for _, id := range ids {
		taken[id] = true
	}
	for _, stage := range stages {
		taken[stage.ConfigurationID] = true
	}
	changed := len(l.Stages) == 0
	for i := range stages {
		if stages[i].ConfigurationID == "" {
			stages[i].ConfigurationID = availableConfigurationID(stages[i].Name, taken)
			taken[stages[i].ConfigurationID] = true
			changed = true
		}
	}
	if !changed {
		return nil
	}
	l.Stages = datatypes.JSONSlice[FactoryLineStage](stages)
	return tx.Model(l).UpdateColumn("stages", l.Stages).Error
}

// AssignConfigurationAutomationIDs adopts existing automations without publishing new revisions.
// The caller holds the workspace lock in the export transaction.
func (f *Factory) AssignConfigurationAutomationIDs(tx *gorm.DB) error {
	canvases, err := f.ListCanvases(tx)
	if err != nil {
		return err
	}
	taken := make(map[string]bool, len(canvases))
	for _, canvas := range canvases {
		taken[canvas.ConfigurationID] = true
	}
	byCanvas := make(map[uuid.UUID]string, len(canvases))
	for i := range canvases {
		canvas := &canvases[i]
		if canvas.ConfigurationID == "" {
			canvas.ConfigurationID = availableConfigurationID(canvas.Name, taken)
			if err := tx.Model(canvas).UpdateColumn("configuration_id", canvas.ConfigurationID).Error; err != nil {
				return err
			}
			taken[canvas.ConfigurationID] = true
		}
		byCanvas[canvas.ID] = canvas.ConfigurationID
	}
	owners, err := ListFactoryConfigurationAutomations(tx, f.OrganizationID, f.ID)
	if err != nil {
		return err
	}
	adopted := make([]FactoryConfigurationAutomation, 0, len(owners))
	seen := make(map[uuid.UUID]bool, len(owners))
	changed := false
	for _, owner := range owners {
		id := byCanvas[owner.CanvasID]
		if id == "" {
			id = owner.ManifestID
		}
		if seen[owner.CanvasID] {
			changed = true
			continue
		}
		seen[owner.CanvasID] = true
		changed = changed || owner.ManifestID != id
		owner.ManifestID = id
		adopted = append(adopted, owner)
	}
	if !changed {
		return nil
	}
	// Replace the complete set so swapped legacy IDs cannot overwrite bindings.
	if err := tx.Where("factory_id = ?", f.ID).Delete(&FactoryConfigurationAutomation{}).Error; err != nil {
		return err
	}
	return tx.Create(&adopted).Error
}

func (l *FactoryLine) retainConfigurationStageIDs(tx *gorm.DB, stages []FactoryLineStage) error {
	factory := &Factory{ID: l.FactoryID, OrganizationID: l.OrganizationID}
	if err := factory.LockForUpdate(tx); err != nil {
		return err
	}
	var ids []string
	if err := tx.Raw(`
		SELECT stage->>'configuration_id'
		FROM factory_lines line, jsonb_array_elements(line.stages) stage
		WHERE line.factory_id = ? AND line.id <> ?
		  AND stage->>'configuration_id' IS NOT NULL
	`, l.FactoryID, l.ID).Scan(&ids).Error; err != nil {
		return err
	}
	taken := make(map[string]bool, len(ids))
	for _, id := range ids {
		taken[id] = true
	}
	var saved FactoryLine
	if err := tx.Select("stages").Where("id = ?", l.ID).Find(&saved).Error; err != nil {
		return err
	}
	existing := map[uuid.UUID]string{}
	for _, stage := range saved.Stages {
		existing[stage.ID] = stage.ConfigurationID
	}
	for i := range stages {
		stage := &stages[i]
		if stage.ConfigurationID == "" {
			stage.ConfigurationID = existing[stage.ID]
		}
		if stage.ConfigurationID == "" {
			continue
		}
		if err := validateConfigurationID(stage.ConfigurationID); err != nil {
			return err
		}
		if taken[stage.ConfigurationID] {
			return fmt.Errorf("stage ID %q is already used in this workspace", stage.ConfigurationID)
		}
		taken[stage.ConfigurationID] = true
	}
	return nil
}

func validateConfigurationID(id string) error {
	if id == "" || strings.TrimSpace(id) != id {
		return fmt.Errorf("use a nonempty public ID without surrounding whitespace")
	}
	return nil
}

func availableConfigurationID(name string, taken map[string]bool) string {
	base := Slugify(name)
	for suffix := 1; ; suffix++ {
		candidate := base
		if suffix > 1 {
			candidate = fmt.Sprintf("%s-%d", base, suffix)
		}
		if !taken[candidate] {
			return candidate
		}
	}
}
