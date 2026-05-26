package canvasstorage

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	codestorage "github.com/pierrecomputer/sdk/packages/code-storage-go"
)

const testCodeStorageKey = "-----BEGIN PRIVATE KEY-----\nMIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgy3DPdzzsP6tOOvmorjbx6L7mpFmKKL2hNWNW3urkN8ehRANCAAQ7/DPhGH3kaWl0YEIO+W9WmhyCclDGyTh6suablSura7ZDG8hpm3oNsq/ykC3Scfsw6ZTuuVuLlXKV/be/Xr0d\n-----END PRIVATE KEY-----\n"

func TestCodeStorageProviderEnsureRepositoryInitializesReadme(t *testing.T) {
	var createBody map[string]any
	var commitMetadata struct {
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
	blobChunks := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/repos":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected create repo method: %s", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create repo body: %v", err)
			}
			_, _ = w.Write([]byte(`{"repo_id":"repo","url":"https://repo.git"}`))

		case "/api/v1/repos/commit-pack":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected commit method: %s", r.Method)
			}
			scanner := bufio.NewScanner(r.Body)
			if !scanner.Scan() {
				t.Fatalf("expected commit metadata")
			}
			if err := json.Unmarshal(scanner.Bytes(), &commitMetadata); err != nil {
				t.Fatalf("decode commit metadata: %v", err)
			}
			for scanner.Scan() {
				blobChunks++
			}
			if err := scanner.Err(); err != nil {
				t.Fatalf("scan commit body: %v", err)
			}
			_, _ = w.Write([]byte(`{
				"commit":{"commit_sha":"abc123","tree_sha":"tree123","target_branch":"main","pack_bytes":1,"blob_count":1},
				"result":{"branch":"main","old_sha":"","new_sha":"abc123","success":true,"status":"ok"}
			}`))

		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := codestorage.NewClient(codestorage.Options{Name: "acme", Key: testCodeStorageKey, APIBaseURL: server.URL})
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	provider := &CodeStorageProvider{client: client, defaultBranch: "main"}

	repo, err := provider.EnsureRepository(context.Background(), RepositorySpec{
		OrganizationID: uuid.New(),
		CanvasID:       uuid.New(),
	})
	if err != nil {
		t.Fatalf("ensure repository: %v", err)
	}

	if repo.DefaultBranch != "main" {
		t.Fatalf("expected default branch main, got %q", repo.DefaultBranch)
	}
	if createBody["default_branch"] != "main" {
		t.Fatalf("expected default branch main, got %#v", createBody["default_branch"])
	}
	if commitMetadata.Metadata.TargetBranch != "main" ||
		commitMetadata.Metadata.CommitMessage != initialRepositoryCommitMessage ||
		commitMetadata.Metadata.Author.Name != initialRepositoryAuthorName ||
		commitMetadata.Metadata.Author.Email != initialRepositoryAuthorEmail {
		t.Fatalf("unexpected commit metadata: %+v", commitMetadata.Metadata)
	}
	if len(commitMetadata.Metadata.Files) != 1 ||
		commitMetadata.Metadata.Files[0].Path != initialRepositoryFilePath ||
		commitMetadata.Metadata.Files[0].Operation != "upsert" {
		t.Fatalf("unexpected files: %+v", commitMetadata.Metadata.Files)
	}
	if blobChunks != 1 {
		t.Fatalf("expected one blob chunk for empty README.md, got %d", blobChunks)
	}
}

