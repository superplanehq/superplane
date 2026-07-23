package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplane/runner/shared/api"
)

// RegisterRunner exchanges a one-time bootstrap token for a runner-scoped bearer.
func RegisterRunner(ctx context.Context, client *http.Client, baseURL, registrationToken, runnerID, fleetID string) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	body, err := json.Marshal(api.RegisterRunnerRequest{RunnerID: runnerID, FleetID: fleetID})
	if err != nil {
		return "", err
	}
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/v1/runners/register"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(registrationToken))
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("register runner: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("register runner: status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out api.RegisterRunnerResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("register runner: decode: %w", err)
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return "", fmt.Errorf("register runner: empty access token")
	}
	return out.AccessToken, nil
}
