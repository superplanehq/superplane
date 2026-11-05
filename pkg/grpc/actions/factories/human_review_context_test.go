package factories

import (
	"context"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/factories/humanreview"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestHumanReviewContextResolvesEligibleIndividualCodeOwners(t *testing.T) {
	author := reviewTestUser(1, "author", "User")
	owner := reviewTestUser(8, "owner", "User")
	reader := reviewTestUser(9, "reader", "User")
	bot := reviewTestUser(10, "robot[bot]", "Bot")
	client := &humanReviewOwnershipFake{humanReviewLifecycleFake: humanReviewLifecycleFake{permissions: map[string]*github.RepositoryPermissionLevel{
		"owner":      {User: owner, Permission: github.Ptr("write")},
		"author":     {User: author, Permission: github.Ptr("admin")},
		"reader":     {User: reader, Permission: github.Ptr("read")},
		"robot[bot]": {User: bot, Permission: github.Ptr("write")},
	}}, ownership: "# @ignored\npkg/** @owner @author @reader @robot[bot] owners@example.com\nweb/** @OWNER # @ignored\n"}
	current := &github.PullRequest{User: author, Base: &github.PullRequestBranch{SHA: github.Ptr("target-branch-commit")}}
	gathered, err := gatherHumanReviewContext(t.Context(), client, &models.FactoryPullRequest{Repository: "acme/app", Number: 42}, current, "acme/fallback")
	require.NoError(t, err)
	candidates, ok := gathered["ownershipCandidates"].([]humanreview.Reviewer)
	require.True(t, ok)
	require.Len(t, candidates, 1)
	require.Equal(t, int64(8), candidates[0].ID)
	require.Equal(t, "owner", candidates[0].Login)
	require.NotEmpty(t, candidates[0].Reason)
	require.Equal(t, "target-branch-commit", client.ownershipRef)
	require.Empty(t, gathered["fallbackCandidates"])
	require.Empty(t, gathered["existingRequests"])
	require.Empty(t, gathered["recentContributions"])
}

type humanReviewOwnershipFake struct {
	humanReviewLifecycleFake
	ownership    string
	ownershipRef string
}

func (f *humanReviewOwnershipFake) HumanReviewOwnershipFile(_ context.Context, _ string, path, ref string) (string, error) {
	if path != ".github/CODEOWNERS" {
		return "", nil
	}
	f.ownershipRef = ref
	return f.ownership, nil
}
