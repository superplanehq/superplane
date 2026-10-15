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
	if !r.IsSystemSkill() {
		return ErrFactorySystemSkillRequired
	}
	if version < 1 {
		return ErrFactorySystemSkillMetadataInvalid
	}
	config := r.Config.Data()
	config.Markdown = markdown
	if err := config.ValidateSkill(); err != nil {
		return err
	}
	config = config.NormalizedSkill()
	metadata := *config.SystemSkill
	metadata.Version = version
	metadata.ContentHash = config.SkillContentHash()
	config.SystemSkill = &metadata
	return tx.Model(r).Clauses(clause.Returning{}).Updates(map[string]any{
		"config": datatypes.NewJSONType(config), "updated_at": time.Now(),
	}).Error
}
