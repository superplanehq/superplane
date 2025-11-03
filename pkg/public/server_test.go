package public

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
)

func Test__HealthCheckEndpoint(t *testing.T) {
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	registry := registry.NewRegistry(&crypto.NoOpEncryptor{})
	signer := jwt.NewSigner("test")
	server, err := NewServer(&crypto.NoOpEncryptor{}, registry, signer, crypto.NewOIDCVerifier(), "", "", "/app/templates", authService)
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method: "GET",
		path:   "/health",
	})

	require.Equal(t, 200, response.Code)
}

func Test__HandleExecutionOutputs(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
	})

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	executorType, executorSpec, resource := support.Executor(t, r)
	stage, err := builders.NewStageBuilder(r.Registry).
		WithEncryptor(r.Encryptor).
		InCanvas(r.Canvas.ID).
		WithName("stage-1").
		WithRequester(r.User).
		WithConnections([]models.Connection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}).
		WithOutputs([]models.OutputDefinition{
			{Name: "version", Required: true},
			{Name: "sha", Required: true},
		}).
		WithExecutorType(executorType).
		WithExecutorSpec(executorSpec).
		ForResource(resource).
		ForIntegration(r.Integration).
		Create()

	require.NoError(t, err)
	signer := jwt.NewSigner("test")
	server, err := NewServer(&crypto.NoOpEncryptor{}, r.Registry, signer, crypto.NewOIDCVerifier(), "", "", "/app/templates", authService)
	require.NoError(t, err)

	execution := support.CreateExecution(t, r.Source, stage)
	superplaneToken, err := signer.Generate(execution.ID.String(), time.Hour)
	require.NoError(t, err)

	workflowID := uuid.NewString()
	_, err = execution.AddResource(workflowID, "workflow", *stage.ResourceID)
	require.NoError(t, err)

	outputs := map[string]any{"version": "v1.0.0", "sha": "078fc8755c051"}

	goodBody, _ := json.Marshal(&ExecutionOutputRequest{
		ExternalID:  workflowID,
		ExecutionID: execution.ID.String(),
		Outputs:     outputs,
	})

	t.Run("event for invalid execution -> 404", func(t *testing.T) {
		body, _ := json.Marshal(&ExecutionOutputRequest{
			ExecutionID: "not-a-uuid",
			Outputs:     outputs,
		})

		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        body,
			authToken:   superplaneToken,
			contentType: "application/json",
		})

		assert.Equal(t, 404, response.Code)
		assert.Equal(t, "Execution not found\n", response.Body.String())
	})

	t.Run("missing Content-Type header -> 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        goodBody,
			contentType: "",
			authToken:   superplaneToken,
		})

		assert.Equal(t, 404, response.Code)
	})

	t.Run("unsupported Content-Type header -> 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        goodBody,
			contentType: "application/x-www-form-urlencoded",
			authToken:   superplaneToken,
		})

		assert.Equal(t, 404, response.Code)
	})

	t.Run("execution that does not exist -> 401", func(t *testing.T) {
		body, _ := json.Marshal(&ExecutionOutputRequest{
			ExecutionID: uuid.NewString(),
			Outputs:     outputs,
		})

		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        body,
			contentType: "application/json",
			authToken:   superplaneToken,
		})

		assert.Equal(t, 404, response.Code)
	})

	t.Run("event with missing authorization header -> 401", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        goodBody,
			signature:   "",
			authToken:   "",
			contentType: "application/json",
		})

		assert.Equal(t, 401, response.Code)
	})

	t.Run("invalid auth token -> 403", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        goodBody,
			authToken:   "invalid",
			contentType: "application/json",
		})

		assert.Equal(t, 401, response.Code)
		assert.Equal(t, "Unauthorized\n", response.Body.String())
	})

	t.Run("superplane token is used -> 200 and execution outputs are updated", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        goodBody,
			authToken:   superplaneToken,
			contentType: "application/json",
		})

		assert.Equal(t, 200, response.Code)
		execution, err := models.FindExecutionByID(execution.ID, execution.StageID)
		require.NoError(t, err)
		assert.Equal(t, outputs, execution.Outputs.Data())
	})

	t.Run("integration OIDC ID token is used -> 200 and execution outputs are updated", func(t *testing.T) {
		token := r.SemaphoreAPIMock.GenerateIDToken(resource.Id(), workflowID)
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        goodBody,
			authToken:   token,
			contentType: "application/json",
		})

		assert.Equal(t, 200, response.Code)
		execution, err := models.FindExecutionByID(execution.ID, execution.StageID)
		require.NoError(t, err)
		assert.Equal(t, outputs, execution.Outputs.Data())
	})

	t.Run("output not defined in stage is ignored", func(t *testing.T) {
		// 'time' output is not defined in the stage
		body, _ := json.Marshal(&ExecutionOutputRequest{
			ExecutionID: execution.ID.String(),
			Outputs: map[string]any{
				"sha":     "078fc8755c051",
				"time":    1748555264,
				"version": "v1.0.0",
			},
		})

		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        body,
			authToken:   superplaneToken,
			contentType: "application/json",
		})

		assert.Equal(t, 200, response.Code)
		execution, err := models.FindExecutionByID(execution.ID, execution.StageID)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"sha": "078fc8755c051", "version": "v1.0.0"}, execution.Outputs.Data())
	})

	t.Run("outputs are limited to 4k", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        generateBigBody(t),
			authToken:   superplaneToken,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
		assert.Equal(t, "Request body is too large - must be up to 4096 bytes\n", response.Body.String())
	})
}

func Test__OpenAPIEndpoints(t *testing.T) {
	checkSwaggerFiles(t)

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	signer := jwt.NewSigner("test")
	registry := registry.NewRegistry(&crypto.NoOpEncryptor{})
	server, err := NewServer(&crypto.NoOpEncryptor{}, registry, signer, crypto.NewOIDCVerifier(), "", "", "/app/templates", authService)
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
	authService.EnableCache(false)

	signer := jwt.NewSigner("test")
	registry := registry.NewRegistry(&crypto.NoOpEncryptor{})
	server, err := NewServer(&crypto.NoOpEncryptor{}, registry, signer, crypto.NewOIDCVerifier(), "", "", "/app/templates", authService)
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

	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)
	return res
}

func generateBigBody(t *testing.T) []byte {
	b := make([]byte, 64*1024*1024)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return b
}
