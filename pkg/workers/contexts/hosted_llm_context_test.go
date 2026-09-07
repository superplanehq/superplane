package contexts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__HostedLLMContext__AssertModelSelectable(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	t.Cleanup(func() {
		_ = database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBYOKModelAllowlist{})
	})

	hosted := NewHostedLLMContext(db, nil, r.Organization.ID, nil)
	err := hosted.AssertModelSelectable(models.UsageProviderOpenAI, models.UsageFundingSourceBYOK, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is required")

	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderOpenAI, datatypes.JSONSlice[string]{
		"gpt-4.1",
	})
	require.NoError(t, err)

	require.NoError(t, hosted.AssertModelSelectable(models.UsageProviderOpenAI, models.UsageFundingSourceBYOK, "gpt-4.1"))
	err = hosted.AssertModelSelectable(models.UsageProviderOpenAI, models.UsageFundingSourceBYOK, "gpt-4o")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "selected-model list")
}
