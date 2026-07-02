package github

import (
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// NOTE: we use raw strings here to avoid a circular dependency issue.
func IntegrationContextForNewSetupFlow() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		NewSetupFlow: true,
		CurrentProperties: map[string]any{
			"authMethod": "pat",
			"owner":      "testhq",
			"ownerType":  "organization",
		},
		CurrentSecrets: map[string]core.IntegrationSecret{
			"pat": {Name: "pat", Value: []byte("test-token")},
		},
	}
}

func IntegrationContextForLegacySetupFlow(pem []byte) *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Metadata: map[string]any{
			"installationId": "67890",
			"owner":          "testhq",
			"githubApp": map[string]any{
				"id": int64(12345),
			},
		},
		Configuration: map[string]any{
			"organization": "testhq",
		},
		CurrentSecrets: map[string]core.IntegrationSecret{
			"pem": {Name: "pem", Value: pem},
		},
	}
}

func GitHubResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
