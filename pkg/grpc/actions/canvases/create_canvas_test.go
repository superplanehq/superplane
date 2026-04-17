package canvases

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	githubintegration "github.com/superplanehq/superplane/pkg/integrations/github"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/usage"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"

	_ "github.com/superplanehq/superplane/pkg/triggers/webhook"
)

type fakeCanvasUsageService struct {
	checkOrganizationResp *usagepb.CheckOrganizationLimitsResponse
}

func (s *fakeCanvasUsageService) Enabled() bool {
	return true
}

func (s *fakeCanvasUsageService) SetupAccount(context.Context, string) (*usagepb.SetupAccountResponse, error) {
	return &usagepb.SetupAccountResponse{}, nil
}

func (s *fakeCanvasUsageService) SetupOrganization(context.Context, string, string) (*usagepb.SetupOrganizationResponse, error) {
	return &usagepb.SetupOrganizationResponse{}, nil
}

func (s *fakeCanvasUsageService) DescribeAccountLimits(context.Context, string) (*usagepb.DescribeAccountLimitsResponse, error) {
	return &usagepb.DescribeAccountLimitsResponse{}, nil
}

func (s *fakeCanvasUsageService) DescribeOrganizationLimits(context.Context, string) (*usagepb.DescribeOrganizationLimitsResponse, error) {
	return &usagepb.DescribeOrganizationLimitsResponse{}, nil
}

func (s *fakeCanvasUsageService) DescribeOrganizationUsage(context.Context, string) (*usagepb.DescribeOrganizationUsageResponse, error) {
	return &usagepb.DescribeOrganizationUsageResponse{}, nil
}

func (s *fakeCanvasUsageService) CheckAccountLimits(
	context.Context,
	string,
	*usagepb.AccountState,
) (*usagepb.CheckAccountLimitsResponse, error) {
	return &usagepb.CheckAccountLimitsResponse{Allowed: true}, nil
}

func (s *fakeCanvasUsageService) CheckOrganizationLimits(
	context.Context,
	string,
	*usagepb.OrganizationState,
	*usagepb.CanvasState,
) (*usagepb.CheckOrganizationLimitsResponse, error) {
	if s.checkOrganizationResp != nil {
		return s.checkOrganizationResp, nil
	}

	return &usagepb.CheckOrganizationLimitsResponse{Allowed: true}, nil
}

var _ usage.Service = (*fakeCanvasUsageService)(nil)

func createGitHubIntegrationSecret(
	t *testing.T,
	r *support.ResourceRegistry,
	integrationID uuid.UUID,
) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	secretValue := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	require.NotEmpty(t, secretValue)

	encryptedValue, err := r.Encryptor.Encrypt(
		context.Background(),
		secretValue,
		[]byte(integrationID.String()),
	)
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, database.Conn().Create(&models.IntegrationSecret{
		OrganizationID: r.Organization.ID,
		InstallationID: integrationID,
		Name:           githubintegration.GitHubAppPEM,
		Value:          encryptedValue,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}).Error)
}

func TestCreateCanvasDuplicateName(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	workflow := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: "Duplicate Canvas",
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), workflow)
	require.NoError(t, err)

	_, err = CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), workflow)
	require.Error(t, err)
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestCreateCanvasInheritsOrganizationChangeManagementWhenEnabled(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	nowEnabled := true
	require.NoError(t, database.Conn().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("change_management_enabled", nowEnabled).Error)

	workflow := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: "Change management default canvas",
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}

	response, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), workflow)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Canvas)
	require.NotNil(t, response.Canvas.Metadata)
	// New canvases inherit organization change management setting.
	require.NotNil(t, response.Canvas.Spec)
	require.NotNil(t, response.Canvas.Spec.ChangeManagement)
	require.True(t, response.Canvas.Spec.ChangeManagement.Enabled)

	require.NotEmpty(t, response.Canvas.Metadata.Id)
	createdCanvasUUID, parseErr := uuid.Parse(response.Canvas.Metadata.Id)
	require.NoError(t, parseErr)
	createdCanvas, findErr := models.FindCanvas(r.Organization.ID, createdCanvasUUID)
	require.NoError(t, findErr)
	require.True(t, createdCanvas.ChangeManagementEnabled)
}

func TestCreateCanvasOnFreshOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        "Health Check Monitor",
			Description: "Quick start canvas on a fresh organization",
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}

	response, err := CreateCanvas(ctx, r.Registry, r.Organization.ID.String(), canvas)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Canvas)
	require.NotNil(t, response.Canvas.Metadata)
	require.Equal(t, "Health Check Monitor", response.Canvas.Metadata.Name)
	require.Equal(t, r.Organization.ID.String(), response.Canvas.Metadata.OrganizationId)
	require.NotEmpty(t, response.Canvas.Metadata.Id)

	canvasID, err := uuid.Parse(response.Canvas.Metadata.Id)
	require.NoError(t, err)
	persisted, err := models.FindCanvas(r.Organization.ID, canvasID)
	require.NoError(t, err)
	require.Equal(t, "Health Check Monitor", persisted.Name)
	require.Equal(t, r.Organization.ID, persisted.OrganizationID)
}

func TestCreateCanvasWithUsageRejectsLimitViolation(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	workflow := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: "Limited Canvas",
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}

	service := &fakeCanvasUsageService{
		checkOrganizationResp: &usagepb.CheckOrganizationLimitsResponse{
			Allowed: false,
			Violations: []*usagepb.LimitViolation{
				{
					Limit:           usagepb.LimitName_LIMIT_NAME_MAX_CANVASES,
					ConfiguredLimit: 1,
					CurrentValue:    2,
				},
			},
		},
	}

	_, err := CreateCanvasWithAutoLayoutAndUsage(ctx, service, r.Registry, r.Organization.ID.String(), workflow, nil)
	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.Equal(t, "organization canvas limit exceeded", status.Convert(err).Message())
}

func TestCreateCanvasTemplateSkipsSetupValidationForOrgSpecificIntegrationNodes(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		support.RandomName("integration"),
		nil,
	)
	require.NoError(t, err)

	integration.State = models.IntegrationStateReady
	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"installationId": "12345",
		"owner":          "testhq",
		"githubApp": map[string]any{
			"id":       1,
			"slug":     "test-app",
			"clientId": "client-id",
		},
		"repositories": []map[string]any{
			{
				"id":   123456,
				"name": "hello",
				"url":  "https://github.com/testhq/hello",
			},
		},
	})
	require.NoError(t, database.Conn().Save(integration).Error)
	createGitHubIntegrationSecret(t, r, integration.ID)

	canvas := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:       "Template without setup validation",
			IsTemplate: true,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "node-a",
					Name: "Get issue",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "github.getIssue",
					},
					Integration: &componentpb.IntegrationRef{
						Id: integration.ID.String(),
					},
					Configuration: mustStruct(t, map[string]any{
						"repository":  "world",
						"issueNumber": "42",
					}),
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	response, err := CreateCanvasWithAutoLayoutAndUsageAndSetup(
		ctx,
		nil,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas,
		nil,
		"",
		r.AuthService,
	)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Canvas)
	require.NotNil(t, response.Canvas.Metadata)
	require.NotNil(t, response.Canvas.Spec)
	require.Equal(t, models.TemplateOrganizationID.String(), response.Canvas.Metadata.OrganizationId)
	require.True(t, response.Canvas.Metadata.IsTemplate)
	require.Len(t, response.Canvas.Spec.Nodes, 1)
	require.Empty(t, response.Canvas.Spec.Nodes[0].GetErrorMessage())

	canvasID, err := uuid.Parse(response.Canvas.Metadata.Id)
	require.NoError(t, err)

	persistedCanvas, err := models.FindCanvas(models.TemplateOrganizationID, canvasID)
	require.NoError(t, err)
	require.True(t, persistedCanvas.IsTemplate)

	node, err := models.FindCanvasNode(database.Conn(), canvasID, "node-a")
	require.NoError(t, err)
	require.Equal(t, models.CanvasNodeStateReady, node.State)
	require.Nil(t, node.StateReason)
	require.Nil(t, node.AppInstallationID)
}

