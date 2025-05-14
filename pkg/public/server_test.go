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
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/encryptor"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__HealthCheckEndpoint(t *testing.T) {
	signer := jwt.NewSigner("test")
	server, err := NewServer(&encryptor.NoOpEncryptor{}, signer, "")
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
	server, err := NewServer(&encryptor.NoOpEncryptor{}, signer, "")
	require.NoError(t, err)

	orgID := uuid.New()
	userID := uuid.New()
	canvas, err := models.CreateCanvas(orgID, userID, "test")
	require.NoError(t, err)

	eventSource, err := canvas.CreateEventSource("github-repo-1", []byte("my-key"))
	require.NoError(t, err)

	validEvent := []byte(`{"action": "created"}`)
	validSignature := "sha256=ee9f99fa8d06b44ffc69ee1c2a7e32e848e8b40536bb5e8405dabb3bbbcaf619"
	validURL := "/sources/" + eventSource.ID.String() + "/github"

	t.Run("missing organization header -> 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method: "POST",
			path:   validURL,
			orgID:  "",
		})

		require.Equal(t, 404, response.Code)
	})

	t.Run("invalid organization header -> 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			orgID:       "not-a-uuid",
			body:        validEvent,
			signature:   validSignature,
			contentType: "application/json",
		})

		require.Equal(t, 404, response.Code)
	})

	t.Run("event for invalid source -> 404", func(t *testing.T) {
		invalidURL := "/sources/invalidsource/github"
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        invalidURL,
			orgID:       orgID.String(),
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
			orgID:       orgID.String(),
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
			orgID:       orgID.String(),
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
			orgID:       orgID.String(),
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
			orgID:       orgID.String(),
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
			orgID:       orgID.String(),
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
			orgID:       orgID.String(),
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
			orgID:       orgID.String(),
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
	server, err := NewServer(&encryptor.NoOpEncryptor{}, signer, "")
	require.NoError(t, err)

	orgID := uuid.New()
	userID := uuid.New()
	canvas, err := models.CreateCanvas(orgID, userID, "test")
	require.NoError(t, err)

	eventSource, err := canvas.CreateEventSource("semaphore-source-1", []byte("my-key"))
	require.NoError(t, err)

	// No need to include organization ID in the payload anymore
	validEvent := []byte(`{"version": "1.0.0", "event_type": "workflow_completed"}`)

	key := []byte("my-key")
	mac := hmac.New(sha256.New, key)
	mac.Write(validEvent)
	validSignatureBytes := mac.Sum(nil)
	validSignature := "sha256=" + hex.EncodeToString(validSignatureBytes)

	validURL := "/sources/" + eventSource.ID.String() + "/semaphore"

	t.Run("missing organization header -> 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			orgID:       "",
			body:        validEvent,
			signature:   validSignature,
			contentType: "application/json",
		})

		require.Equal(t, 404, response.Code)
	})

	t.Run("invalid organization header -> 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			orgID:       "not-a-uuid",
			body:        validEvent,
			signature:   validSignature,
			contentType: "application/json",
		})

		require.Equal(t, 404, response.Code)
	})

	t.Run("event for invalid source -> 404", func(t *testing.T) {
		invalidURL := "/sources/invalidsource/semaphore"
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        invalidURL,
			orgID:       orgID.String(),
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
			orgID:     orgID.String(),
			body:      validEvent,
			signature: validSignature,
		})

		assert.Equal(t, 404, response.Code)
	})

	t.Run("unsupported Content-Type header -> 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			orgID:       orgID.String(),
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
			orgID:       orgID.String(),
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
			orgID:       orgID.String(),
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
			orgID:       orgID.String(),
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
			orgID:       orgID.String(),
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
			orgID:       orgID.String(),
			body:        generateBigBody(t),
			signature:   validSignature,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
		assert.Equal(t, "Request body is too large - must be up to 65536 bytes\n", response.Body.String())
	})
}

