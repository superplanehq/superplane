package buildkite

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func Test_VerifyWebhook_ValidSignature(t *testing.T) {
	secret := []byte("test-secret")
	body := []byte(`{"event": "build.finished", "build": {"id": "test"}}`)
	ts := time.Now().Unix()

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(fmt.Sprintf("%d.%s", ts, body)))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	headers := http.Header{}
	headers.Set(SignatureHeader, fmt.Sprintf("timestamp=%d,signature=%s", ts, expectedSignature))

	err := VerifyWebhook(headers, body, secret)
	if err != nil {
		t.Errorf("Expected valid signature to pass, got error: %v", err)
	}
}

func Test_VerifyWebhook_InvalidSignature(t *testing.T) {
	secret := []byte("test-secret")
	body := []byte(`{"event": "build.finished"}`)

	headers := http.Header{}
	headers.Set(SignatureHeader, "timestamp=123,signature=invalid_signature")

	err := VerifyWebhook(headers, body, secret)
	if err == nil {
		t.Error("Expected invalid signature to fail")
	}
}

func Test_VerifyWebhook_MissingSignatureParts(t *testing.T) {
	secret := []byte("test-secret")
	body := []byte(`{"event": "build.finished"}`)

	headers := http.Header{}
	headers.Set(SignatureHeader, "timestamp=123")

	err := VerifyWebhook(headers, body, secret)
	if err == nil {
		t.Error("Expected missing signature part to fail")
	}
}

func Test_VerifyWebhook_ReplayWindow(t *testing.T) {
	secret := []byte("test-secret")
	body := []byte(`{"event": "build.finished"}`)

	ts := time.Now().Add(-10 * time.Minute).Unix()
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(fmt.Sprintf("%d.%s", ts, body)))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	headers := http.Header{}
	headers.Set(SignatureHeader, fmt.Sprintf("timestamp=%d,signature=%s", ts, expectedSignature))

	err := VerifyWebhook(headers, body, secret)
	if err == nil {
		t.Error("Expected old timestamp to fail replay window check")
	}
}

func Test_VerifyWebhook_ValidToken(t *testing.T) {
	secret := []byte("test-token")
	body := []byte(`{"event": "build.finished"}`)

	headers := http.Header{}
	headers.Set(TokenHeader, "test-token")

	err := VerifyWebhook(headers, body, secret)
	if err != nil {
		t.Errorf("Expected valid token to pass, got error: %v", err)
	}
}

func Test_VerifyWebhook_InvalidToken(t *testing.T) {
	secret := []byte("test-secret")
	body := []byte(`{"event": "build.finished"}`)

	headers := http.Header{}
	headers.Set(TokenHeader, "wrong-token")

	err := VerifyWebhook(headers, body, secret)
	if err == nil {
		t.Error("Expected invalid token to fail")
	}
}

func Test_VerifyWebhook_NoVerificationHeader(t *testing.T) {
	secret := []byte("test-secret")
	body := []byte(`{"event": "build.finished"}`)

	headers := http.Header{}

	err := VerifyWebhook(headers, body, secret)
	if err == nil {
		t.Error("Expected missing verification headers to fail")
	}
}

func Test_VerifyWebhook_SignaturePreferredOverToken(t *testing.T) {
	secret := []byte("test-secret")
	body := []byte(`{"event": "build.finished"}`)
	ts := time.Now().Unix()

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(fmt.Sprintf("%d.%s", ts, body)))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	headers := http.Header{}
	headers.Set(SignatureHeader, fmt.Sprintf("timestamp=%d,signature=%s", ts, expectedSignature))
	headers.Set(TokenHeader, "wrong-token")

	err := VerifyWebhook(headers, body, secret)
	if err != nil {
		t.Errorf("Expected signature verification to succeed even with wrong token, got error: %v", err)
	}
}

func Test_verifySignature_InvalidTimestamp(t *testing.T) {
	secret := []byte("test-secret")
	body := []byte(`{"event": "build.finished"}`)

	headers := http.Header{}
	headers.Set(SignatureHeader, "timestamp=invalid,signature=abc123")

	err := verifySignature(headers.Get(SignatureHeader), body, secret)
	if err == nil {
		t.Error("Expected invalid timestamp to fail")
	}
}

func Test_verifyToken_ConstantTimeComparison(t *testing.T) {
	secret := []byte("test-secret-token")

	err := verifyToken("test-secret-token", secret)
	if err != nil {
		t.Errorf("Expected matching token to pass, got error: %v", err)
	}

	err = verifyToken("wrong-token", secret)
	if err == nil {
		t.Error("Expected wrong token to fail")
	}
}
