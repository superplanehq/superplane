package codestorage

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	codestorage "github.com/pierrecomputer/sdk/packages/code-storage-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/git/provider"
)

const testKey = "-----BEGIN PRIVATE KEY-----\nMIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgy3DPdzzsP6tOOvmorjbx6L7mpFmKKL2hNWNW3urkN8ehRANCAAQ7/DPhGH3kaWl0YEIO+W9WmhyCclDGyTh6suablSura7ZDG8hpm3oNsq/ykC3Scfsw6ZTuuVuLlXKV/be/Xr0d\n-----END PRIVATE KEY-----\n"

const commitPackAck = `{"commit":{"commit_sha":"sha-1"},"result":{"success":true,"branch":"main","new_sha":"sha-1"}}`

type commitPackMetadata struct {
	Metadata struct {
		TargetBranch  string `json:"target_branch"`
		CommitMessage string `json:"commit_message"`
		Author        struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
		Files []struct {
			Path      string `json:"path"`
			Operation string `json:"operation"`
		} `json:"files"`
	} `json:"metadata"`
}

func newTestProvider(t *testing.T, handler http.HandlerFunc) *Provider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := codestorage.NewClient(codestorage.Options{
		Name:       "test-org",
		Key:        testKey,
		APIBaseURL: server.URL,
	})
	require.NoError(t, err)

	return &Provider{client: client, defaultBranch: "main"}
}

func writeJSON(w http.ResponseWriter, payload string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(payload))
}

func parseCommitPackMetadata(t *testing.T, body string) commitPackMetadata {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(body), "\n")
	require.NotEmpty(t, lines)

	var payload commitPackMetadata
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &payload))
	return payload
}

func findFile(metadata commitPackMetadata, path string) (string, bool) {
	for _, file := range metadata.Metadata.Files {
		if file.Path == path {
			return file.Operation, true
		}
	}
	return "", false
}

func TestNewProviderRequiresName(t *testing.T) {
	t.Setenv("GIT_STORAGE_CODE_STORAGE_NAME", "")
	t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY", "")
	t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY_PATH", "")
	_, err := NewProvider()
	require.Error(t, err)
	require.Contains(t, err.Error(), "GIT_STORAGE_CODE_STORAGE_NAME is required")
}

func TestNewProviderReadsKeyFromFile(t *testing.T) {
	t.Setenv("GIT_STORAGE_CODE_STORAGE_NAME", "test-org")
	t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY", "")
	t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY_PATH", "")

	keyFile := filepath.Join(t.TempDir(), "key.pem")
	require.NoError(t, os.WriteFile(keyFile, []byte(testKey), 0o600))
	t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY_PATH", keyFile)

	p, err := NewProvider()
	require.NoError(t, err)
	require.Equal(t, provider.CodeStorageProvider, p.Name())
}

func TestGetPrivateKeyPrefersEnv(t *testing.T) {
	t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY", "env-key")
	t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY_PATH", "/nonexistent/key.pem")

	key, err := getPrivateKey()
	require.NoError(t, err)
	require.Equal(t, "env-key", string(key))
}

func TestGetPrivateKeyMissing(t *testing.T) {
	t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY", "")
	t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY_PATH", "")
	_, err := getPrivateKey()
	require.Error(t, err)
	require.Contains(t, err.Error(), "either GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY")
}

func TestGetPrivateKeyReadError(t *testing.T) {
	t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY", "")
	t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY_PATH", "/nonexistent/key.pem")

	_, err := getPrivateKey()
	require.Error(t, err)
	require.Contains(t, err.Error(), "error reading private key")
}

func TestGetRepositoryID(t *testing.T) {
	p := &Provider{}
	orgID := uuid.New()
	canvasID := uuid.New()

	require.Equal(
		t,
		"orgs/"+orgID.String()+"/canvases/"+canvasID.String(),
		p.GetRepositoryID(provider.RepositoryOptions{
			OrganizationID: orgID,
			CanvasID:       canvasID,
		}),
	)
}

