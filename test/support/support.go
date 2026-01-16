package support

import (
	"encoding/json"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/secrets"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	// Import components, triggers, and applications to register them via init()
	_ "github.com/superplanehq/superplane/pkg/applications/github"
	_ "github.com/superplanehq/superplane/pkg/applications/semaphore"
	_ "github.com/superplanehq/superplane/pkg/components/approval"
	_ "github.com/superplanehq/superplane/pkg/components/filter"
	_ "github.com/superplanehq/superplane/pkg/components/http"
	_ "github.com/superplanehq/superplane/pkg/components/if"
	_ "github.com/superplanehq/superplane/pkg/components/merge"
	_ "github.com/superplanehq/superplane/pkg/components/noop"
	_ "github.com/superplanehq/superplane/pkg/components/wait"
	_ "github.com/superplanehq/superplane/pkg/triggers/schedule"
	_ "github.com/superplanehq/superplane/pkg/triggers/start"
	_ "github.com/superplanehq/superplane/pkg/widgets/annotation"
)

type ResourceRegistry struct {
	User         uuid.UUID
	UserModel    *models.User
	Organization *models.Organization
	Account      *models.Account
	Integration  *models.Integration
	Encryptor    crypto.Encryptor
	AuthService  *authorization.AuthService
	Registry     *registry.Registry
}

func (r *ResourceRegistry) Close() {}

type SetupOptions struct {
	Source    bool
	Stage     bool
	Approvals int
}

func Setup(t *testing.T) *ResourceRegistry {
	return SetupWithOptions(t, SetupOptions{
		Source:    true,
		Stage:     true,
		Approvals: 1,
	})
}

func SetupWithOptions(t *testing.T, options SetupOptions) *ResourceRegistry {
	require.NoError(t, database.TruncateTables())

	//
	// Set up initial test resource registry
	//
	encryptor := crypto.NewNoOpEncryptor()
	r := ResourceRegistry{
		Encryptor:   encryptor,
		Registry:    registry.NewRegistry(encryptor),
		AuthService: AuthService(t),
	}

	//
	// Create organization and user
	//
	var err error
	tx := database.Conn().Begin()
	organization, err := models.CreateOrganizationInTransaction(tx, RandomName("org"), RandomName("org-display"))
	if !assert.NoError(t, err) {
		tx.Rollback()
		t.FailNow()
	}

	account, err := models.CreateAccountInTransaction(tx, "test@example.com", "test")
	if !assert.NoError(t, err) {
		tx.Rollback()
		t.FailNow()
	}

	user, err := models.CreateUserInTransaction(tx, organization.ID, account.ID, account.Email, account.Name)
	if !assert.NoError(t, err) {
		tx.Rollback()
		t.FailNow()
	}

	err = r.AuthService.SetupOrganization(tx, organization.ID.String(), user.ID.String())
	if !assert.NoError(t, err) {
		tx.Rollback()
		t.FailNow()
	}

	err = tx.Commit().Error
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	r.Account = account
	r.User = user.ID
	r.UserModel = user
	r.Organization = organization

	return &r
}

func CreateSecret(t *testing.T, r *ResourceRegistry, secretData map[string]string) (*models.Secret, error) {
	data, err := json.Marshal(secretData)
	require.NoError(t, err)
	secret, err := models.CreateSecret(RandomName("secret"), secrets.ProviderLocal, r.User.String(), models.DomainTypeOrganization, r.Organization.ID, data)
	require.NoError(t, err)
	return secret, nil
}

func RandomName(prefix string) string {
	return prefix + "-" + uuid.New().String()
}

func AuthService(t *testing.T) *authorization.AuthService {
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	return authService
}

func CreateOrganization(t *testing.T, r *ResourceRegistry, userID uuid.UUID) *models.Organization {
	tx := database.Conn().Begin()
	organization, err := models.CreateOrganizationInTransaction(tx, RandomName("org"), RandomName("org-display"))
	if !assert.NoError(t, err) {
		tx.Rollback()
		t.FailNow()
	}

	err = r.AuthService.SetupOrganization(tx, organization.ID.String(), userID.String())
	if !assert.NoError(t, err) {
		tx.Rollback()
		t.FailNow()
	}

	err = tx.Commit().Error
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	return organization
}

func CreateUser(t *testing.T, r *ResourceRegistry, organizationID uuid.UUID) *models.User {
	name := RandomName("user")
	account, err := models.CreateAccount(name, name+"@test.com")
	require.NoError(t, err)
	user, err := models.CreateUser(organizationID, account.ID, account.Name, account.Email)
	require.NoError(t, err)
	err = r.AuthService.AssignRole(user.ID.String(), models.RoleOrgViewer, organizationID.String(), models.DomainTypeOrganization)
	require.NoError(t, err)
	return user
}

func EmitWorkflowEventForNode(t *testing.T, workflowID uuid.UUID, nodeID string, channel string, executionID *uuid.UUID) *models.WorkflowEvent {
	return EmitWorkflowEventForNodeWithData(t, workflowID, nodeID, channel, executionID, map[string]any{"key": "value"})
}

