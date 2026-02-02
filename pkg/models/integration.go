package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	IntegrationStatePending = "pending"
	IntegrationStateReady   = "ready"
	IntegrationStateError   = "error"
)

type Integration struct {
	ID               uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	OrganizationID   uuid.UUID
	AppName          string
	InstallationName string
	State            string
	StateDescription string
	Configuration    datatypes.JSONType[map[string]any]
	Metadata         datatypes.JSONType[map[string]any]
	BrowserAction    *datatypes.JSONType[BrowserAction]
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

func (a *Integration) TableName() string {
	return "app_installations"
}

type IntegrationSecret struct {
	ID             uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	OrganizationID uuid.UUID
	InstallationID uuid.UUID
	Name           string
	Value          []byte
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

func (a *IntegrationSecret) TableName() string {
	return "app_installation_secrets"
}

type BrowserAction struct {
	URL         string
	Method      string
	FormFields  map[string]string
	Description string
}

func CreateIntegration(id, orgID uuid.UUID, appName string, installationName string, config map[string]any) (*Integration, error) {
	now := time.Now()
	integration := Integration{
		ID:               id,
		OrganizationID:   orgID,
		AppName:          appName,
		InstallationName: installationName,
		State:            IntegrationStatePending,
		Configuration:    datatypes.NewJSONType(config),
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}

	err := database.Conn().Create(&integration).Error
	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func ListIntegrations(orgID uuid.UUID) ([]Integration, error) {
	var integrations []Integration
	err := database.Conn().Where("organization_id = ?", orgID).Find(&integrations).Error
	if err != nil {
		return nil, err
	}
	return integrations, nil
}

func ListIntegrationWebhooks(tx *gorm.DB, integrationID uuid.UUID) ([]Webhook, error) {
	var webhooks []Webhook
	err := tx.
		Where("app_installation_id = ?", integrationID).
		Find(&webhooks).
		Error

	if err != nil {
		return nil, err
	}
	return webhooks, nil
}

func ListUnscopedIntegrationWebhooks(tx *gorm.DB, integrationID uuid.UUID) ([]Webhook, error) {
	var webhooks []Webhook
	err := tx.Unscoped().
		Where("app_installation_id = ?", integrationID).
		Find(&webhooks).
		Error

	if err != nil {
		return nil, err
	}
	return webhooks, nil
}

func CreateCleanupRequestsForIntegrationNodes(tx *gorm.DB, integrationID uuid.UUID) error {
	var nodes []CanvasNode
	err := tx.
		Where("app_installation_id = ?", integrationID).
		Where("type IN ?", []string{NodeTypeComponent, NodeTypeTrigger}).
		Find(&nodes).
		Error
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		return nil
	}

	runAt := time.Now()
	requests := make([]CanvasNodeRequest, 0, len(nodes))
	for _, node := range nodes {
		requests = append(requests, CanvasNodeRequest{
			ID:         uuid.New(),
			WorkflowID: node.WorkflowID,
			NodeID:     node.NodeID,
			State:      NodeExecutionRequestStatePending,
			Type:       NodeRequestTypeCleanup,
			Spec:       datatypes.NewJSONType(NodeExecutionRequestSpec{}),
			RunAt:      runAt,
			CreatedAt:  runAt,
			UpdatedAt:  runAt,
		})
	}

	return tx.Create(&requests).Error
}

func CountPendingCleanupRequestsForIntegrationNodes(tx *gorm.DB, integrationID uuid.UUID) (int64, error) {
	var count int64
	err := tx.Unscoped().
		Table("workflow_node_requests").
		Joins("JOIN workflow_nodes ON workflow_node_requests.workflow_id = workflow_nodes.workflow_id AND workflow_node_requests.node_id = workflow_nodes.node_id").
		Where("workflow_nodes.app_installation_id = ?", integrationID).
		Where("workflow_node_requests.state = ?", NodeExecutionRequestStatePending).
		Where("workflow_node_requests.type = ?", NodeRequestTypeCleanup).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

type WorkflowNodeReference struct {
	WorkflowID   uuid.UUID
	WorkflowName string
	NodeID       string
	NodeName     string
}

func ListIntegrationNodeReferences(integrationID uuid.UUID) ([]WorkflowNodeReference, error) {
	var nodeReferences []WorkflowNodeReference
	err := database.Conn().
		Table("workflow_nodes AS wn").
		Joins("JOIN workflows AS w ON w.id = wn.workflow_id").
		Select("w.id as workflow_id, w.name as workflow_name, wn.node_id as node_id, wn.name as node_name").
		Where("wn.app_installation_id = ?", integrationID).
		Where("wn.deleted_at IS NULL").
		Find(&nodeReferences).
		Error

	if err != nil {
		return nil, err
	}
	return nodeReferences, nil
}

func FindUnscopedIntegration(integrationID uuid.UUID) (*Integration, error) {
	return FindUnscopedIntegrationInTransaction(database.Conn(), integrationID)
}

func FindUnscopedIntegrationInTransaction(tx *gorm.DB, integrationID uuid.UUID) (*Integration, error) {
	var integration Integration
	err := tx.
		Where("id = ?", integrationID).
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func FindMaybeDeletedIntegrationInTransaction(tx *gorm.DB, integrationID uuid.UUID) (*Integration, error) {
	var integration Integration
	err := tx.Unscoped().
		Where("id = ?", integrationID).
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func FindIntegration(orgID, integrationID uuid.UUID) (*Integration, error) {
	var integration Integration
	err := database.Conn().
		Where("id = ?", integrationID).
		Where("organization_id = ?", orgID).
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func FindIntegrationByName(orgID uuid.UUID, integrationName string) (*Integration, error) {
	var integration Integration
	err := database.Conn().
		Where("organization_id = ?", orgID).
		Where("installation_name = ?", integrationName).
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func ListDeletedIntegrations() ([]Integration, error) {
	var integrations []Integration
	err := database.Conn().Unscoped().
		Where("deleted_at IS NOT NULL").
		Find(&integrations).
		Error

	if err != nil {
		return nil, err
	}

	return integrations, nil
}

func LockIntegration(tx *gorm.DB, ID uuid.UUID) (*Integration, error) {
	var integration Integration

	err := tx.Unscoped().
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", ID).
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func (a *Integration) SoftDelete() error {
	return a.SoftDeleteInTransaction(database.Conn())
}

func (a *Integration) SoftDeleteInTransaction(tx *gorm.DB) error {
	now := time.Now()
	timestamp := now.Unix()

	newName := fmt.Sprintf("%s (deleted-%d)", a.InstallationName, timestamp)
	return tx.Model(a).Updates(map[string]interface{}{
		"deleted_at":        now,
		"installation_name": newName,
	}).Error
}

func (a *Integration) GetRequest(ID string) (*IntegrationRequest, error) {
	var request IntegrationRequest

	err := database.Conn().
		Where("id = ?", ID).
		Where("app_installation_id = ?", a.ID).
		First(&request).
		Error

	if err != nil {
		return nil, err
	}

	return &request, nil
}

func (a *Integration) ListRequests(reqType string) ([]IntegrationRequest, error) {
	requests := []IntegrationRequest{}

	err := database.Conn().
		Where("app_installation_id = ?", a.ID).
		Find(&requests).
		Error

	if err != nil {
		return nil, err
	}

	return requests, nil
}

func (a *Integration) CreateSyncRequest(tx *gorm.DB, runAt *time.Time) error {
	now := time.Now()
	return tx.Create(&IntegrationRequest{
		ID:                uuid.New(),
		AppInstallationID: a.ID,
		State:             IntegrationRequestStatePending,
		Type:              IntegrationRequestTypeSync,
		RunAt:             *runAt,
		CreatedAt:         now,
		UpdatedAt:         now,
	}).Error
}

func (a *Integration) CreateActionRequest(tx *gorm.DB, actionName string, parameters any, runAt *time.Time) error {
	now := time.Now()
	return tx.Create(&IntegrationRequest{
		ID:                uuid.New(),
		AppInstallationID: a.ID,
		State:             IntegrationRequestStatePending,
		Type:              IntegrationRequestTypeInvokeAction,
		RunAt:             *runAt,
		CreatedAt:         now,
		UpdatedAt:         now,
		Spec: datatypes.NewJSONType(IntegrationRequestSpec{
			InvokeAction: &IntegrationInvokeAction{
				ActionName: actionName,
				Parameters: parameters,
			},
		}),
	}).Error
}
