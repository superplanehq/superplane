package buildkite

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	SignatureHeader = "X-Buildkite-Signature"
	TokenHeader     = "X-Buildkite-Token"
	MaxReplayAge    = 5 * time.Minute
)

func VerifyWebhook(headers http.Header, body []byte, secret []byte) error {
	if signature := headers.Get(SignatureHeader); signature != "" {
		return verifySignature(signature, body, secret)
	}

	if token := headers.Get(TokenHeader); token != "" {
		return verifyToken(token, secret)
	}

	return fmt.Errorf("no verification header found: expected %s or %s", SignatureHeader, TokenHeader)
}

func verifySignature(signatureHeader string, body []byte, secret []byte) error {
	parts := strings.Split(signatureHeader, ",")
	if len(parts) != 2 {
		return fmt.Errorf("invalid signature format")
	}

	var timestampStr, signature string
	for _, part := range parts {
		kv := strings.Split(strings.TrimSpace(part), "=")
		if len(kv) != 2 {
			return fmt.Errorf("invalid signature part format: %s", part)
		}
		switch kv[0] {
		case "timestamp":
			timestampStr = kv[1]
		case "signature":
			signature = kv[1]
		}
	}

	if timestampStr == "" || signature == "" {
		return fmt.Errorf("missing timestamp or signature")
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %v", err)
	}

	sigTime := time.Unix(timestamp, 0)
	if time.Since(sigTime) > MaxReplayAge {
		return fmt.Errorf("timestamp too old: max age is %v", MaxReplayAge)
	}

	expectedMAC := hmac.New(sha256.New, secret)
	expectedMAC.Write([]byte(fmt.Sprintf("%s.%s", timestampStr, string(body))))
	expectedSignature := hex.EncodeToString(expectedMAC.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

func verifyToken(token string, secret []byte) error {
	tokenBytes := []byte(token)

	if subtle.ConstantTimeCompare(tokenBytes, secret) != 1 {
		return fmt.Errorf("token mismatch")
	}

	return nil
}
