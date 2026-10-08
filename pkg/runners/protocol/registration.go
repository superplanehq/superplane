package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxErrorResponseBytes = 4 * 1024

type RegistrationDetails struct {
	Version  string            `json:"version"`
	OS       string            `json:"os"`
	Arch     string            `json:"arch"`
	Hostname string            `json:"hostname"`
	Tags     map[string]string `json:"tags,omitempty"`
}

// Registration is the identity and revocable credential returned after the
// server consumes a generic or task-specific registration token.
type Registration struct {
	RunnerID    string `json:"runner_id"`
	FleetID     string `json:"fleet_id"`
	AccessToken string `json:"access_token"`
	Ephemeral   bool   `json:"ephemeral"`
}

func Register(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	registrationToken string,
	details RegistrationDetails,
) (Registration, error) {
	if client == nil {
		client = http.DefaultClient
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	registrationToken = strings.TrimSpace(registrationToken)
	details.Version = strings.TrimSpace(details.Version)
	switch {
	case baseURL == "":
		return Registration{}, errors.New("runner API URL is required")
	case registrationToken == "":
		return Registration{}, errors.New("runner registration token is required")
	case details.Version == "":
		return Registration{}, errors.New("runner version is required")
	}

	body, err := json.Marshal(details)
	if err != nil {
		return Registration{}, fmt.Errorf("encode runner registration: %w", err)
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+"/runner/v1/register",
		bytes.NewReader(body),
	)
	if err != nil {
		return Registration{}, fmt.Errorf("create runner registration request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+registrationToken)
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return Registration{}, fmt.Errorf("register runner: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		content, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorResponseBytes))
		return Registration{}, fmt.Errorf(
			"register runner: status %d: %s",
			response.StatusCode,
			strings.TrimSpace(string(content)),
		)
	}

	var registration Registration
	if err := json.NewDecoder(response.Body).Decode(&registration); err != nil {
		return Registration{}, fmt.Errorf("decode runner registration: %w", err)
	}
	if strings.TrimSpace(registration.RunnerID) == "" ||
		strings.TrimSpace(registration.FleetID) == "" ||
		strings.TrimSpace(registration.AccessToken) == "" {
		return Registration{}, errors.New("register runner: response is incomplete")
	}
	return registration, nil
}
