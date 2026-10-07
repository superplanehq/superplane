package openrouter

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// AuthorizeURL is where the user approves the connection. OpenRouter's OAuth
	// is PKCE-only: there is no app to register, so no client ID or secret.
	AuthorizeURL = "https://openrouter.ai/auth"

	// KeysURL exchanges the authorization code for an inference API key.
	KeysURL = baseURL + "/auth/keys"
)

const (
	// SecretAPIKey holds the inference key returned by the OAuth exchange.
	SecretAPIKey = "apiKey"

	// SecretCodeVerifier holds the PKCE verifier between the authorize redirect
	// and the callback. It is a secret rather than metadata because integration
	// metadata is serialized to the browser in plaintext.
	SecretCodeVerifier = "codeVerifier"
)

type Metadata struct {
	State string `json:"state" mapstructure:"state"`
}

type Auth struct {
	http core.HTTPContext
}

func NewAuth(httpClient core.HTTPContext) *Auth {
	return &Auth{http: httpClient}
}

// newCodeVerifier returns a PKCE code verifier. RFC 7636 requires unpadded
// base64url, so this cannot reuse crypto.Base64String, which pads.
func newCodeVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// codeChallenge derives the S256 challenge from a verifier.
func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// authorizeURL builds the consent URL. The state is carried inside callback_url
// rather than as its own parameter, because OpenRouter's authorize endpoint only
// accepts callback_url, code_challenge and code_challenge_method.
func authorizeURL(callbackURL, state, verifier string) string {
	callback := callbackURL
	if state != "" {
		callback = fmt.Sprintf("%s?state=%s", callbackURL, url.QueryEscape(state))
	}

	return fmt.Sprintf(
		"%s?callback_url=%s&code_challenge=%s&code_challenge_method=S256",
		AuthorizeURL,
		url.QueryEscape(callback),
		url.QueryEscape(codeChallenge(verifier)),
	)
}

type exchangeRequest struct {
	Code                string `json:"code"`
	CodeVerifier        string `json:"code_verifier"`
	CodeChallengeMethod string `json:"code_challenge_method"`
}

type exchangeResponse struct {
	Key string `json:"key"`
}

// ExchangeCode trades the authorization code for an inference API key. The
// response carries a normal sk-or-v1 key; OpenRouter's OAuth cannot issue a
// provisioning key, which is why Get Credits still takes one from configuration.
func (a *Auth) ExchangeCode(code, verifier string) (string, error) {
	body, err := json.Marshal(exchangeRequest{
		Code:                code,
		CodeVerifier:        verifier,
		CodeChallengeMethod: "S256",
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal exchange request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, KeysURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to build exchange request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := a.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchange request failed: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := readAll(res)
	if err != nil {
		return "", err
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", apiError(res.StatusCode, responseBody)
	}

	var parsed exchangeResponse
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return "", fmt.Errorf("failed to unmarshal exchange response: %v", err)
	}

	if parsed.Key == "" {
		return "", fmt.Errorf("exchange returned an empty key")
	}

	return parsed.Key, nil
}

// findSecret returns a stored secret by name, or "" when it is not set.
func findSecret(integration core.IntegrationContext, name string) (string, error) {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, secret := range secrets {
		if secret.Name == name {
			return string(secret.Value), nil
		}
	}

	return "", nil
}
