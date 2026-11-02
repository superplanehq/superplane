package enterprise

// GitHubEnterprise authorizes connecting an organization to a GitHub host
// other than github.com. The implementation is in ee/githubenterprise under
// the SuperPlane Enterprise license. Read, delete, webhooks, and API calls
// on an existing connection stay in Community code so that they keep working
// after a license expires.
type GitHubEnterprise interface {
	AuthorizeHost(host string) error
}

type communityGitHubEnterprise struct{}

func (communityGitHubEnterprise) AuthorizeHost(string) error {
	return notLicensed()
}
