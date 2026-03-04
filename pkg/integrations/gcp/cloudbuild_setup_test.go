package gcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

type rewriteHTTPContext struct {
	baseURL *url.URL
	client  *http.Client
}

func (c *rewriteHTTPContext) Do(request *http.Request) (*http.Response, error) {
	rewritten := request.Clone(request.Context())
	targetURL := *request.URL
	targetURL.Scheme = c.baseURL.Scheme
	targetURL.Host = c.baseURL.Host
	rewritten.URL = &targetURL
	rewritten.Host = ""
	return c.client.Do(rewritten)
}

func TestHandleEnsureCloudBuildCreatesTopicAndSubscription(t *testing.T) {
	integrationID := "11111111-1111-1111-1111-111111111111"
	expectedSubscriptionID := "sp-cb-sub-" + sanitizeID(integrationID)

	var subscriptionRequestBody struct {
		Topic      string `json:"topic"`
		PushConfig struct {
			PushEndpoint string `json:"pushEndpoint"`
		} `json:"pushConfig"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/projects/demo-project/services/pubsub.googleapis.com":
			_, _ = w.Write([]byte(`{"state":"ENABLED"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/projects/demo-project/services/cloudbuild.googleapis.com":
			_, _ = w.Write([]byte(`{"state":"ENABLED"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/projects/demo-project/topics/cloud-builds":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && r.URL.Path == "/v1/projects/demo-project/subscriptions/"+expectedSubscriptionID:
			require.NoError(t, json.NewDecoder(r.Body).Decode(&subscriptionRequestBody))
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	integrationCtx := &testcontexts.IntegrationContext{
		IntegrationID: integrationID,
		Metadata: gcpcommon.Metadata{
			ProjectID:  "demo-project",
			AuthMethod: gcpcommon.AuthMethodWIF,
		},
		Secrets: map[string]core.IntegrationSecret{
			gcpcommon.SecretNameAccessToken: {
				Name:  gcpcommon.SecretNameAccessToken,
				Value: []byte("test-access-token"),
			},
		},
	}

	err = (&GCP{}).HandleAction(core.IntegrationActionContext{
		Name:            gcpcommon.ActionNameEnsureCloudBuild,
		WebhooksBaseURL: "https://superplane.example",
		HTTP: &rewriteHTTPContext{
			baseURL: baseURL,
			client:  server.Client(),
		},
		Integration: integrationCtx,
	})
	require.NoError(t, err)

	var metadata gcpcommon.Metadata
	require.NoError(t, mapstructure.Decode(integrationCtx.GetMetadata(), &metadata))
	assert.Equal(t, expectedSubscriptionID, metadata.CloudBuildSubscription)

	secret, ok := integrationCtx.Secrets[CloudBuildSecretName]
	require.True(t, ok)
	assert.NotEmpty(t, secret.Value)
	assert.Equal(t, "projects/demo-project/topics/cloud-builds", subscriptionRequestBody.Topic)
	assert.Equal(
		t,
		"https://superplane.example/api/v1/integrations/"+integrationID+"/cloud-build-events?token="+string(secret.Value),
		subscriptionRequestBody.PushConfig.PushEndpoint,
	)
}
