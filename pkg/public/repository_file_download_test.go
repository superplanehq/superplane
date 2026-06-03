package public

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func downloadFile(
	t *testing.T,
	server *Server,
	signer *jwt.Signer,
	organizationID uuid.UUID,
	accountID *uuid.UUID,
	canvasID string,
	path string,
) *httptest.ResponseRecorder {
	return downloadFileOnBranch(t, server, signer, organizationID, accountID, canvasID, path, "")
}

func downloadFileOnBranch(
	t *testing.T,
	server *Server,
	signer *jwt.Signer,
	organizationID uuid.UUID,
	accountID *uuid.UUID,
	canvasID string,
	path string,
	branch string,
) *httptest.ResponseRecorder {
	t.Helper()

	query := url.Values{}
	if path != "" {
		query.Set("path", path)
	}
	if branch != "" {
		query.Set("branch", branch)
	}

	requestURL := fmt.Sprintf("/api/v1/canvases/%s/repository/file", canvasID)
	if encoded := query.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}

	req := httptest.NewRequest(http.MethodGet, requestURL, nil)
	if accountID != nil {
		req.Header.Set("x-organization-id", organizationID.String())
		token, err := signer.Generate(accountID.String(), time.Hour)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	}

	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	return rec
}

func Test__RepositoryFileDownload(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test")
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		signer,
		support.NewOIDCProvider(),
		r.GitProvider,
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService,
		nil,
		false,
	)

	require.NoError(t, err)
	require.NoError(t, server.RegisterGRPCGateway("localhost:50051"))

	authenticated := &r.Account.ID

	t.Run("missing path -> bad request", func(t *testing.T) {
		canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
		response := downloadFile(t, server, signer, r.Organization.ID, authenticated, canvas.ID.String(), "")
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "path is required")
	})

	t.Run("invalid canvas id -> bad request", func(t *testing.T) {
		response := downloadFile(t, server, signer, r.Organization.ID, authenticated, "invalid-id", "README.md")
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "Invalid canvas_id")
	})

	t.Run("unauthenticated -> unauthorized", func(t *testing.T) {
		canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
		response := downloadFile(t, server, signer, r.Organization.ID, nil, canvas.ID.String(), "README.md")
		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("user without canvas access -> forbidden", func(t *testing.T) {
		canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)

		restrictedAccount, err := models.CreateAccount("restricted@example.com", "Restricted User")
		require.NoError(t, err)
		_, err = models.CreateUser(r.Organization.ID, restrictedAccount.ID, restrictedAccount.Email, restrictedAccount.Name)
		require.NoError(t, err)

		response := downloadFile(t, server, signer, r.Organization.ID, &restrictedAccount.ID, canvas.ID.String(), "README.md")
		assert.Equal(t, http.StatusForbidden, response.Code)
		assert.Contains(t, response.Body.String(), "Unauthorized")
	})

	t.Run("canvas not found -> not found", func(t *testing.T) {
		invalidID := uuid.NewString()
		response := downloadFile(t, server, signer, r.Organization.ID, authenticated, invalidID, "README.md")
		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Contains(t, response.Body.String(), "Canvas not found")
	})

	t.Run("repository not found -> not found", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		response := downloadFile(t, server, signer, r.Organization.ID, authenticated, canvas.ID.String(), "README.md")
		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Contains(t, response.Body.String(), "Repository not found")
	})

	t.Run("git provider error -> internal server error", func(t *testing.T) {
		canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
		response := downloadFile(t, server, signer, r.Organization.ID, authenticated, canvas.ID.String(), "missing.txt")
		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.Contains(t, response.Body.String(), "Failed to get file")
	})

	t.Run("returns file contents", func(t *testing.T) {
		canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
		headSHA, err := r.GitProvider.Head(context.Background(), repository.RepoID, "")
		require.NoError(t, err)

		_, err = r.GitProvider.Commit(context.Background(), repository.RepoID, git.CommitOptions{
			ExpectedHeadSHA: headSHA,
			Message:         "seed readme",
			Operations: []git.FileOperation{
				{
					Path:      "README.md",
					Content:   bytes.NewReader([]byte("updated readme")),
					SizeBytes: 14,
				},
			},
		})
		require.NoError(t, err)

		response := downloadFile(t, server, signer, r.Organization.ID, authenticated, canvas.ID.String(), "README.md")
		assert.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, "updated readme", response.Body.String())
		assert.Equal(t, "application/octet-stream", response.Header().Get("Content-Type"))
		assert.Equal(t, "nosniff", response.Header().Get("X-Content-Type-Options"))
		assert.Contains(t, response.Header().Get("Content-Disposition"), "README.md")
	})

	t.Run("reads file contents from the requested branch", func(t *testing.T) {
		canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
		ctx := context.Background()

		mainSHA, err := r.GitProvider.Head(ctx, repository.RepoID, "")
		require.NoError(t, err)

		_, err = r.GitProvider.Commit(ctx, repository.RepoID, git.CommitOptions{
			ExpectedHeadSHA: mainSHA,
			Message:         "seed readme on main",
			Operations: []git.FileOperation{
				{
					Path:      "README.md",
					Content:   bytes.NewReader([]byte("live content")),
					SizeBytes: int64(len("live content")),
				},
			},
		})
		require.NoError(t, err)

		const branch = "drafts/test"
		require.NoError(t, r.GitProvider.CreateBranch(ctx, repository.RepoID, branch, ""))

		branchSHA, err := r.GitProvider.Head(ctx, repository.RepoID, branch)
		require.NoError(t, err)

		_, err = r.GitProvider.Commit(ctx, repository.RepoID, git.CommitOptions{
			Branch:          branch,
			ExpectedHeadSHA: branchSHA,
			Message:         "draft change",
			Operations: []git.FileOperation{
				{
					Path:      "README.md",
					Content:   bytes.NewReader([]byte("draft content")),
					SizeBytes: int64(len("draft content")),
				},
			},
		})
		require.NoError(t, err)

		// Without a branch param we read the default branch (live).
		liveResponse := downloadFile(t, server, signer, r.Organization.ID, authenticated, canvas.ID.String(), "README.md")
		assert.Equal(t, http.StatusOK, liveResponse.Code)
		assert.Equal(t, "live content", liveResponse.Body.String())

		// With the branch param we read the draft branch tip.
		branchResponse := downloadFileOnBranch(t, server, signer, r.Organization.ID, authenticated, canvas.ID.String(), "README.md", branch)
		assert.Equal(t, http.StatusOK, branchResponse.Code)
		assert.Equal(t, "draft content", branchResponse.Body.String())
	})
}
