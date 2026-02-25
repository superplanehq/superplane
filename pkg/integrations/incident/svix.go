package incident

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Svix timestamp tolerance for replay protection (seconds).
const svixTimestampToleranceSec = 300

// VerifySvixSignature verifies the incident.io (Svix) webhook signature.
// Headers: webhook-id, webhook-timestamp, webhook-signature (incident.io uses webhook- prefix).
// Signed content: id + "." + timestamp + "." + raw_body
// Secret: if it has "whsec_" prefix, the key is the base64-decoded part after it; otherwise raw bytes.
func VerifySvixSignature(webhookID, webhookTimestamp, webhookSignature string, body []byte, secret []byte) error {
	if webhookID == "" || webhookTimestamp == "" || webhookSignature == "" {
		return fmt.Errorf("missing required webhook headers")
	}

	key := secret
	if strings.HasPrefix(string(secret), "whsec_") {
		encoded := strings.TrimPrefix(string(secret), "whsec_")
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return fmt.Errorf("invalid secret base64: %w", err)
		}
		key = decoded
	}

	signedContent := webhookID + "." + webhookTimestamp + "." + string(body)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(signedContent))
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	signatures := strings.Fields(webhookSignature)
	for _, part := range signatures {
		sig := part
		if strings.HasPrefix(part, "v1,") {
			sig = strings.TrimPrefix(part, "v1,")
		}
		if hmac.Equal([]byte(sig), []byte(expectedSig)) {
			// Optional: reject old timestamps to limit replay
			ts, err := strconv.ParseInt(webhookTimestamp, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid webhook timestamp: %w", err)
			}
			now := time.Now().Unix()
			if now-ts > svixTimestampToleranceSec || ts-now > svixTimestampToleranceSec {
				return fmt.Errorf("webhook timestamp too old or in future")
			}
			return nil
		}
	}

	return fmt.Errorf("signature mismatch")
}
