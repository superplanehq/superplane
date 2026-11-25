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

// RevocationError is one failed check of a Universal Logout JWT.
// Detail is the check that failed. It is never a token or claim value.
type RevocationError struct {
	Detail string
}

func (e *RevocationError) Error() string {
	return "revocation token is not valid"
}

func revocationError(detail string) error {
	return &RevocationError{Detail: detail}
}

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
		return "", time.Time{}, revocationError(verificationDetail(err))
	}
	tokenType, err := signedJWTType(raw)
	if err != nil {
		return "", time.Time{}, revocationError("signature")
	}
	if !acceptedRevocationType(tokenType) {
		return "", time.Time{}, revocationError("type")
	}
	if len(idToken.Audience) != 1 || idToken.Audience[0] != audience {
		return "", time.Time{}, revocationError("audience")
	}
	if idToken.Subject != cfg.ClientID {
		return "", time.Time{}, revocationError("subject")
	}
	now := a.now()
	if idToken.IssuedAt.IsZero() || idToken.IssuedAt.After(now.Add(globalRevocationSkew)) {
		return "", time.Time{}, revocationError("not_yet_valid")
	}
	if idToken.Expiry.IsZero() || idToken.Expiry.Before(now.Add(-globalRevocationSkew)) {
		return "", time.Time{}, revocationError("expired")
	}
	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return "", time.Time{}, revocationError("signature")
	}
	if notBefore, ok := claimTime(claims, "nbf"); ok && notBefore.After(now.Add(globalRevocationSkew)) {
		return "", time.Time{}, revocationError("not_yet_valid")
	}
	jti, _ := claims["jti"].(string)
	jti = strings.TrimSpace(jti)
	if jti == "" {
		return "", time.Time{}, revocationError("jti_missing")
	}
	return jti, idToken.Expiry, nil
}

func verificationDetail(err error) string {
	text := err.Error()
	switch {
	case strings.Contains(text, "issued by a different provider"):
		return "issuer"
	case strings.Contains(text, "expected audience"):
		return "audience"
	default:
		return "signature"
	}
}

func acceptedRevocationType(tokenType string) bool {
	switch strings.ToLower(strings.TrimSpace(tokenType)) {
	case "", "jwt", globalRevocationType:
		return true
	default:
		return false
	}
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
