package incident

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// svixSign builds a v1 signature: signed = id + "." + timestamp + "." + body, sig = base64(HMAC-SHA256(signed, key)).
func svixSign(key []byte, id, timestamp string, body []byte) string {
	signed := id + "." + timestamp + "." + string(body)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(signed))
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifySvixSignature(t *testing.T) {
	secret := []byte("test-secret-32-bytes-long-enough!!")
	body := []byte(`{"event_type":"public_incident.incident_created_v2"}`)
	now := time.Now()
	tsValid := strconv.FormatInt(now.Unix(), 10)

	t.Run("missing webhook-id", func(t *testing.T) {
		sig := svixSign(secret, "id", tsValid, body)
		err := VerifySvixSignature("", tsValid, sig, body, secret)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required webhook headers")
	})

	t.Run("missing webhook-timestamp", func(t *testing.T) {
		sig := svixSign(secret, "id", tsValid, body)
		err := VerifySvixSignature("id", "", sig, body, secret)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required webhook headers")
	})

	t.Run("missing webhook-signature", func(t *testing.T) {
		err := VerifySvixSignature("id", tsValid, "", body, secret)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required webhook headers")
	})

	t.Run("valid signature with raw secret", func(t *testing.T) {
		id := "msg_123"
		sig := svixSign(secret, id, tsValid, body)
		err := VerifySvixSignature(id, tsValid, sig, body, secret)
		require.NoError(t, err)
	})

	t.Run("valid signature with whsec_ prefix", func(t *testing.T) {
		id := "msg_456"
		encoded := base64.StdEncoding.EncodeToString(secret)
		secretWithPrefix := []byte("whsec_" + encoded)
		sig := svixSign(secret, id, tsValid, body)
		err := VerifySvixSignature(id, tsValid, sig, body, secretWithPrefix)
		require.NoError(t, err)
	})

	t.Run("invalid base64 in whsec_ secret", func(t *testing.T) {
		id := "msg_789"
		secretWithPrefix := []byte("whsec_!!!not-valid-base64!!!")
		sig := svixSign(secret, id, tsValid, body)
		err := VerifySvixSignature(id, tsValid, sig, body, secretWithPrefix)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid secret base64")
	})

	t.Run("signature mismatch", func(t *testing.T) {
		id := "msg_abc"
		sig := svixSign(secret, id, tsValid, body)
		wrongSecret := []byte("wrong-secret-32-bytes-long-enough!!")
		err := VerifySvixSignature(id, tsValid, sig, body, wrongSecret)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signature mismatch")
	})

	t.Run("invalid webhook timestamp", func(t *testing.T) {
		id := "msg_badts"
		sig := svixSign(secret, id, "not-a-number", body)
		err := VerifySvixSignature(id, "not-a-number", sig, body, secret)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid webhook timestamp")
	})

	t.Run("timestamp too old", func(t *testing.T) {
		id := "msg_old"
		tsOld := strconv.FormatInt(now.Unix()-400, 10)
		sig := svixSign(secret, id, tsOld, body)
		err := VerifySvixSignature(id, tsOld, sig, body, secret)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timestamp too old or in future")
	})

	t.Run("timestamp in future", func(t *testing.T) {
		id := "msg_future"
		tsFuture := strconv.FormatInt(now.Unix()+400, 10)
		sig := svixSign(secret, id, tsFuture, body)
		err := VerifySvixSignature(id, tsFuture, sig, body, secret)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timestamp too old or in future")
	})

	t.Run("multiple signature parts with one valid", func(t *testing.T) {
		id := "msg_multi"
		validSig := svixSign(secret, id, tsValid, body)
		// Svix can send multiple parts, e.g. "v1,sig1 v1,sig2"; we accept if any matches.
		multiSig := "v1,dGhpcyBpcyB3cm9uZw== " + validSig
		err := VerifySvixSignature(id, tsValid, multiSig, body, secret)
		require.NoError(t, err)
	})
}
