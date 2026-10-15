package models

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/features"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ListSystemSkillsForSync visits active workspaces across all eligible organizations.
// UUID pagination bounds each query without excluding customized or unknown templates.
func ListSystemSkillsForSync(tx *gorm.DB, after uuid.UUID, limit int) ([]FactoryAgentResource, error) {
	var resources []FactoryAgentResource
	query := tx.Model(&FactoryAgentResource{}).Select("factory_agent_resources.*").
		Joins("JOIN factories ON factories.id = factory_agent_resources.factory_id AND factories.deleted_at IS NULL").
		Joins("JOIN organizations ON organizations.id = factory_agent_resources.organization_id AND organizations.deleted_at IS NULL").
		Where("factory_agent_resources.kind = ? AND factory_agent_resources.id > ?", FactoryAgentResourceKindSkill, after).
		Where("factory_agent_resources.config->'systemSkill' IS NOT NULL AND factory_agent_resources.config->'systemSkill' <> 'null'::jsonb")
	if !features.IsReleased(features.FeatureWorkspaceSkills) {
		query = query.Where("organizations.enabled_experimental_features @> ?", datatypes.NewJSONSlice([]string{features.FeatureWorkspaceSkills}))
	}
	err := query.Order("factory_agent_resources.id ASC").Limit(limit).Find(&resources).Error
	return resources, err
}
