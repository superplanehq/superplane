package public

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__HealthCheckEndpoint(t *testing.T) {
	signer := jwt.NewSigner("test")
	server, err := NewServer(&crypto.NoOpEncryptor{}, signer, "", "")
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method: "GET",
		path:   "/",
	})

	require.Equal(t, 200, response.Code)
}

func Test__ReceiveGitHubEvent(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	signer := jwt.NewSigner("test")
	server, err := NewServer(&crypto.NoOpEncryptor{}, signer, "", "")
	require.NoError(t, err)

	org, err := models.CreateOrganization(uuid.New(), "test", "test", "")
	require.NoError(t, err)

	userID := uuid.New()
	canvas, err := models.CreateCanvas(userID, org.ID, "test")
	require.NoError(t, err)

	eventSource, err := canvas.CreateEventSource("github-repo-1", []byte("my-key"), models.EventSourceScopeExternal, nil)
	require.NoError(t, err)

	validEvent := []byte(`{"action": "created"}`)
	validSignature := "sha256=ee9f99fa8d06b44ffc69ee1c2a7e32e848e8b40536bb5e8405dabb3bbbcaf619"
	validURL := "/sources/" + eventSource.ID.String() + "/github"

	t.Run("event for invalid source -> 404", func(t *testing.T) {
		invalidURL := "/sources/invalidsource/github"
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        invalidURL,
			body:        validEvent,
			signature:   validSignature,
			contentType: "application/json",
		})

		assert.Equal(t, 404, response.Code)
		assert.Equal(t, "source ID not found\n", response.Body.String())
	})

	t.Run("missing Content-Type header -> 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			body:        validEvent,
			signature:   validSignature,
			contentType: "",
		})

		assert.Equal(t, 404, response.Code)
	})

	t.Run("unsupported Content-Type header -> 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			body:        validEvent,
			signature:   validSignature,
			contentType: "application/x-www-form-urlencoded",
		})

		assert.Equal(t, 404, response.Code)
	})

	t.Run("event for source that does not exist -> 404", func(t *testing.T) {
		invalidURL := "/sources/" + uuid.New().String() + "/github"
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        invalidURL,
			body:        validEvent,
			signature:   validSignature,
			contentType: "application/json",
		})

		assert.Equal(t, 404, response.Code)
		assert.Equal(t, "source ID not found\n", response.Body.String())
	})

	t.Run("event with missing signature header -> 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			body:        validEvent,
			signature:   "",
			contentType: "application/json",
		})

		assert.Equal(t, 400, response.Code)
		assert.Equal(t, "Missing X-Hub-Signature-256 header\n", response.Body.String())
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			body:        validEvent,
			signature:   "sha256=823a7b73b066321f4f644e70e1d32c15dc8f4677968149c1f35eb07639013271",
			contentType: "application/json",
		})

		assert.Equal(t, 403, response.Code)
		assert.Equal(t, "Invalid signature\n", response.Body.String())
	})

	t.Run("properly signed event is received -> 200", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			body:        validEvent,
			signature:   validSignature,
			contentType: "application/json",
		})

		assert.Equal(t, 200, response.Code)
		events, err := models.ListEventsBySourceID(eventSource.ID)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, eventSource.ID, events[0].SourceID)
		assert.Equal(t, models.EventStatePending, events[0].State)
		assert.Equal(t, []byte(`{"action": "created"}`), []byte(events[0].Raw))
		assert.NotNil(t, events[0].ReceivedAt)
	})

	t.Run("event data is limited to 64k", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			body:        generateBigBody(t),
			signature:   validSignature,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
		assert.Equal(t, "Request body is too large - must be up to 65536 bytes\n", response.Body.String())
	})
}

