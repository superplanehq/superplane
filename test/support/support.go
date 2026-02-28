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

	// Import components, triggers, and integrations to register them via init()
	_ "github.com/superplanehq/superplane/pkg/components/approval"
	_ "github.com/superplanehq/superplane/pkg/components/deletememory"
	_ "github.com/superplanehq/superplane/pkg/components/filter"
	_ "github.com/superplanehq/superplane/pkg/components/http"
	_ "github.com/superplanehq/superplane/pkg/components/if"
	_ "github.com/superplanehq/superplane/pkg/components/merge"
	_ "github.com/superplanehq/superplane/pkg/components/noop"
	_ "github.com/superplanehq/superplane/pkg/components/readmemory"
	_ "github.com/superplanehq/superplane/pkg/components/ssh"
	_ "github.com/superplanehq/superplane/pkg/components/wait"
	_ "github.com/superplanehq/superplane/pkg/integrations/circleci"
	_ "github.com/superplanehq/superplane/pkg/integrations/github"
	_ "github.com/superplanehq/superplane/pkg/integrations/semaphore"
	_ "github.com/superplanehq/superplane/pkg/triggers/schedule"
	_ "github.com/superplanehq/superplane/pkg/triggers/start"
	_ "github.com/superplanehq/superplane/pkg/widgets/annotation"
)

