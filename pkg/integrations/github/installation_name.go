package github

import (
	"regexp"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
)

var generatedGitHubInstallationName = regexp.MustCompile(`^github(-[0-9]+)?$`)

// ownerGitHubInstallationName matches names this package generated from a
// GitHub owner (github-acme, github-acme (2), ...). A rebind to another
// account regenerates such a name from the new owner.
var ownerGitHubInstallationName = regexp.MustCompile(`^github-[a-z0-9][a-z0-9-]*( \([0-9]+\))?$`)

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
// when the connection uses the generated placeholder or a name generated
// from a previous owner.
func GeneratedOwnerInstallationName(integration *models.Integration) (string, bool) {
	if integration == nil || integration.AppName != "github" {
		return "", false
	}
	name := strings.TrimSpace(integration.InstallationName)
	if !IsGeneratedInstallationName(name) && !ownerGitHubInstallationName.MatchString(name) {
		return "", false
	}

	owner := strings.TrimSpace(githubOwnerFromMetadata(integration.Metadata.Data()))
	if owner == "" {
		return "", false
	}

	target := OwnerInstallationName(owner)
	// The name already comes from this owner (with or without a uniqueness
	// suffix), so a new assignment would only take the lock for nothing.
	if name == target || strings.HasPrefix(name, target+" (") {
		return "", false
	}

	return target, true
}

func githubOwnerFromMetadata(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}
	owner, _ := metadata["owner"].(string)
	return owner
}
