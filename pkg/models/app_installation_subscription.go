package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AppInstallationSubscription struct {
	ID             uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	InstallationID uuid.UUID
	WorkflowID     uuid.UUID
	NodeID         string
	Configuration  datatypes.JSONType[any]
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

func CreateAppSubscription(node *WorkflowNode, installation *AppInstallation, configuration any) (*AppInstallationSubscription, error) {
	return CreateAppSubscriptionInTransaction(database.Conn(), node, installation, configuration)
}

func CreateAppSubscriptionInTransaction(tx *gorm.DB, node *WorkflowNode, installation *AppInstallation, configuration any) (*AppInstallationSubscription, error) {
	now := time.Now()
	s := AppInstallationSubscription{
		InstallationID: installation.ID,
		WorkflowID:     node.WorkflowID,
		NodeID:         node.NodeID,
		Configuration:  datatypes.NewJSONType(configuration),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	err := tx.Create(&s).Error
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func DeleteAppSubscriptionsForNodeInTransaction(tx *gorm.DB, workflowID uuid.UUID, nodeID string) error {
	return tx.
		Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).
		Delete(&AppInstallationSubscription{}).
		Error
}

type NodeSubscription struct {
	WorkflowID    uuid.UUID
	NodeID        string
	NodeType      string
	NodeRef       datatypes.JSONType[NodeRef]
	Configuration datatypes.JSONType[any]
}

func FindAppSubscriptionForNodeInTransaction(tx *gorm.DB, workflowID uuid.UUID, nodeID string, installationID uuid.UUID) (*NodeSubscription, error) {
	var subscription NodeSubscription
	err := tx.
		Where("workflow_id = ? AND node_id = ? AND installation_id = ?", workflowID, nodeID, installationID).
		First(&subscription).
		Error

	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func ListAppSubscriptions(tx *gorm.DB, installationID uuid.UUID) ([]NodeSubscription, error) {
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
