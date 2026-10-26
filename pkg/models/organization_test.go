package models

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

func TestCreateOrganizationWithoutInviteLink(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	organization, err := CreateOrganization("Email invitations", "")
	require.NoError(t, err)
	require.NotNil(t, organization)

	_, err = FindInviteLinkByOrganizationID(database.DB(t.Context()), organization.ID.String())
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
