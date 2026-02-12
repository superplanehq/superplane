package jenkins

import "github.com/superplanehq/superplane/pkg/core"

// WebhookConfiguration is the configuration for a Jenkins webhook.
// Jenkins does not have a programmatic webhook management API,
// so the user configures the Jenkins Notification Plugin manually
// to POST to the SuperPlane webhook URL.
type WebhookConfiguration struct{}

type JenkinsWebhookHandler struct{}

func (h *JenkinsWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	return nil, nil
}

func (h *JenkinsWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}

func (h *JenkinsWebhookHandler) CompareConfig(a, b any) (bool, error) {
	return true, nil
}

func (h *JenkinsWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}
