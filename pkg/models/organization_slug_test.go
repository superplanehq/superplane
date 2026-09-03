package models

import (
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func Test__Slugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "simple name", in: "Acme Corp", want: "acme-corp"},
		{name: "extra whitespace", in: "  Acme   Corp  ", want: "acme-corp"},
		{name: "punctuation", in: "Acme, Corp!!", want: "acme-corp"},
		{name: "leading and trailing dashes trimmed", in: "-Acme Corp-", want: "acme-corp"},
		{name: "mixed case", in: "ACME Corp", want: "acme-corp"},
		{name: "accented characters", in: "Café Org", want: "cafe-org"},
		{name: "non-ascii characters become dashes", in: "組織 Org", want: "org"},
		{name: "already a slug", in: "already-a-slug", want: "already-a-slug"},
		{name: "empty name falls back to default", in: "   ", want: defaultOrganizationSlug},
		{name: "only punctuation falls back to default", in: "!!!", want: defaultOrganizationSlug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Slugify(tt.in))
		})
	}

	t.Run("caps length", func(t *testing.T) {
		longName := ""
		for i := 0; i < 100; i++ {
			longName += "a"
		}

		slug := Slugify(longName)
		assert.LessOrEqual(t, len(slug), maxOrganizationSlugLength)
	})
}

func Test__IsReservedOrganizationSlug(t *testing.T) {
	assert.True(t, IsReservedOrganizationSlug("admin"))
	assert.True(t, IsReservedOrganizationSlug("login"))
	assert.True(t, IsReservedOrganizationSlug("api"))
	assert.False(t, IsReservedOrganizationSlug("acme-corp"))
}

func Test__GenerateUniqueOrganizationSlug(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("slugifies the base name", func(t *testing.T) {
		slug, err := GenerateUniqueOrganizationSlug(database.Conn(), "Acme Corp", uuid.Nil)
		require.NoError(t, err)
		assert.Equal(t, "acme-corp", slug)
	})

	t.Run("appends a numeric suffix on collision", func(t *testing.T) {
		// Different names that slugify to the same base, since organization
		// names must themselves be unique.
		org1, err := CreateOrganization("Collision Org", "")
		require.NoError(t, err)
		assert.Equal(t, "collision-org", org1.Slug)

		slug, err := GenerateUniqueOrganizationSlug(database.Conn(), "Collision Org", uuid.Nil)
		require.NoError(t, err)
		assert.Equal(t, "collision-org-2", slug)

		org2, err := CreateOrganization("Collision  Org", "")
		require.NoError(t, err)
		assert.Equal(t, "collision-org-2", org2.Slug)

		slug, err = GenerateUniqueOrganizationSlug(database.Conn(), "Collision Org", uuid.Nil)
		require.NoError(t, err)
		assert.Equal(t, "collision-org-3", slug)
	})

	t.Run("excludes the given organization ID from the collision check", func(t *testing.T) {
		org, err := CreateOrganization("Self Collision Org", "")
		require.NoError(t, err)

		slug, err := GenerateUniqueOrganizationSlug(database.Conn(), "Self Collision Org", org.ID)
		require.NoError(t, err)
		assert.Equal(t, "self-collision-org", slug, "excluding the org's own ID should allow it to keep its slug")
	})

	t.Run("ignores soft-deleted organizations", func(t *testing.T) {
		org, err := CreateOrganization("Deleted Collision Org", "")
		require.NoError(t, err)
		require.NoError(t, SoftDeleteOrganization(org.ID.String()))

		slug, err := GenerateUniqueOrganizationSlug(database.Conn(), "Deleted Collision Org", uuid.Nil)
		require.NoError(t, err)
		assert.Equal(t, "deleted-collision-org", slug)
	})

	t.Run("appends a suffix for reserved slugs", func(t *testing.T) {
		slug, err := GenerateUniqueOrganizationSlug(database.Conn(), "Admin", uuid.Nil)
		require.NoError(t, err)
		assert.False(t, IsReservedOrganizationSlug(slug))
		assert.Equal(t, "admin-org", slug)
	})
}

func Test__FindOrganizationBySlug(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org, err := CreateOrganization("Findable Org", "")
	require.NoError(t, err)

	t.Run("finds an organization by slug", func(t *testing.T) {
		found, err := FindOrganizationBySlug(database.Conn(), org.Slug)
		require.NoError(t, err)
		assert.Equal(t, org.ID, found.ID)
	})

	t.Run("returns an error for an unknown slug", func(t *testing.T) {
		_, err := FindOrganizationBySlug(database.Conn(), "does-not-exist")
		assert.Error(t, err)
	})
}

func Test__FindOrganizationByIDOrSlug(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org, err := CreateOrganization("ID Or Slug Org", "")
	require.NoError(t, err)

	t.Run("resolves by UUID", func(t *testing.T) {
		found, err := FindOrganizationByIDOrSlug(database.Conn(), org.ID.String())
		require.NoError(t, err)
		assert.Equal(t, org.ID, found.ID)
	})

	t.Run("resolves by slug", func(t *testing.T) {
		found, err := FindOrganizationByIDOrSlug(database.Conn(), org.Slug)
		require.NoError(t, err)
		assert.Equal(t, org.ID, found.ID)
	})

	t.Run("returns an error for an unknown reference", func(t *testing.T) {
		_, err := FindOrganizationByIDOrSlug(database.Conn(), "does-not-exist")
		assert.Error(t, err)
	})
}
