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
			"instanceUrl": "https://dev12345.service-now.com",
			"authType":    "basicAuth",
			"username":    "admin",
			"password":    "password",
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

	t.Run("returns list of assignment groups", func(t *testing.T) {
		ctx := core.ListResourcesContext{
			Integration: integrationCtx,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"result": [
								{"sys_id": "grp1", "name": "Network"},
								{"sys_id": "grp2", "name": "Database"}
							]
						}`)),
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
}
