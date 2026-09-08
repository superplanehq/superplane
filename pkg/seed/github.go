package seed

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

// ErrNoGitHubInstallations means the hosted App is not installed on any account.
var ErrNoGitHubInstallations = fmt.Errorf("the GitHub App has no installations")

// ErrNoGitHubRepositories means the selected installation has no repositories.
var ErrNoGitHubRepositories = fmt.Errorf("the GitHub App installation has no repositories")

// SelectInstallation picks one hosted App installation.
// When requestedID is set, it must match an installation.
// When it is empty and exactly one installation exists, that installation is used.
func SelectInstallation(installations []common.PendingInstallation, requestedID string) (common.PendingInstallation, error) {
	if len(installations) == 0 {
		return common.PendingInstallation{}, ErrNoGitHubInstallations
	}

	requestedID = strings.TrimSpace(requestedID)
	if requestedID != "" {
		for _, installation := range installations {
			if installation.ID == requestedID {
				return installation, nil
			}
		}
		return common.PendingInstallation{}, fmt.Errorf(
			"GitHub App installation %s was not found",
			requestedID,
		)
	}

	if len(installations) == 1 {
		return installations[0], nil
	}

	return common.PendingInstallation{}, multipleInstallationsError(installations)
}

// SelectRepository returns owner/name for the app repository.
func SelectRepository(owner string, repos []common.Repository, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		return repositoryFullName(owner, requested), nil
	}
	if len(repos) == 0 {
		return "", ErrNoGitHubRepositories
	}
	return repositoryFullName(owner, repos[0].Name), nil
}

func repositoryFullName(owner, name string) string {
	name = strings.TrimSpace(name)
	if strings.Contains(name, "/") {
		return name
	}
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return name
	}
	return owner + "/" + name
}

func multipleInstallationsError(installations []common.PendingInstallation) error {
	var b strings.Builder
	b.WriteString("the GitHub App has more than one installation; set SUPERPLANE_SEED_GITHUB_INSTALLATION_ID to one of:")
	for _, installation := range installations {
		b.WriteString("\n  ")
		b.WriteString(installation.ID)
		if installation.AccountLogin != "" {
			b.WriteString(" (")
			b.WriteString(installation.AccountLogin)
			b.WriteString(")")
		}
	}
	return fmt.Errorf("%s", b.String())
}
