package provider

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// RepositorySlug converts a canvas (app) name into a single path segment suitable
// for git repository IDs and clone URLs.
func RepositorySlug(canvasName string) (string, error) {
	name := strings.TrimSpace(canvasName)
	if name == "" {
		return "", ErrInvalidRepositoryID
	}

	slug := strings.ReplaceAll(name, " ", "-")
	return NormalizePath(slug)
}

// RepositoryPath returns the provider-side repository identifier for an app.
func RepositoryPath(organizationID uuid.UUID, canvasName string) (string, error) {
	slug, err := RepositorySlug(canvasName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("orgs/%s/%s", organizationID.String(), slug), nil
}
