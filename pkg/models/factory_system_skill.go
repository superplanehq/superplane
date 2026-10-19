package models

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrFactorySystemSkillRequired = errors.New("only system skills can be restored")

func (c FactoryAgentResourceConfig) SkillContentHash() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.TrimSpace(c.Markdown))))
}

// RestoreSystemSkill replaces instructions and advances the supplied baseline.
// The caller resolves the template from the current SuperPlane catalog.
func (r *FactoryAgentResource) RestoreSystemSkill(tx *gorm.DB, version int32, markdown string) error {
	config, err := r.systemSkillConfig(version, markdown)
	if err != nil {
		return err
	}
	return tx.Model(r).Clauses(clause.Returning{}).Updates(map[string]any{
		"config": datatypes.NewJSONType(config), "updated_at": time.Now(),
	}).Error
}

// UpdateSystemSkillIfUnmodified advances only an untouched supplied baseline.
// The conditional write also protects edits and newer releases made after scanning.
func (r *FactoryAgentResource) UpdateSystemSkillIfUnmodified(tx *gorm.DB, version int32, markdown string) (bool, error) {
	if !r.IsSystemSkill() {
		return false, nil
	}
	original := r.Config.Data()
	baseline := original.SystemSkill
	if baseline.Version > version || original.SkillContentHash() != baseline.ContentHash {
		return false, nil
	}
	config, err := r.systemSkillConfig(version, markdown)
	if err != nil {
		return false, err
	}
	if baseline.Version == version && original.SkillContentHash() == config.SkillContentHash() {
		return false, nil
	}
	result := tx.Model(&FactoryAgentResource{}).
		Where("id = ? AND config = ?", r.ID, r.Config).
		Updates(map[string]any{"config": datatypes.NewJSONType(config), "updated_at": time.Now()})
	return result.RowsAffected > 0, result.Error
}

func (r *FactoryAgentResource) systemSkillConfig(version int32, markdown string) (FactoryAgentResourceConfig, error) {
	if !r.IsSystemSkill() {
		return FactoryAgentResourceConfig{}, ErrFactorySystemSkillRequired
	}
	if version < 1 {
		return FactoryAgentResourceConfig{}, ErrFactorySystemSkillMetadataInvalid
	}
	config := r.Config.Data()
	config.Markdown = markdown
	if err := config.ValidateSkill(); err != nil {
		return FactoryAgentResourceConfig{}, err
	}
	config = config.NormalizedSkill()
	metadata := *config.SystemSkill
	metadata.Version = version
	metadata.ContentHash = config.SkillContentHash()
	config.SystemSkill = &metadata
	return config, nil
}
