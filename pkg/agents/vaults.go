package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	anthropicAPIVersion = "2023-06-01"
	vaultCacheTTL       = 1 * time.Hour
)

// VaultManager creates and manages per-user Anthropic vaults for MCP authentication.
type VaultManager struct {
	apiKey       string
	baseURL      string
	httpClient   *http.Client
	mcpServerURL string
}

// NewVaultManager creates a vault manager for the given Anthropic API configuration.
func NewVaultManager(apiKey, baseURL, mcpServerURL string) *VaultManager {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	return &VaultManager{
		apiKey:       apiKey,
		baseURL:      baseURL,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		mcpServerURL: mcpServerURL,
	}
}

// EnsureVaultForUser returns the vault ID for the given user, creating one if needed.
// It also ensures the vault has an up-to-date JWT credential.
func (vm *VaultManager) EnsureVaultForUser(ctx context.Context, organizationID, userID uuid.UUID, jwtToken string) (string, error) {
	// Check if vault already exists in DB
	existing, err := models.FindAgentVaultForUser(userID, organizationID, "anthropic")
	if err == nil && existing != nil {
		// Vault exists — update credential with fresh JWT
		if err := vm.updateVaultCredential(ctx, existing.ProviderVaultID, existing.CredentialID, jwtToken); err != nil {
			log.WithError(err).Warn("failed to update vault credential, will recreate")
			// Don't fail hard — we'll recreate below if needed
		} else {
			return existing.ProviderVaultID, nil
		}
	}

	// Need to create or recreate vault
	return vm.provisionVault(ctx, organizationID, userID, jwtToken)
}

// provisionVault creates a new Anthropic vault + credential and persists to DB.
func (vm *VaultManager) provisionVault(ctx context.Context, organizationID, userID uuid.UUID, jwtToken string) (string, error) {
	// Create vault via Anthropic API
	vaultID, err := vm.createVault(ctx, organizationID, userID)
	if err != nil {
		return "", fmt.Errorf("create vault: %w", err)
	}

	// Create static_bearer credential
	credentialID, err := vm.createVaultCredential(ctx, vaultID, jwtToken)
	if err != nil {
		return "", fmt.Errorf("create credential: %w", err)
	}

	// Persist to DB
	if err := database.Conn().Transaction(func(tx *gorm.DB) error {
		_, err := models.FindOrCreateAgentVault(tx, userID, organizationID, "anthropic", vaultID, credentialID, vm.mcpServerURL)
		return err
	}); err != nil {
		log.WithError(err).Warn("failed to persist vault to database")
		// Don't fail — vault is created upstream, DB is just caching
	}

	return vaultID, nil
}

// createVault calls POST /v1/vaults to create a new vault.
func (vm *VaultManager) createVault(ctx context.Context, organizationID, userID uuid.UUID) (string, error) {
	body := map[string]interface{}{
		"display_name": fmt.Sprintf("SuperPlane MCP (%s)", userID.String()[:8]),
		"metadata": map[string]string{
			"organization_id": organizationID.String(),
			"user_id":         userID.String(),
			"purpose":         "mcp-auth",
		},
	}

	var resp struct {
		ID string `json:"id"`
	}
	if err := vm.doAnthropicRequest(ctx, "POST", "/vaults", body, &resp); err != nil {
		return "", err
	}
	if resp.ID == "" {
		return "", fmt.Errorf("anthropic returned empty vault id")
	}
	return resp.ID, nil
}

// createVaultCredential calls POST /v1/vaults/{id}/credentials to add a static_bearer credential.
func (vm *VaultManager) createVaultCredential(ctx context.Context, vaultID, jwtToken string) (string, error) {
	body := map[string]interface{}{
		"auth": map[string]interface{}{
			"type":           "static_bearer",
			"mcp_server_url": vm.mcpServerURL,
			"token":          jwtToken,
		},
	}

	var resp struct {
		ID string `json:"id"`
	}
	if err := vm.doAnthropicRequest(ctx, "POST", fmt.Sprintf("/vaults/%s/credentials", vaultID), body, &resp); err != nil {
		return "", err
	}
	if resp.ID == "" {
		return "", fmt.Errorf("anthropic returned empty credential id")
	}
	return resp.ID, nil
}

// updateVaultCredential calls PUT /v1/vaults/{vaultID}/credentials/{credentialID} to update the token.
func (vm *VaultManager) updateVaultCredential(ctx context.Context, vaultID, credentialID, jwtToken string) error {
	body := map[string]interface{}{
		"auth": map[string]interface{}{
			"type":           "static_bearer",
			"mcp_server_url": vm.mcpServerURL,
			"token":          jwtToken,
		},
	}

	// Delete old credential and create new one (update not supported)
	_ = vm.doAnthropicRequest(ctx, "DELETE", fmt.Sprintf("/vaults/%s/credentials/%s", vaultID, credentialID), nil, nil)
	var resp struct{ ID string `json:"id"` }
	if err := vm.doAnthropicRequest(ctx, "POST", fmt.Sprintf("/vaults/%s/credentials", vaultID), body, &resp); err != nil {
		return err
	}
	return nil
}

// doAnthropicRequest is a helper for calling the Anthropic API.
func (vm *VaultManager) doAnthropicRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, vm.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("x-api-key", vm.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	req.Header.Set("anthropic-beta", "managed-agents-2026-04-01")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := vm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("anthropic API %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
