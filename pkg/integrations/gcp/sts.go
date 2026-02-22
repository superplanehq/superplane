package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	stsTokenURL = "https://sts.googleapis.com/v1/token"

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
