package githubapp

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/jwt"
)

const (
	createAppIntent   = "github-app-create"
	createAppStateTTL = 30 * time.Minute
	createAppPrefix   = "github-create:"
	defaultReturnPath = "/"
)

func SignCreateState(secret, returnPath string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("GitHub App create state secret is required")
	}
	path := SafeReturnPath(returnPath)
	token, err := jwt.NewSigner(secret).GenerateWithClaims(createAppStateTTL, map[string]string{
		"intent": createAppIntent,
		"jti":    uuid.NewString(),
		"path":   path,
	})
	if err != nil {
		return "", fmt.Errorf("sign GitHub App create state: %w", err)
	}
	return createAppPrefix + token, nil
}

func VerifyCreateState(secret, state string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("GitHub App create state secret is required")
	}
	token, ok := strings.CutPrefix(strings.TrimSpace(state), createAppPrefix)
	if !ok {
		return "", errors.New("invalid GitHub App create state")
	}
	claims, err := jwt.NewSigner(secret).ValidateAndGetClaims(token)
	if err != nil {
		return "", fmt.Errorf("verify GitHub App create state: %w", err)
	}
	intent, _ := claims["intent"].(string)
	nonce, _ := claims["jti"].(string)
	path, _ := claims["path"].(string)
	if intent != createAppIntent || nonce == "" {
		return "", errors.New("invalid GitHub App create state claims")
	}
	return SafeReturnPath(path), nil
}

func SafeReturnPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return defaultReturnPath
	}
	if strings.ContainsAny(path, "\r\n") {
		return defaultReturnPath
	}
	return path
}
