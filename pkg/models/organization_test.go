package models

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestCreateOrganizationWithoutInviteLink(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	organization, err := CreateOrganization("Email invitations", "")
	require.NoError(t, err)
	require.NotNil(t, organization)

	var links int64
	require.NoError(t, database.DB(t.Context()).
		Model(&OrganizationInviteLink{}).
		Where("organization_id = ?", organization.ID).
		Count(&links).
		Error)
	require.Zero(t, links)
}
