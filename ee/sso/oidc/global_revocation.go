package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	globalRevocationType = "global-token-revocation+jwt"
	globalRevocationSkew = 2 * time.Minute
)

// VerifyGlobalRevocation checks an Okta Universal Logout JWT.
// audience is the revocation endpoint URL. The token subject must be the
// connection client ID. The returned expiry is the token exp claim.
func (a *Adapter) VerifyGlobalRevocation(ctx context.Context, conn *models.SSOConnection, raw, audience string) (string, time.Time, error) {
	cfg, err := conn.OIDCConfig()
	if err != nil {
		return "", time.Time{}, err
	}
	if strings.TrimSpace(cfg.Issuer) == "" || strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(audience) == "" {
		return "", time.Time{}, fmt.Errorf("revocation token is not valid")
	}
	provider, err := a.provider(ctx, conn, cfg.Issuer)
	if err != nil {
		return "", time.Time{}, err
	}
	idToken, err := provider.Verifier(&oidc.Config{
		ClientID:        audience,
		SkipExpiryCheck: true,
	}).Verify(a.clientContext(ctx), raw)
	if err != nil {
		return "", time.Time{}, err
	}
	tokenType, err := signedJWTType(raw)
	if err != nil || tokenType != globalRevocationType {
		return "", time.Time{}, fmt.Errorf("revocation token type is not valid")
	}
	if len(idToken.Audience) != 1 || idToken.Audience[0] != audience {
		return "", time.Time{}, fmt.Errorf("revocation token audience is not valid")
	}
	if idToken.Subject != cfg.ClientID {
		return "", time.Time{}, fmt.Errorf("revocation token subject is not valid")
	}
	now := a.now()
	if idToken.IssuedAt.IsZero() || idToken.IssuedAt.After(now.Add(globalRevocationSkew)) {
		return "", time.Time{}, fmt.Errorf("revocation token is not valid yet")
	}
	if idToken.Expiry.IsZero() || idToken.Expiry.Before(now.Add(-globalRevocationSkew)) {
		return "", time.Time{}, fmt.Errorf("revocation token is expired")
	}
	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return "", time.Time{}, err
	}
	if notBefore, ok := claimTime(claims, "nbf"); ok && notBefore.After(now.Add(globalRevocationSkew)) {
		return "", time.Time{}, fmt.Errorf("revocation token is not valid yet")
	}
	jti, _ := claims["jti"].(string)
	jti = strings.TrimSpace(jti)
	if jti == "" {
		return "", time.Time{}, fmt.Errorf("revocation token id is missing")
	}
	return jti, idToken.Expiry, nil
}

func signedJWTType(raw string) (string, error) {
	header, _, ok := strings.Cut(raw, ".")
	if !ok || header == "" {
		return "", fmt.Errorf("revocation token is not valid")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(header)
	if err != nil {
		return "", err
	}
	var document struct {
		Type string `json:"typ"`
	}
	if err := json.Unmarshal(decoded, &document); err != nil {
		return "", err
	}
	return document.Type, nil
}

func claimTime(claims map[string]any, name string) (time.Time, bool) {
	value, ok := claims[name]
	if !ok || value == nil {
		return time.Time{}, false
	}
	seconds, ok := value.(float64)
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(int64(seconds), 0), true
}