func EmitWorkflowEventForNodeWithData(
	t *testing.T,
	workflowID uuid.UUID,
	nodeID string,
	channel string,
	executionID *uuid.UUID,
	data map[string]any,
) *models.WorkflowEvent {
	ensureWorkflowNodeExists(t, workflowID, nodeID)

	now := time.Now()
	event := models.WorkflowEvent{
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		Channel:     channel,
		Data:        datatypes.NewJSONType[any](data),
		State:       models.WorkflowEventStatePending,
		ExecutionID: executionID,
		CreatedAt:   &now,
	}
	require.NoError(t, database.Conn().Clauses(clause.Returning{}).Create(&event).Error)
	return &event
}

func CreateWorkflowQueueItem(t *testing.T, workflowID uuid.UUID, nodeID string, rootEventID uuid.UUID, eventID uuid.UUID) *models.WorkflowNodeQueueItem {
	now := time.Now()

	queueItem := models.WorkflowNodeQueueItem{
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		RootEventID: rootEventID,
		EventID:     eventID,
		CreatedAt:   &now,
	}

	err := database.Conn().Create(&queueItem).Error
	require.NoError(t, err)
	return &queueItem
}

func CreateNodeExecutionWithConfiguration(
	t *testing.T,
	workflowID uuid.UUID,
	nodeID string,
	rootEventID uuid.UUID,
	eventID uuid.UUID,
	parentExecutionID *uuid.UUID,
	configuration map[string]any,
) *models.WorkflowNodeExecution {
	ensureWorkflowNodeExists(t, workflowID, nodeID)

	now := time.Now()
	execution := models.WorkflowNodeExecution{
		WorkflowID:        workflowID,
		NodeID:            nodeID,
		RootEventID:       rootEventID,
		EventID:           eventID,
		ParentExecutionID: parentExecutionID,
		State:             models.WorkflowNodeExecutionStatePending,
		Configuration:     datatypes.NewJSONType(configuration),
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}

	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}

func CreateWorkflowNodeExecution(
	t *testing.T,
	workflowID uuid.UUID,
	nodeID string,
	rootEventID uuid.UUID,
	eventID uuid.UUID,
	parentExecutionID *uuid.UUID,
) *models.WorkflowNodeExecution {
	ensureWorkflowNodeExists(t, workflowID, nodeID)

	now := time.Now()
	execution := models.WorkflowNodeExecution{
		WorkflowID:        workflowID,
		NodeID:            nodeID,
		RootEventID:       rootEventID,
		EventID:           eventID,
		ParentExecutionID: parentExecutionID,
		State:             models.WorkflowNodeExecutionStatePending,
		Configuration:     datatypes.NewJSONType(map[string]any{}),
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}

	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}

func CreateNextNodeExecution(
	t *testing.T,
	workflowID uuid.UUID,
	nodeID string,
	rootEventID uuid.UUID,
	eventID uuid.UUID,
	previous *uuid.UUID,
) *models.WorkflowNodeExecution {
	ensureWorkflowNodeExists(t, workflowID, nodeID)

	now := time.Now()
	execution := models.WorkflowNodeExecution{
		WorkflowID:          workflowID,
		NodeID:              nodeID,
		RootEventID:         rootEventID,
		EventID:             eventID,
		PreviousExecutionID: previous,
		State:               models.WorkflowNodeExecutionStatePending,
		Configuration:       datatypes.NewJSONType(map[string]any{}),
		CreatedAt:           &now,
		UpdatedAt:           &now,
	}

	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}

