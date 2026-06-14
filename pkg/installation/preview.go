package installation

import (
	"errors"
	"fmt"
	"strings"

	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
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
func BuildPreview(repoParam string, reg *registry.Registry) (*Preview, error) {
	repo, err := ParseRepository(repoParam)
	if err != nil {
		return nil, err
	}

	// Fetch raw canvas to resolve ref, then check for params.
	var canvasBody []byte
	var ref string
	if repo.Ref == "" {
		for _, r := range defaultRefs {
			body, fetchErr := fetchURL(rawFileURL(repo, r, canvasFileName))
			if fetchErr == nil {
				ref = r
				repo.Ref = r
				canvasBody = body
				break
			}
			if !errors.Is(fetchErr, errFileNotFound) {
				return nil, fetchErr
			}
		}
		if canvasBody == nil {
			return nil, fmt.Errorf("canvas.yaml not found on main or master branch")
		}
	} else {
		ref = repo.Ref
		canvasBody, err = fetchURL(rawFileURL(repo, ref, canvasFileName))
		if err != nil {
			return nil, err
		}
	}

	// Fetch params and substitute with defaults/placeholders so canvas parses.
	params, _ := FetchParams(repo, ref)
	if params != nil && len(params.InstallParams) > 0 {
		defaults := make(map[string]string, len(params.InstallParams))
		for _, p := range params.InstallParams {
			if p.Default != "" {
				defaults[p.Name] = p.Default
			} else if p.Placeholder != "" {
				defaults[p.Name] = p.Placeholder
			} else {
				defaults[p.Name] = p.Name
			}
		}
		canvasBody = SubstituteInstallParams(canvasBody, defaults)
	}

	canvas, err := parseCanvasYAML(canvasBody)
	if err != nil {
		return nil, err
	}

	preview := previewFromCanvas(repo, canvas, ref)
	if params != nil {
		preview.InstallParams = params.InstallParams
	}

	if reg != nil {
		preview.Integrations = detectIntegrations(canvas, reg)
	}

	return preview, nil
}

// detectIntegrations returns a deduplicated list of integration type names
// required by the canvas nodes.
func detectIntegrations(canvas *pb.Canvas, reg *registry.Registry) []string {
	if canvas.Spec == nil {
		return nil
	}

	seen := make(map[string]bool)
	var result []string
	for _, node := range canvas.Spec.Nodes {
		name := findIntegrationForComponent(node, reg)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}

	return result
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
