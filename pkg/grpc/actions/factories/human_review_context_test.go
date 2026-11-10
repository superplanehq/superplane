package factories

import (
	"context"
	"errors"
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

func TestHumanReviewContextSkipsUnavailableOwnershipHints(t *testing.T) {
	for _, owner := range []string{"@acme/deleted-team", "@deleted-user"} {
		t.Run(owner, func(t *testing.T) {
			reviewer := reviewTestUser(8, "reviewer", "User")
			client := &unavailableHumanReviewOwnerFake{humanReviewOwnershipFake: humanReviewOwnershipFake{
				humanReviewLifecycleFake: humanReviewLifecycleFake{members: []*github.User{reviewer}, permissions: map[string]*github.RepositoryPermissionLevel{"reviewer": {User: reviewer, Permission: github.Ptr("write")}}},
				ownership:                "unrelated/** " + owner + "\n",
			}}
			current := &github.PullRequest{User: reviewTestUser(1, "author", "User")}
			gathered, err := gatherHumanReviewContext(t.Context(), client, &models.FactoryPullRequest{Repository: "acme/app", Number: 42}, current, "acme/fallback")
			require.NoError(t, err)
			candidates := gathered["fallbackCandidates"].([]humanreview.Reviewer)
			require.Len(t, candidates, 1)
			require.Equal(t, int64(8), candidates[0].ID)
			require.Empty(t, gathered["ownershipCandidates"])
			for _, candidates := range gathered["ownershipTeamCandidates"].(map[string][]humanreview.Reviewer) {
				require.Empty(t, candidates)
			}

			client.fallbackFailure = errors.New("fallback team is inaccessible")
			_, err = gatherHumanReviewContext(t.Context(), client, &models.FactoryPullRequest{Repository: "acme/app", Number: 42}, current, "acme/fallback")
			require.ErrorIs(t, err, client.fallbackFailure)
		})
	}
}

type unavailableHumanReviewOwnerFake struct {
	humanReviewOwnershipFake
	fallbackFailure error
}

func (f *unavailableHumanReviewOwnerFake) HumanReviewTeamMembers(ctx context.Context, organization, slug string) ([]*github.User, error) {
	if slug != "fallback" {
		return nil, errors.New("ownership team is unavailable")
	}
	if f.fallbackFailure != nil {
		return nil, f.fallbackFailure
	}
	return f.humanReviewLifecycleFake.HumanReviewTeamMembers(ctx, organization, slug)
}

func (f *unavailableHumanReviewOwnerFake) GetRepositoryPermissionLevel(ctx context.Context, repository, login string) (*github.RepositoryPermissionLevel, *github.Response, error) {
	if login == "deleted-user" {
		return nil, nil, errors.New("ownership user is unavailable")
	}
	return f.humanReviewLifecycleFake.GetRepositoryPermissionLevel(ctx, repository, login)
}
