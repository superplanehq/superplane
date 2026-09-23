package agent

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ResolveAccessTokenOptions controls how a runner obtains its broker access token.
type ResolveAccessTokenOptions struct {
	BaseURL           string
	AccessToken       string
	RegistrationToken string
	RunnerID          string
	FleetID           string
	// AccessTokenPath, when set, is read before registration and written after a
	// successful exchange so multi-shot runners survive process restarts.
	AccessTokenPath string
}

// ResolveAccessToken returns a runner-scoped bearer from env, a persisted file,
// or by exchanging a one-time registration JWT with the broker.
func ResolveAccessToken(ctx context.Context, client *http.Client, opts ResolveAccessTokenOptions) (string, error) {
	if token := strings.TrimSpace(opts.AccessToken); token != "" {
		return token, nil
	}
	path := strings.TrimSpace(opts.AccessTokenPath)
	if path != "" {
		if raw, err := os.ReadFile(path); err == nil {
			if token := strings.TrimSpace(string(raw)); token != "" {
				return token, nil
			}
		}
	}
	registrationToken := strings.TrimSpace(opts.RegistrationToken)
	if registrationToken == "" {
		return "", fmt.Errorf("RUNNER_ACCESS_TOKEN or RUNNER_REGISTRATION_TOKEN is required")
	}
	token, err := RegisterRunner(ctx, client, opts.BaseURL, registrationToken, opts.RunnerID, opts.FleetID)
	if err != nil {
		return "", err
	}
	// Best-effort: this boot already has a token. A persist failure only hurts restarts.
	if path != "" {
		_ = persistAccessToken(path, token)
	}
	return token, nil
}

func persistAccessToken(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(token+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
