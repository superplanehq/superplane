package contexts

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
)

// repositoryFilesContext implements core.RepositoryFilesContext by reading
// files from the canvas git repository via supergit.
type repositoryFilesContext struct {
	gitProvider gitprovider.Provider
	canvasID    uuid.UUID

	once   sync.Once
	repoID string
	err    error
}

// NewRepositoryFilesContext creates a RepositoryFilesContext that lazily
// resolves the git repository for the given canvas.
func NewRepositoryFilesContext(
	gitProvider gitprovider.Provider,
	canvasID uuid.UUID,
) core.RepositoryFilesContext {
	return &repositoryFilesContext{
		gitProvider: gitProvider,
		canvasID:    canvasID,
	}
}

func (c *repositoryFilesContext) resolveRepo() (string, error) {
	c.once.Do(func() {
		repo, err := models.FindRepositoryUnscoped(c.canvasID)
		if err != nil {
			c.err = fmt.Errorf("find repository for canvas %s: %w", c.canvasID, err)
			return
		}
		c.repoID = repo.RepoID
	})
	return c.repoID, c.err
}

func (c *repositoryFilesContext) List() ([]string, error) {
	repoID, err := c.resolveRepo()
	if err != nil {
		return nil, err
	}
	return c.gitProvider.ListFiles(context.Background(), repoID)
}

func (c *repositoryFilesContext) Read(path string) (io.ReadCloser, error) {
	repoID, err := c.resolveRepo()
	if err != nil {
		return nil, err
	}
	return c.gitProvider.GetFile(context.Background(), repoID, path)
}
