package repositoryurl

import (
	"os"
	"strings"

	"github.com/google/uuid"
)

const defaultBranch = "main"

// DefaultBranch returns the default branch name for canvas repositories.
func DefaultBranch() string {
	branch := strings.TrimSpace(os.Getenv("CANVAS_STORAGE_DEFAULT_BRANCH"))
	if branch == "" {
		return defaultBranch
	}
	return branch
}

// SuperplaneCloneURL returns the git remote served by SuperPlane when using supergit:
// {BASE_URL}/git/{canvas-id}.git
func SuperplaneCloneURL(canvasID string) string {
	parsed, err := uuid.Parse(strings.TrimSpace(canvasID))
	if err != nil {
		return ""
	}

	base := strings.TrimRight(strings.TrimSpace(appBaseURL()), "/")
	if base == "" {
		return ""
	}

	return base + "/git/" + parsed.String() + ".git"
}

func appBaseURL() string {
	if base := strings.TrimSpace(os.Getenv("BASE_URL")); base != "" {
		return strings.TrimRight(base, "/")
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8000"
	}

	return "http://localhost:" + port
}
