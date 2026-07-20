package linear

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Linear__Sync(t *testing.T) {
	integration := &Linear{}

	t.Run("stores the viewer, workspace and teams", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"viewer":{"id":"u1","name":"Jane Doe","displayName":"jane","email":"jane@example.com"},"organization":{"id":"o1","name":"Acme","urlKey":"acme"}}}`),
				jsonResponse(`{"data":{"teams":{"nodes":[{"id":"t1","key":"ENG","name":"Engineering"}]}}}`),
			},
		}

		integrationContext := newAuthorizedIntegration()
		err := integration.Sync(core.SyncContext{
			HTTP:        httpContext,
			Integration: integrationContext,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationContext.State)

		metadata, ok := integrationContext.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, metadata.User)
		assert.Equal(t, "Jane Doe", metadata.User.Name)
		assert.Equal(t, "Acme", metadata.Organization)
		assert.Equal(t, "acme", metadata.URLKey)
		require.Len(t, metadata.Teams, 1)
		assert.Equal(t, "ENG", metadata.Teams[0].Key)
	})

	t.Run("invalid credentials -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"errors":[{"message":"Authentication required, not authenticated"}]}`),
			},
		}

		err := integration.Sync(core.SyncContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
		})

		require.ErrorContains(t, err, "error verifying Linear credentials")
	})
}

func Test__Linear__Definition(t *testing.T) {
	integration := &Linear{}

	assert.Equal(t, "linear", integration.Name())
	assert.Equal(t, "Linear", integration.Label())
	assert.Equal(t, "linear", integration.Icon())

	actions := integration.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "linear.createIssue", actions[0].Name())

	triggers := integration.Triggers()
	require.Len(t, triggers, 1)
	assert.Equal(t, "linear.onIssue", triggers[0].Name())
}
