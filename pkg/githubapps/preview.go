package githubapps

import (
	"fmt"
	"strings"

	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

// Preview describes an installable GitHub app before the user confirms installation.
type Preview struct {
	Repo        string `json:"repo"`
	Owner       string `json:"owner"`
	Repository  string `json:"repository"`
	Ref         string `json:"ref"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CanvasName  string `json:"canvasName"`
	DefaultName string `json:"defaultName"`
}

// BuildPreview loads app metadata from GitHub and prepares install defaults.
func BuildPreview(repoParam string) (*Preview, error) {
	repo, err := ParseRepository(repoParam)
	if err != nil {
		return nil, err
	}

	canvas, ref, err := FetchCanvas(repo)
	if err != nil {
		return nil, err
	}

	repo.Ref = ref

	return previewFromCanvas(repo, canvas, ref), nil
}

func previewFromCanvas(repo *Repository, canvas *pb.Canvas, ref string) *Preview {
	canvasName := strings.TrimSpace(canvas.GetMetadata().GetName())
	description := strings.TrimSpace(canvas.GetMetadata().GetDescription())

	defaultName := truncateInstallationName(canvasName)
	if defaultName == "" {
		defaultName = DefaultInstallationName(repo.Name)
	}

	displayName := canvasName
	if displayName == "" {
		displayName = DefaultInstallationName(repo.Name)
	}

	return &Preview{
		Repo:        repo.String(),
		Owner:       repo.Owner,
		Repository:  repo.Name,
		Ref:         ref,
		Title:       fmt.Sprintf("Install %s", displayName),
		Description: description,
		CanvasName:  canvasName,
		DefaultName: defaultName,
	}
}
