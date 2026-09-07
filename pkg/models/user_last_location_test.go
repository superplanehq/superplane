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

func Test__SetUserLastLocation__CreatesAndUpdates(t *testing.T) {
	r := support.Setup(t)

	location, err := models.SetUserLastLocation(database.Conn(), r.Organization.ID, r.User, "/acme/apps/deploy?run=1")
	require.NoError(t, err)
	assert.Equal(t, "/acme/apps/deploy?run=1", location.Path)

	location, err = models.SetUserLastLocation(database.Conn(), r.Organization.ID, r.User, "/acme/apps/deploy?run=2&node=approve-1")
	require.NoError(t, err)
	assert.Equal(t, "/acme/apps/deploy?run=2&node=approve-1", location.Path)

	found, err := models.FindUserLastLocation(database.Conn(), r.Organization.ID, r.User)
	require.NoError(t, err)
	assert.Equal(t, location.Path, found.Path)
}

func Test__SetUserLastLocation__RejectsUnsafePaths(t *testing.T) {
	r := support.Setup(t)

	for _, path := range []string{"", "no-leading-slash", "//evil.com", "https://evil.com"} {
		_, err := models.SetUserLastLocation(database.Conn(), r.Organization.ID, r.User, path)
		assert.ErrorIs(t, err, models.ErrUserLastLocationPathInvalid, "path %q should be rejected", path)
	}
}

func Test__FindUserLastLocation__NotFound(t *testing.T) {
	r := support.Setup(t)

	_, err := models.FindUserLastLocation(database.Conn(), r.Organization.ID, r.User)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
