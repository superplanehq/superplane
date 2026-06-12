package anthropic

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/docs"
)

//go:embed static/rich-ui-widgets.md
var richUIWidgetsContent []byte

const skillsRepoRawBaseURL = "https://raw.githubusercontent.com/superplanehq/skills/main"
const superplaneRepoRawBaseURL = "https://raw.githubusercontent.com/superplanehq/superplane/main"

var resourceNameCamelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

type resourceSource struct {
	MountPath  string
	SourceKey  string
	SourceData []byte
	SourcePath string
	SourceURL  string
}

type resolvedResourceSource struct {
	Content   []byte
	Filename  string
	MountPath string
}

func LoadDefaultSessionResources(ctx context.Context, cfg Config) ([]agents.FileResource, error) {
	sources, err := defaultResourceSources()
	if err != nil {
		return nil, err
	}

	return loadSessionResources(ctx, cfg, sources)
}

func loadSessionResources(ctx context.Context, cfg Config, sources []resourceSource) ([]agents.FileResource, error) {
	client, err := newClient(cfg)
	if err != nil {
		return nil, err
	}

	existing, err := client.listFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	resolvedSources, err := resolveResourceSources(ctx, sources)
	if err != nil {
		return nil, err
	}

	return syncSessionResources(ctx, client, existing, resolvedSources)
}

func defaultResourceSources() ([]resourceSource, error) {
	return defaultResourceSourcesForSkillsBaseURL(skillsRepoRawBaseURL)
}

func defaultResourceSourcesForSkillsBaseURL(skillsBaseURL string) ([]resourceSource, error) {
	sources := []resourceSource{
		{
			MountPath: "ref/skills/superplane-app-builder/SKILL.md",
			SourceKey: filepath.ToSlash(filepath.Join("skills", "superplane-app-builder", "SKILL.md")),
			SourceURL: skillsRawURL(
				skillsBaseURL,
				"skills",
				"superplane-app-builder",
				"SKILL.md",
			),
		},
		{
			MountPath: "ref/skills/superplane-app-builder/references/components-and-triggers.md",
			SourceKey: filepath.ToSlash(filepath.Join("skills", "superplane-app-builder", "references", "components-and-triggers.md")),
			SourceURL: skillsRawURL(
				skillsBaseURL,
				"skills",
				"superplane-app-builder",
				"references",
				"components-and-triggers.md",
			),
		},
		{
			MountPath: "ref/skills/superplane-cli/references/canvas-yaml-spec.md",
			SourceKey: filepath.ToSlash(filepath.Join("skills", "superplane-cli", "references", "canvas-yaml-spec.md")),
			SourceURL: skillsRawURL(
				skillsBaseURL,
				"skills",
				"superplane-cli",
				"references",
				"canvas-yaml-spec.md",
			),
		},
		{
			MountPath: "ref/skills/superplane-cli/references/console-yaml-spec.md",
			SourceKey: filepath.ToSlash(filepath.Join("skills", "superplane-cli", "references", "console-yaml-spec.md")),
			SourceURL: skillsRawURL(
				skillsBaseURL,
				"skills",
				"superplane-cli",
				"references",
				"console-yaml-spec.md",
			),
		},
		{
			MountPath: "ref/skills/superplane-monitor/SKILL.md",
			SourceKey: filepath.ToSlash(filepath.Join("skills", "superplane-monitor", "SKILL.md")),
			SourceURL: skillsRawURL(
				skillsBaseURL,
				"skills",
				"superplane-monitor",
				"SKILL.md",
			),
		},
		{
			MountPath: "ref/docs/prd/console-and-widgets.md",
			SourceKey: filepath.ToSlash(filepath.Join("docs", "prd", "console-and-widgets.md")),
			SourceURL: skillsRawURL(
				superplaneRepoRawBaseURL,
				"docs",
				"prd",
				"console-and-widgets.md",
			),
		},
	}

	// Static files bundled in the binary
	staticSources := []resourceSource{
		{
			MountPath:  "ref/rich-ui-widgets.md",
			SourceKey:  "static/rich-ui-widgets.md",
			SourceData: richUIWidgetsContent,
		},
	}
	sources = append(sources, staticSources...)

	componentSources, err := componentResourceSources()
	if err != nil {
		return nil, err
	}

	return append(sources, componentSources...), nil
}

func componentResourceSources() ([]resourceSource, error) {
	files, err := docs.GenerateFiles()
	if err != nil {
		return nil, err
	}

	indexFile, err := docs.GenerateComponentIndexFile()
	if err != nil {
		return nil, err
	}

	sources := make([]resourceSource, 0, len(files)+1)
	sources = append(sources, resourceSource{
		MountPath:  filepath.ToSlash(filepath.Join("ref", "components", indexFile.Name)),
		SourceKey:  filepath.ToSlash(filepath.Join("docs", "components", indexFile.Name)),
		SourceData: indexFile.Content,
	})

	for _, file := range files {
		sources = append(sources, resourceSource{
			MountPath:  filepath.ToSlash(filepath.Join("ref", "components", file.Name)),
			SourceKey:  filepath.ToSlash(filepath.Join("docs", "components", file.Name)),
			SourceData: file.Content,
		})
	}

	sort.Slice(sources, func(i, j int) bool {
		return sources[i].SourceKey < sources[j].SourceKey
	})

	return sources, nil
}

