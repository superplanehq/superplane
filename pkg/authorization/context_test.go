package authorization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestOrganizationIDFromContext(t *testing.T) {
	t.Run("returns empty when context is nil", func(t *testing.T) {
		assert.Equal(t, "", OrganizationIDFromContext(nil))
	})

	t.Run("returns empty when key is missing", func(t *testing.T) {
		assert.Equal(t, "", OrganizationIDFromContext(context.Background()))
	})

	t.Run("returns empty when value is not a string", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), OrganizationContextKey, 42)
		assert.Equal(t, "", OrganizationIDFromContext(ctx))
	})

	t.Run("returns the value when set", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), OrganizationContextKey, "org-123")
		assert.Equal(t, "org-123", OrganizationIDFromContext(ctx))
	})
}

func TestDomainIDFromContext(t *testing.T) {
	t.Run("returns empty when context is nil", func(t *testing.T) {
		assert.Equal(t, "", DomainIDFromContext(nil))
	})

	t.Run("returns empty when key is missing", func(t *testing.T) {
		assert.Equal(t, "", DomainIDFromContext(context.Background()))
	})

	t.Run("returns the value when set", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), DomainIdContextKey, "domain-1")
		assert.Equal(t, "domain-1", DomainIDFromContext(ctx))
	})
}

func TestDomainTypeFromContext(t *testing.T) {
	t.Run("returns empty when context is nil", func(t *testing.T) {
		assert.Equal(t, "", DomainTypeFromContext(nil))
	})

	t.Run("returns empty when key is missing", func(t *testing.T) {
		assert.Equal(t, "", DomainTypeFromContext(context.Background()))
	})

	t.Run("returns the value when set", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), DomainTypeContextKey, models.DomainTypeOrganization)
		assert.Equal(t, models.DomainTypeOrganization, DomainTypeFromContext(ctx))
	})
}
