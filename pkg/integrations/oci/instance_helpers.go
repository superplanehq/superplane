package oci

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
)

func extractWebhookID(webhookURL string) (string, error) {
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse webhook URL: %w", err)
	}

	webhookID := strings.Trim(strings.TrimRight(parsedURL.Path, "/"), "/")
	if webhookID == "" {
		return "", fmt.Errorf("webhook URL path is empty")
	}

	segments := strings.Split(webhookID, "/")
	return segments[len(segments)-1], nil
}

func enrichInstanceWithVNICIPs(logger *log.Entry, client *Client, instance *Instance, payload map[string]any) {
	attachments, err := client.ListVNICAttachments(instance.CompartmentID, instance.ID)
	if err != nil {
		if logger != nil {
			logger.Warnf("failed to list VNIC attachments for instance %s: %v", instance.ID, err)
		}
		return
	}

	for _, att := range attachments {
		if att.LifecycleState != "ATTACHED" || att.VNICID == "" {
			continue
		}

		vnic, err := client.GetVNIC(att.VNICID)
		if err != nil {
			if logger != nil {
				logger.Warnf("failed to get VNIC %s for instance %s: %v", att.VNICID, instance.ID, err)
			}
			return
		}

		payload["publicIp"] = vnic.PublicIP
		payload["privateIp"] = vnic.PrivateIP
		return
	}
}

func confirmONSSubscription(ctx core.WebhookRequestContext, confirmURL string) (int, *core.WebhookResponseBody, error) {
	if err := validateONSConfirmationURL(confirmURL); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("refusing ONS confirmation URL: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, confirmURL, nil)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to build ONS confirmation request: %w", err)
	}

	resp, err := ctx.HTTP.Do(req)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to confirm ONS subscription: %w", err)
	}

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return http.StatusInternalServerError, nil, fmt.Errorf("ONS confirmation returned %d", resp.StatusCode)
	}

	return http.StatusOK, nil, nil
}
