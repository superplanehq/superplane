package licensing_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
)

type grantLookupFunc func(context.Context, uuid.UUID, licensing.Feature) (bool, error)

func (f grantLookupFunc) HasGrant(ctx context.Context, orgID uuid.UUID, feature licensing.Feature) (bool, error) {
	return f(ctx, orgID, feature)
}

func serviceWithGrant(t *testing.T, issuer *licensingtest.Issuer, raw []byte, lookup licensing.GrantLookup) *licensing.Service {
	t.Helper()
	options := []licensing.ServiceOption{}
	if lookup != nil {
		options = append(options, licensing.WithGrantLookup(lookup))
	}
	service := licensing.NewService(
		licensing.NewVerifier(licensingtest.KeySet(issuer)),
		&memorySource{raw: raw},
		options...,
	)
	require.NoError(t, service.Refresh(context.Background()))
	return service
}

func TestOrganizationEntitlement(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	org := uuid.New()
	other := uuid.New()
	ctx := context.Background()
	allowOrg := grantLookupFunc(func(_ context.Context, id uuid.UUID, feature licensing.Feature) (bool, error) {
		return id == org && feature == licensing.FeatureGroups, nil
	})
	deny := grantLookupFunc(func(context.Context, uuid.UUID, licensing.Feature) (bool, error) {
		return false, nil
	})
	fail := grantLookupFunc(func(context.Context, uuid.UUID, licensing.Feature) (bool, error) {
		return true, errors.New("database unavailable")
	})

	t.Run("no license", func(t *testing.T) {
		service := serviceWithGrant(t, issuer, nil, allowOrg)
		assert.False(t, service.IsEntitled(ctx, org, licensing.FeatureGroups))
	})

	t.Run("expired license", func(t *testing.T) {
		claims := licensingtest.Claims(licensing.FeatureGroups)
		claims["iat"] = time.Now().Add(-48 * time.Hour).Unix()
		claims["nbf"] = time.Now().Add(-48 * time.Hour).Unix()
		claims["exp"] = time.Now().Add(-time.Hour).Unix()
		service := serviceWithGrant(t, issuer, issuer.Sign(claims), allowOrg)
		assert.False(t, service.IsEntitled(ctx, org, licensing.FeatureGroups))
	})

	t.Run("revoked license", func(t *testing.T) {
		root := licensingtest.NewIssuer("root-2026")
		raw := issuer.License(licensing.FeatureGroups)
		verified, err := licensing.NewVerifier(licensingtest.KeySet(issuer)).Verify(raw)
		require.NoError(t, err)
		server := newKeyServer(t, root.RevocationList(1, verified.ID))
		revocations := licensing.NewRevocationSync(licensingtest.KeySet(root), &memoryRevocationCache{}, server.URL)
		service := licensing.NewService(
			licensing.NewVerifier(licensingtest.KeySet(issuer)),
			&memorySource{raw: raw},
			licensing.WithRevocationSync(revocations),
			licensing.WithGrantLookup(allowOrg),
		)
		require.NoError(t, service.Refresh(ctx))
		_, err = revocations.Sync(ctx)
		require.NoError(t, err)
		assert.Equal(t, licensing.StateRevoked, service.Status().State)
		assert.False(t, service.IsEntitled(ctx, org, licensing.FeatureGroups))
	})

	t.Run("invalid license", func(t *testing.T) {
		otherIssuer := licensingtest.NewIssuer("untrusted")
		service := serviceWithGrant(t, issuer, otherIssuer.License(licensing.FeatureGroups), allowOrg)
		assert.Equal(t, licensing.StateInvalid, service.Status().State)
		assert.False(t, service.IsEntitled(ctx, org, licensing.FeatureGroups))
	})

	t.Run("license lacks the feature", func(t *testing.T) {
		service := serviceWithGrant(t, issuer, issuer.License(licensing.FeatureCustomRoles), allowOrg)
		assert.False(t, service.IsEntitled(ctx, org, licensing.FeatureGroups))
	})

	t.Run("grant lookup returns false", func(t *testing.T) {
		service := serviceWithGrant(t, issuer, issuer.License(licensing.FeatureGroups), deny)
		assert.False(t, service.IsEntitled(ctx, org, licensing.FeatureGroups))
	})

	t.Run("grant lookup returns an error", func(t *testing.T) {
		service := serviceWithGrant(t, issuer, issuer.License(licensing.FeatureGroups), fail)
		assert.False(t, service.IsEntitled(ctx, org, licensing.FeatureGroups))
	})

	t.Run("nil organization", func(t *testing.T) {
		service := serviceWithGrant(t, issuer, issuer.License(licensing.FeatureGroups), allowOrg)
		assert.False(t, service.IsEntitled(ctx, uuid.Nil, licensing.FeatureGroups))
	})

	t.Run("no grant lookup", func(t *testing.T) {
		service := serviceWithGrant(t, issuer, issuer.License(licensing.FeatureGroups), nil)
		assert.False(t, service.IsEntitled(ctx, org, licensing.FeatureGroups))
	})

	t.Run("active license and grant", func(t *testing.T) {
		service := serviceWithGrant(t, issuer, issuer.License(licensing.FeatureGroups), allowOrg)
		assert.True(t, service.IsEntitled(ctx, org, licensing.FeatureGroups))
		assert.False(t, service.IsEntitled(ctx, other, licensing.FeatureGroups))
	})

	t.Run("development license skips the grant lookup", func(t *testing.T) {
		dev := licensingtest.NewIssuer(licensing.DevelopmentKeyID)
		called := false
		lookup := grantLookupFunc(func(context.Context, uuid.UUID, licensing.Feature) (bool, error) {
			called = true
			return false, errors.New("lookup must not run")
		})
		handler := &recordingLifecycle{}
		service := licensing.NewService(
			licensing.NewVerifier(licensingtest.KeySet(dev)),
			&memorySource{raw: dev.License(licensing.FeatureGroups)},
			licensing.WithGrantLookup(lookup),
			licensing.WithLifecycleHandler(handler),
		)
		require.NoError(t, service.Refresh(ctx))
		assert.True(t, service.IsEntitled(ctx, uuid.Nil, licensing.FeatureGroups))
		assert.True(t, service.IsEntitled(ctx, org, licensing.FeatureGroups))
		assert.False(t, called)
		assert.Equal(t, 0, handler.count())
	})
}