type ResourceRegistry struct {
	User         uuid.UUID
	UserModel    *models.User
	Organization *models.Organization
	Account      *models.Account
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

	encryptor := crypto.NewNoOpEncryptor()
	registry, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
	require.NoError(t, err)

	//
	// Set up initial test resource registry
	//
	r := ResourceRegistry{
		Encryptor:   encryptor,
		Registry:    registry,
		AuthService: AuthService(t),
	}

	//
	// Create organization and user
	//
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

func EmitCanvasEventForNode(t *testing.T, canvasID uuid.UUID, nodeID string, channel string, executionID *uuid.UUID) *models.CanvasEvent {
	return EmitCanvasEventForNodeWithData(t, canvasID, nodeID, channel, executionID, map[string]any{"key": "value"})
}

func EmitCanvasEventForNodeWithData(
	t *testing.T,
	canvasID uuid.UUID,
	nodeID string,
	channel string,
	executionID *uuid.UUID,
	data map[string]any,
) *models.CanvasEvent {
	ensureCanvasNodeExists(t, canvasID, nodeID)

	now := time.Now()
	event := models.CanvasEvent{
		WorkflowID:  canvasID,
		NodeID:      nodeID,
		Channel:     channel,
		Data:        datatypes.NewJSONType[any](data),
		State:       models.CanvasEventStatePending,
		ExecutionID: executionID,
		CreatedAt:   &now,
	}
	require.NoError(t, database.Conn().Clauses(clause.Returning{}).Create(&event).Error)
	return &event
}

func CreateQueueItem(t *testing.T, workflowID uuid.UUID, nodeID string, rootEventID uuid.UUID, eventID uuid.UUID) *models.CanvasNodeQueueItem {
	now := time.Now()

	queueItem := models.CanvasNodeQueueItem{
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
) *models.CanvasNodeExecution {
	ensureCanvasNodeExists(t, workflowID, nodeID)

	now := time.Now()
	execution := models.CanvasNodeExecution{
		WorkflowID:        workflowID,
		NodeID:            nodeID,
		RootEventID:       rootEventID,
		EventID:           eventID,
		ParentExecutionID: parentExecutionID,
		State:             models.CanvasNodeExecutionStatePending,
		Configuration:     datatypes.NewJSONType(configuration),
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}

	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}

func CreateCanvasNodeExecution(
	t *testing.T,
	canvasID uuid.UUID,
	nodeID string,
	rootEventID uuid.UUID,
	eventID uuid.UUID,
	parentExecutionID *uuid.UUID,
) *models.CanvasNodeExecution {
	ensureCanvasNodeExists(t, canvasID, nodeID)

	now := time.Now()
	execution := models.CanvasNodeExecution{
		WorkflowID:        canvasID,
		NodeID:            nodeID,
		RootEventID:       rootEventID,
		EventID:           eventID,
		ParentExecutionID: parentExecutionID,
		State:             models.CanvasNodeExecutionStatePending,
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
) *models.CanvasNodeExecution {
	ensureCanvasNodeExists(t, workflowID, nodeID)

	now := time.Now()
	execution := models.CanvasNodeExecution{
		WorkflowID:          workflowID,
		NodeID:              nodeID,
		RootEventID:         rootEventID,
		EventID:             eventID,
		PreviousExecutionID: previous,
		State:               models.CanvasNodeExecutionStatePending,
		Configuration:       datatypes.NewJSONType(map[string]any{}),
		CreatedAt:           &now,
		UpdatedAt:           &now,
	}

	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}

func CreateCanvas(t *testing.T, orgID uuid.UUID, userID uuid.UUID, nodes []models.CanvasNode, edges []models.Edge) (*models.Canvas, []models.CanvasNode) {
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
	// Create canvas
	//
	workflow := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           RandomName("canvas"),
		Description:    "Test canvas",
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

	var createdNodes []models.CanvasNode
	for _, node := range expandedNodes {
		var parentNodeID *string
		if idx := strings.Index(node.ID, ":"); idx != -1 {
			parent := node.ID[:idx]
			parentNodeID = &parent
		}

		canvasNode := models.CanvasNode{
			WorkflowID:    workflow.ID,
			NodeID:        node.ID,
			ParentNodeID:  parentNodeID,
			Name:          node.Name,
			State:         models.CanvasNodeStateReady,
			Type:          node.Type,
			Ref:           datatypes.NewJSONType(node.Ref),
			Configuration: datatypes.NewJSONType(node.Configuration),
			Position:      datatypes.NewJSONType(node.Position),
			Metadata:      datatypes.NewJSONType(node.Metadata),
			IsCollapsed:   node.IsCollapsed,
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}

		require.NoError(t, database.Conn().Clauses(clause.Returning{}).Create(&canvasNode).Error)
		createdNodes = append(createdNodes, canvasNode)
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

func VerifyCanvasEventsCount(t *testing.T, canvasID uuid.UUID, expected int) {
	var actual int64

	err := database.Conn().
		Model(&models.CanvasEvent{}).
		Where("workflow_id = ?", canvasID).
		Count(&actual).
		Error

	require.NoError(t, err)
	require.Equal(t, expected, int(actual))
}

func VerifyCanvasNodeEventsCount(t *testing.T, canvasID uuid.UUID, nodeID string, expected int) {
	var actual int64

	err := database.Conn().
		Model(&models.CanvasEvent{}).
		Where("workflow_id = ?", canvasID).
		Where("node_id = ?", nodeID).
		Count(&actual).
		Error

	require.NoError(t, err)
	require.Equal(t, expected, int(actual))
}

func VerifyNodeExecutionsCount(t *testing.T, workflowID uuid.UUID, expected int) {
	var actual int64

	err := database.Conn().
		Model(&models.CanvasNodeExecution{}).
		Where("workflow_id = ?", workflowID).
		Count(&actual).
		Error

	require.NoError(t, err)
	require.Equal(t, expected, int(actual))
}

func VerifyNodeQueueCount(t *testing.T, workflowID uuid.UUID, expected int) {
	var actual int64

	err := database.Conn().
		Model(&models.CanvasNodeQueueItem{}).
		Where("workflow_id = ?", workflowID).
		Count(&actual).
		Error

	require.NoError(t, err)
	require.Equal(t, expected, int(actual))
}

func VerifyNodeExecutionKVCount(t *testing.T, workflowID uuid.UUID, expected int) {
	var actual int64

	err := database.Conn().
		Model(&models.CanvasNodeExecutionKV{}).
		Where("workflow_id = ?", workflowID).
		Count(&actual).
		Error

	require.NoError(t, err)
	require.Equal(t, expected, int(actual))
}

func VerifyNodeRequestCount(t *testing.T, workflowID uuid.UUID, expected int) {
	var actual int64

	err := database.Conn().
		Model(&models.CanvasNodeRequest{}).
		Where("workflow_id = ?", workflowID).
		Count(&actual).
		Error

	require.NoError(t, err)
	require.Equal(t, expected, int(actual))
}

func ensureCanvasNodeExists(t *testing.T, workflowID uuid.UUID, nodeID string) {
	var existingNode models.CanvasNode
	err := database.Conn().
		Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).
		First(&existingNode).Error

	if err == nil {
		return
	}

	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	now := time.Now()
	node := models.CanvasNode{
		WorkflowID:    workflowID,
		NodeID:        nodeID,
		Name:          "Auto-created node for test",
		State:         models.CanvasNodeStateReady,
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
