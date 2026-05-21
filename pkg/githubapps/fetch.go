package githubapps

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/encoding/protojson"
)

const canvasFileName = "canvas.yaml"

var defaultRefs = []string{"main", "master"}

// FetchCanvas loads and parses canvas.yaml from a public GitHub repository.
func FetchCanvas(repo *Repository) (*pb.Canvas, string, error) {
	if repo.Ref == "" {
		for _, ref := range defaultRefs {
			canvas, err := fetchCanvasAtRef(repo, ref)
			if err == nil {
				repo.Ref = ref
				return canvas, ref, nil
			}
		}

		return nil, "", fmt.Errorf("canvas.yaml not found on main or master branch")
	}

	canvas, err := fetchCanvasAtRef(repo, repo.Ref)
	return canvas, repo.Ref, err
}

func fetchCanvasAtRef(repo *Repository, ref string) (*pb.Canvas, error) {
	rawURL := fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/%s/%s",
		repo.Owner,
		repo.Name,
		ref,
		canvasFileName,
	)

	body, err := fetchURL(rawURL)
	if err != nil {
		return nil, err
	}

	return parseCanvasYAML(body)
}

// FetchReadmeTitle returns the first markdown H1 from README.md, if present.
func FetchReadmeTitle(repo *Repository) (string, error) {
	ref := repo.Ref
	if ref == "" {
		ref = defaultRefs[0]
	}

	rawURL := fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/%s/README.md",
		repo.Owner,
		repo.Name,
		ref,
	)

	body, err := fetchURL(rawURL)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# ")), nil
		}
	}

	return "", fmt.Errorf("readme title not found")
}

func fetchURL(rawURL string) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if parsed.Scheme != "https" || parsed.Host != "raw.githubusercontent.com" {
		return nil, fmt.Errorf("unsupported fetch host %q", parsed.Host)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	response, err := client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s not found", rawURL)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %s: unexpected status %d", rawURL, response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rawURL, err)
	}

	return body, nil
}

func parseCanvasYAML(data []byte) (*pb.Canvas, error) {
	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, fmt.Errorf("parse canvas yaml: %w", err)
	}

	var canvas pb.Canvas
	if err := protojson.Unmarshal(jsonData, &canvas); err != nil {
		return nil, fmt.Errorf("parse canvas definition: %w", err)
	}

	if canvas.Metadata == nil {
		return nil, fmt.Errorf("canvas metadata is required")
	}

	if canvas.Metadata.GetIsTemplate() {
		return nil, fmt.Errorf("repository canvas is marked as a template")
	}

	canvas.Metadata.Id = ""
	canvas.Metadata.IsTemplate = false

	return &canvas, nil
}
