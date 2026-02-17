package public

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__HealthCheckEndpoint(t *testing.T) {
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	registry, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)
	signer := jwt.NewSigner("test")
	oidcProvider := support.NewOIDCProvider()
	server, err := NewServer(&crypto.NoOpEncryptor{}, registry, signer, oidcProvider, "", "", "", "test", "/app/templates", authService, false)
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method: "GET",
		path:   "/health",
	})

	require.Equal(t, 200, response.Code)
}

func Test__OpenAPIEndpoints(t *testing.T) {
	checkSwaggerFiles(t)

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	signer := jwt.NewSigner("test")
	registry, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)
	oidcProvider := support.NewOIDCProvider()
	server, err := NewServer(&crypto.NoOpEncryptor{}, registry, signer, oidcProvider, "", "", "", "test", "/app/templates", authService, false)
	require.NoError(t, err)

	server.RegisterOpenAPIHandler()

	t.Run("OpenAPI JSON spec is accessible", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method: "GET",
			path:   "/docs/superplane.swagger.json",
		})

		require.Equal(t, 200, response.Code)
		require.NotEmpty(t, response.Body.String())
		require.Contains(t, response.Header().Get("Content-Type"), "application/json")

		var jsonData map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &jsonData)
		require.NoError(t, err, "Response should be valid JSON")

		assert.Contains(t, jsonData, "swagger", "Should contain 'swagger' field")
		assert.Contains(t, jsonData, "paths", "Should contain 'paths' field")
	})

	t.Run("Swagger UI HTML is accessible", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method: "GET",
			path:   "/docs",
		})

		require.Equal(t, 200, response.Code)
		require.NotEmpty(t, response.Body.String())
		require.Contains(t, response.Header().Get("Content-Type"), "text/html")

		require.Contains(t, response.Body.String(), "<html")
		require.Contains(t, response.Body.String(), "swagger-ui")
		require.Contains(t, response.Body.String(), "SwaggerUIBundle")
	})

	t.Run("OpenAPI spec is accessible via directory path", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method: "GET",
			path:   "/docs/superplane.swagger.json",
		})

		require.Equal(t, 200, response.Code)
		require.NotEmpty(t, response.Body.String())
		require.Contains(t, response.Header().Get("Content-Type"), "application/json")

		var jsonData map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &jsonData)
		require.NoError(t, err, "Response should be valid JSON")
	})

	t.Run("Non-existent file returns 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method: "GET",
			path:   "/docs/non-existent-file.json",
		})

		require.Equal(t, 404, response.Code)
	})
}

func Test__GRPCGatewayRegistration(t *testing.T) {
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	signer := jwt.NewSigner("test")
	registry, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)
	oidcProvider := support.NewOIDCProvider()
	server, err := NewServer(&crypto.NoOpEncryptor{}, registry, signer, oidcProvider, "", "", "", "test", "/app/templates", authService, false)
	require.NoError(t, err)

	err = server.RegisterGRPCGateway("localhost:50051")
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method: "GET",
		path:   "/api/v1/canvases/is-alive",
	})

	require.Equal(t, "", response.Body.String())
	require.Equal(t, 200, response.Code)
}

// Helper function to check if the required Swagger files exist
func checkSwaggerFiles(t *testing.T) {
	apiDir := os.Getenv("SWAGGER_BASE_PATH")

	// Check if the directory exists
	dirInfo, err := os.Stat(apiDir)
	require.NoError(t, err, "api/swagger directory should exist")
	require.True(t, dirInfo.IsDir(), "api/swagger should be a directory")

	// Check for the OpenAPI spec JSON file
	specPath := filepath.Join(apiDir, "superplane.swagger.json")
	fileInfo, err := os.Stat(specPath)
	require.NoError(t, err, "superplane.swagger.json should exist")
	require.False(t, fileInfo.IsDir(), "superplane.swagger.json should be a file")
	require.Greater(t, fileInfo.Size(), int64(0), "superplane.swagger.json should not be empty")

	// Check for the Swagger UI HTML file
	htmlPath := filepath.Join(apiDir, "swagger-ui.html")
	fileInfo, err = os.Stat(htmlPath)
	require.NoError(t, err, "swagger-ui.html should exist")
	require.False(t, fileInfo.IsDir(), "swagger-ui.html should be a file")
	require.Greater(t, fileInfo.Size(), int64(0), "swagger-ui.html should not be empty")

	// Check that the JSON file is valid
	jsonData, err := os.ReadFile(specPath)
	require.NoError(t, err, "Should be able to read swagger JSON file")

	var data map[string]interface{}
	err = json.Unmarshal(jsonData, &data)
	require.NoError(t, err, "superplane.swagger.json should contain valid JSON")

	// Check that the HTML file contains expected content
	htmlData, err := os.ReadFile(htmlPath)
	require.NoError(t, err, "Should be able to read swagger UI HTML file")
	require.Contains(t, string(htmlData), "swagger-ui", "HTML should contain swagger-ui reference")
}

type requestParams struct {
	method       string
	path         string
	body         []byte
	signature    string
	authToken    string
	authCookie   string
	contentType  string
	customSource bool
}

