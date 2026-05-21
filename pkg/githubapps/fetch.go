package githubapps

import (
	"encoding/json"
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
		var lastErr error
		for _, ref := range defaultRefs {
			canvas, err := fetchCanvasAtRef(repo, ref)
			if err == nil {
				repo.Ref = ref
				return canvas, ref, nil
			}

			lastErr = err
		}

		if lastErr != nil && strings.Contains(lastErr.Error(), "not found") {
			return nil, "", fmt.Errorf("canvas.yaml not found on main or master branch")
		}

		if lastErr != nil {
			return nil, "", lastErr
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

	canvasJSON, err := canvasJSONFromResource(jsonData)
	if err != nil {
		return nil, err
	}

	var canvas pb.Canvas
	if err := protojson.Unmarshal(canvasJSON, &canvas); err != nil {
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

func canvasJSONFromResource(jsonData []byte) ([]byte, error) {
	var resource map[string]json.RawMessage
	if err := json.Unmarshal(jsonData, &resource); err != nil {
		return nil, fmt.Errorf("parse canvas yaml: %w", err)
	}

	if kindRaw, ok := resource["kind"]; ok {
		var kind string
		if err := json.Unmarshal(kindRaw, &kind); err != nil {
			return nil, fmt.Errorf("parse canvas definition: %w", err)
		}

		if kind != "" && kind != "Canvas" {
			return nil, fmt.Errorf("unsupported resource kind %q", kind)
		}
	}

	canvasPayload := make(map[string]json.RawMessage)
	if metadata, ok := resource["metadata"]; ok {
		canvasPayload["metadata"] = metadata
	}
	if spec, ok := resource["spec"]; ok {
		canvasPayload["spec"] = spec
	}

	if len(canvasPayload) == 0 {
		return jsonData, nil
	}

	canvasJSON, err := json.Marshal(canvasPayload)
	if err != nil {
		return nil, fmt.Errorf("parse canvas definition: %w", err)
	}

	return canvasJSON, nil
}
