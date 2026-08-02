package authorization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCanvasAccess(t *testing.T) {
	userID := "22222222-2222-4222-8222-222222222222"
	orgID := "11111111-1111-4111-8111-111111111111"
	canvasA := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	canvasB := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"

	t.Run("denies when organization permission is missing", func(t *testing.T) {
		allowed, err := CheckCanvasAccess(
			context.Background(),
			denyingPermissionChecker{},
			userID,
			orgID,
			canvasA,
			"canvases",
			"read",
			nil,
			nil,
		)
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("allows unscoped caller with organization permission", func(t *testing.T) {
		allowed, err := CheckCanvasAccess(
			context.Background(),
			allowingPermissionChecker{},
			userID,
			orgID,
			canvasA,
			"canvases",
			"read",
			nil,
			nil,
		)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("denies API key scoped to a different canvas", func(t *testing.T) {
		allowed, err := CheckCanvasAccess(
			context.Background(),
			allowingPermissionChecker{},
			userID,
			orgID,
			canvasB,
			"canvases",
			"read",
			nil,
			[]string{canvasA},
		)
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("allows API key scoped to the requested canvas", func(t *testing.T) {
		allowed, err := CheckCanvasAccess(
			context.Background(),
			allowingPermissionChecker{},
			userID,
			orgID,
			canvasA,
			"canvases",
			"read",
			nil,
			[]string{canvasA},
		)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("denies scoped token for a different canvas", func(t *testing.T) {
		allowed, err := CheckCanvasAccess(
			context.Background(),
			allowingPermissionChecker{},
			userID,
			orgID,
			canvasB,
			"canvases",
			"read",
			[]string{"canvases:read:" + canvasA},
			nil,
		)
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("allows scoped token for the requested canvas", func(t *testing.T) {
		allowed, err := CheckCanvasAccess(
			context.Background(),
			allowingPermissionChecker{},
			userID,
			orgID,
			canvasA,
			"canvases",
			"read",
			[]string{"canvases:read:" + canvasA},
			nil,
		)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("denies scoped token missing canvases read action", func(t *testing.T) {
		allowed, err := CheckCanvasAccess(
			context.Background(),
			allowingPermissionChecker{},
			userID,
			orgID,
			canvasA,
			"canvases",
			"read",
			[]string{"canvases:update:" + canvasA},
			nil,
		)
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}
