package aws

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__AWS__Sync(t *testing.T) {
	a := &AWS{}

	t.Run("missing role arn -> browser action", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}

		err := a.Sync(core.SyncContext{
			Configuration: map[string]any{"region": "us-east-1"},
			Integration:   integrationCtx,
			BaseURL:       "http://localhost:8000",
		})

		require.NoError(t, err)
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Create Identity Provider")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "IAM Role")
	})

	t.Run("role arn -> sets secrets, metadata, and schedules resync", func(t *testing.T) {
		expiration := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
		stsResponse := fmt.Sprintf(`
<AssumeRoleWithWebIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <AssumeRoleWithWebIdentityResult>
    <Credentials>
      <AccessKeyId>AKIA_TEST</AccessKeyId>
      <SecretAccessKey>secret</SecretAccessKey>
      <SessionToken>token</SessionToken>
      <Expiration>%s</Expiration>
    </Credentials>
  </AssumeRoleWithWebIdentityResult>
</AssumeRoleWithWebIdentityResponse>
`, expiration)

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(stsResponse)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"roleArn":                "arn:aws:iam::123456789012:role/test-role",
				"region":                 "us-east-1",
				"sessionDurationSeconds": 3600,
			},
			Secrets:       map[string]core.IntegrationSecret{},
			BrowserAction: &core.BrowserAction{},
		}

		err := a.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			OIDC:          support.NewOIDCProvider(),
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		assert.Nil(t, integrationCtx.BrowserAction)

		require.Contains(t, integrationCtx.Secrets, "accessKeyId")
		require.Contains(t, integrationCtx.Secrets, "secretAccessKey")
		require.Contains(t, integrationCtx.Secrets, "sessionToken")
		assert.Equal(t, []byte("AKIA_TEST"), integrationCtx.Secrets["accessKeyId"].Value)
		assert.Equal(t, []byte("secret"), integrationCtx.Secrets["secretAccessKey"].Value)
		assert.Equal(t, []byte("token"), integrationCtx.Secrets["sessionToken"].Value)

		metadata, ok := integrationCtx.Metadata.(SessionMetadata)
		require.True(t, ok)
		assert.Equal(t, "arn:aws:iam::123456789012:role/test-role", metadata.RoleArn)
		assert.Equal(t, "us-east-1", metadata.Region)
		assert.Equal(t, expiration, metadata.ExpiresAt)

		require.Len(t, integrationCtx.ResyncRequests, 1)
		assert.GreaterOrEqual(t, integrationCtx.ResyncRequests[0], time.Minute)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://sts.us-east-1.amazonaws.com", httpContext.Requests[0].URL.String())
	})
}

func Test__AWS__ListResources(t *testing.T) {
	a := &AWS{}

	t.Run("unknown resource type returns empty list", func(t *testing.T) {
		resources, err := a.ListResources("unknown", core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("lambda.function without credentials returns error", func(t *testing.T) {
		_, err := a.ListResources("lambda.function", core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{},
			},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("lambda.function without region returns error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"region": "   ",
			},
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		_, err := a.ListResources("lambda.function", core.ListResourcesContext{
			Integration: integrationCtx,
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("lambda.function returns functions", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"Functions": [
								{
									"FunctionName": "runFunction",
									"FunctionArn": "arn:aws:lambda:us-east-1:123456789012:function:runFunction"
								}
							]
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		resources, err := a.ListResources("lambda.function", core.ListResourcesContext{
			Integration: integrationCtx,
			HTTP:        httpContext,
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "lambda.function", resources[0].Type)
		assert.Equal(t, "runFunction", resources[0].Name)
		assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:runFunction", resources[0].ID)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://lambda.us-east-1.amazonaws.com/2015-03-31/functions?MaxItems=50", httpContext.Requests[0].URL.String())
	})
}