func CreateWorkflow(t *testing.T, orgID uuid.UUID, userID uuid.UUID, nodes []models.WorkflowNode, edges []models.Edge) (*models.Workflow, []models.WorkflowNode) {
	now := time.Now()

	inputNodes := make([]models.Node, len(nodes))
	for i, node := range nodes {
		inputNodes[i] = models.Node{
			ID:            node.NodeID,
			Name:          node.Name,
			Type:          node.Type,
			Ref:           node.Ref.Data(),
			Configuration: node.Configuration.Data(),
			Metadata:      node.Metadata.Data(),
			Position:      node.Position.Data(),
			IsCollapsed:   node.IsCollapsed,
		}
	}

	//
	// Create workflow
	//
	workflow := &models.Workflow{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           RandomName("workflow"),
		Description:    "Test workflow",
		Nodes:          datatypes.NewJSONSlice(inputNodes),
		Edges:          datatypes.NewJSONSlice(edges),
		CreatedBy:      &userID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	require.NoError(t, database.Conn().Create(workflow).Error)

	//
	// Expand blueprint nodes (convert WorkflowNode to Node, expand, then back to WorkflowNode)
	//
	expandedNodes, err := expandBlueprintNodes(t, orgID, inputNodes)
	require.NoError(t, err)

	var createdNodes []models.WorkflowNode
	for _, node := range expandedNodes {
		var parentNodeID *string
		if idx := strings.Index(node.ID, ":"); idx != -1 {
			parent := node.ID[:idx]
			parentNodeID = &parent
		}

		workflowNode := models.WorkflowNode{
			WorkflowID:    workflow.ID,
			NodeID:        node.ID,
			ParentNodeID:  parentNodeID,
			Name:          node.Name,
			State:         models.WorkflowNodeStateReady,
			Type:          node.Type,
			Ref:           datatypes.NewJSONType(node.Ref),
			Configuration: datatypes.NewJSONType(node.Configuration),
			Position:      datatypes.NewJSONType(node.Position),
			Metadata:      datatypes.NewJSONType(node.Metadata),
			IsCollapsed:   node.IsCollapsed,
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}

		require.NoError(t, database.Conn().Clauses(clause.Returning{}).Create(&workflowNode).Error)
		createdNodes = append(createdNodes, workflowNode)
	}

	return workflow, createdNodes
}

func CreateBlueprint(t *testing.T, orgID uuid.UUID, nodes []models.Node, edges []models.Edge, outputChannels []models.BlueprintOutputChannel) *models.Blueprint {
	now := time.Now()

	blueprint := models.Blueprint{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           RandomName("blueprint"),
		Nodes:          datatypes.NewJSONSlice(nodes),
		Edges:          datatypes.NewJSONSlice(edges),
		OutputChannels: datatypes.NewJSONSlice(outputChannels),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	require.NoError(t, database.Conn().Create(&blueprint).Error)

	return &blueprint
}

func VerifyWorkflowEventsCount(t *testing.T, workflowID uuid.UUID, expected int) {
	var actual int64

	err := database.Conn().
		Model(&models.WorkflowEvent{}).
		Where("workflow_id = ?", workflowID).
		Count(&actual).
		Error

	require.NoError(t, err)
	require.Equal(t, expected, int(actual))
}

func VerifyWorkflowNodeExecutionsCount(t *testing.T, workflowID uuid.UUID, expected int) {
	var actual int64

	err := database.Conn().
		Model(&models.WorkflowNodeExecution{}).
		Where("workflow_id = ?", workflowID).
		Count(&actual).
		Error

	require.NoError(t, err)
	require.Equal(t, expected, int(actual))
}

func VerifyWorkflowNodeQueueCount(t *testing.T, workflowID uuid.UUID, expected int) {
	var actual int64

	err := database.Conn().
		Model(&models.WorkflowNodeQueueItem{}).
		Where("workflow_id = ?", workflowID).
		Count(&actual).
		Error

	require.NoError(t, err)
	require.Equal(t, expected, int(actual))
}

func VerifyWorkflowNodeExecutionKVCount(t *testing.T, workflowID uuid.UUID, expected int) {
	var actual int64

	err := database.Conn().
		Model(&models.WorkflowNodeExecutionKV{}).
		Where("workflow_id = ?", workflowID).
		Count(&actual).
		Error

	require.NoError(t, err)
	require.Equal(t, expected, int(actual))
}

func VerifyWorkflowNodeRequestCount(t *testing.T, workflowID uuid.UUID, expected int) {
	var actual int64

	err := database.Conn().
		Model(&models.WorkflowNodeRequest{}).
		Where("workflow_id = ?", workflowID).
		Count(&actual).
		Error

	require.NoError(t, err)
	require.Equal(t, expected, int(actual))
}

// ensureWorkflowNodeExists creates a minimal workflow node if it doesn't exist
// This is needed to satisfy FK constraints when creating events/executions
func ensureWorkflowNodeExists(t *testing.T, workflowID uuid.UUID, nodeID string) {
	var existingNode models.WorkflowNode
	err := database.Conn().
		Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).
		First(&existingNode).Error

	if err == nil {
		return
	}

	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	now := time.Now()
	node := models.WorkflowNode{
		WorkflowID:    workflowID,
		NodeID:        nodeID,
		Name:          "Auto-created node for test",
		State:         models.WorkflowNodeStateReady,
		Type:          models.NodeTypeComponent,
		Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
		Configuration: datatypes.NewJSONType(map[string]any{}),
		Position:      datatypes.NewJSONType(models.Position{}),
		Metadata:      datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	require.NoError(t, database.Conn().Create(&node).Error)
}

func expandBlueprintNodes(t *testing.T, orgID uuid.UUID, nodes []models.Node) ([]models.Node, error) {
	expanded := make([]models.Node, 0, len(nodes))

	for _, n := range nodes {
		expanded = append(expanded, n)

		if n.Type != models.NodeTypeBlueprint || n.Ref.Blueprint == nil {
			continue
		}

		blueprintID := n.Ref.Blueprint.ID
		if blueprintID == "" {
			continue
		}

		b, err := models.FindBlueprint(orgID.String(), blueprintID)
		if err != nil {
			continue
		}

		for _, bn := range b.Nodes {
			internal := models.Node{
				ID:            n.ID + ":" + bn.ID,
				Name:          bn.Name,
				Type:          bn.Type,
				Ref:           bn.Ref,
				Configuration: bn.Configuration,
				Metadata:      maps.Clone(bn.Metadata),
				Position:      bn.Position,
				IsCollapsed:   bn.IsCollapsed,
			}

			expanded = append(expanded, internal)
		}
	}

	return expanded, nil
}
