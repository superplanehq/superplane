package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/registry"
)

func setupTestHandler(t *testing.T) (*Handler, *jwt.Signer) {
	jwtSecret := "test-secret-key-123"
	jwtSigner := jwt.NewSigner(jwtSecret)

	reg, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)

	handler := NewHandler(jwtSigner, reg)

	return handler, jwtSigner
}

func createTestToken(t *testing.T, signer *jwt.Signer) string {
	token, err := signer.GenerateWithClaims(24*time.Hour, map[string]string{
		"sub": "test-user-id",
	})
	require.NoError(t, err)
	return token
}

func TestInitialize(t *testing.T) {
	handler, signer := setupTestHandler(t)
	token := createTestToken(t, signer)

	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp jsonRPCResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, float64(1), resp.ID)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)

	// Verify result structure
	result, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "2024-11-05", result["protocolVersion"])
	assert.NotNil(t, result["capabilities"])
	assert.NotNil(t, result["serverInfo"])
}

func TestToolsList(t *testing.T) {
	handler, signer := setupTestHandler(t)
	token := createTestToken(t, signer)

	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp jsonRPCResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)

	tools, ok := result["tools"].([]interface{})
	require.True(t, ok)
	assert.Len(t, tools, 3)

	// Verify tool names
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolMap := tool.(map[string]interface{})
		toolNames[i] = toolMap["name"].(string)
	}
	assert.Contains(t, toolNames, "canvas_get")
	assert.Contains(t, toolNames, "canvas_list_versions")
	assert.Contains(t, toolNames, "integrations_list")
}

func TestUnauthorized(t *testing.T) {
	handler, _ := setupTestHandler(t)

	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      6,
		Method:  "tools/list",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp jsonRPCResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotNil(t, resp.Error)
	assert.Equal(t, InvalidRequest, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "Unauthorized")
}

func TestInvalidMethod(t *testing.T) {
	handler, signer := setupTestHandler(t)
	token := createTestToken(t, signer)

	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      7,
		Method:  "invalid/method",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp jsonRPCResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotNil(t, resp.Error)
	assert.Equal(t, MethodNotFound, resp.Error.Code)
}

func TestInvalidJSONRPCVersion(t *testing.T) {
	handler, signer := setupTestHandler(t)
	token := createTestToken(t, signer)

	reqBody := jsonRPCRequest{
		JSONRPC: "1.0",
		ID:      8,
		Method:  "initialize",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp jsonRPCResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotNil(t, resp.Error)
	assert.Equal(t, InvalidRequest, resp.Error.Code)
}

func TestNotificationNoResponse(t *testing.T) {
	handler, signer := setupTestHandler(t)
	token := createTestToken(t, signer)

	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Notifications should not return a response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Body.String())
}
