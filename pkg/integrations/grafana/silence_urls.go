package grafana

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// grafanaManagedAlertmanagerName is the Alertmanager source name Grafana uses for the built-in
// Alertmanager (same as /api/alertmanager/grafana/...). The alerting UI requires this query
// param on silence routes; without it, /alerting/silence/{id} returns not found.
const grafanaManagedAlertmanagerName = "grafana"

func buildSilenceWebURL(integrationCtx core.IntegrationContext, silenceID string) (string, error) {
	id := strings.TrimSpace(silenceID)
	if id == "" {
		return "", fmt.Errorf("empty silence id")
	}

	baseURL, err := readBaseURL(integrationCtx)
	if err != nil {
		return "", err
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// Matches Grafana's makeAMLink(`/alerting/silence/${id}/edit`, alertManagerSourceName).
	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + "/alerting/silence/" + url.PathEscape(id) + "/edit"
	q := url.Values{}
	q.Set("alertmanager", grafanaManagedAlertmanagerName)
	parsed.RawQuery = q.Encode()
	parsed.Fragment = ""

	return parsed.String(), nil
}
