package authorization

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestOrgHasExperimentalFeature(t *testing.T) {
	t.Run("released feature short-circuits without cache or database", func(t *testing.T) {
		t.Cleanup(clearOrgExperimentalFeatureCacheForTest)

		enabled, err := orgHasExperimentalFeature(uuid.New(), features.FeatureClaudeManagedAgents)
		require.NoError(t, err)
		assert.True(t, enabled)
	})

	t.Run("cached enabled short-circuits without loading organization", func(t *testing.T) {
		t.Cleanup(clearOrgExperimentalFeatureCacheForTest)

		orgID := uuid.New()
		const featureID = "exp-feature"
		SetOrgExperimentalFeatureCache(orgID, featureID, true)

		enabled, err := orgHasExperimentalFeature(orgID, featureID)
		require.NoError(t, err)
		assert.True(t, enabled)
	})

	t.Run("cached enabled expires after ttl", func(t *testing.T) {
		t.Cleanup(clearOrgExperimentalFeatureCacheForTest)
		originalNow := experimentalFeatureCacheNow
		t.Cleanup(func() { experimentalFeatureCacheNow = originalNow })

		start := time.Now()
		experimentalFeatureCacheNow = func() time.Time { return start }

		orgID := uuid.New()
		const featureID = "exp-feature"
		SetOrgExperimentalFeatureCache(orgID, featureID, true)

		cached, ok := cachedOrgExperimentalFeatureEnabled(orgID, featureID)
		require.True(t, ok)
		assert.True(t, cached)

		experimentalFeatureCacheNow = func() time.Time { return start.Add(experimentalFeatureCacheTTL + time.Minute) }

		cached, ok = cachedOrgExperimentalFeatureEnabled(orgID, featureID)
		assert.False(t, ok)
		assert.False(t, cached)
	})

	t.Run("cached disabled reloads from database", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		t.Cleanup(clearOrgExperimentalFeatureCacheForTest)

		org, err := models.CreateOrganization("auth-expfeat-cache-disabled", "")
		require.NoError(t, err)

		require.NoError(t, models.EnableExperimentalFeature(org.ID, "exp-feature"))
		SetOrgExperimentalFeatureCache(org.ID, "exp-feature", false)

		enabled, err := orgHasExperimentalFeature(org.ID, "exp-feature")
		require.NoError(t, err)
		assert.True(t, enabled)
	})

	t.Run("database result is cached after load", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		t.Cleanup(clearOrgExperimentalFeatureCacheForTest)

		org, err := models.CreateOrganization("auth-expfeat-cache-store", "")
		require.NoError(t, err)
		require.NoError(t, models.EnableExperimentalFeature(org.ID, "exp-feature"))

		enabled, err := orgHasExperimentalFeature(org.ID, "exp-feature")
		require.NoError(t, err)
		assert.True(t, enabled)

		cached, ok := cachedOrgExperimentalFeatureEnabled(org.ID, "exp-feature")
		require.True(t, ok)
		assert.True(t, cached)
	})
}

func TestCheckRequiredExperimentalFeatures(t *testing.T) {
	t.Run("released feature passes without organization lookup", func(t *testing.T) {
		err := checkRequiredExperimentalFeatures(context.Background(), uuid.NewString(), AuthorizationRule{
			RequiredExperimentalFeatures: []string{features.FeatureClaudeManagedAgents},
		})
		require.NoError(t, err)
	})

	t.Run("unreleased feature denied when not enabled", func(t *testing.T) {
		const unreleasedFeature = "test_unreleased_feature"
		t.Cleanup(features.WithRegistryForTest([]features.Feature{{ID: unreleasedFeature, Label: unreleasedFeature}}))
		t.Cleanup(clearOrgExperimentalFeatureCacheForTest)
		require.NoError(t, database.TruncateTables())

		org, err := models.CreateOrganization("auth-expfeat-deny", "")
		require.NoError(t, err)

		err = checkRequiredExperimentalFeatures(context.Background(), org.ID.String(), AuthorizationRule{
			RequiredExperimentalFeatures: []string{unreleasedFeature},
		})
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("unreleased feature allowed when enabled", func(t *testing.T) {
		const unreleasedFeature = "test_unreleased_feature"
		t.Cleanup(features.WithRegistryForTest([]features.Feature{{ID: unreleasedFeature, Label: unreleasedFeature}}))
		t.Cleanup(clearOrgExperimentalFeatureCacheForTest)
		require.NoError(t, database.TruncateTables())

		org, err := models.CreateOrganization("auth-expfeat-allow", "")
		require.NoError(t, err)
		require.NoError(t, models.EnableExperimentalFeature(org.ID, unreleasedFeature))

		err = checkRequiredExperimentalFeatures(context.Background(), org.ID.String(), AuthorizationRule{
			RequiredExperimentalFeatures: []string{unreleasedFeature},
		})
		require.NoError(t, err)
	})

	t.Run("cache update after disable forces database reload", func(t *testing.T) {
		const unreleasedFeature = "test_unreleased_feature"
		t.Cleanup(features.WithRegistryForTest([]features.Feature{{ID: unreleasedFeature, Label: unreleasedFeature}}))
		t.Cleanup(clearOrgExperimentalFeatureCacheForTest)
		require.NoError(t, database.TruncateTables())

		org, err := models.CreateOrganization("auth-expfeat-disable-cache", "")
		require.NoError(t, err)
		require.NoError(t, models.EnableExperimentalFeature(org.ID, unreleasedFeature))
		SetOrgExperimentalFeatureCache(org.ID, unreleasedFeature, true)

		require.NoError(t, models.DisableExperimentalFeature(org.ID, unreleasedFeature))
		SetOrgExperimentalFeatureCache(org.ID, unreleasedFeature, false)

		err = checkRequiredExperimentalFeatures(context.Background(), org.ID.String(), AuthorizationRule{
			RequiredExperimentalFeatures: []string{unreleasedFeature},
		})
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})
}
