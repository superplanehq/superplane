package cloudsmith

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// signatureHeader is the header Cloudsmith sets on each delivery, formatted as
// "sha1=<hex>" where the digest is HMAC-SHA1 of the request body keyed by the
// webhook's signature_key. Shared by the repository-scoped webhook triggers.
const signatureHeader = "X-Cloudsmith-Signature"

// RepositoryRef identifies the repository a webhook trigger watches.
type RepositoryRef struct {
	Namespace string `json:"namespace" mapstructure:"namespace"`
	Slug      string `json:"slug" mapstructure:"slug"`
}

// verifyCloudsmithSignature checks the X-Cloudsmith-Signature header (formatted
// "sha1=<hex>") against HMAC-SHA1 of the body keyed by secret. When no secret is
// configured, verification is skipped (mirrors the other integrations).
func verifyCloudsmithSignature(signature string, body, secret []byte) error {
	if len(secret) == 0 {
		return nil
	}
	if signature == "" {
		return fmt.Errorf("missing signature")
	}

	mac := hmac.New(sha1.New, secret)
	mac.Write(body)
	expected := "sha1=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

// parsePackageFromWebhook extracts the package from a JSON-object webhook body.
// Cloudsmith delivers the package under a "data" key; we fall back to a
// top-level object so the trigger is resilient to payload-shape differences.
func parsePackageFromWebhook(body []byte) (*Package, error) {
	var envelope struct {
		Data Package `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && (envelope.Data.SlugPerm != "" || envelope.Data.Name != "") {
		return &envelope.Data, nil
	}

	var pkg Package
	if err := json.Unmarshal(body, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}