func Test__HandleExecutionTags(t *testing.T) {
	r := support.Setup(t)

	signer := jwt.NewSigner("test")
	server, err := NewServer(&encryptor.NoOpEncryptor{}, signer, "")
	require.NoError(t, err)

	execution := support.CreateExecution(t, r.Source, r.Stage)
	validURL := "/executions/" + execution.ID.String() + "/tags"
	validToken, err := signer.Generate(execution.ID.String(), time.Hour)
	require.NoError(t, err)
	tags := []byte(`{"version":"1.0.0","sha":"078fc8755c051"}`)

	t.Run("missing organization header -> 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			orgID:       "",
			body:        tags,
			contentType: "application/json",
			authToken:   validToken,
		})

		require.Equal(t, 404, response.Code)
	})

	t.Run("invalid organization header -> 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			orgID:       "not-a-uuid",
			body:        tags,
			contentType: "application/json",
			authToken:   validToken,
		})

		require.Equal(t, 404, response.Code)
	})

	t.Run("event for invalid execution -> 404", func(t *testing.T) {
		invalidURL := "/executions/invalidsource/tags"
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        invalidURL,
			orgID:       r.Org.String(),
			body:        tags,
			authToken:   validToken,
			contentType: "application/json",
		})

		assert.Equal(t, 404, response.Code)
		assert.Equal(t, "execution not found\n", response.Body.String())
	})

	t.Run("missing Content-Type header -> 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			orgID:       r.Org.String(),
			body:        tags,
			contentType: "",
			authToken:   validToken,
		})

		assert.Equal(t, 404, response.Code)
	})

	t.Run("unsupported Content-Type header -> 400", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			orgID:       r.Org.String(),
			body:        tags,
			contentType: "application/x-www-form-urlencoded",
			authToken:   validToken,
		})

		assert.Equal(t, 404, response.Code)
	})

	t.Run("event for execution that does not exist -> 404", func(t *testing.T) {
		invalidURL := "/executions/" + uuid.New().String() + "/tags"
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        invalidURL,
			orgID:       r.Org.String(),
			body:        tags,
			contentType: "application/json",
			authToken:   validToken,
		})

		assert.Equal(t, 404, response.Code)
		assert.Equal(t, "execution not found\n", response.Body.String())
	})

	t.Run("event with missing authorization header -> 401", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			orgID:       r.Org.String(),
			body:        tags,
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
			path:        validURL,
			orgID:       r.Org.String(),
			body:        tags,
			authToken:   "invalid",
			contentType: "application/json",
		})

		assert.Equal(t, 401, response.Code)
		assert.Equal(t, "Invalid token\n", response.Body.String())
	})

	t.Run("proper request -> 200 and tags are saved in execution", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			orgID:       r.Org.String(),
			body:        tags,
			authToken:   validToken,
			contentType: "application/json",
		})

		assert.Equal(t, 200, response.Code)
		execution, err := models.FindExecutionByID(execution.ID)
		require.NoError(t, err)
		compareJSONB(t, tags, []byte(execution.Tags))
	})

	t.Run("tags are limited to 4k", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        validURL,
			orgID:       r.Org.String(),
			body:        generateBigBody(t),
			authToken:   validToken,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
		assert.Equal(t, "Request body is too large - must be up to 4096 bytes\n", response.Body.String())
	})
}

type requestParams struct {
	method      string
	path        string
	orgID       string
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

	if params.orgID != "" {
		req.Header.Add("x-semaphore-org-id", params.orgID)
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

func compareJSONB(t *testing.T, a, b []byte) {
	var dataA map[string]any
	err := json.Unmarshal(a, &dataA)
	require.NoError(t, err)

	var dataB map[string]any
	err = json.Unmarshal(b, &dataB)
	require.NoError(t, err)

	for k, v := range dataA {
		assert.Equal(t, v, dataB[k])
	}
}
