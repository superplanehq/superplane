package factories

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func setMultiRepoTasks(t *testing.T, r *support.ResourceRegistry, enabled bool) (BuiltInAutomationReset, error) {
	t.Helper()
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	return SetOrganizationMultiRepoTasks(ctx, r.Organization.ID, enabled, testIntakeDependencies(r))
}

func multiRepoTasksStored(t *testing.T, r *support.ResourceRegistry) bool {
	t.Helper()
	organization, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	return slices.Contains(organization.EnabledExperimentalFeatures, features.FeatureFactoryMultiRepoTasks)
}

func hasPullRequestNode(nodes []models.Node) bool {
	return slices.ContainsFunc(nodes, func(node models.Node) bool { return isFactoryPullRequestComponent(node.ComponentName()) })
}

func TestSetOrganizationMultiRepoTasksSwitchesTheImplementTemplate(t *testing.T) {
	r, factory := resetTestFactory(t)
	github := resetTestTemplate(t, r, factory, lineImplementationTemplateID, models.ProviderGitHub, func(doc *yaml.Canvas) {
		setResetTestDraftPullRequests(t, doc, true)
	})
	bitbucket := resetTestTemplate(t, r, factory, lineImplementationTemplateID, models.ProviderBitbucket, func(*yaml.Canvas) {})
	bitbucketVersion := resetTestLiveVersion(t, bitbucket).ID

	result, err := setMultiRepoTasks(t, r, true)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Reset, "only the GitHub template has a multi-repo variant")
	assert.True(t, multiRepoTasksStored(t, r))
	current := resetTestLiveVersion(t, github)
	assert.False(t, hasPullRequestNode(current.Nodes), "multi-repo agents open their own pull requests")
	assert.True(t, resetTestDraftPullRequests(t, current.Nodes))
	assert.Equal(t, bitbucketVersion, resetTestLiveVersion(t, bitbucket).ID)

	enabledVersion := current.ID
	result, err = setMultiRepoTasks(t, r, true)
	require.NoError(t, err)
	assert.Zero(t, result.Reset, "enabling twice must not publish again")
	assert.Equal(t, enabledVersion, resetTestLiveVersion(t, github).ID)

	result, err = setMultiRepoTasks(t, r, false)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Reset)
	assert.False(t, multiRepoTasksStored(t, r))
	current = resetTestLiveVersion(t, github)
	assert.True(t, hasPullRequestNode(current.Nodes))
	assert.True(t, resetTestDraftPullRequests(t, current.Nodes))
}

func TestSetOrganizationMultiRepoTasksKeepsTheFlagWhenSettingsCannotApply(t *testing.T) {
	r, factory := resetTestFactory(t)
	canvas := resetTestTemplate(t, r, factory, lineImplementationTemplateID, models.ProviderGitHub, func(*yaml.Canvas) {})
	version := resetTestLiveVersion(t, canvas)
	nodes := slices.Clone(version.Nodes)
	zero := 0
	for i := range nodes {
		if nodes[i].ID == implementationAgentNodeID {
			nodes[i].Concurrency = &models.ConcurrencySpec{Max: &zero}
		}
	}
	require.NoError(t, database.DB(t.Context()).Model(&models.CanvasVersion{}).
		Where("id = ?", version.ID).
		Update("nodes", datatypes.NewJSONSlice(nodes)).Error)

	_, err := setMultiRepoTasks(t, r, true)
	var settingsError *BuiltInAutomationSettingsError
	require.ErrorAs(t, err, &settingsError)
	assert.False(t, multiRepoTasksStored(t, r))
	current := resetTestLiveVersion(t, canvas)
	assert.Equal(t, version.ID, current.ID)
	assert.True(t, hasPullRequestNode(current.Nodes))
}
