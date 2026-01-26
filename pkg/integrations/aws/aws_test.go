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
		appCtx := &contexts.AppInstallationContext{}

		err := a.Sync(core.SyncContext{
			Configuration:   map[string]any{"region": "us-east-1"},
			AppInstallation: appCtx,
			BaseURL:         "http://localhost:8000",
		})

		require.NoError(t, err)
		require.NotNil(t, appCtx.BrowserAction)
		assert.Contains(t, appCtx.BrowserAction.Description, "Create Identity Provider")
		assert.Contains(t, appCtx.BrowserAction.Description, "IAM Role")
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

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"roleArn":                "arn:aws:iam::123456789012:role/test-role",
				"region":                 "us-east-1",
				"sessionDurationSeconds": 3600,
			},
			Secrets:       map[string]core.InstallationSecret{},
			BrowserAction: &core.BrowserAction{},
		}

		err := a.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			OIDC:            support.NewOIDCProvider(),
			AppInstallation: appCtx,
			InstallationID:  "installation-123",
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		assert.Nil(t, appCtx.BrowserAction)

		require.Contains(t, appCtx.Secrets, "accessKeyId")
		require.Contains(t, appCtx.Secrets, "secretAccessKey")
		require.Contains(t, appCtx.Secrets, "sessionToken")
		assert.Equal(t, []byte("AKIA_TEST"), appCtx.Secrets["accessKeyId"].Value)
		assert.Equal(t, []byte("secret"), appCtx.Secrets["secretAccessKey"].Value)
		assert.Equal(t, []byte("token"), appCtx.Secrets["sessionToken"].Value)

		metadata, ok := appCtx.Metadata.(SessionMetadata)
		require.True(t, ok)
		assert.Equal(t, "arn:aws:iam::123456789012:role/test-role", metadata.RoleArn)
		assert.Equal(t, "us-east-1", metadata.Region)
		assert.Equal(t, expiration, metadata.ExpiresAt)

		require.Len(t, appCtx.ResyncRequests, 1)
		assert.GreaterOrEqual(t, appCtx.ResyncRequests[0], time.Minute)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://sts.us-east-1.amazonaws.com", httpContext.Requests[0].URL.String())
	})
}

func Test__AWS__ListResources(t *testing.T) {
	a := &AWS{}

	t.Run("unknown resource type returns empty list", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}

		resources, err := a.ListResources("unknown", core.ListResourcesContext{
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("lambda.function without credentials returns error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Secrets: map[string]core.InstallationSecret{},
		}

		_, err := a.ListResources("lambda.function", core.ListResourcesContext{
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("lambda.function without region returns error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"region": "   ",
			},
			Secrets: map[string]core.InstallationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		_, err := a.ListResources("lambda.function", core.ListResourcesContext{
			AppInstallation: appCtx,
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

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"region": "us-east-1",
			},
			Secrets: map[string]core.InstallationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		resources, err := a.ListResources("lambda.function", core.ListResourcesContext{
			AppInstallation: appCtx,
			HTTP:            httpContext,
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
