package authorization

import (
	"context"
	"sync"
	"time"
)

const organizationPermissionCacheTTL = 15 * time.Minute

var (
	organizationPermissionCache    sync.Map
	organizationPermissionCacheNow = time.Now
)

type organizationPermissionCacheEntry struct {
	allowed   bool
	expiresAt time.Time
}

func organizationPermissionCacheKey(userID, orgID, resource, action string) string {
	return userID + "/" + orgID + "/" + resource + "/" + action
}

func cachedOrganizationPermissionAllowed(userID, orgID, resource, action string) (allowed bool, ok bool) {
	key := organizationPermissionCacheKey(userID, orgID, resource, action)
	value, loaded := organizationPermissionCache.Load(key)
	if !loaded {
		return false, false
	}

	entry, ok := value.(organizationPermissionCacheEntry)
	if !ok {
		organizationPermissionCache.Delete(key)
		return false, false
	}

	if organizationPermissionCacheNow().After(entry.expiresAt) {
		organizationPermissionCache.Delete(key)
		return false, false
	}

	return entry.allowed, true
}

func setOrganizationPermissionCache(userID, orgID, resource, action string, allowed bool) {
	organizationPermissionCache.Store(organizationPermissionCacheKey(userID, orgID, resource, action), organizationPermissionCacheEntry{
		allowed:   allowed,
		expiresAt: organizationPermissionCacheNow().Add(organizationPermissionCacheTTL),
	})
}

func clearOrganizationPermissionCacheForTest() {
	organizationPermissionCache.Range(func(key, _ any) bool {
		organizationPermissionCache.Delete(key)
		return true
	})
}

type organizationPermissionChecker interface {
	CheckOrganizationPermission(ctx context.Context, userID, orgID, resource, action string) (bool, error)
}

func checkOrganizationPermission(ctx context.Context, auth organizationPermissionChecker, userID, orgID, resource, action string) (bool, error) {
	if allowed, ok := cachedOrganizationPermissionAllowed(userID, orgID, resource, action); ok && allowed {
		return true, nil
	}

	allowed, err := auth.CheckOrganizationPermission(ctx, userID, orgID, resource, action)
	if err != nil {
		return false, err
	}

	setOrganizationPermissionCache(userID, orgID, resource, action, allowed)
	return allowed, nil
}