func TestCreateCanvasTemplateExpandsBlueprintsUsingCreatorOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	blueprint := support.CreateBlueprint(
		t,
		r.Organization.ID,
		[]models.Node{
			{
				ID:   "inner",
				Name: "Inner noop",
				Type: models.NodeTypeComponent,
				Ref: models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				},
				Configuration: map[string]any{},
				Metadata:      map[string]any{},
			},
		},
		[]models.Edge{},
		nil,
	)

	canvas := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:       "Template with org blueprint",
			IsTemplate: true,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "node-a",
					Name: "Org blueprint",
					Type: componentpb.Node_TYPE_BLUEPRINT,
					Blueprint: &componentpb.Node_BlueprintRef{
						Id: blueprint.ID.String(),
					},
					Configuration: mustStruct(t, map[string]any{}),
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	response, err := CreateCanvasWithAutoLayoutAndUsageAndSetup(
		ctx,
		nil,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas,
		nil,
		"",
		r.AuthService,
	)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Canvas)
	require.NotNil(t, response.Canvas.Spec)
	require.Len(t, response.Canvas.Spec.Nodes, 2)

	nodeIDs := map[string]bool{}
	for _, node := range response.Canvas.Spec.Nodes {
		nodeIDs[node.Id] = true
	}
	require.True(t, nodeIDs["node-a"])
	require.True(t, nodeIDs["node-a:inner"])

	canvasID, err := uuid.Parse(response.Canvas.Metadata.Id)
	require.NoError(t, err)

	internalNode, err := models.FindCanvasNode(database.Conn(), canvasID, "node-a:inner")
	require.NoError(t, err)
	require.Equal(t, "Inner noop", internalNode.Name)
}

func TestCreateCanvasSkipsRuntimeSetupForNonTemplateNodes(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: "Canvas without runtime setup",
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "node-a",
					Name: "Webhook trigger",
					Type: componentpb.Node_TYPE_TRIGGER,
					Trigger: &componentpb.Node_TriggerRef{
						Name: "webhook",
					},
					Configuration: mustStruct(t, map[string]any{
						"authentication": "none",
					}),
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	response, err := CreateCanvasWithAutoLayoutAndUsageAndSetup(
		ctx,
		nil,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas,
		nil,
		"",
		r.AuthService,
	)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Canvas)
	require.NotNil(t, response.Canvas.Metadata)
	require.Equal(t, r.Organization.ID.String(), response.Canvas.Metadata.OrganizationId)
	require.False(t, response.Canvas.Metadata.IsTemplate)

	canvasID, err := uuid.Parse(response.Canvas.Metadata.Id)
	require.NoError(t, err)

	node, err := models.FindCanvasNode(database.Conn(), canvasID, "node-a")
	require.NoError(t, err)
	require.Equal(t, models.CanvasNodeStateReady, node.State)
	require.Nil(t, node.StateReason)
	require.Nil(t, node.WebhookID)
	require.Empty(t, node.Metadata.Data())

	webhooks, err := models.ListPendingWebhooks()
	require.NoError(t, err)
	require.Empty(t, webhooks)
}

func TestCreateCanvasTemplateAutoLayoutReturnsInvalidArgument(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:       "Template invalid auto layout",
			IsTemplate: true,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err := CreateCanvasWithAutoLayoutAndUsageAndSetup(
		ctx,
		nil,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas,
		&pb.CanvasAutoLayout{
			Algorithm: pb.CanvasAutoLayout_ALGORITHM_UNSPECIFIED,
		},
		"",
		r.AuthService,
	)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), "failed to apply layout")
}

func TestTemplateCanvasAutoLayoutErrorUnwrapMatchesSentinelAndCause(t *testing.T) {
	cause := errors.New("layout failed")
	err := &templateCanvasAutoLayoutError{cause: cause}

	require.True(t, errors.Is(err, errTemplateCanvasAutoLayout))
	require.True(t, errors.Is(err, cause))
}

func mustStruct(t *testing.T, value map[string]any) *structpb.Struct {
	t.Helper()

	result, err := structpb.NewStruct(value)
	require.NoError(t, err)
	return result
}
