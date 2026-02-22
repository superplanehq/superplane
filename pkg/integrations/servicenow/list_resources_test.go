package servicenow

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ServiceNow__ListResources(t *testing.T) {
	s := &ServiceNow{}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"instanceUrl":  "https://dev12345.service-now.com",
			"clientId":     "client-123",
			"clientSecret": "secret-123",
		},
		Secrets: map[string]core.IntegrationSecret{
			OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("access-token-123")},
		},
	}

	t.Run("returns empty list for unknown resource type", func(t *testing.T) {
		resources, err := s.ListResources("unknown", core.ListResourcesContext{
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("returns list of users", func(t *testing.T) {
		ctx := core.ListResourcesContext{
			Integration: integrationCtx,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"result": [
								{"sys_id": "user1", "name": "John Smith", "email": "john@example.com"},
								{"sys_id": "user2", "name": "Jane Doe", "email": ""}
							]
						}`)),
					},
				},
			},
		}

		resources, err := s.ListResources("user", ctx)

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "user1", resources[0].ID)
		assert.Equal(t, "John Smith (john@example.com)", resources[0].Name)
		assert.Equal(t, "user", resources[0].Type)
		assert.Equal(t, "user2", resources[1].ID)
		assert.Equal(t, "Jane Doe", resources[1].Name)
	})

	t.Run("returns list of categories from metadata", func(t *testing.T) {
		ctx := core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{
				Metadata: Metadata{
					Categories: []ChoiceRecord{
						{Label: "Software", Value: "software"},
						{Label: "Hardware", Value: "hardware"},
					},
				},
			},
		}

		resources, err := s.ListResources("category", ctx)

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "software", resources[0].ID)
		assert.Equal(t, "Software", resources[0].Name)
		assert.Equal(t, "category", resources[0].Type)
		assert.Equal(t, "hardware", resources[1].ID)
		assert.Equal(t, "Hardware", resources[1].Name)
	})

	t.Run("returns list of assignment groups from metadata", func(t *testing.T) {
		ctx := core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{
				Metadata: Metadata{
					AssignmentGroups: []AssignmentGroupRecord{
						{SysID: "grp1", Name: "Network"},
						{SysID: "grp2", Name: "Database"},
					},
				},
			},
		}

		resources, err := s.ListResources("assignment_group", ctx)

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "grp1", resources[0].ID)
		assert.Equal(t, "Network", resources[0].Name)
		assert.Equal(t, "assignment_group", resources[0].Type)
		assert.Equal(t, "grp2", resources[1].ID)
		assert.Equal(t, "Database", resources[1].Name)
	})

	t.Run("returns users filtered by assignment group", func(t *testing.T) {
		ctx := core.ListResourcesContext{
			Integration: integrationCtx,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					// First call: ListGroupMembers (sys_user_grmember)
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"result": [
								{"user": {"value": "user1"}},
								{"user": {"value": "user2"}}
							]
						}`)),
					},
					// Second call: fetch user details
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"result": [
								{"sys_id": "user1", "name": "John Smith", "email": "john@example.com"},
								{"sys_id": "user2", "name": "Jane Doe", "email": "jane@example.com"}
							]
						}`)),
					},
				},
			},
			Parameters: map[string]string{
				"assignmentGroup": "grp1",
			},
		}

		resources, err := s.ListResources("user", ctx)

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "user1", resources[0].ID)
		assert.Equal(t, "John Smith (john@example.com)", resources[0].Name)
		assert.Equal(t, "user", resources[0].Type)
		assert.Equal(t, "user2", resources[1].ID)
		assert.Equal(t, "Jane Doe (jane@example.com)", resources[1].Name)
	})

	t.Run("returns empty list when group has no members", func(t *testing.T) {
		ctx := core.ListResourcesContext{
			Integration: integrationCtx,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"result": []
						}`)),
					},
				},
			},
			Parameters: map[string]string{
				"assignmentGroup": "empty-group",
			},
		}

		resources, err := s.ListResources("user", ctx)

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("returns hardcoded state resources", func(t *testing.T) {
		resources, err := s.ListResources("state", core.ListResourcesContext{
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 6)
		assert.Equal(t, "1", resources[0].ID)
		assert.Equal(t, "New", resources[0].Name)
		assert.Equal(t, "state", resources[0].Type)
		assert.Equal(t, "6", resources[3].ID)
		assert.Equal(t, "Resolved", resources[3].Name)
	})

	t.Run("returns hardcoded urgency resources", func(t *testing.T) {
		resources, err := s.ListResources("urgency", core.ListResourcesContext{
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 3)
		assert.Equal(t, "1", resources[0].ID)
		assert.Equal(t, "High", resources[0].Name)
		assert.Equal(t, "urgency", resources[0].Type)
		assert.Equal(t, "3", resources[2].ID)
		assert.Equal(t, "Low", resources[2].Name)
	})

	t.Run("returns hardcoded impact resources", func(t *testing.T) {
		resources, err := s.ListResources("impact", core.ListResourcesContext{
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 3)
		assert.Equal(t, "1", resources[0].ID)
		assert.Equal(t, "High", resources[0].Name)
		assert.Equal(t, "impact", resources[0].Type)
	})

	t.Run("returns hardcoded on_hold_reason resources", func(t *testing.T) {
		resources, err := s.ListResources("on_hold_reason", core.ListResourcesContext{
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 4)
		assert.Equal(t, "1", resources[0].ID)
		assert.Equal(t, "Awaiting Caller", resources[0].Name)
		assert.Equal(t, "on_hold_reason", resources[0].Type)
	})

	t.Run("returns hardcoded resolution_code resources", func(t *testing.T) {
		resources, err := s.ListResources("resolution_code", core.ListResourcesContext{
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 10)
		assert.Equal(t, "resolution_code", resources[0].Type)
		assert.Equal(t, "Duplicate", resources[0].ID)
		assert.Equal(t, "Duplicate", resources[0].Name)
	})

	t.Run("returns list of subcategories for a category", func(t *testing.T) {
		ctx := core.ListResourcesContext{
			Integration: integrationCtx,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"result": [
								{"label": "Email", "value": "email"},
								{"label": "Operating System", "value": "os"}
							]
						}`)),
					},
				},
			},
			Parameters: map[string]string{
				"category": "software",
			},
		}

		resources, err := s.ListResources("subcategory", ctx)

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "email", resources[0].ID)
		assert.Equal(t, "Email", resources[0].Name)
		assert.Equal(t, "subcategory", resources[0].Type)
	})
}
