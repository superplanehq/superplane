package pagerduty

import (
	"crypto/hmac"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/crypto"
)

// PagerDuty signs a delivery with every active secret, so a rotation in progress produces
// a handful of signatures at most. Bounding the scan keeps an oversized header cheap to reject.
const maxWebhookSignatures = 32

type NodeMetadata struct {
	Service *Service `json:"service"`
}

// parseWebhookSignatures returns the v1 signatures carried by the X-PagerDuty-Signature header.
//
// The header holds one or more comma-separated signatures, because PagerDuty signs each delivery
// with every secret that is currently active for the subscription.
func parseWebhookSignatures(header string) []string {
	signatures := []string{}

	remaining := header
	for range maxWebhookSignatures {
		if remaining == "" {
			break
		}

		candidate, rest, _ := strings.Cut(remaining, ",")
		remaining = rest

		if value, found := strings.CutPrefix(strings.TrimSpace(candidate), "v1="); found {
			signatures = append(signatures, value)
		}
	}

	return signatures
}

// verifyWebhookSignatures reports whether any of the signatures matches the body.
//
// The body is hashed once and compared against each signature, so a delivery carrying several
// signatures costs no more to verify than one carrying a single signature.
func verifyWebhookSignatures(signatures []string, secret []byte, body []byte) error {
	expected := []byte(crypto.Sign(secret, body))
	for _, signature := range signatures {
		if hmac.Equal(expected, []byte(signature)) {
			return nil
		}
	}

	return fmt.Errorf("invalid signature")
}
