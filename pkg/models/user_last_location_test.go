package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func orgPath(r *support.ResourceRegistry, suffix string) string {
	return "/" + r.Organization.Slug + suffix
}

func Test__PathBelongsToOrganization(t *testing.T) {
	assert.True(t, models.PathBelongsToOrganization("/acme", "acme"))
	assert.True(t, models.PathBelongsToOrganization("/acme/apps/deploy", "acme"))
	assert.True(t, models.PathBelongsToOrganization("/acme?run=1", "acme"))
	assert.True(t, models.PathBelongsToOrganization("/acme#node", "acme"))
	assert.False(t, models.PathBelongsToOrganization("/acme-2/apps", "acme"))
	assert.False(t, models.PathBelongsToOrganization("/other/apps", "acme"))
	assert.False(t, models.PathBelongsToOrganization("//evil.com", "acme"))
	assert.False(t, models.PathBelongsToOrganization("/acme", ""))
}

func Test__SetUserLastLocation__CreatesAndUpdates(t *testing.T) {
	r := support.Setup(t)

	location, err := models.SetUserLastLocation(database.Conn(), r.Organization.ID, r.User, orgPath(r, "/apps/deploy?run=1"))
	require.NoError(t, err)
	assert.Equal(t, orgPath(r, "/apps/deploy?run=1"), location.Path)

	location, err = models.SetUserLastLocation(database.Conn(), r.Organization.ID, r.User, orgPath(r, "/apps/deploy?run=2&node=approve-1"))
	require.NoError(t, err)
	assert.Equal(t, orgPath(r, "/apps/deploy?run=2&node=approve-1"), location.Path)

	found, err := models.FindUserLastLocation(database.Conn(), r.Organization.ID, r.User)
	require.NoError(t, err)
	assert.Equal(t, location.Path, found.Path)
}

func Test__SetUserLastLocation__RejectsUnsafePaths(t *testing.T) {
	r := support.Setup(t)

	for _, path := range []string{"", "no-leading-slash", "//evil.com", "https://evil.com", "/other-org/apps"} {
		_, err := models.SetUserLastLocation(database.Conn(), r.Organization.ID, r.User, path)
		assert.ErrorIs(t, err, models.ErrUserLastLocationPathInvalid, "path %q should be rejected", path)
	}
}

func Test__FindUserLastLocation__NotFound(t *testing.T) {
	r := support.Setup(t)

	_, err := models.FindUserLastLocation(database.Conn(), r.Organization.ID, r.User)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func Test__ListUserLastLocationsForAccount(t *testing.T) {
	r := support.Setup(t)
	require.NotNil(t, r.Account)

	_, err := models.SetUserLastLocation(database.Conn(), r.Organization.ID, r.User, orgPath(r, "/apps/deploy"))
	require.NoError(t, err)

	locations, err := models.ListUserLastLocationsForAccount(database.Conn(), r.Account.ID)
	require.NoError(t, err)
	require.Len(t, locations, 1)
	assert.Equal(t, r.Organization.ID, locations[0].OrganizationID)
	assert.Equal(t, orgPath(r, "/apps/deploy"), locations[0].Path)
}
