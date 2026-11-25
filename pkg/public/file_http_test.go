package public

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestFileContentUploadCredentialScopes(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })
	signer := jwt.NewSigner("test")
	server, err := NewServer(r.Encryptor, r.Registry, signer, support.NewOIDCProvider(), "",
		"http://localhost", "http://localhost", "test", "/app/templates", r.AuthService, false)
	require.NoError(t, err)
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Synthetic task", "", &r.User, nil, nil)
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	adminToken, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)

	newKey := func(t *testing.T) (*models.User, string) {
		t.Helper()
		expires := time.Now().Add(time.Hour)
		key, err := models.CreateAPIKey(db, r.Organization.ID, support.RandomName("upload-key"), nil, r.User, &expires, nil)
		require.NoError(t, err)
		token, err := crypto.Base64String(32)
		require.NoError(t, err)
		require.NoError(t, key.UpdateTokenHash(crypto.HashToken(token)))
		require.NoError(t, r.AuthService.AssignRole(key.ID.String(), models.RoleOrgOperator,
			r.Organization.ID.String(), models.DomainTypeOrganization))
		return key, token
	}
	createRequest := func(path, token, purpose string) *httptest.ResponseRecorder {
		body := `{"filename":"dummy.txt","contentType":"text/plain"}`
		if purpose == models.FilePurposeArtifact {
			body = `{"filename":"dummy.jpg","contentType":"image/jpeg","purpose":"artifact"}`
		}
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		return rec
	}
	createFile := func(t *testing.T, path, token, purpose string) *models.File {
		t.Helper()
		rec := createRequest(path, token, purpose)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var response struct {
			File struct {
				ID string `json:"id"`
			} `json:"file"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		id, err := uuid.Parse(response.File.ID)
		require.NoError(t, err)
		file, err := models.FindFile(db, id)
		require.NoError(t, err)
		require.Equal(t, models.FileStatePending, file.State)
		return file
	}
	narrowKey := func(t *testing.T, key *models.User) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/api-keys/"+key.ID.String(),
			bytes.NewBufferString(fmt.Sprintf(`{"canvasIds":[%q]}`, canvas.ID.String())))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-organization-id", r.Organization.ID.String())
		req.AddCookie(&http.Cookie{Name: "account_token", Value: adminToken})
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		current, err := models.FindActiveUserByID(r.Organization.ID.String(), key.ID.String())
		require.NoError(t, err)
		require.Equal(t, []string{canvas.ID.String()}, []string(current.APIKeyCanvasIDs))
	}
	assertUpload := func(t *testing.T, file *models.File, token string, status int) {
		t.Helper()
		resource, action := fileUploadPermission(file.Scope)
		allowed, err := r.AuthService.CheckOrganizationPermission(t.Context(), file.CreatedByID.String(),
			r.Organization.ID.String(), resource, action)
		require.NoError(t, err)
		require.True(t, allowed, "the subject must retain upload RBAC permission")
		content := []byte("dummy")
		if file.Purpose == models.FilePurposeArtifact {
			content = []byte("\xff\xd8\xffab")
		}
		body := &countingUploadBody{Reader: bytes.NewReader(content)}
		req := httptest.NewRequest(http.MethodPut, "/api/v1/files/"+file.ID.String()+"/content", body)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("x-Token-Scopes", `["work_orders:create","work_orders:update"]`)
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		assert.Equal(t, status, rec.Code, rec.Body.String())
		current, err := models.FindFile(db, file.ID)
		require.NoError(t, err)
		if status == http.StatusForbidden {
			assert.Zero(t, body.reads)
			assert.Equal(t, models.FileStatePending, current.State)
			assert.Equal(t, file.SizeBytes, current.SizeBytes)
			assert.Equal(t, file.Checksum, current.Checksum)
			reader, err := store.Get(t.Context(), file.StorageKey)
			if reader != nil {
				reader.Close()
			}
			assert.ErrorIs(t, err, blob.ErrNotFound)
			return
		}
		assert.Equal(t, models.FileStateReady, current.State)
		assert.Equal(t, int64(5), current.SizeBytes)
		reader, err := store.Get(t.Context(), file.StorageKey)
		require.NoError(t, err)
		defer reader.Close()
		stored, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, content, stored)
	}

	workspacePath := "/api/v1/factories/" + factory.ID.String() + "/files"
	taskPath := "/api/v1/factories/" + factory.ID.String() + "/orders/" + order.ID.String() + "/files"
	for _, scenario := range []struct{ name, path, purpose string }{
		{"workspace", workspacePath, ""},
		{"task", taskPath, ""},
		{"artifact", taskPath, models.FilePurposeArtifact},
	} {
		t.Run("narrowed_key/"+scenario.name, func(t *testing.T) {
			key, token := newKey(t)
			file := createFile(t, scenario.path, token, scenario.purpose)
			narrowKey(t, key)
			assertUpload(t, file, token, http.StatusForbidden)
			var before, after int64
			require.NoError(t, db.Model(&models.File{}).Where("created_by_id = ?", key.ID).Count(&before).Error)
			rec := createRequest(scenario.path, token, scenario.purpose)
			assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
			require.NoError(t, db.Model(&models.File{}).Where("created_by_id = ?", key.ID).Count(&after).Error)
			assert.Equal(t, before, after)
		})
	}
	t.Run("unrestricted_key", func(t *testing.T) {
		_, token := newKey(t)
		assertUpload(t, createFile(t, workspacePath, token, ""), token, http.StatusNoContent)
	})
	for _, scenario := range []struct {
		name, path, scope string
		narrowed          bool
		status            int
	}{
		{"workspace_read", workspacePath, "work_orders:read", false, http.StatusForbidden},
		{"workspace_update", workspacePath, "work_orders:update", false, http.StatusForbidden},
		{"workspace_create", workspacePath, "work_orders:create", false, http.StatusNoContent},
		{"task_read", taskPath, "work_orders:read", false, http.StatusForbidden},
		{"task_create", taskPath, "work_orders:create", false, http.StatusForbidden},
		{"task_update", taskPath, "work_orders:update", false, http.StatusNoContent},
		{"resource_specific", taskPath, "work_orders:update:" + order.ID.String(), false, http.StatusForbidden},
		{"claims_cannot_override_key", taskPath, "work_orders:update", true, http.StatusForbidden},
	} {
		t.Run("signed_jwt/"+scenario.name, func(t *testing.T) {
			key, token := newKey(t)
			file := createFile(t, scenario.path, token, "")
			token, err := signer.GenerateScopedToken(jwt.ScopedTokenClaims{
				Subject: key.ID.String(), OrgID: r.Organization.ID.String(), Purpose: "upload",
				Scopes: []string{scenario.scope},
			}, time.Hour)
			require.NoError(t, err)
			if scenario.narrowed {
				narrowKey(t, key)
			}
			assertUpload(t, file, token, scenario.status)
		})
	}
}

type countingUploadBody struct {
	*bytes.Reader
	reads int
}

func (b *countingUploadBody) Read(p []byte) (int, error) {
	b.reads++
	return b.Reader.Read(p)
}

func (b *countingUploadBody) Close() error { return nil }

func TestFileContentUploadAndPublicDownload(t *testing.T) {
	r := support.Setup(t)
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

	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/files/"+file.ID.String()+"/content", bytes.NewReader([]byte("png-bytes")))
	req.Header.Set("x-organization-id", r.Organization.ID.String())
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)

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
	assert.Equal(t, "bytes", getRec.Header().Get("Accept-Ranges"))

	rangeReq := httptest.NewRequest(http.MethodGet, parsed.RequestURI(), nil)
	rangeReq.Header.Set("Range", "bytes=4-6")
	rangeRec := httptest.NewRecorder()
	server.Router.ServeHTTP(rangeRec, rangeReq)
	assert.Equal(t, http.StatusPartialContent, rangeRec.Code)
	assert.Equal(t, "byt", rangeRec.Body.String())
	assert.Contains(t, rangeRec.Header().Get("Content-Range"), "bytes 4-6/")

	badReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/public/files/%s?expires=1&sig=deadbeef&sp_file=1", file.ID), nil)
	badRec := httptest.NewRecorder()
	server.Router.ServeHTTP(badRec, badReq)
	assert.Equal(t, http.StatusForbidden, badRec.Code)
}

func TestWorkspaceFileContentUploadUsesWorkOrderCreatePermission(t *testing.T) {
	r := support.Setup(t)
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	operatorName := support.RandomName("operator")
	operatorAccount, err := models.CreateAccount(operatorName, operatorName+"@test.com")
	require.NoError(t, err)
	operator, err := models.CreateUser(r.Organization.ID, operatorAccount.ID, operatorAccount.Email, operatorAccount.Name)
	require.NoError(t, err)
	require.NoError(t, r.AuthService.AssignRole(
		operator.ID.String(), models.RoleOrgOperator, r.Organization.ID.String(), models.DomainTypeOrganization,
	))
	canCreateWorkOrders, err := r.AuthService.CheckOrganizationPermission(
		t.Context(), operator.ID.String(), r.Organization.ID.String(), "work_orders", "create",
	)
	require.NoError(t, err)
	require.True(t, canCreateWorkOrders)
	canUpdateFactories, err := r.AuthService.CheckOrganizationPermission(
		t.Context(), operator.ID.String(), r.Organization.ID.String(), "factories", "update",
	)
	require.NoError(t, err)
	require.False(t, canUpdateFactories)

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

	factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	file, err := models.CreatePendingFile(database.Conn(), models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "rows.csv",
		ContentType:    "text/csv",
		CreatedByID:    operator.ID,
	})
	require.NoError(t, err)
	token, err := authentication.GenerateAccountToken(signer, operatorAccount.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/files/"+file.ID.String()+"/content", bytes.NewReader([]byte("a,b\n1,2")))
	req.Header.Set("x-organization-id", r.Organization.ID.String())
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestFileContentUploadRejectsOversizedContentLength(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
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

	factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	file, err := models.CreatePendingFile(database.Conn(), models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "clip.mp4",
		ContentType:    "video/mp4",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/files/"+file.ID.String()+"/content", bytes.NewReader([]byte("x")))
	req.ContentLength = models.MaxFileBytes + 1
	req.Header.Set("x-organization-id", r.Organization.ID.String())
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestFileContentUploadAcceptsExactMaxContentLength(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
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

	factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	file, err := models.CreatePendingFile(database.Conn(), models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "clip.mp4",
		ContentType:    "video/mp4",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)

	body := bytes.Repeat([]byte{'a'}, int(models.MaxFileBytes))
	req := httptest.NewRequest(http.MethodPut, "/api/v1/files/"+file.ID.String()+"/content", bytes.NewReader(body))
	require.Equal(t, int64(models.MaxFileBytes), req.ContentLength)
	req.Header.Set("x-organization-id", r.Organization.ID.String())
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	ready, err := models.FindFile(database.Conn(), file.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FileStateReady, ready.State)
	assert.Equal(t, int64(models.MaxFileBytes), ready.SizeBytes)
}

func TestParseBytesRange(t *testing.T) {
	start, length, ok := parseBytesRange("bytes=4-6", 9)
	assert.True(t, ok)
	assert.Equal(t, int64(4), start)
	assert.Equal(t, int64(3), length)

	start, length, ok = parseBytesRange("bytes=8-", 9)
	assert.True(t, ok)
	assert.Equal(t, int64(8), start)
	assert.Equal(t, int64(1), length)

	_, _, ok = parseBytesRange("bytes=9-10", 9)
	assert.False(t, ok)
}
