package supergit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/git/provider"
)

func TestCreateRepositoryCreatesInitialCommit(t *testing.T) {
	var commitBody string
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/repos":
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"id":"repo-id","default_branch":"main"}`))
			require.NoError(t, err)

		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/commits"):
			require.Equal(t, "application/x-ndjson", r.Header.Get("Content-Type"))
			body := readRequestBody(t, r)
			commitBody = body
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"commit":{"commit_sha":"abc123"}}`))
			require.NoError(t, err)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	p := &Provider{
		client:        NewClient(server.URL),
		defaultBranch: "main",
	}

	repoID := p.GetRepositoryID(provider.RepositoryOptions{
		OrganizationID: uuid.New(),
		CanvasID:       uuid.New(),
	})

	repo, err := p.CreateRepository(context.Background(), repoID)
	require.NoError(t, err)
	require.Equal(t, "repo-id", repo.ID)
	require.Equal(t, []string{
		"POST /repos",
		"POST /repos/repo-id/commits",
	}, requests)

	lines := strings.Split(strings.TrimSpace(commitBody), "\n")
	require.Len(t, lines, 2)

	var metadata struct {
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
				ContentID string `json:"content_id"`
				Mode      string `json:"mode"`
			} `json:"files"`
		} `json:"metadata"`
	}
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &metadata))
	require.Equal(t, "main", metadata.Metadata.TargetBranch)
	require.Equal(t, "Initialize repository", metadata.Metadata.CommitMessage)
	require.Equal(t, "SuperPlane", metadata.Metadata.Author.Name)
	require.Equal(t, "bot@superplane.local", metadata.Metadata.Author.Email)
	require.Len(t, metadata.Metadata.Files, 1)
	require.Equal(t, "README.md", metadata.Metadata.Files[0].Path)
	require.Equal(t, "upsert", metadata.Metadata.Files[0].Operation)
	require.Equal(t, "blob-1", metadata.Metadata.Files[0].ContentID)
	require.Equal(t, "100644", metadata.Metadata.Files[0].Mode)

	var chunk struct {
		BlobChunk struct {
			ContentID string `json:"content_id"`
			Data      string `json:"data"`
			EOF       bool   `json:"eof"`
		} `json:"blob_chunk"`
	}
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &chunk))
	require.Equal(t, "blob-1", chunk.BlobChunk.ContentID)
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte("")), chunk.BlobChunk.Data)
	require.True(t, chunk.BlobChunk.EOF)
}

func readRequestBody(t *testing.T, r *http.Request) string {
	t.Helper()

	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	return string(body)
}
