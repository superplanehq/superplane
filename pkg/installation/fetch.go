package installation

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	canvasyaml "github.com/superplanehq/superplane/pkg/canvas/yaml"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

const (
	canvasFileName  = "canvas.yaml"
	consoleFileName = "console.yaml"
)

// errFileNotFound is returned (wrapped) by fetchURL when the upstream
// raw.githubusercontent.com responds with 404. Callers that treat a missing
// file as a non-error (for example, the optional console.yaml in installable
// apps) can detect it with errors.Is.
var errFileNotFound = errors.New("file not found")

var defaultRefs = []string{"main", "master"}

// httpGet performs the raw file GET. It is a package-level variable so tests
// can stub upstream responses instead of depending on mutable external repos.
var httpGet = func(rawURL string) (*http.Response, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	return client.Get(rawURL)
}

// fetchRawCanvasFile resolves the ref and downloads the raw canvas.yaml bytes.
// It tries defaultRefs (main, master) when repo.Ref is empty. On success it
// sets repo.Ref to the resolved ref and returns the raw YAML content.
func fetchRawCanvasFile(repo *Repository) ([]byte, string, error) {
	if repo.Ref == "" {
		for _, ref := range defaultRefs {
			body, fetchErr := fetchURL(rawFileURL(repo, ref, canvasFileName))
			if fetchErr == nil {
				repo.Ref = ref
				return body, ref, nil
			}
			if !errors.Is(fetchErr, errFileNotFound) {
				return nil, "", fetchErr
			}
		}
		return nil, "", fmt.Errorf("canvas.yaml not found on main or master branch")
	}

	body, err := fetchURL(rawFileURL(repo, repo.Ref, canvasFileName))
	if err != nil {
		return nil, "", err
	}
	return body, repo.Ref, nil
}

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
	body, err := fetchURL(rawFileURL(repo, ref, canvasFileName))
	if err != nil {
		return nil, err
	}

	return parseCanvasYAML(body)
}

// FetchConsole loads and parses an optional console.yaml from a public GitHub
// repository at the given ref. The console is opt-in: a missing file returns
// (nil, nil) so callers can install the app without one. Parse and validation
// errors from models.ConsoleFromYML are wrapped and surfaced to the caller.
//
// Callers must pass a non-empty ref. Resolve it via FetchCanvas first so the
// canvas and console are read from the same commit.
func FetchConsole(repo *Repository, ref string) (*models.ConsoleYAML, error) {
	if ref == "" {
		return nil, fmt.Errorf("console fetch requires a resolved ref")
	}

	body, err := fetchURL(rawFileURL(repo, ref, consoleFileName))
	if err != nil {
		if errors.Is(err, errFileNotFound) {
			return nil, nil
		}
		return nil, err
	}

	console, err := models.ConsoleFromYML(body)
	if err != nil {
		return nil, fmt.Errorf("parse console yaml: %w", err)
	}

	return console, nil
}

func rawFileURL(repo *Repository, ref, filename string) string {
	return fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/%s/%s",
		repo.Owner,
		repo.Name,
		ref,
		filename,
	)
}

func fetchURL(rawURL string) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if parsed.Scheme != "https" || parsed.Host != "raw.githubusercontent.com" {
		return nil, fmt.Errorf("unsupported fetch host %q", parsed.Host)
	}

	response, err := httpGet(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s not found: %w", rawURL, errFileNotFound)
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
	canvas, err := canvasyaml.ParseCanvasResource(data)
	if err != nil {
		return nil, err
	}

	canvas.Metadata.Id = ""

	return canvas, nil
}
