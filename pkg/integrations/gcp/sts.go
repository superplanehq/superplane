package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	stsTokenURL                                = "https://sts.googleapis.com/v1/token"
	iamCredentialsGenerateAccessTokenURLFormat = "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/%s:generateAccessToken"

	grantTypeTokenExchange   = "urn:ietf:params:oauth:grant-type:token-exchange"
	requestedTokenTypeAccess = "urn:ietf:params:oauth:token-type:access_token"
	subjectTokenTypeJWT      = "urn:ietf:params:oauth:token-type:jwt"
	scopeCloudPlatformSTS    = "https://www.googleapis.com/auth/cloud-platform"
)

type stsTokenRequest struct {
	GrantType          string `json:"grantType"`
	Audience           string `json:"audience"`
	Scope              string `json:"scope"`
	RequestedTokenType string `json:"requestedTokenType"`
	SubjectToken       string `json:"subjectToken"`
	SubjectTokenType   string `json:"subjectTokenType"`
}

type stsTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type stsErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

type serviceAccountTokenRequest struct {
	Scope    []string `json:"scope"`
	Lifetime string   `json:"lifetime,omitempty"`
}

type serviceAccountTokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpireTime  string `json:"expireTime"`
}

func ExchangeToken(ctx context.Context, httpClient core.HTTPContext, oidcToken, audience string) (accessToken string, expiresIn time.Duration, err error) {
	reqBody := stsTokenRequest{
		GrantType:          grantTypeTokenExchange,
		Audience:           audience,
		Scope:              scopeCloudPlatformSTS,
		RequestedTokenType: requestedTokenTypeAccess,
		SubjectToken:       oidcToken,
		SubjectTokenType:   subjectTokenTypeJWT,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("marshal STS request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, stsTokenURL, bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("create STS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("STS request failed: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", 0, fmt.Errorf("read STS response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		var errResp stsErrorResponse
		msg := string(resBody)
		if json.Unmarshal(resBody, &errResp) == nil && errResp.Error.Message != "" {
			msg = errResp.Error.Message
		}
		return "", 0, fmt.Errorf("STS token exchange failed (%d): %s", res.StatusCode, msg)
	}

	var tokResp stsTokenResponse
	if err := json.Unmarshal(resBody, &tokResp); err != nil {
		return "", 0, fmt.Errorf("parse STS response: %w", err)
	}
	if tokResp.AccessToken == "" {
		return "", 0, fmt.Errorf("STS response missing access_token")
	}

	expiresIn = time.Duration(tokResp.ExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = time.Hour
	}
	return tokResp.AccessToken, expiresIn, nil
}

func GenerateServiceAccountAccessToken(
	ctx context.Context,
	httpClient core.HTTPContext,
	federatedAccessToken,
	serviceAccountEmail string,
	scopes ...string,
) (accessToken string, expiresIn time.Duration, err error) {
	federatedAccessToken = strings.TrimSpace(federatedAccessToken)
	if federatedAccessToken == "" {
		return "", 0, fmt.Errorf("federated access token is required")
	}

	serviceAccountEmail = strings.TrimSpace(serviceAccountEmail)
	if serviceAccountEmail == "" {
		return "", 0, fmt.Errorf("service account email is required")
	}

	if len(scopes) == 0 {
		scopes = []string{scopeCloudPlatformSTS}
	}

	reqBody := serviceAccountTokenRequest{
		Scope:    scopes,
		Lifetime: "3600s",
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("marshal service account token request: %w", err)
	}

	tokenURL := fmt.Sprintf(iamCredentialsGenerateAccessTokenURLFormat, url.PathEscape(serviceAccountEmail))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("create service account token request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+federatedAccessToken)
	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("service account token request failed: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", 0, fmt.Errorf("read service account token response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		var errResp stsErrorResponse
		msg := string(resBody)
		if json.Unmarshal(resBody, &errResp) == nil && errResp.Error.Message != "" {
			msg = errResp.Error.Message
		}
		return "", 0, fmt.Errorf("service account token generation failed (%d): %s", res.StatusCode, msg)
	}

	var tokResp serviceAccountTokenResponse
	if err := json.Unmarshal(resBody, &tokResp); err != nil {
		return "", 0, fmt.Errorf("parse service account token response: %w", err)
	}

	if tokResp.AccessToken == "" {
		return "", 0, fmt.Errorf("service account token response missing accessToken")
	}

	expireTime, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(tokResp.ExpireTime))
	if err != nil {
		return "", 0, fmt.Errorf("parse service account token expiry: %w", err)
	}

	expiresIn = time.Until(expireTime)
	if expiresIn <= 0 {
		return "", 0, fmt.Errorf("service account token is already expired")
	}

	return tokResp.AccessToken, expiresIn, nil
}
