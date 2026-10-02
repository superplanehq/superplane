package pagerduty

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/crypto"
)

func Test__parseWebhookSignatures(t *testing.T) {
	t.Run("single signature", func(t *testing.T) {
		assert.Equal(t, []string{"abc"}, parseWebhookSignatures("v1=abc"))
	})

	t.Run("several signatures, as sent while a secret is being rotated", func(t *testing.T) {
		assert.Equal(t, []string{"abc", "def"}, parseWebhookSignatures("v1=abc,v1=def"))
	})

	t.Run("other signature versions are ignored", func(t *testing.T) {
		assert.Equal(t, []string{"abc"}, parseWebhookSignatures("v2=zzz,v1=abc"))
	})

	t.Run("surrounding whitespace is tolerated", func(t *testing.T) {
		assert.Equal(t, []string{"abc", "def"}, parseWebhookSignatures("v1=abc, v1=def"))
	})

	t.Run("header without a v1 signature", func(t *testing.T) {
		assert.Empty(t, parseWebhookSignatures("garbage"))
		assert.Empty(t, parseWebhookSignatures("v2=abc"))
		assert.Empty(t, parseWebhookSignatures(""))
	})

	t.Run("an oversized header is not scanned indefinitely", func(t *testing.T) {
		header := strings.TrimSuffix(strings.Repeat("v1=abc,", 100_000), ",")
		assert.Len(t, parseWebhookSignatures(header), maxWebhookSignatures)
	})
}

func Test__verifyWebhookSignatures(t *testing.T) {
	secret := []byte("test-secret")
	body := []byte(`{"event":{"event_type":"incident.triggered"}}`)
	valid := crypto.Sign(secret, body)
	other := crypto.Sign([]byte("rotated-secret"), body)

	t.Run("single valid signature", func(t *testing.T) {
		assert.NoError(t, verifyWebhookSignatures([]string{valid}, secret, body))
	})

	t.Run("matching signature listed second", func(t *testing.T) {
		assert.NoError(t, verifyWebhookSignatures([]string{other, valid}, secret, body))
	})

	t.Run("matching signature listed first", func(t *testing.T) {
		assert.NoError(t, verifyWebhookSignatures([]string{valid, other}, secret, body))
	})

	t.Run("no signature matches", func(t *testing.T) {
		err := verifyWebhookSignatures([]string{other}, secret, body)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("body must match the signed payload", func(t *testing.T) {
		err := verifyWebhookSignatures([]string{valid}, secret, []byte("tampered"))
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("forged signatures are rejected", func(t *testing.T) {
		for i := range 100 {
			forged := crypto.Sign([]byte(fmt.Sprintf("guess-%d", i)), body)
			err := verifyWebhookSignatures([]string{forged}, secret, body)
			assert.ErrorContains(t, err, "invalid signature")
		}
	})
}
