package installation

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/yaml"
)

// Preview describes an installable GitHub app before the user confirms installation.
type Preview struct {
	Repo          string         `json:"repo"`
	Owner         string         `json:"owner"`
	Repository    string         `json:"repository"`
	Ref           string         `json:"ref"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	CanvasName    string         `json:"canvasName"`
	DefaultName   string         `json:"defaultName"`
	InstallParams []InstallParam `json:"installParams,omitempty"`
	Integrations  []string       `json:"integrations,omitempty"`
}

// BuildPreview loads app metadata from GitHub and prepares install defaults.
// If reg is non-nil, it also detects which integrations the canvas needs.
func BuildPreview(repoParam string, registry *registry.Registry) (*Preview, error) {
	repo, err := ParseRepository(repoParam)
	if err != nil {
		return nil, err
	}

	canvasBody, ref, err := fetchRawCanvasFile(repo)
	if err != nil {
		return nil, err
	}

	params, err := FetchParams(repo, ref)
	if err != nil {
		log.Warnf("failed to load params.json for %s: %v", repo.String(), err)
	}
	if params != nil && len(params.InstallParams) > 0 {
		canvasBody = SubstituteInstallParams(canvasBody, DefaultParamValues(params.InstallParams))
	}

	canvas, err := yaml.CanvasFromYAML(canvasBody)
	if err != nil {
		return nil, err
	}

	preview := previewFromCanvas(repo, canvas, ref)
	if params != nil {
		preview.InstallParams = params.InstallParams
	}

	preview.Integrations = detectIntegrations(canvas, registry)
	return preview, nil
}

// detectIntegrations returns a deduplicated list of integration type names
// required by the canvas nodes.
func detectIntegrations(canvas *yaml.Canvas, registry *registry.Registry) []string {
	if canvas.Spec == nil {
		return nil
	}

	componentToIntegration := buildComponentIntegrationMap(registry)
	seen := make(map[string]bool)
	var result []string

	for _, node := range canvas.Spec.Nodes {
		name := componentToIntegration[node.Component]
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}

	return result
}

func previewFromCanvas(repo *Repository, canvas *yaml.Canvas, ref string) *Preview {
	canvasName := strings.TrimSpace(canvas.Metadata.Name)
	description := strings.TrimSpace(canvas.Metadata.Description)

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
