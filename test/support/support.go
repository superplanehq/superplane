package support

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/semaphore"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"

	// Import components and triggers to register them via init()
	_ "github.com/superplanehq/superplane/pkg/components/approval"
	_ "github.com/superplanehq/superplane/pkg/components/filter"
	_ "github.com/superplanehq/superplane/pkg/components/http"
	_ "github.com/superplanehq/superplane/pkg/components/if"
	_ "github.com/superplanehq/superplane/pkg/components/merge"
	_ "github.com/superplanehq/superplane/pkg/components/noop"
	_ "github.com/superplanehq/superplane/pkg/components/wait"
	_ "github.com/superplanehq/superplane/pkg/triggers/github"
	_ "github.com/superplanehq/superplane/pkg/triggers/schedule"
	_ "github.com/superplanehq/superplane/pkg/triggers/semaphore"
	_ "github.com/superplanehq/superplane/pkg/triggers/start"
)

type ResourceRegistry struct {
	User             uuid.UUID
	UserModel        *models.User
	Canvas           *models.Canvas
	Organization     *models.Organization
	Account          *models.Account
	Integration      *models.Integration
	Encryptor        crypto.Encryptor
	AuthService      *authorization.AuthService
	Registry         *registry.Registry
	SemaphoreAPIMock *semaphore.SemaphoreAPIMock
}

func (r *ResourceRegistry) Close() {
	if r.SemaphoreAPIMock != nil {
		r.SemaphoreAPIMock.Close()
	}
}

type SetupOptions struct {
	Source      bool
	Stage       bool
	Approvals   int
	Integration bool
}

func Setup(t *testing.T) *ResourceRegistry {
	return SetupWithOptions(t, SetupOptions{
		Source:      true,
		Stage:       true,
		Integration: true,
		Approvals:   1,
	})
}

func SetupWithOptions(t *testing.T, options SetupOptions) *ResourceRegistry {
	require.NoError(t, database.TruncateTables())

	//
	// Set up initial test resource registry
	//
	encryptor := crypto.NewNoOpEncryptor()
	r := ResourceRegistry{
		Encryptor:        encryptor,
		Registry:         registry.NewRegistry(encryptor),
		AuthService:      AuthService(t),
		SemaphoreAPIMock: semaphore.NewSemaphoreAPIMock(),
	}

	require.NoError(t, r.SemaphoreAPIMock.Init())

	//
	// Create organization and user
	//
	var err error
	organization, err := models.CreateOrganization(RandomName("org"), RandomName("org-display"))
	require.NoError(t, err)
	r.AuthService.SetupOrganizationRoles(organization.ID.String())
	require.NoError(t, err)

	account, err := models.CreateAccount("test@example.com", "test")
	require.NoError(t, err)
	user, err := models.CreateUser(organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	err = r.AuthService.AssignRole(user.ID.String(), models.RoleOrgOwner, organization.ID.String(), models.DomainTypeOrganization)
	require.NoError(t, err)

	r.Account = account
	r.User = user.ID
	r.UserModel = user
	r.Organization = organization

	//
	// Create canvas
	//
	r.Canvas = CreateCanvas(t, &r, r.Organization.ID, r.User)

	//
	// Create integration
	//
	if options.Integration {
		secret, err := CreateCanvasSecret(t, &r, map[string]string{"key": "test"})
		require.NoError(t, err)
		integration, err := models.CreateIntegration(&models.Integration{
			Name:       RandomName("integration"),
			CreatedBy:  r.User,
			Type:       models.IntegrationTypeSemaphore,
			DomainType: models.DomainTypeCanvas,
			DomainID:   r.Canvas.ID,
			URL:        r.SemaphoreAPIMock.Server.URL,
			AuthType:   models.IntegrationAuthTypeToken,
			Auth: datatypes.NewJSONType(models.IntegrationAuth{
				Token: &models.IntegrationAuthToken{
					ValueFrom: models.ValueDefinitionFrom{
						Secret: &models.ValueDefinitionFromSecret{
							DomainType: models.DomainTypeCanvas,
							Name:       secret.Name,
							Key:        "key",
						},
					},
				},
			}),
		})

		require.NoError(t, err)
		r.Integration = integration
	}

	return &r
}

func CreateCanvasSecret(t *testing.T, r *ResourceRegistry, secretData map[string]string) (*models.Secret, error) {
	data, err := json.Marshal(secretData)
	require.NoError(t, err)
	secret, err := models.CreateSecret(RandomName("secret"), secrets.ProviderLocal, r.User.String(), models.DomainTypeCanvas, r.Canvas.ID, data)
	require.NoError(t, err)
	return secret, nil
}

func CreateOrganizationSecret(t *testing.T, r *ResourceRegistry, secretData map[string]string) (*models.Secret, error) {
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

// TODO: this needs to be refactored
func CreateOrganization(t *testing.T, r *ResourceRegistry, userID uuid.UUID) *models.Organization {
	organization, err := models.CreateOrganization(RandomName("org"), RandomName("org-display"))
	require.NoError(t, err)
	r.AuthService.SetupOrganizationRoles(organization.ID.String())
	require.NoError(t, err)
	err = r.AuthService.AssignRole(userID.String(), models.RoleOrgOwner, organization.ID.String(), models.DomainTypeOrganization)
	require.NoError(t, err)
	return organization
}

func CreateCanvas(t *testing.T, r *ResourceRegistry, organizationID, userID uuid.UUID) *models.Canvas {
	canvas, err := models.CreateCanvas(userID, organizationID, RandomName("canvas"), "Test Canvas")
	require.NoError(t, err)
	err = r.AuthService.SetupCanvasRoles(canvas.ID.String())
	require.NoError(t, err)
	err = r.AuthService.AssignRole(userID.String(), models.RoleCanvasOwner, canvas.ID.String(), models.DomainTypeCanvas)
	require.NoError(t, err)
	return canvas
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

func CreateWorkflowQueueItem(t *testing.T, workflowID uuid.UUID, nodeID string, rootEventID uuid.UUID, eventID uuid.UUID) {
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

	//
	// Create workflow
	//
	workflow := &models.Workflow{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           RandomName("workflow"),
		Description:    "Test workflow",
		Edges:          datatypes.NewJSONSlice(edges),
		CreatedBy:      &userID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	require.NoError(t, database.Conn().Create(workflow).Error)

	//
	// Create workflow nodes
	//
	for _, node := range nodes {
		node.WorkflowID = workflow.ID
		node.State = models.WorkflowNodeStateReady
		node.CreatedAt = &now
		node.UpdatedAt = &now
		require.NoError(t, database.Conn().Clauses(clause.Returning{}).Create(&node).Error)
	}

	return workflow, nodes
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
