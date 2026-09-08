package github

import (
	"regexp"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
)

var generatedGitHubInstallationName = regexp.MustCompile(`^github(-[0-9]+)?$`)

// IsGeneratedInstallationName reports whether the name is the placeholder
// created before GitHub bind (github, github-2, ...).
func IsGeneratedInstallationName(name string) bool {
	return generatedGitHubInstallationName.MatchString(strings.TrimSpace(name))
}

// OwnerInstallationName is the preferred display name after bind.
func OwnerInstallationName(owner string) string {
	return "github-" + strings.ToLower(strings.TrimSpace(owner))
}

// GeneratedOwnerInstallationName returns the preferred github-<owner> name
// when the connection still uses the generated placeholder.
func GeneratedOwnerInstallationName(integration *models.Integration) (string, bool) {
	if integration == nil || integration.AppName != "github" {
		return "", false
	}
	if !IsGeneratedInstallationName(integration.InstallationName) {
		return "", false
	}

	owner := strings.TrimSpace(githubOwnerFromMetadata(integration.Metadata.Data()))
	if owner == "" {
		return "", false
	}

	return OwnerInstallationName(owner), true
}

func githubOwnerFromMetadata(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}
	owner, _ := metadata["owner"].(string)
	return owner
}
