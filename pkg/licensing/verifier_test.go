package licensing_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
)

func newVerifier(t *testing.T, issuers ...*licensingtest.Issuer) *licensing.Verifier {
	t.Helper()
	return licensing.NewVerifier(licensingtest.KeySet(issuers...))
}

func requireReason(t *testing.T, err error, reason licensing.Reason) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, reason, licensing.ReasonOf(err))
}

func replaceSegment(t *testing.T, token []byte, index int, value []byte) []byte {
	t.Helper()
	segments := strings.Split(string(token), ".")
	require.Len(t, segments, 3)
	segments[index] = base64.RawURLEncoding.EncodeToString(value)
	return []byte(strings.Join(segments, "."))
}

func TestVerifierAcceptsValidLicense(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	claims := licensingtest.Claims(licensing.FeatureGroups, licensing.FeatureCustomRoles)
	token := issuer.Sign(claims)

	license, err := newVerifier(t, issuer).Verify(token)
	require.NoError(t, err)

	assert.Equal(t, claims["jti"], license.ID.String())
	assert.Equal(t, claims["sub"], license.CustomerID.String())
	assert.Equal(t, "test-2026-01", license.KeyID)
	assert.Equal(t, licensing.ExpectedIssuer, license.Issuer)
	assert.Equal(t, licensing.EditionEnterprise, license.Edition)
	assert.Equal(t, []licensing.Feature{licensing.FeatureCustomRoles, licensing.FeatureGroups}, license.Features)
	assert.Equal(t, licensing.ValidityActive, license.ValidityAt(time.Now()))
}

func TestVerifierAcceptsSurroundingWhitespace(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	token := issuer.License(licensing.FeatureGroups)

	_, err := newVerifier(t, issuer).Verify(append(append([]byte("\n  "), token...), '\n'))
	require.NoError(t, err)
}

func TestVerifierRejectsTampering(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	verifier := newVerifier(t, issuer)
	token := issuer.License(licensing.FeatureGroups)

	t.Run("modified claims", func(t *testing.T) {
		claims := licensingtest.Claims(licensing.FeatureGroups, licensing.FeatureCustomRoles)
		claimsJSON, err := json.Marshal(claims)
		require.NoError(t, err)

		_, err = verifier.Verify(replaceSegment(t, token, 1, claimsJSON))
		requireReason(t, err, licensing.ReasonInvalidSignature)
	})

	t.Run("modified protected header", func(t *testing.T) {
		// Same values in a different order change the signed bytes.
		reordered := []byte(`{"typ":"` + licensing.TokenType + `","kid":"` + issuer.KeyID + `","alg":"ES256"}`)

		_, err := verifier.Verify(replaceSegment(t, token, 0, reordered))
		requireReason(t, err, licensing.ReasonInvalidSignature)
	})

	t.Run("modified signature", func(t *testing.T) {
		segments := strings.Split(string(token), ".")
		signature, err := base64.RawURLEncoding.DecodeString(segments[2])
		require.NoError(t, err)
		signature[10] ^= 0xFF

		_, err = verifier.Verify(replaceSegment(t, token, 2, signature))
		requireReason(t, err, licensing.ReasonInvalidSignature)
	})

	t.Run("signature with a different key under a trusted key ID", func(t *testing.T) {
		impostor := licensingtest.NewIssuer(issuer.KeyID)
		_, err := verifier.Verify(impostor.License(licensing.FeatureGroups))
		requireReason(t, err, licensing.ReasonInvalidSignature)
	})

	t.Run("short signature", func(t *testing.T) {
		_, err := verifier.Verify(replaceSegment(t, token, 2, make([]byte, 63)))
		requireReason(t, err, licensing.ReasonInvalidSignature)
	})
}

