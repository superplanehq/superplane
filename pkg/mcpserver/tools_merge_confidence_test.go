package mcpserver

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunMergeConfidenceToolAuthorization(t *testing.T) {
	for _, scenario := range []string{"read-only scope", "read-only role", "other workspace", "other task workspace", "factories disabled", "missing task", "outside Verify"} {
		t.Run(scenario, func(t *testing.T) {
			r := support.Setup(t)
			enableFactories(t, r.Organization.ID)
			db := database.DB(t.Context())
			factory, err := models.CreateFactory(db, r.Organization.ID, "Merge confidence", "", "MCP")
			require.NoError(t, err)
			order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
			require.NoError(t, err)
			claims := toolClaims(r, factory.ID)
			args := map[string]any{"task": order.ID.String()}
			runtime := &Runtime{Auth: r.AuthService}
			expected := "Not found"
			switch scenario {
			case "read-only scope":
				claims.Scopes = []string{"work_orders:read"}
			case "read-only role":
				reader := createHumanMember(t, r.Organization.ID, "Reader", "reader-"+uuid.NewString()+"@example.com")
				roleName := "mcp-reader-" + uuid.NewString()
				require.NoError(t, r.AuthService.CreateCustomRole(r.Organization.ID.String(), &authorization.RoleDefinition{Name: roleName, DisplayName: "Reader", DomainType: models.DomainTypeOrganization, Permissions: []*authorization.Permission{{Resource: "work_orders", Action: "read", DomainType: models.DomainTypeOrganization}}}))
				require.NoError(t, r.AuthService.AssignRole(reader.ID.String(), roleName, r.Organization.ID.String(), models.DomainTypeOrganization))
				claims.UserID = reader.ID
			case "other workspace":
				args["workspace"] = uuid.NewString()
			case "other task workspace":
				other, err := models.CreateFactory(db, r.Organization.ID, "Other", "", "OTH")
				require.NoError(t, err)
				claims = toolClaims(r, other.ID)
			case "factories disabled":
				require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
			case "missing task":
				delete(args, "task")
				expected = "task is required"
			case "outside Verify":
				expected = "The task is not in Verify."
			}
			_, err = runtime.CallTool(t.Context(), claims, "run_merge_confidence", args)
			require.Error(t, err)
			assert.Equal(t, expected, err.Error())
		})
	}
}

func TestRunMergeConfidenceDiscovery(t *testing.T) {
	response := HandleJSONRPC(t.Context(), &Runtime{}, &AccessClaims{}, &JSONRPCRequest{JSONRPC: "2.0", ID: 1, Method: "tools/list"})
	require.Nil(t, response.Error)
	for _, tool := range response.Result.(map[string]any)["tools"].([]Tool) {
		if tool.Name != "run_merge_confidence" {
			continue
		}
		assert.Equal(t, []string{"task"}, tool.InputSchema["required"])
		properties := tool.InputSchema["properties"].(map[string]any)
		assert.Contains(t, properties, "workspace")
		assert.Contains(t, properties, "task")
		return
	}
	t.Fatal("run_merge_confidence is not available in tool discovery")
}
