package authorization

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckOrganizationPermission(t *testing.T) {
	t.Run("cached allowed short-circuits without loading policies", func(t *testing.T) {
		t.Cleanup(clearOrganizationPermissionCacheForTest)

		userID := "user-123"
		orgID := "org-456"
		setOrganizationPermissionCache(userID, orgID, "canvases", "read", true)

		allowed, err := checkOrganizationPermission(context.Background(), denyingPermissionChecker{}, userID, orgID, "canvases", "read")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("cached allowed expires after ttl", func(t *testing.T) {
		t.Cleanup(clearOrganizationPermissionCacheForTest)
		originalNow := organizationPermissionCacheNow
		t.Cleanup(func() { organizationPermissionCacheNow = originalNow })

		start := time.Now()
		organizationPermissionCacheNow = func() time.Time { return start }

		userID := "user-123"
		orgID := "org-456"
		setOrganizationPermissionCache(userID, orgID, "canvases", "read", true)

		cached, ok := cachedOrganizationPermissionAllowed(userID, orgID, "canvases", "read")
		require.True(t, ok)
		assert.True(t, cached)

		organizationPermissionCacheNow = func() time.Time { return start.Add(organizationPermissionCacheTTL + time.Minute) }

		cached, ok = cachedOrganizationPermissionAllowed(userID, orgID, "canvases", "read")
		assert.False(t, ok)
		assert.False(t, cached)
	})

	t.Run("cached denied reloads from authorization service", func(t *testing.T) {
		t.Cleanup(clearOrganizationPermissionCacheForTest)

		userID := "user-123"
		orgID := "org-456"
		setOrganizationPermissionCache(userID, orgID, "canvases", "read", false)

		allowed, err := checkOrganizationPermission(context.Background(), allowingPermissionChecker{}, userID, orgID, "canvases", "read")
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("authorization service result is cached after load", func(t *testing.T) {
		t.Cleanup(clearOrganizationPermissionCacheForTest)

		userID := "user-123"
		orgID := "org-456"

		allowed, err := checkOrganizationPermission(context.Background(), allowingPermissionChecker{}, userID, orgID, "canvases", "read")
		require.NoError(t, err)
		assert.True(t, allowed)

		cached, ok := cachedOrganizationPermissionAllowed(userID, orgID, "canvases", "read")
		require.True(t, ok)
		assert.True(t, cached)
	})
}

type denyingPermissionChecker struct{}

func (denyingPermissionChecker) CheckOrganizationPermission(_ context.Context, _, _, _, _ string) (bool, error) {
	return false, nil
}

type allowingPermissionChecker struct{}

func (allowingPermissionChecker) CheckOrganizationPermission(_ context.Context, _, _, _, _ string) (bool, error) {
	return true, nil
}