func Test__ReceiveSemaphoreEvent(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	signer := jwt.NewSigner("test")
	server, err := NewServer(&crypto.NoOpEncryptor{}, signer, "", "")
	require.NoError(t, err)

	org, err := models.CreateOrganization(uuid.New(), "test", "test", "")
	require.NoError(t, err)

	userID := uuid.New()
	canvas, err := models.CreateCanvas(userID, org.ID, "test")
	require.NoError(t, err)

	eventSource, err := canvas.CreateEventSource("semaphore-source-1", []byte("my-key"), models.EventSourceScopeExternal, nil)
	require.NoError(t, err)

	// No need to include organization ID in the payload anymore
	validEvent := []byte(`{"version": "1.0.0", "event_type": "workflow_completed"}`)

	key := []byte("my-key")
	mac := hmac.New(sha256.New, key)
	mac.Write(validEvent)
	validSignatureBytes := mac.Sum(nil)
	validSignature := "sha256=" + hex.EncodeToString(validSignatureBytes)

	validURL := "/sources/" + eventSource.ID.String() + "/semaphore"

	t.Run("event for invalid source -> 404", func(t *testing.T) {
		invalidURL := "/sources/invalidsource/semaphore"
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        invalidURL,
			body:        validEvent,
			signature:   validSignature,
			contentType: "application/json",
		})

		assert.Equal(t, 404, response.Code)
		assert.Equal(t, "source ID not found\n", response.Body.String())
	})

	t.Run("missing Content-Type header -> 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:    "POST",
			path:      validURL,
			body:      validEvent,
			signature: validSignature,
		})

		assert.Equal(t, 404, response.Code)
	})

	t.Run("unsupported Content-Type header -> 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			body:        validEvent,
			signature:   validSignature,
			contentType: "application/x-www-form-urlencoded",
		})

		assert.Equal(t, 404, response.Code)
	})

	t.Run("event for source that does not exist -> 404", func(t *testing.T) {
		invalidURL := "/sources/" + uuid.New().String() + "/semaphore"
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        invalidURL,
			body:        validEvent,
			signature:   validSignature,
			contentType: "application/json",
		})

		assert.Equal(t, 404, response.Code)
		assert.Equal(t, "source ID not found\n", response.Body.String())
	})

	t.Run("event with missing signature header -> 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			body:        validEvent,
			contentType: "application/json",
		})

		assert.Equal(t, 400, response.Code)
		assert.Equal(t, "Missing X-Semaphore-Signature-256 header\n", response.Body.String())
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			body:        validEvent,
			signature:   "sha256=invalid-signature",
			contentType: "application/json",
		})

		assert.Equal(t, 403, response.Code)
		assert.Equal(t, "Invalid signature\n", response.Body.String())
	})

	t.Run("properly signed event is received -> 200", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			body:        validEvent,
			signature:   validSignature,
			contentType: "application/json",
		})

		assert.Equal(t, 200, response.Code)
		events, err := models.ListEventsBySourceID(eventSource.ID)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, eventSource.ID, events[0].SourceID)
		assert.Equal(t, models.EventStatePending, events[0].State)

		// Compare the event payload
		var savedEvent map[string]interface{}
		err = json.Unmarshal([]byte(events[0].Raw), &savedEvent)
		require.NoError(t, err)

		var expectedEvent map[string]interface{}
		err = json.Unmarshal(validEvent, &expectedEvent)
		require.NoError(t, err)
		assert.Equal(t, expectedEvent["event_type"], savedEvent["event_type"])
		assert.NotNil(t, events[0].ReceivedAt)
	})

	t.Run("event data is limited to 64k", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			body:        generateBigBody(t),
			signature:   validSignature,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
		assert.Equal(t, "Request body is too large - must be up to 65536 bytes\n", response.Body.String())
	})
}

