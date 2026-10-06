package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestListFactoryVelocityMembersIncludesBitbucketLogin(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.SaveAccountLinkedAccount(database.Conn(), models.NewAccountLinkedAccount(
		r.Account.ID,
		models.ProviderBitbucket,
		"11111111-1111-1111-1111-111111111111",
		"ada-bb",
		"Ada",
		"",
	)))

	members, err := models.ListFactoryVelocityMembers(database.Conn(), r.Organization.ID)
	require.NoError(t, err)

	var member *models.FactoryVelocityMember
	for index := range members {
		if members[index].UserID == r.User {
			member = &members[index]
			break
		}
	}
	require.NotNil(t, member)
	assert.Equal(t, "ada-bb", member.BitbucketLogin)
	assert.Equal(t, "testuser", member.GitHubLogin)
}
