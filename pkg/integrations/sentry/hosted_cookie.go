package sentry

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	SetupCookieName = "sp_sentry_app_setup"
	setupCookieTTL  = 15 * time.Minute
)

type setupCookiePayload struct {
	IntegrationID string `json:"integrationId"`
	Nonce         string `json:"nonce"`
	ExpiresAt     int64  `json:"expiresAt"`
}

func SignSetupCookie(secret, integrationID, nonce string, now time.Time) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("missing cookie secret")
	}
	if strings.TrimSpace(integrationID) == "" || strings.TrimSpace(nonce) == "" {
		return "", fmt.Errorf("missing setup cookie fields")
	}

	payload := setupCookiePayload{
		IntegrationID: integrationID,
		Nonce:         nonce,
		ExpiresAt:     now.Add(setupCookieTTL).Unix(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(raw)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + hex.EncodeToString(mac.Sum(nil)), nil
}

func ParseSetupCookie(secret, value string, now time.Time) (string, string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", "", fmt.Errorf("missing cookie secret")
	}

	encoded, signature, ok := strings.Cut(strings.TrimSpace(value), ".")
	if !ok || encoded == "" || signature == "" {
		return "", "", fmt.Errorf("invalid setup cookie")
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", fmt.Errorf("invalid setup cookie")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(raw)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(strings.ToLower(expected))) {
		return "", "", fmt.Errorf("invalid setup cookie signature")
	}

	var payload setupCookiePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", "", fmt.Errorf("invalid setup cookie")
	}
	if payload.IntegrationID == "" || payload.Nonce == "" {
		return "", "", fmt.Errorf("invalid setup cookie")
	}
	if payload.ExpiresAt <= now.Unix() {
		return "", "", fmt.Errorf("setup cookie expired")
	}

	return payload.IntegrationID, payload.Nonce, nil
}
