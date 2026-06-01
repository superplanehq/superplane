package anthropic

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultResourceSourcesForSkillsBaseURL(t *testing.T) {
	sources, err := defaultResourceSourcesForSkillsBaseURL("https://example.test/root/")
	require.NoError(t, err)

	byMountPath := make(map[string]resourceSource, len(sources))
	for _, source := range sources {
		byMountPath[source.MountPath] = source
	}

	assert.Contains(t, byMountPath, "ref/skills/superplane-app-builder/SKILL.md")
	assert.Equal(
		t,
		"https://example.test/root/skills/superplane-app-builder/SKILL.md",
		byMountPath["ref/skills/superplane-app-builder/SKILL.md"].SourceURL,
	)
	assert.Equal(
		t,
		"skills/superplane-app-builder/SKILL.md",
		byMountPath["ref/skills/superplane-app-builder/SKILL.md"].SourceKey,
	)
	assert.Equal(
		t,
		"https://example.test/root/skills/superplane-cli/references/canvas-yaml-spec.md",
		byMountPath["ref/skills/superplane-cli/references/canvas-yaml-spec.md"].SourceURL,
	)
	assert.Equal(
		t,
		"https://example.test/root/skills/superplane-cli/references/console-yaml-spec.md",
		byMountPath["ref/skills/superplane-cli/references/console-yaml-spec.md"].SourceURL,
	)
	assert.Equal(
		t,
		"https://raw.githubusercontent.com/superplanehq/superplane/main/docs/prd/console-and-widgets.md",
		byMountPath["ref/docs/prd/console-and-widgets.md"].SourceURL,
	)
	assert.Equal(
		t,
		"docs/prd/console-and-widgets.md",
		byMountPath["ref/docs/prd/console-and-widgets.md"].SourceKey,
	)
	assert.Equal(
		t,
		"https://example.test/root/skills/superplane-monitor/SKILL.md",
		byMountPath["ref/skills/superplane-monitor/SKILL.md"].SourceURL,
	)
	assert.Contains(t, byMountPath, "ref/components/Index.md")
	assert.Contains(t, string(byMountPath["ref/components/Index.md"].SourceData), "aws.ec2.createInstance")

	componentSourceCount := 0
	for _, source := range sources {
		if !strings.HasPrefix(source.MountPath, "ref/components/") {
			continue
		}

		componentSourceCount++
		assert.NotEmpty(t, source.SourceData)
		assert.Empty(t, source.SourcePath)
		assert.Equal(t, filepath.ToSlash(filepath.Join("docs", "components", filepath.Base(source.MountPath))), source.SourceKey)
		assert.Empty(t, source.SourceURL)
	}
	assert.NotZero(t, componentSourceCount)
}

func TestResourceFilenameUsesSourceKeyAndContentHash(t *testing.T) {
	t.Parallel()

	filenameA := resourceFilename("docs/components/Telegram.mdx", []byte("same-content"))
	filenameB := resourceFilename("skills/superplane-cli/SKILL.md", []byte("same-content"))
	filenameUpdated := resourceFilename("docs/components/Telegram.mdx", []byte("updated-content"))

	assert.NotEqual(t, filenameA, filenameB)
	assert.NotEqual(t, filenameA, filenameUpdated)
	assert.Contains(t, filenameA, "TELEGRAM_")
	assert.Contains(t, filenameB, "SUPERPLANE_CLI_SKILL_")
}

func TestLoadSessionResourcesUploadsRemoteAndLocalSources(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	localSourcePath := filepath.Join(tempDir, "local.md")
	require.NoError(t, os.WriteFile(localSourcePath, []byte("local-content"), 0o600))

	uploadedContents := map[string]string{}
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/files":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[],"has_more":false,"last_id":""}`))
		case r.Method == http.MethodPost && r.URL.Path == "/files":
			filename, content := readUploadedFile(t, r)
			uploadedContents[filename] = content
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"id":"file-%s","filename":"%s"}`, filename, filename)))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer apiServer.Close()

	rawServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/skills/remote.md", r.URL.Path)
		_, _ = w.Write([]byte("remote-content"))
	}))
	defer rawServer.Close()

	resources, err := loadSessionResources(context.Background(), Config{
		APIKey:  "test-key",
		BaseURL: apiServer.URL,
	}, []resourceSource{
		{
			MountPath: "ref/skills/remote.md",
			SourceKey: "skills/remote.md",
			SourceURL: rawServer.URL + "/skills/remote.md",
		},
		{
			MountPath:  "ref/components/local.md",
			SourceKey:  "docs/components/local.md",
			SourcePath: localSourcePath,
		},
	})
	require.NoError(t, err)

	remoteFilename := resourceFilename("skills/remote.md", []byte("remote-content"))
	localFilename := resourceFilename("docs/components/local.md", []byte("local-content"))
	assert.Equal(t, map[string]string{
		remoteFilename: "remote-content",
		localFilename:  "local-content",
	}, uploadedContents)
	assert.Equal(t, "file-"+remoteFilename, resources[0].FileID)
	assert.Equal(t, "ref/skills/remote.md", resources[0].MountPath)
	assert.Equal(t, "file-"+localFilename, resources[1].FileID)
	assert.Equal(t, "ref/components/local.md", resources[1].MountPath)
}

func TestLoadSessionResourcesReusesExistingManagedFiles(t *testing.T) {
	t.Parallel()

	const (
		sourceKey   = "skills/remote.md"
		currentBody = "remote-content"
	)

	currentFilename := resourceFilename(sourceKey, []byte(currentBody))

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/files":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{
				"data":[
					{"id":"current-id","filename":"%s"}
				],
				"has_more":false,
				"last_id":""
			}`, currentFilename)))
		case r.Method == http.MethodPost && r.URL.Path == "/files":
			t.Fatalf("unexpected upload request")
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer apiServer.Close()

	rawServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/skills/remote.md", r.URL.Path)
		_, _ = w.Write([]byte(currentBody))
	}))
	defer rawServer.Close()

	resources, err := loadSessionResources(context.Background(), Config{
		APIKey:  "test-key",
		BaseURL: apiServer.URL,
	}, []resourceSource{
		{
			MountPath: "ref/skills/remote.md",
			SourceKey: sourceKey,
			SourceURL: rawServer.URL + "/skills/remote.md",
		},
	})
	require.NoError(t, err)

	require.Len(t, resources, 1)
	assert.Equal(t, "current-id", resources[0].FileID)
	assert.Equal(t, "ref/skills/remote.md", resources[0].MountPath)
}

func readUploadedFile(t *testing.T, r *http.Request) (string, string) {
	t.Helper()

	reader, err := r.MultipartReader()
	require.NoError(t, err)

	part, err := reader.NextPart()
	require.NoError(t, err)
	require.Equal(t, "file", part.FormName())

	content, err := io.ReadAll(part)
	require.NoError(t, err)

	require.NoError(t, consumeMultipartRemainder(reader))
	return part.FileName(), string(content)
}

func consumeMultipartRemainder(reader *multipart.Reader) error {
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err := io.Copy(io.Discard, part); err != nil {
			return err
		}
	}
}
