package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	grpcCanvases "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/usage"
	"github.com/superplanehq/superplane/test/support"
)

type rejectingCanvasUsageService struct{}

func (rejectingCanvasUsageService) Enabled() bool { return true }
func (rejectingCanvasUsageService) SetupAccount(context.Context, string) (*usagepb.SetupAccountResponse, error) {
	return nil, nil
}
func (rejectingCanvasUsageService) SetupOrganization(context.Context, string, string, usage.SetupOrganizationDetails) (*usagepb.SetupOrganizationResponse, error) {
	return nil, nil
}
func (rejectingCanvasUsageService) DescribeAccountLimits(context.Context, string) (*usagepb.DescribeAccountLimitsResponse, error) {
	return nil, nil
}
func (rejectingCanvasUsageService) DescribeOrganizationLimits(context.Context, string) (*usagepb.DescribeOrganizationLimitsResponse, error) {
	return nil, nil
}
func (rejectingCanvasUsageService) DescribeOrganizationUsage(context.Context, string) (*usagepb.DescribeOrganizationUsageResponse, error) {
	return nil, nil
}
func (rejectingCanvasUsageService) CheckAccountLimits(context.Context, string, *usagepb.AccountState) (*usagepb.CheckAccountLimitsResponse, error) {
	return &usagepb.CheckAccountLimitsResponse{Allowed: true}, nil
}
func (rejectingCanvasUsageService) CheckOrganizationLimits(context.Context, string, *usagepb.OrganizationState, *usagepb.CanvasState) (*usagepb.CheckOrganizationLimitsResponse, error) {
	return &usagepb.CheckOrganizationLimitsResponse{
		Allowed: false,
		Violations: []*usagepb.LimitViolation{
			{
				Limit:           usagepb.LimitName_LIMIT_NAME_MAX_NODES_PER_CANVAS,
				ConfiguredLimit: 1,
				CurrentValue:    2,
			},
		},
	}, nil
}

func TestResolveCustomToolAutoLayout_DefaultsGraphUpdatesToFullCanvas(t *testing.T) {
	layout := resolveCustomToolAutoLayout(nil, true)

	require.NotNil(t, layout)
	assert.Equal(t, pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL, layout.Algorithm)
	assert.Equal(t, pb.CanvasAutoLayout_SCOPE_FULL_CANVAS, layout.Scope)
	assert.Empty(t, layout.NodeIds)
}

func TestResolveCustomToolAutoLayout_SkipsConsoleOnlyUpdates(t *testing.T) {
	assert.Nil(t, resolveCustomToolAutoLayout(nil, false))
}

func TestResolveCustomToolAutoLayout_PreservesExplicitSettings(t *testing.T) {
	layout := resolveCustomToolAutoLayout(&superPlaneCanvasAutoLayoutInput{
		Scope:   "connected_component",
		NodeIDs: []string{"node-1"},
	}, true)

	require.NotNil(t, layout)
	assert.Equal(t, pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL, layout.Algorithm)
	assert.Equal(t, pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT, layout.Scope)
	assert.Equal(t, []string{"node-1"}, layout.NodeIds)
}

func TestSummarizeNodes_UsesYamlComponentFieldName(t *testing.T) {
	summary := summarizeNodes([]models.Node{
		{
			ID:   "node-1",
			Name: "Notify",
			Type: "TYPE_ACTION",
			Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "slack.sendTextMessage"}},
		},
	}, 20)

	require.Len(t, summary, 1)
	assert.Equal(t, "slack.sendTextMessage", summary[0].Component)

	data, err := json.Marshal(summary[0])
	require.NoError(t, err)
	assert.Contains(t, string(data), `"component":"slack.sendTextMessage"`)
	assert.NotContains(t, string(data), `"ref"`)
}

func TestSuperPlaneCanvasTool_UpdateDraftEnforcesUsageLimits(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	canvasYAML, err := grpcCanvases.ReadRepositorySpecFile(
		context.Background(),
		r.Organization.ID.String(),
		canvas.ID.String(),
		"",
		grpcCanvases.CanvasYAMLRepositoryPath,
	)
	require.NoError(t, err)

	input, err := json.Marshal(map[string]string{
		"action":      "update_draft",
		"canvas_yaml": canvasYAML,
	})
	require.NoError(t, err)

	tool := NewSuperPlaneCanvasTool(SuperPlaneCanvasToolOptions{
		Encryptor:      r.Encryptor,
		Registry:       r.Registry,
		AuthService:    r.AuthService,
		UsageService:   rejectingCanvasUsageService{},
		WebhookBaseURL: "https://hooks.example.test",
	})

	result := tool.ExecuteCustomTool(context.Background(), agents.AgentSessionContext{
		SessionID:      "session-1",
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		CanvasID:       canvas.ID.String(),
	}, agents.CustomToolUse{
		ID:    "tool-1",
		Name:  SuperPlaneCanvasToolName,
		Input: string(input),
	})

	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "canvas node limit exceeded")
}
