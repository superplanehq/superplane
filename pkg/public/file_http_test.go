package public

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestFileContentUploadAndPublicDownload(t *testing.T) {
	r := support.Setup(t)
	server, signer := newFileHTTPTestServer(t, r)

	factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	file, err := models.CreatePendingFile(database.Conn(), models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)

	putFileContent(t, server, signer, r.Organization.ID, r.Account.ID, file.ID, []byte("png-bytes"), http.StatusNoContent)

	ready, err := models.FindFile(database.Conn(), file.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FileStateReady, ready.State)

	downloadURL, err := blob.FileAccessURL(file.ID, time.Hour, []byte("test-signing-key"))
	require.NoError(t, err)
	parsed, err := url.Parse(downloadURL)
	require.NoError(t, err)
	getReq := httptest.NewRequest(http.MethodGet, parsed.RequestURI(), nil)
	getRec := httptest.NewRecorder()
	server.Router.ServeHTTP(getRec, getReq)
	assert.Equal(t, http.StatusOK, getRec.Code)
	assert.Equal(t, "image/png", getRec.Header().Get("Content-Type"))
	assert.Equal(t, "png-bytes", getRec.Body.String())

	badReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/public/files/%s?expires=1&sig=deadbeef&sp_file=1", file.ID), nil)
	badRec := httptest.NewRecorder()
	server.Router.ServeHTTP(badRec, badReq)
	assert.Equal(t, http.StatusForbidden, badRec.Code)
}

func TestTaskCreatorUploadsWorkspaceFileWithoutFactoryUpdate(t *testing.T) {
	r := support.Setup(t)
	server, signer := newFileHTTPTestServer(t, r)
	factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	t.Run("task creator stores a csv", func(t *testing.T) {
		creator, account := createOrgMember(t, r, models.RoleOrgOperator)
		requireOrganizationPermission(t, r, creator.ID, "work_orders", "create", true)
		requireOrganizationPermission(t, r, creator.ID, "factories", "update", false)

		fileID := createWorkspaceFile(t, server, signer, r.Organization.ID, account.ID, factoryModel.ID, "rows.csv", "text/csv")
		putFileContent(t, server, signer, r.Organization.ID, account.ID, fileID, []byte("a,b\n1,2\n"), http.StatusNoContent)

		ready, err := models.FindFile(database.Conn(), fileID)
		require.NoError(t, err)
		assert.Equal(t, models.FileStateReady, ready.State)
		assert.Equal(t, blob.ScopeWorkspace, ready.Scope)
		assert.Equal(t, "text/csv", ready.ContentType)
		require.NotNil(t, ready.CreatedByID)
		assert.Equal(t, creator.ID, *ready.CreatedByID)
	})

	t.Run("admin still attaches", func(t *testing.T) {
		fileID := createWorkspaceFile(t, server, signer, r.Organization.ID, r.Account.ID, factoryModel.ID, "note.txt", "text/plain")
		putFileContent(t, server, signer, r.Organization.ID, r.Account.ID, fileID, []byte("hello"), http.StatusNoContent)

		ready, err := models.FindFile(database.Conn(), fileID)
		require.NoError(t, err)
		assert.Equal(t, models.FileStateReady, ready.State)
	})

	t.Run("user who cannot create tasks is denied", func(t *testing.T) {
		reader, account := createOrgMember(t, r, "")
		requireOrganizationPermission(t, r, reader.ID, "work_orders", "create", false)

		denied := postWorkspaceFile(t, server, signer, r.Organization.ID, account.ID, factoryModel.ID, "rows.csv", "text/csv")
		assert.Equal(t, http.StatusNotFound, denied.Code)
		assert.Contains(t, denied.Body.String(), "Not found")

		file, err := models.CreatePendingFile(database.Conn(), models.CreateFileParams{
			Scope:          blob.ScopeWorkspace,
			OrganizationID: r.Organization.ID,
			FactoryID:      factoryModel.ID,
			Filename:       "rows.csv",
			ContentType:    "text/csv",
			CreatedByID:    reader.ID,
		})
		require.NoError(t, err)
		putFileContent(t, server, signer, r.Organization.ID, account.ID, file.ID, []byte("a,b\n"), http.StatusForbidden)
	})
}

