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

func SignCreateState(secret, returnPath string, accountID uuid.UUID) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("GitHub App create state secret is required")
	}
	if accountID == uuid.Nil {
		return "", errors.New("account id is required")
	}
	path := SafeReturnPath(returnPath)
	token, err := jwt.NewSigner(secret).GenerateWithClaims(createAppStateTTL, map[string]string{
		"intent": createAppIntent,
		"jti":    uuid.NewString(),
		"path":   path,
		"sub":    accountID.String(),
	})
	if err != nil {
		return "", fmt.Errorf("sign GitHub App create state: %w", err)
	}
	return createAppPrefix + token, nil
}

func VerifyCreateState(secret, state string) (string, uuid.UUID, error) {
	if strings.TrimSpace(secret) == "" {
		return "", uuid.Nil, errors.New("GitHub App create state secret is required")
	}
	token, ok := strings.CutPrefix(strings.TrimSpace(state), createAppPrefix)
	if !ok {
		return "", uuid.Nil, errors.New("invalid GitHub App create state")
	}
	claims, err := jwt.NewSigner(secret).ValidateAndGetClaims(token)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("verify GitHub App create state: %w", err)
	}
	intent, _ := claims["intent"].(string)
	nonce, _ := claims["jti"].(string)
	path, _ := claims["path"].(string)
	subject, _ := claims["sub"].(string)
	if intent != createAppIntent || nonce == "" || subject == "" {
		return "", uuid.Nil, errors.New("invalid GitHub App create state claims")
	}
	accountID, err := uuid.Parse(subject)
	if err != nil || accountID == uuid.Nil {
		return "", uuid.Nil, errors.New("invalid account id in GitHub App create state")
	}
	return SafeReturnPath(path), accountID, nil
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