func TestCodeStorageProviderEnsureRepositoryDoesNotInitializeExistingRepo(t *testing.T) {
	commitCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/repos":
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"repository already exists"}`))

		case "/api/v1/repo":
			_, _ = w.Write([]byte(`{"default_branch":"main","created_at":"2026-05-25T12:00:00Z"}`))

		case "/api/v1/repos/commit-pack":
			commitCalled = true
			w.WriteHeader(http.StatusInternalServerError)

		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := codestorage.NewClient(codestorage.Options{Name: "acme", Key: testCodeStorageKey, APIBaseURL: server.URL})
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	provider := &CodeStorageProvider{client: client, defaultBranch: "main"}

	_, err = provider.EnsureRepository(context.Background(), RepositorySpec{
		OrganizationID: uuid.New(),
		CanvasID:       uuid.New(),
	})
	if err != nil {
		t.Fatalf("ensure repository: %v", err)
	}
	if commitCalled {
		t.Fatal("existing repository should not be initialized")
	}
}

func TestCodeStorageProviderDeleteRepository(t *testing.T) {
	canvasID := uuid.New()
	repoID := CanvasRepoID(uuid.New(), canvasID)
	deleteCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/delete" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected delete repo method: %s", r.Method)
		}

		deleteCalled = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"repo_id":"deleted","message":"ok"}`))
	}))
	defer server.Close()

	client, err := codestorage.NewClient(codestorage.Options{Name: "acme", Key: testCodeStorageKey, APIBaseURL: server.URL})
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	provider := &CodeStorageProvider{client: client, defaultBranch: "main"}

	err = provider.DeleteRepository(context.Background(), RepositoryRef{RepoID: repoID, DefaultBranch: "main"})
	if err != nil {
		t.Fatalf("delete repository: %v", err)
	}
	if !deleteCalled {
		t.Fatal("expected delete repo to be called")
	}
}

func TestCodeStorageProviderDeleteRepositoryAlreadyDeleted(t *testing.T) {
	repoID := CanvasRepoID(uuid.New(), uuid.New())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := codestorage.NewClient(codestorage.Options{Name: "acme", Key: testCodeStorageKey, APIBaseURL: server.URL})
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	provider := &CodeStorageProvider{client: client, defaultBranch: "main"}

	err = provider.DeleteRepository(context.Background(), RepositoryRef{RepoID: repoID, DefaultBranch: "main"})
	if err != nil {
		t.Fatalf("delete repository: %v", err)
	}
}

func TestCodeStorageProviderCurrentHead(t *testing.T) {
	repoID := CanvasRepoID(uuid.New(), uuid.New())
	rawQuery := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/branches" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"branches":[
				{"name":"feature","head_sha":"feature-sha"},
				{"name":"main","head_sha":"main-sha"}
			],
			"has_more":false
		}`))
	}))
	defer server.Close()

	client, err := codestorage.NewClient(codestorage.Options{Name: "acme", Key: testCodeStorageKey, APIBaseURL: server.URL})
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	provider := &CodeStorageProvider{client: client, defaultBranch: "main"}

	head, err := provider.CurrentHead(context.Background(), RepositoryRef{RepoID: repoID, DefaultBranch: "main"}, "")
	if err != nil {
		t.Fatalf("current head: %v", err)
	}
	if head != "main-sha" {
		t.Fatalf("expected main head, got %q", head)
	}
	if !strings.Contains(rawQuery, "limit=100") {
		t.Fatalf("expected branch query limit, got %q", rawQuery)
	}
}

func TestCodeStorageProviderGitURLAndCredentials(t *testing.T) {
	client, err := codestorage.NewClient(codestorage.Options{
		Name:           "acme",
		Key:            testCodeStorageKey,
		StorageBaseURL: "acme.code.storage",
	})
	if err != nil {
		t.Fatalf("client error: %v", err)
	}
	provider := &CodeStorageProvider{client: client, defaultBranch: "main"}
	ref := RepositoryRef{RepoID: CanvasRepoID(uuid.New(), uuid.New()), DefaultBranch: "main"}

	gitURL, err := provider.GitURL(context.Background(), ref)
	if err != nil {
		t.Fatalf("git url: %v", err)
	}
	if strings.Contains(gitURL, "@") || !strings.HasPrefix(gitURL, "https://acme.code.storage/") {
		t.Fatalf("unexpected git url: %s", gitURL)
	}

	credentials, err := provider.GenerateGitCredentials(context.Background(), ref, GitCredentialsOptions{})
	if err != nil {
		t.Fatalf("generate credentials: %v", err)
	}
	if credentials.Username != "t" || credentials.Password == "" {
		t.Fatalf("unexpected credentials: %+v", credentials)
	}
}