func execRequest(server *Server, params requestParams) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(params.method, params.path, bytes.NewReader(params.body))

	if params.contentType != "" {
		req.Header.Add("Content-Type", params.contentType)
	}

	// Set the appropriate signature header based on the path
	if params.signature != "" {
		if params.customSource {
			req.Header.Add("X-Signature-256", params.signature)
		} else {
			req.Header.Add("X-Semaphore-Signature-256", params.signature)
		}
	}

	if params.authToken != "" {
		req.Header.Add("Authorization", "Bearer "+params.authToken)
	}

	if params.authCookie != "" {
		req.AddCookie(&http.Cookie{Name: "account_token", Value: params.authCookie})
	}

	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)
	return res
}

// mockAuthService wraps a real auth service and allows us to inject errors
type mockAuthService struct {
	*authorization.AuthService
	setupOrgError error
}

func (m *mockAuthService) SetupOrganization(tx *gorm.DB, orgID, ownerID string) error {
	if m.setupOrgError != nil {
		return m.setupOrgError
	}
	return m.AuthService.SetupOrganization(tx, orgID, ownerID)
}

func Test__CreateOrganization(t *testing.T) {
	t.Run("organization creation fails due to RBAC setup failure", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		//
		// Set up account
		//
		signer := jwt.NewSigner("test")
		account, err := models.CreateAccount("test@example.com", "Test User")
		require.NoError(t, err)
		token, err := signer.Generate(account.ID.String(), time.Hour)
		require.NoError(t, err)

		//
		// Initial server and dependencies.
		// Here, we use a mocked auth service that will fail to setup organization.
		//
		authService, err := authorization.NewAuthService()
		require.NoError(t, err)

		mockedAuthService := &mockAuthService{
			AuthService:   authService,
			setupOrgError: errors.New("simulated authorization setup failure"),
		}

		encryptor := &crypto.NoOpEncryptor{}
		r, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
		require.NoError(t, err)
		oidcProvider := support.NewOIDCProvider()
		server, err := NewServer(encryptor, r, signer, oidcProvider, "", "localhost", "", "test", "/app/templates", mockedAuthService, false)
		require.NoError(t, err)

		//
		// Request to create organization returns 500
		//
		body, err := json.Marshal(OrganizationCreationRequest{Name: "Test Organization"})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/organizations",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.Contains(t, response.Body.String(), "Failed to set up organization roles")

		//
		// Organization and user records to not exist
		//
		_, err = models.FindOrganizationByName("Test Organization")
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = models.FindAnyUserByEmail(account.Email)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("organization is created successfully", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		//
		// Set up account
		//
		account, err := models.CreateAccount("success@example.com", "Success User")
		require.NoError(t, err)
		signer := jwt.NewSigner("test")
		token, err := signer.Generate(account.ID.String(), time.Hour)
		require.NoError(t, err)

		//
		// Initial server and dependencies.
		// Here, we use the real authentication service, which should not fail.
		//
		authService, err := authorization.NewAuthService()
		require.NoError(t, err)

		encryptor := &crypto.NoOpEncryptor{}
		r, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
		require.NoError(t, err)
		oidcProvider := support.NewOIDCProvider()
		server, err := NewServer(encryptor, r, signer, oidcProvider, "", "localhost", "", "test", "/app/templates", authService, false)
		require.NoError(t, err)

		//
		// Request to create organization should succeed
		//
		body, err := json.Marshal(OrganizationCreationRequest{Name: "Success Organization"})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/organizations",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)

		//
		// Verify organization and user records were created,
		// and RBAC policies were set up for the organization and user.
		//
		var responseData map[string]interface{}
		err = json.Unmarshal(response.Body.Bytes(), &responseData)
		require.NoError(t, err)
		orgID := responseData["id"].(string)

		org, err := models.FindOrganizationByID(orgID)
		require.NoError(t, err)
		assert.Equal(t, "Success Organization", org.Name)

		user, err := models.FindActiveUserByEmail(orgID, account.Email)
		require.NoError(t, err)
		assert.Equal(t, account.Email, user.GetEmail())

		roles, err := authService.GetUserRolesForOrg(user.ID.String(), orgID)
		require.NoError(t, err)
		assert.NotEmpty(t, roles)
	})

	t.Run("organization creation fails with 409 when name already exists", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		account, err := models.CreateAccount("duplicate@example.com", "Duplicate User")
		require.NoError(t, err)
		signer := jwt.NewSigner("test")
		token, err := signer.Generate(account.ID.String(), time.Hour)
		require.NoError(t, err)

		authService, err := authorization.NewAuthService()
		require.NoError(t, err)

		encryptor := &crypto.NoOpEncryptor{}
		r, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
		require.NoError(t, err)
		oidcProvider := support.NewOIDCProvider()
		server, err := NewServer(encryptor, r, signer, oidcProvider, "", "localhost", "", "test", "/app/templates", authService, false)
		require.NoError(t, err)

		body, err := json.Marshal(OrganizationCreationRequest{Name: "Duplicate Organization"})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/organizations",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)

		response = execRequest(server, requestParams{
			method:      "POST",
			path:        "/organizations",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusConflict, response.Code)
		assert.Contains(t, response.Body.String(), "Organization name already in use")
	})
}