func TestCreateRepository(t *testing.T) {
	var requests []string
	var commitBody string
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos":
			writeJSON(w, `{}`)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repos/commit-pack"):
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			commitBody = string(body)
			writeJSON(w, commitPackAck)
		default:
			http.NotFound(w, r)
		}
	})

	repo, err := p.CreateRepository(context.Background(), "repo-1")
	require.NoError(t, err)
	require.Equal(t, "repo-1", repo.ID)
	require.Equal(t, []string{"POST /api/v1/repos", "POST /api/v1/repos/commit-pack"}, requests)

	metadata := parseCommitPackMetadata(t, commitBody)
	require.Equal(t, "main", metadata.Metadata.TargetBranch)
	require.NotEmpty(t, metadata.Metadata.CommitMessage)
	require.NotEmpty(t, metadata.Metadata.Author.Name)
	operation, ok := findFile(metadata, "README.md")
	require.True(t, ok)
	require.Equal(t, "upsert", operation)
}

func TestCommitValidationBeforeClientUse(t *testing.T) {
	p := &Provider{}
	_, err := p.Commit(context.Background(), "repo-1", provider.CommitOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "commit message is required")
}

func TestCommitSendsOperations(t *testing.T) {
	var requests []string
	var commitBody string
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		commitBody = string(body)
		writeJSON(w, commitPackAck)
	})

	sha, err := p.Commit(context.Background(), "repo-1", provider.CommitOptions{
		Message: "update files",
		Author:  provider.CommitAuthor{Name: "bot", Email: "bot@example.com"},
		Operations: []provider.FileOperation{
			{Path: "a.txt", Content: strings.NewReader("hello"), SizeBytes: 5},
			{Path: "b.txt", Delete: true},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "sha-1", sha)
	require.Equal(t, []string{"POST /api/v1/repos/commit-pack"}, requests)

	metadata := parseCommitPackMetadata(t, commitBody)
	require.Equal(t, "update files", metadata.Metadata.CommitMessage)
	require.Equal(t, "bot", metadata.Metadata.Author.Name)

	operation, ok := findFile(metadata, "a.txt")
	require.True(t, ok)
	require.Equal(t, "upsert", operation)

	operation, ok = findFile(metadata, "b.txt")
	require.True(t, ok)
	require.Equal(t, "delete", operation)
}

func TestDeleteRepositoryNoopWhenMissing(t *testing.T) {
	var requests []string
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/repo" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.NotFound(w, r)
	})

	require.NoError(t, p.DeleteRepository(context.Background(), "repo-1"))
	require.Equal(t, []string{"GET /api/v1/repo"}, requests)
}

func TestDeleteRepository(t *testing.T) {
	var requests []string
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repo":
			writeJSON(w, `{"default_branch":"main"}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/repos/delete":
			writeJSON(w, `{"repo_id":"repo-1"}`)
		default:
			http.NotFound(w, r)
		}
	})

	require.NoError(t, p.DeleteRepository(context.Background(), "repo-1"))
	require.Equal(t, []string{"GET /api/v1/repo", "DELETE /api/v1/repos/delete"}, requests)
}

func TestListBranchesPaginationAndFilter(t *testing.T) {
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/repos/branches", r.URL.Path)
		switch r.URL.Query().Get("cursor") {
		case "":
			writeJSON(w, `{"branches":[{"name":"feature-b"},{"name":"main"}],"has_more":true,"next_cursor":"c2"}`)
		case "c2":
			writeJSON(w, `{"branches":[{"name":"feature-a"},{"name":"zeta"}],"has_more":false}`)
		default:
			t.Fatalf("unexpected cursor %q", r.URL.Query().Get("cursor"))
		}
	})

	names, err := p.ListBranches(context.Background(), "repo-1", "feature-")
	require.NoError(t, err)
	require.Equal(t, []string{"feature-a", "feature-b"}, names)
}
