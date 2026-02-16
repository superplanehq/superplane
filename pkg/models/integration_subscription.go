package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IntegrationSubscription struct {
	ID             uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	InstallationID uuid.UUID
	WorkflowID     uuid.UUID
	NodeID         string
	Configuration  datatypes.JSONType[any]
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

func (a *IntegrationSubscription) TableName() string {
	return "app_installation_subscriptions"
}

func CreateIntegrationSubscription(node *CanvasNode, integration *Integration, configuration any) (*IntegrationSubscription, error) {
	return CreateIntegrationSubscriptionInTransaction(database.Conn(), node, integration, configuration)
}

func CreateIntegrationSubscriptionInTransaction(tx *gorm.DB, node *CanvasNode, integration *Integration, configuration any) (*IntegrationSubscription, error) {
	now := time.Now()
	s := IntegrationSubscription{
		InstallationID: integration.ID,
		WorkflowID:     node.WorkflowID,
		NodeID:         node.NodeID,
		Configuration:  datatypes.NewJSONType(configuration),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	err := tx.
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "installation_id"},
				{Name: "workflow_id"},
				{Name: "node_id"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"configuration": s.Configuration,
				"updated_at":    now,
			}),
		}).
		Create(&s).
		Error
	if err != nil {
		return nil, err
	}

	var subscription IntegrationSubscription
	err = tx.
		Where("installation_id = ?", integration.ID).
		Where("workflow_id = ?", node.WorkflowID).
		Where("node_id = ?", node.NodeID).
		First(&subscription).
		Error
	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func DeleteIntegrationSubscriptionsForNodeInTransaction(tx *gorm.DB, workflowID uuid.UUID, nodeID string) error {
	return tx.
		Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).
		Delete(&IntegrationSubscription{}).
		Error
}

type NodeSubscription struct {
	WorkflowID    uuid.UUID
	NodeID        string
	NodeType      string
	NodeRef       datatypes.JSONType[NodeRef]
	Configuration datatypes.JSONType[any]
}

func ListIntegrationSubscriptions(tx *gorm.DB, installationID uuid.UUID) ([]NodeSubscription, error) {
	var subscriptions []NodeSubscription

	err := tx.
		Table("app_installation_subscriptions AS s").
		Select("wn.workflow_id as workflow_id, wn.node_id as node_id, wn.type as node_type, wn.ref as node_ref, s.configuration as configuration").
		Joins("INNER JOIN workflow_nodes AS wn ON wn.workflow_id = s.workflow_id AND wn.node_id = s.node_id").
		Where("s.installation_id = ?", installationID).
		Where("wn.deleted_at IS NULL").
		Scan(&subscriptions).
		Error

	if err != nil {
		return nil, err
	}

	return subscriptions, nil
}
