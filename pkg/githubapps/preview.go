package githubapps

import (
	"fmt"
	"strings"
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

	defaultName, err := GenerateInstallationName()
	if err != nil {
		return nil, err
	}

	canvasName := strings.TrimSpace(canvas.GetMetadata().GetName())
	description := strings.TrimSpace(canvas.GetMetadata().GetDescription())

	title := canvasName
	if readmeTitle, readmeErr := FetchReadmeTitle(repo); readmeErr == nil && readmeTitle != "" {
		title = readmeTitle
	}

	if title == "" {
		title = repo.Name
	}

	installTitle := fmt.Sprintf("Install %s", title)

	return &Preview{
		Repo:        repo.String(),
		Owner:       repo.Owner,
		Repository:  repo.Name,
		Ref:         ref,
		Title:       installTitle,
		Description: description,
		CanvasName:  canvasName,
		DefaultName: defaultName,
	}, nil
}
