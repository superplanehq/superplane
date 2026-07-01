package canvases

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
)

// ResolveRepositoryGitRef returns the git ref for reading a non-spec repository
// file at a canvas version. An empty versionID means the repository default
// branch head. When versionID is set, commit SHA is preferred over branch name.
func ResolveRepositoryGitRef(ctx context.Context, organizationID, canvasID, versionID string) (string, error) {
	if strings.TrimSpace(versionID) == "" {
		return "", nil
	}

	_, version, err := loadRepositorySpecVersionForRead(ctx, organizationID, canvasID, versionID)
	if err != nil {
		return "", err
	}

	return versionGitRef(version), nil
}

func versionGitRef(version *models.CanvasVersion) string {
	if version == nil {
		return ""
	}
	if sha := strings.TrimSpace(version.CommitSHA); sha != "" {
		return sha
	}
	return strings.TrimSpace(version.GitBranch)
}
