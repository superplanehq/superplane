package models

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// BeforeCreate assigns a public identity once, before a factory automation exists.
func (c *Canvas) BeforeCreate(tx *gorm.DB) error {
	if c.FactoryID == nil {
		return nil
	}
	factory := &Factory{ID: *c.FactoryID, OrganizationID: c.OrganizationID}
	if err := factory.LockForUpdate(tx); err != nil {
		return err
	}
	if c.ConfigurationID != "" {
		return validateConfigurationID(c.ConfigurationID)
	}
	var ids []string
	if err := tx.Model(&Canvas{}).Where("factory_id = ?", *c.FactoryID).Pluck("configuration_id", &ids).Error; err != nil {
		return err
	}
	taken := make(map[string]bool, len(ids))
	for _, id := range ids {
		taken[id] = true
	}
	c.ConfigurationID = availableConfigurationID(c.Name, taken)
	return nil
}

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

func (l *FactoryLine) BeforeCreate(tx *gorm.DB) error {
	stages := l.StagesValue()
	if len(stages) == 0 {
		var err error
		stages, err = defaultFactoryLineStages(tx, l.Steps)
		if err != nil {
			return err
		}
	}
	if err := l.assignConfigurationStageIDs(tx, stages); err != nil {
		return err
	}
	l.Stages = datatypes.JSONSlice[FactoryLineStage](stages)
	return nil
}

func (l *FactoryLine) assignConfigurationStageIDs(tx *gorm.DB, stages []FactoryLineStage) error {
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
	for i := range stages {
		stage := &stages[i]
		if stage.ConfigurationID != "" {
			continue
		}
		stage.ConfigurationID = availableConfigurationID(stage.Name, taken)
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
