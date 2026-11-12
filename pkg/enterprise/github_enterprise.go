package enterprise

import (
	"context"

	"github.com/google/uuid"
)

// GitHubEnterprise authorizes connecting an organization to a GitHub host
// other than github.com. The implementation is in ee/githubenterprise under
// the SuperPlane Enterprise license. Read, delete, webhooks, and API calls
// on an existing connection stay in Community code so that they keep working
// after a license expires.
type GitHubEnterprise interface {
	AuthorizeHost(ctx context.Context, orgID uuid.UUID, host string) error
}

type communityGitHubEnterprise struct{}

func (communityGitHubEnterprise) AuthorizeHost(context.Context, uuid.UUID, string) error {
	return notLicensed()
}
