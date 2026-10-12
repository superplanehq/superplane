package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

func TestIntakeAnalysisCloneCommandProvider(t *testing.T) {
	github := intakeAnalysisCloneCommand("")
	assert.Contains(t, github, "GITHUB_TOKEN")

	bitbucket := intakeAnalysisCloneCommand(models.ProviderBitbucket)
	assert.NotContains(t, bitbucket, "GITHUB_TOKEN")
	assert.Contains(t, bitbucket, "git ls-remote --heads")
}

func TestRejectVCSProviderChange(t *testing.T) {
	github := "github"
	bitbucket := models.ProviderBitbucket
	factory := &models.Factory{}
	factory.OnboardingConfig = datatypes.NewJSONType(models.FactoryOnboardingConfig{
		VCSIntegrationID: "11111111-1111-1111-1111-111111111111",
		VCSProvider:      models.ProviderBitbucket,
	})

	require.NoError(t, rejectVCSProviderChange(factory, models.FactoryOnboardingPatch{}))
	require.NoError(t, rejectVCSProviderChange(factory, models.FactoryOnboardingPatch{VCSProvider: &bitbucket}))
	require.ErrorContains(t, rejectVCSProviderChange(factory, models.FactoryOnboardingPatch{VCSProvider: &github}), "cannot be changed")

	require.NoError(t, rejectFactoryProviderSwitch(factory, models.ProviderBitbucket))
	require.ErrorContains(t, rejectFactoryProviderSwitch(factory, models.ProviderGitHub), "cannot be changed")
}