func TestVerifierRejectsMalformedInput(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	verifier := newVerifier(t, issuer)
	token := string(issuer.License(licensing.FeatureGroups))

	cases := map[string]string{
		"empty":                "",
		"two segments":         "a.b",
		"four segments":        token + ".extra",
		"empty segment":        strings.Replace(token, ".", "..", 1)[1:],
		"padded base64":        strings.Replace(token, ".", "=.", 1),
		"header is not base64": "%%%." + strings.SplitN(token, ".", 2)[1],
		"header is not JSON":   base64.RawURLEncoding.EncodeToString([]byte("nope")) + "." + strings.SplitN(token, ".", 2)[1],
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := verifier.Verify([]byte(input))
			requireReason(t, err, licensing.ReasonMalformed)
		})
	}

	t.Run("input larger than the limit", func(t *testing.T) {
		_, err := verifier.Verify(bytes.Repeat([]byte("a"), licensing.MaxLicenseBytes+1))
		requireReason(t, err, licensing.ReasonMalformed)
	})
}

func TestVerifierRejectsUnsafeHeaders(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	verifier := newVerifier(t, issuer)
	claims := licensingtest.Claims(licensing.FeatureGroups)

	cases := map[string]struct {
		mutate func(map[string]any)
		reason licensing.Reason
	}{
		"alg none":        {func(h map[string]any) { h["alg"] = "none" }, licensing.ReasonUnsupportedAlgorithm},
		"alg HS256":       {func(h map[string]any) { h["alg"] = "HS256" }, licensing.ReasonUnsupportedAlgorithm},
		"alg ES384":       {func(h map[string]any) { h["alg"] = "ES384" }, licensing.ReasonUnsupportedAlgorithm},
		"missing alg":     {func(h map[string]any) { delete(h, "alg") }, licensing.ReasonUnsupportedAlgorithm},
		"wrong typ":       {func(h map[string]any) { h["typ"] = "JWT" }, licensing.ReasonMalformed},
		"missing typ":     {func(h map[string]any) { delete(h, "typ") }, licensing.ReasonMalformed},
		"missing kid":     {func(h map[string]any) { delete(h, "kid") }, licensing.ReasonMalformed},
		"unknown kid":     {func(h map[string]any) { h["kid"] = "attacker-key" }, licensing.ReasonUnknownKey},
		"crit parameter":  {func(h map[string]any) { h["crit"] = []string{"exp"} }, licensing.ReasonMalformed},
		"embedded jwk":    {func(h map[string]any) { h["jwk"] = map[string]string{"kty": "EC"} }, licensing.ReasonMalformed},
		"remote key URL":  {func(h map[string]any) { h["jku"] = "https://attacker.example/jwks.json" }, licensing.ReasonMalformed},
		"certificate URL": {func(h map[string]any) { h["x5u"] = "https://attacker.example/cert.pem" }, licensing.ReasonMalformed},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			header := licensingtest.Header(issuer.KeyID)
			tc.mutate(header)

			_, err := verifier.Verify(issuer.SignWithHeader(header, claims))
			requireReason(t, err, tc.reason)
		})
	}
}

func TestVerifierRejectsInvalidClaims(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	verifier := newVerifier(t, issuer)
	now := time.Now()

	cases := map[string]func(map[string]any){
		"wrong issuer":               func(c map[string]any) { c["iss"] = "https://attacker.example" },
		"missing issuer":             func(c map[string]any) { delete(c, "iss") },
		"wrong audience":             func(c map[string]any) { c["aud"] = "superplane-cloud" },
		"audience array":             func(c map[string]any) { c["aud"] = []string{licensing.ExpectedAudience} },
		"unsupported schema version": func(c map[string]any) { c["schema_version"] = 2 },
		"missing schema version":     func(c map[string]any) { delete(c, "schema_version") },
		"unsupported edition":        func(c map[string]any) { c["edition"] = "community" },
		"malformed subject":          func(c map[string]any) { c["sub"] = "customer-1" },
		"nil subject":                func(c map[string]any) { c["sub"] = "00000000-0000-0000-0000-000000000000" },
		"malformed license ID":       func(c map[string]any) { c["jti"] = "not-a-uuid" },
		"missing license ID":         func(c map[string]any) { delete(c, "jti") },
		"missing expiry":             func(c map[string]any) { delete(c, "exp") },
		"expiry before start":        func(c map[string]any) { c["exp"] = now.Add(-2 * time.Hour).Unix() },
		"expiry before issuance": func(c map[string]any) {
			c["iat"] = now.Add(time.Hour).Unix()
			c["exp"] = now.Add(time.Minute).Unix()
		},
		"string timestamp":      func(c map[string]any) { c["exp"] = "2030-01-01" },
		"empty features":        func(c map[string]any) { c["features"] = []string{} },
		"missing features":      func(c map[string]any) { delete(c, "features") },
		"duplicate features":    func(c map[string]any) { c["features"] = []string{"groups", "groups"} },
		"malformed feature key": func(c map[string]any) { c["features"] = []string{"Groups"} },
		"feature is not string": func(c map[string]any) { c["features"] = []any{1} },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			claims := licensingtest.Claims(licensing.FeatureGroups)
			mutate(claims)

			_, err := verifier.Verify(issuer.Sign(claims))
			requireReason(t, err, licensing.ReasonInvalidClaims)
		})
	}
}

