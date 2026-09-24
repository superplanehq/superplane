package blob

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectKeyBuildsScopedPaths(t *testing.T) {
	installationID := "install-1"
	orgID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	factoryID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	workOrderID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	fileID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	t.Run("task", func(t *testing.T) {
		key, err := ObjectKey(installationID, ScopeTask, orgID, factoryID, workOrderID, fileID)
		require.NoError(t, err)
		assert.Equal(
			t,
			"install-1/orgs/11111111-1111-1111-1111-111111111111/workspaces/22222222-2222-2222-2222-222222222222/tasks/33333333-3333-3333-3333-333333333333/44444444-4444-4444-4444-444444444444",
			key,
		)
	})

	t.Run("workspace", func(t *testing.T) {
		key, err := ObjectKey(installationID, ScopeWorkspace, orgID, factoryID, uuid.Nil, fileID)
		require.NoError(t, err)
		assert.Equal(
			t,
			"install-1/orgs/11111111-1111-1111-1111-111111111111/workspaces/22222222-2222-2222-2222-222222222222/44444444-4444-4444-4444-444444444444",
			key,
		)
	})

	t.Run("rejects unknown scope", func(t *testing.T) {
		_, err := ObjectKey(installationID, "run", orgID, factoryID, workOrderID, fileID)
		require.Error(t, err)
	})
}
