package cloudflare

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// buildWorkerProvisionRequestBody builds the JSON body for POST .../workers/workers (Cloudflare Create Worker API).
func buildWorkerProvisionRequestBody(
	name string,
	tags string,
	logpush *bool,
	observabilityEnabled *bool,
	observabilityHeadSamplingRate string,
	subdomainEnabled *bool,
	subdomainPreviewsEnabled *bool,
	tailConsumers string,
) (map[string]any, error) {
	body := map[string]any{
		"name": strings.TrimSpace(name),
	}

	if tagsList := parseCommaOrNewlineList(tags); len(tagsList) > 0 {
		body["tags"] = tagsList
	}

	if logpush != nil {
		body["logpush"] = *logpush
	}

	obsIncluded := observabilityEnabled != nil || strings.TrimSpace(observabilityHeadSamplingRate) != ""
	if obsIncluded {
		obs := map[string]any{}
		if observabilityEnabled != nil {
			obs["enabled"] = *observabilityEnabled
		}
		if s := strings.TrimSpace(observabilityHeadSamplingRate); s != "" {
			rate, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, fmt.Errorf("observabilityHeadSamplingRate: %w", err)
			}
			obs["head_sampling_rate"] = rate
		}
		body["observability"] = obs
	}

	if subdomainEnabled != nil || subdomainPreviewsEnabled != nil {
		sub := map[string]any{}
		if subdomainEnabled != nil {
			sub["enabled"] = *subdomainEnabled
		}
		if subdomainPreviewsEnabled != nil {
			sub["previews_enabled"] = *subdomainPreviewsEnabled
		}
		body["subdomain"] = sub
	}

	if names := parseCommaOrNewlineList(tailConsumers); len(names) > 0 {
		consumers := make([]map[string]string, 0, len(names))
		for _, n := range names {
			consumers = append(consumers, map[string]string{"name": n})
		}
		body["tail_consumers"] = consumers
	}

	return body, nil
}

// isWorkerProvisionConflict reports whether err indicates the Worker already exists (safe to ignore before upload).
// We only match on HTTP 409 Conflict, which is the unambiguous signal from Cloudflare that the resource
// already exists. Broad substring matching on error messages (e.g. "already", "duplicate") is intentionally
// avoided because it would silently swallow unrelated errors whose messages happen to contain those words
// (e.g. "Token already revoked", "Quota already exhausted").
func isWorkerProvisionConflict(err error) bool {
	var apiErr *CloudflareAPIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == 409
}

func ensureWorkerProvisioned(c *Client, accountID string, body map[string]any) error {
	_, err := c.CreateWorkerResource(accountID, body)
	if err == nil {
		return nil
	}
	if isWorkerProvisionConflict(err) {
		return nil
	}
	return err
}
