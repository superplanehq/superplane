package installation

import (
	"fmt"
	"net/url"
	"strings"
)

// Repository identifies a public GitHub repository hosting a SuperPlane app.
type Repository struct {
	Owner string
	Name  string
	// Ref is the git ref used when fetching app files (main or master).
	Ref string
}

// ParseRepository accepts github.com/owner/repo and common variants.
func ParseRepository(raw string) (*Repository, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("repository is required")
	}

	trimmed = strings.TrimPrefix(trimmed, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	trimmed = strings.TrimSuffix(trimmed, ".git")
	trimmed = strings.Trim(trimmed, "/")

	if strings.HasPrefix(trimmed, "github.com/") {
		trimmed = strings.TrimPrefix(trimmed, "github.com/")
	} else if u, err := url.Parse("https://" + trimmed); err == nil && strings.EqualFold(u.Host, "github.com") {
		trimmed = strings.Trim(u.Path, "/")
	}

	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("expected github.com/owner/repository")
	}

	if len(parts) > 2 {
		return nil, fmt.Errorf("expected github.com/owner/repository")
	}

	return &Repository{
		Owner: parts[0],
		Name:  parts[1],
	}, nil
}

func (r *Repository) String() string {
	return fmt.Sprintf("github.com/%s/%s", r.Owner, r.Name)
}
