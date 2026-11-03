package factories

import (
	"context"
	"errors"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
)

func (g *githubProvider) ReadDescription(ctx context.Context, ref vcs.PullRequestRef) (vcs.PullRequestDescription, error) {
	if err := g.ensureClient(); err != nil {
		return vcs.PullRequestDescription{}, err
	}
	pr, _, err := g.client.GetPullRequest(ctx, ref.Repository, int(ref.Number))
	if err != nil {
		return vcs.PullRequestDescription{}, err
	}
	return vcs.PullRequestDescription{Description: pr.GetBody(), State: pr.GetState(), HeadSHA: pr.GetHead().GetSHA()}, nil
}

func (g *githubProvider) UpdateDescription(ctx context.Context, ref vcs.PullRequestRef, description string) error {
	if err := g.ensureClient(); err != nil {
		return err
	}
	_, _, err := g.client.EditPullRequest(ctx, ref.Repository, int(ref.Number), &github.PullRequest{Body: &description})
	return err
}

func (b *bitbucketProvider) ReadDescription(_ context.Context, ref vcs.PullRequestRef) (vcs.PullRequestDescription, error) {
	if err := b.ensureClient(); err != nil {
		return vcs.PullRequestDescription{}, err
	}
	pr, err := b.client.GetPullRequest(ref.Repository, ref.Number)
	if err != nil {
		return vcs.PullRequestDescription{}, err
	}
	state := models.FactoryPullRequestStateClosed
	if pr.State == bitbucket.PullRequestStateOpen {
		state = models.FactoryPullRequestStateOpen
	}
	if pr.State == bitbucket.PullRequestStateMerged {
		state = models.FactoryPullRequestStateMerged
	}
	return vcs.PullRequestDescription{Description: pr.Description, State: state, HeadSHA: pr.SourceHash}, nil
}

func (b *bitbucketProvider) UpdateDescription(_ context.Context, ref vcs.PullRequestRef, description string) error {
	if err := b.ensureClient(); err != nil {
		return err
	}
	_, err := b.client.UpdatePullRequest(ref.Repository, ref.Number, bitbucket.UpdatePullRequestRequest{Description: &description})
	return err
}

func retryableDescriptionError(err error) bool {
	if errors.Is(err, errFactoryGitHubNotConnected) || errors.Is(err, errFactoryBitbucketNotConnected) {
		return false
	}
	var rate *github.RateLimitError
	var abuse *github.AbuseRateLimitError
	if errors.As(err, &rate) || errors.As(err, &abuse) {
		return true
	}
	var tokenError *ghinstallation.HTTPError
	if errors.As(err, &tokenError) && tokenError.Response != nil {
		response := tokenError.Response
		if response.StatusCode == http.StatusForbidden && (response.Header.Get("Retry-After") != "" || response.Header.Get("X-RateLimit-Remaining") == "0") {
			return true
		}
		return retryableDescriptionStatus(response.StatusCode)
	}
	var githubError *github.ErrorResponse
	if errors.As(err, &githubError) && githubError.Response != nil {
		return retryableDescriptionStatus(githubError.Response.StatusCode)
	}
	var bitbucketError *bitbucket.APIError
	if errors.As(err, &bitbucketError) {
		return retryableDescriptionStatus(bitbucketError.StatusCode)
	}
	// Transport errors and failures without an HTTP response use consumer retry.
	return true
}

func retryableDescriptionStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusConflict || status == http.StatusTooManyRequests || status >= 500
}