func Test__HandleExecutionOutputs(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
	})

	executorType, executorSpec, resource := support.Executor(t, r)
	stage, err := builders.NewStageBuilder(r.Registry).
		WithEncryptor(r.Encryptor).
		InCanvas(r.Canvas).
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
	server, err := NewServer(&crypto.NoOpEncryptor{}, signer, "", "")
	require.NoError(t, err)

	execution := support.CreateExecution(t, r.Source, stage)
	validToken, err := signer.Generate(execution.ID.String(), time.Hour)
	require.NoError(t, err)

	outputs := map[string]any{"version": "v1.0.0", "sha": "078fc8755c051"}

	goodBody, _ := json.Marshal(&OutputsRequest{
		ExecutionID: execution.ID.String(),
		Outputs:     outputs,
	})

	t.Run("event for invalid execution -> 404", func(t *testing.T) {
		body, _ := json.Marshal(&OutputsRequest{
			ExecutionID: "not-a-uuid",
			Outputs:     outputs,
		})

		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        body,
			authToken:   validToken,
			contentType: "application/json",
		})

		assert.Equal(t, 404, response.Code)
		assert.Equal(t, "execution not found\n", response.Body.String())
	})

	t.Run("missing Content-Type header -> 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        goodBody,
			contentType: "",
			authToken:   validToken,
		})

		assert.Equal(t, 404, response.Code)
	})

	t.Run("unsupported Content-Type header -> 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        goodBody,
			contentType: "application/x-www-form-urlencoded",
			authToken:   validToken,
		})

		assert.Equal(t, 404, response.Code)
	})

	t.Run("execution that does not exist -> 401", func(t *testing.T) {
		body, _ := json.Marshal(&OutputsRequest{
			ExecutionID: uuid.NewString(),
			Outputs:     outputs,
		})

		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        body,
			contentType: "application/json",
			authToken:   validToken,
		})

		assert.Equal(t, 401, response.Code)
		assert.Equal(t, "Invalid token\n", response.Body.String())
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
		assert.Equal(t, "Missing Authorization header\n", response.Body.String())
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
		assert.Equal(t, "Invalid token\n", response.Body.String())
	})

	t.Run("proper request -> 200 and execution outputs are updated", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        goodBody,
			authToken:   validToken,
			contentType: "application/json",
		})

		assert.Equal(t, 200, response.Code)
		execution, err := models.FindExecutionByID(execution.ID)
		require.NoError(t, err)
		assert.Equal(t, outputs, execution.Outputs.Data())
	})

	t.Run("output not defined in stage is ignored", func(t *testing.T) {
		// 'time' output is not defined in the stage
		body, _ := json.Marshal(&OutputsRequest{
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
			authToken:   validToken,
			contentType: "application/json",
		})

		assert.Equal(t, 200, response.Code)
		execution, err := models.FindExecutionByID(execution.ID)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"sha": "078fc8755c051", "version": "v1.0.0"}, execution.Outputs.Data())
	})

	t.Run("outputs are limited to 4k", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/outputs",
			body:        generateBigBody(t),
			authToken:   validToken,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
		assert.Equal(t, "Request body is too large - must be up to 4096 bytes\n", response.Body.String())
	})
}

// Test__OpenAPIEndpoints tests that the OpenAPI endpoints serve the files correctly
func Test__OpenAPIEndpoints(t *testing.T) {
	checkSwaggerFiles(t)

	signer := jwt.NewSigner("test")
	server, err := NewServer(&crypto.NoOpEncryptor{}, signer, "", "")
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
	signer := jwt.NewSigner("test")
	server, err := NewServer(&crypto.NoOpEncryptor{}, signer, "", "")
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
	method      string
	path        string
	body        []byte
	signature   string
	authToken   string
	contentType string
}

func execRequest(server *Server, params requestParams) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(params.method, params.path, bytes.NewReader(params.body))

	if params.contentType != "" {
		req.Header.Add("Content-Type", params.contentType)
	}

	// Set the appropriate signature header based on the path
	if params.signature != "" {
		if strings.Contains(params.path, "/github") {
			req.Header.Add("X-Hub-Signature-256", params.signature)
		} else if strings.Contains(params.path, "/semaphore") {
			req.Header.Add("X-Semaphore-Signature-256", params.signature)
		} else {
			// Default to GitHub header for backward compatibility
			req.Header.Add("X-Hub-Signature-256", params.signature)
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