func TestVerifierIgnoresUnknownFeatures(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	claims := licensingtest.Claims(licensing.FeatureGroups)
	claims["features"] = []string{"future_feature", "groups"}

	license, err := newVerifier(t, issuer).Verify(issuer.Sign(claims))
	require.NoError(t, err)
	assert.Equal(t, []licensing.Feature{licensing.FeatureGroups}, license.Features)
	assert.False(t, license.HasFeature("future_feature"))
}

func TestLicenseValidityWindow(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	verifier := newVerifier(t, issuer)
	now := time.Now()

	cases := map[string]struct {
		notBefore time.Time
		expiresAt time.Time
		validity  licensing.Validity
	}{
		"active":                          {now.Add(-time.Hour), now.Add(time.Hour), licensing.ValidityActive},
		"starts within the clock skew":    {now.Add(4 * time.Minute), now.Add(time.Hour), licensing.ValidityActive},
		"starts after the clock skew":     {now.Add(10 * time.Minute), now.Add(time.Hour), licensing.ValidityNotYetValid},
		"expired within the clock skew":   {now.Add(-time.Hour), now.Add(-4 * time.Minute), licensing.ValidityActive},
		"expired beyond the clock skew":   {now.Add(-time.Hour), now.Add(-10 * time.Minute), licensing.ValidityExpired},
		"expired long ago with valid sig": {now.Add(-48 * time.Hour), now.Add(-24 * time.Hour), licensing.ValidityExpired},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			claims := licensingtest.Claims(licensing.FeatureGroups)
			claims["iat"] = tc.notBefore.Unix()
			claims["nbf"] = tc.notBefore.Unix()
			claims["exp"] = tc.expiresAt.Unix()

			license, err := verifier.Verify(issuer.Sign(claims))
			require.NoError(t, err)
			assert.Equal(t, tc.validity, license.ValidityAt(now))
		})
	}
}

func TestVerifierSupportsKeyRotation(t *testing.T) {
	oldIssuer := licensingtest.NewIssuer("license-signing-old")
	newIssuer := licensingtest.NewIssuer("license-signing-new")
	oldLicense := oldIssuer.License(licensing.FeatureGroups)
	newLicense := newIssuer.License(licensing.FeatureGroups)

	t.Run("a release with both keys accepts licenses from both", func(t *testing.T) {
		verifier := newVerifier(t, oldIssuer, newIssuer)

		_, err := verifier.Verify(oldLicense)
		require.NoError(t, err)

		_, err = verifier.Verify(newLicense)
		require.NoError(t, err)
	})

	t.Run("a release without the old key rejects its licenses", func(t *testing.T) {
		verifier := newVerifier(t, newIssuer)

		_, err := verifier.Verify(oldLicense)
		requireReason(t, err, licensing.ReasonUnknownKey)
	})

	t.Run("a release before the new key rejects new licenses", func(t *testing.T) {
		verifier := newVerifier(t, oldIssuer)

		_, err := verifier.Verify(newLicense)
		requireReason(t, err, licensing.ReasonUnknownKey)
	})
}