func TestFileUploadPermissionUsesTaskCreateForWorkspaceFiles(t *testing.T) {
	resource, action := fileUploadPermission(blob.ScopeWorkspace)
	assert.Equal(t, "work_orders", resource)
	assert.Equal(t, "create", action)

	resource, action = fileUploadPermission(blob.ScopeTask)
	assert.Equal(t, "work_orders", resource)
	assert.Equal(t, "update", action)
}

func newFileHTTPTestServer(t *testing.T, r *support.ResourceRegistry) (*Server, *jwt.Signer) {
	t.Helper()
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	signer := jwt.NewSigner("test")
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		signer,
		support.NewOIDCProvider(),
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService, false,
	)
	require.NoError(t, err)
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
	return server, signer
}

func createOrgMember(t *testing.T, r *support.ResourceRegistry, role string) (*models.User, *models.Account) {
	t.Helper()
	name := support.RandomName("user")
	account, err := models.CreateAccount(name, name+"@test.com")
	require.NoError(t, err)
	user, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	if role == "" {
		return user, account
	}
	require.NoError(t, r.AuthService.AssignRole(user.ID.String(), role, r.Organization.ID.String(), models.DomainTypeOrganization))
	return user, account
}

func requireOrganizationPermission(t *testing.T, r *support.ResourceRegistry, userID uuid.UUID, resource, action string, want bool) {
	t.Helper()
	allowed, err := r.AuthService.CheckOrganizationPermission(context.Background(), userID.String(), r.Organization.ID.String(), resource, action)
	require.NoError(t, err)
	require.Equal(t, want, allowed)
}

func createWorkspaceFile(
	t *testing.T,
	server *Server,
	signer *jwt.Signer,
	organizationID uuid.UUID,
	accountID uuid.UUID,
	factoryID uuid.UUID,
	filename string,
	contentType string,
) uuid.UUID {
	t.Helper()
	rec := postWorkspaceFile(t, server, signer, organizationID, accountID, factoryID, filename, contentType)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var body struct {
		File struct {
			ID string `json:"id"`
		} `json:"file"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	fileID, err := uuid.Parse(body.File.ID)
	require.NoError(t, err)
	return fileID
}

func postWorkspaceFile(
	t *testing.T,
	server *Server,
	signer *jwt.Signer,
	organizationID uuid.UUID,
	accountID uuid.UUID,
	factoryID uuid.UUID,
	filename string,
	contentType string,
) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(map[string]string{
		"filename":    filename,
		"contentType": contentType,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/factories/"+factoryID.String()+"/files", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	return serveAuthenticated(t, server, signer, organizationID, accountID, req)
}

func putFileContent(
	t *testing.T,
	server *Server,
	signer *jwt.Signer,
	organizationID uuid.UUID,
	accountID uuid.UUID,
	fileID uuid.UUID,
	body []byte,
	wantStatus int,
) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/files/"+fileID.String()+"/content", bytes.NewReader(body))
	rec := serveAuthenticated(t, server, signer, organizationID, accountID, req)
	assert.Equal(t, wantStatus, rec.Code, rec.Body.String())
}

func serveAuthenticated(
	t *testing.T,
	server *Server,
	signer *jwt.Signer,
	organizationID uuid.UUID,
	accountID uuid.UUID,
	req *http.Request,
) *httptest.ResponseRecorder {
	t.Helper()
	token, err := authentication.GenerateAccountToken(signer, accountID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	req.Header.Set("x-organization-id", organizationID.String())
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	return rec
}
