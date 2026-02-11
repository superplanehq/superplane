package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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

	err := tx.Create(&s).Error
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func DeleteIntegrationSubscription(tx *gorm.DB, id uuid.UUID) error {
	result := tx.Where("id = ?", id).Delete(&IntegrationSubscription{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
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

// FindIntegrationSubscriptionByMessageTS finds a subscription by installation_id and message_ts
// without loading node information. This is useful for webhook handlers that only need
// the subscription configuration.
// Note: This function specifically filters for subscriptions with type='button_click'.
func FindIntegrationSubscriptionByMessageTS(tx *gorm.DB, installationID uuid.UUID, messageTS string) (*IntegrationSubscription, error) {
	var subscription IntegrationSubscription

	err := tx.
		Where("installation_id = ?", installationID).
		Where("configuration->>'message_ts' = ?", messageTS).
		Where("configuration->>'type' = ?", "button_click").
		First(&subscription).
		Error

	if err != nil {
		return nil, err
	}

	return &subscription, nil
}
