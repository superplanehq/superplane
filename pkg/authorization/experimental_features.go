package authorization

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const experimentalFeatureCacheTTL = 15 * time.Minute

var (
	orgExperimentalFeatureCache sync.Map
	experimentalFeatureCacheNow = time.Now
)

type experimentalFeatureCacheEntry struct {
	enabled   bool
	expiresAt time.Time
}

func orgExperimentalFeatureCacheKey(orgID uuid.UUID, featureID string) string {
	return orgID.String() + "/" + featureID
}

// SetOrgExperimentalFeatureCache records whether a feature is enabled for an
// organization in the authorization cache. Entries expire after 15 minutes.
func SetOrgExperimentalFeatureCache(orgID uuid.UUID, featureID string, enabled bool) {
	orgExperimentalFeatureCache.Store(orgExperimentalFeatureCacheKey(orgID, featureID), experimentalFeatureCacheEntry{
		enabled:   enabled,
		expiresAt: experimentalFeatureCacheNow().Add(experimentalFeatureCacheTTL),
	})
}

func cachedOrgExperimentalFeatureEnabled(orgID uuid.UUID, featureID string) (enabled bool, ok bool) {
	key := orgExperimentalFeatureCacheKey(orgID, featureID)
	value, loaded := orgExperimentalFeatureCache.Load(key)
	if !loaded {
		return false, false
	}

	entry, ok := value.(experimentalFeatureCacheEntry)
	if !ok {
		orgExperimentalFeatureCache.Delete(key)
		return false, false
	}

	if experimentalFeatureCacheNow().After(entry.expiresAt) {
		orgExperimentalFeatureCache.Delete(key)
		return false, false
	}

	return entry.enabled, true
}

func clearOrgExperimentalFeatureCacheForTest() {
	orgExperimentalFeatureCache.Range(func(key, _ any) bool {
		orgExperimentalFeatureCache.Delete(key)
		return true
	})
}

func orgHasExperimentalFeature(orgID uuid.UUID, featureID string) (bool, error) {
	if features.IsReleased(featureID) {
		return true, nil
	}

	if enabled, ok := cachedOrgExperimentalFeatureEnabled(orgID, featureID); ok && enabled {
		return true, nil
	}

	enabled, err := models.HasExperimentalFeature(orgID, featureID)
	if err != nil {
		return false, err
	}

	SetOrgExperimentalFeatureCache(orgID, featureID, enabled)
	return enabled, nil
}

func checkRequiredExperimentalFeatures(ctx context.Context, orgID string, rule AuthorizationRule) error {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return status.Error(codes.NotFound, "organization not found")
	}

	for _, featureID := range rule.RequiredExperimentalFeatures {
		enabled, err := orgHasExperimentalFeature(orgUUID, featureID)
		if err != nil {
			return status.Error(codes.NotFound, "organization not found")
		}
		if !enabled {
			return status.Error(codes.PermissionDenied, "required experimental feature is not enabled")
		}
	}

	return nil
}