func resolveResourceSource(ctx context.Context, source resourceSource) (resolvedResourceSource, error) {
	content, err := loadResourceContent(ctx, source)
	if err != nil {
		return resolvedResourceSource{}, err
	}

	return resolvedResourceSource{
		Content:   content,
		Filename:  resourceFilename(source.SourceKey, content),
		MountPath: source.MountPath,
	}, nil
}

func resolveResourceSources(ctx context.Context, sources []resourceSource) ([]resolvedResourceSource, error) {
	resolvedSources := make([]resolvedResourceSource, 0, len(sources))
	for _, source := range sources {
		resolved, err := resolveResourceSource(ctx, source)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", source.MountPath, err)
		}
		resolvedSources = append(resolvedSources, resolved)
	}

	return resolvedSources, nil
}

func syncSessionResources(
	ctx context.Context,
	client *Client,
	existing []fileMetadata,
	resolvedSources []resolvedResourceSource,
) ([]agents.FileResource, error) {
	fileByFilename := indexFilesByFilename(existing)
	resources := make([]agents.FileResource, 0, len(resolvedSources))

	for _, resolved := range resolvedSources {
		file, err := ensureSessionResourceFile(ctx, client, fileByFilename, resolved)
		if err != nil {
			return nil, err
		}

		resources = append(resources, agents.FileResource{
			FileID:    file.ID,
			MountPath: resolved.MountPath,
		})
	}

	return resources, nil
}

func indexFilesByFilename(files []fileMetadata) map[string]fileMetadata {
	indexed := make(map[string]fileMetadata, len(files))
	for _, file := range files {
		if file.Filename == "" || file.ID == "" {
			continue
		}
		if _, found := indexed[file.Filename]; found {
			continue
		}
		indexed[file.Filename] = file
	}

	return indexed
}

func ensureSessionResourceFile(
	ctx context.Context,
	client *Client,
	fileByFilename map[string]fileMetadata,
	resolved resolvedResourceSource,
) (fileMetadata, error) {
	if existingFile := fileByFilename[resolved.Filename]; existingFile.ID != "" {
		return existingFile, nil
	}

	file, err := client.uploadFileContent(ctx, resolved.Content, resolved.Filename)
	if err != nil {
		return fileMetadata{}, fmt.Errorf("upload %s: %w", resolved.Filename, err)
	}
	fileByFilename[resolved.Filename] = file

	return file, nil
}

func loadResourceContent(ctx context.Context, source resourceSource) ([]byte, error) {
	if len(source.SourceData) > 0 {
		return append([]byte(nil), source.SourceData...), nil
	}

	if source.SourcePath != "" {
		content, err := os.ReadFile(filepath.Clean(source.SourcePath))
		if err != nil {
			return nil, fmt.Errorf("read source file: %w", err)
		}
		return content, nil
	}

	return fetchResourceContent(ctx, source.SourceURL)
}

func shortResourceHash(sourceKey string, content []byte) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte(sourceKey))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write(content)

	return hex.EncodeToString(digest.Sum(nil))[:12]
}

func resourceFilename(sourceKey string, content []byte) string {
	return resourceName(sourceKey) + "_" + shortResourceHash(sourceKey, content)
}

func resourceName(sourceKey string) string {
	sourceKey = filepath.ToSlash(sourceKey)
	parts := strings.Split(sourceKey, "/")

	switch {
	case len(parts) >= 3 && parts[0] == "docs" && parts[1] == "components":
		return normalizeResourceNameSegment(strings.TrimSuffix(parts[len(parts)-1], filepath.Ext(parts[len(parts)-1])))
	case len(parts) >= 3 && parts[0] == "skills":
		relevant := make([]string, 0, len(parts)-1)
		for _, part := range parts[1:] {
			if part == "references" {
				continue
			}
			relevant = append(relevant, strings.TrimSuffix(part, filepath.Ext(part)))
		}
		return normalizeResourceNameSegment(strings.Join(relevant, "_"))
	default:
		return normalizeResourceNameSegment(strings.TrimSuffix(filepath.Base(sourceKey), filepath.Ext(sourceKey)))
	}
}

func normalizeResourceNameSegment(value string) string {
	withBoundaries := resourceNameCamelBoundary.ReplaceAllString(strings.TrimSpace(value), "$1_$2")
	replacer := strings.NewReplacer(
		"-", "_",
		".", "_",
		" ", "_",
		"/", "_",
	)
	normalized := replacer.Replace(withBoundaries)
	normalized = strings.ToUpper(normalized)

	var builder strings.Builder
	builder.Grow(len(normalized))
	lastUnderscore := false
	for _, r := range normalized {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if lastUnderscore {
			continue
		}
		builder.WriteByte('_')
		lastUnderscore = true
	}

	return strings.Trim(builder.String(), "_")
}

func fetchResourceContent(ctx context.Context, sourceURL string) ([]byte, error) {
	if strings.TrimSpace(sourceURL) == "" {
		return nil, fmt.Errorf("resource source URL is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build source request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request source: %w", err)
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read source body: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("source returned %d: %s", resp.StatusCode, truncate(string(content), 500))
	}

	return content, nil
}

func skillsRawURL(baseURL string, parts ...string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	return baseURL + "/" + filepath.ToSlash(filepath.Join(parts...))
}
