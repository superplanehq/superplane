package aws

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
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
		stsResponse := stsResponse("token", expiration)
		createRoleResponse := createRoleResponse()
		putRolePolicyResponse := `<PutRolePolicyResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/"></PutRolePolicyResponse>`
		createConnectionResponse := `{"ConnectionArn":"arn:aws:events:us-east-1:123456789012:connection/superplane-test/abc123"}`
		createAPIDestinationResponse := `{"ApiDestinationArn":"arn:aws:events:us-east-1:123456789012:api-destination/superplane-test/def456"}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(stsResponse)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(createRoleResponse)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(putRolePolicyResponse)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(createConnectionResponse)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(createAPIDestinationResponse)),
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
			Configuration:   integrationCtx.Configuration,
			HTTP:            httpContext,
			OIDC:            support.NewOIDCProvider(),
			Integration:     integrationCtx,
			WebhooksBaseURL: "http://localhost:8000",
			Logger:          logrus.NewEntry(logrus.New()),
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

		metadata, ok := integrationCtx.Metadata.(common.IntegrationMetadata)
		require.True(t, ok)
		assert.Equal(t, "arn:aws:iam::123456789012:role/test-role", metadata.Session.RoleArn)
		assert.Equal(t, "123456789012", metadata.Session.AccountID)
		assert.Equal(t, "us-east-1", metadata.Session.Region)
		assert.Equal(t, expiration, metadata.Session.ExpiresAt)
		require.NotNil(t, metadata.IAM)
		require.NotNil(t, metadata.IAM.TargetDestinationRole)
		assert.Equal(t, "arn:aws:iam::123456789012:role/superplane-destination-invoker-test", metadata.IAM.TargetDestinationRole.RoleArn)
		assert.NotEmpty(t, metadata.IAM.TargetDestinationRole.RoleName)

		require.NotNil(t, metadata.EventBridge)
		require.NotEmpty(t, metadata.EventBridge.APIDestinations)

		require.Len(t, integrationCtx.ResyncRequests, 1)
		assert.GreaterOrEqual(t, integrationCtx.ResyncRequests[0], time.Minute)

		require.Len(t, httpContext.Requests, 5)
		assert.Equal(t, "https://sts.us-east-1.amazonaws.com", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://iam.amazonaws.com/", httpContext.Requests[1].URL.String())
		assert.Equal(t, "https://iam.amazonaws.com/", httpContext.Requests[2].URL.String())
		assert.Equal(t, "https://events.us-east-1.amazonaws.com/", httpContext.Requests[3].URL.String())
		assert.Equal(t, "https://events.us-east-1.amazonaws.com/", httpContext.Requests[4].URL.String())
	})

	t.Run("IAM and EventBridge already configured, only session token is refreshed", func(t *testing.T) {
		expiration := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
		stsResponse := stsResponse("token2", expiration)

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
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("AKIA_TEST")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token1")},
			},
			Metadata: common.IntegrationMetadata{
				Session: &common.SessionMetadata{
					RoleArn:   "arn:aws:iam::123456789012:role/test-role",
					AccountID: "123456789012",
					Region:    "us-east-1",
					ExpiresAt: expiration,
				},
				IAM: &common.IAMMetadata{
					TargetDestinationRole: &common.IAMRoleMetadata{
						RoleArn: "arn:aws:iam::123456789012:role/superplane-destination-invoker-test",
					},
				},
				EventBridge: &common.EventBridgeMetadata{
					APIDestinations: map[string]common.APIDestinationMetadata{
						"us-east-1": {
							APIDestinationArn: "arn:aws:events:us-east-1:123456789012:api-destination/superplane-test/def456",
						},
					},
				},
			},
		}

		err := a.Sync(core.SyncContext{
			Configuration:   integrationCtx.Configuration,
			HTTP:            httpContext,
			OIDC:            support.NewOIDCProvider(),
			Integration:     integrationCtx,
			WebhooksBaseURL: "http://localhost:8000",
			Logger:          logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		assert.Nil(t, integrationCtx.BrowserAction)

		require.Len(t, integrationCtx.ResyncRequests, 1)
		assert.GreaterOrEqual(t, integrationCtx.ResyncRequests[0], time.Minute)

		//
		// On the STS request to refresh the session token is done.
		//
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://sts.us-east-1.amazonaws.com", httpContext.Requests[0].URL.String())

		//
		// Session token is refreshed.
		//
		require.Contains(t, integrationCtx.Secrets, "accessKeyId")
		require.Contains(t, integrationCtx.Secrets, "secretAccessKey")
		require.Contains(t, integrationCtx.Secrets, "sessionToken")
		assert.Equal(t, []byte("AKIA_TEST"), integrationCtx.Secrets["accessKeyId"].Value)
		assert.Equal(t, []byte("secret"), integrationCtx.Secrets["secretAccessKey"].Value)
		assert.Equal(t, []byte("token2"), integrationCtx.Secrets["sessionToken"].Value)

		metadata, ok := integrationCtx.Metadata.(common.IntegrationMetadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Session)
		require.NotNil(t, metadata.IAM)
		require.NotNil(t, metadata.IAM.TargetDestinationRole)
		require.NotNil(t, metadata.EventBridge)
		require.NotEmpty(t, metadata.EventBridge.APIDestinations)
	})
}

func Test__AWS__ListResources(t *testing.T) {
	a := &AWS{}

	t.Run("unknown resource type returns empty list", func(t *testing.T) {
		resources, err := a.ListResources("unknown", core.ListResourcesContext{
			Logger:      logrus.NewEntry(logrus.New()),
			Integration: &contexts.IntegrationContext{},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("lambda.function without credentials returns error", func(t *testing.T) {
		_, err := a.ListResources("lambda.function", core.ListResourcesContext{
			Logger: logrus.NewEntry(logrus.New()),
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{},
			},
		})

		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("lambda.function without region returns error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		_, err := a.ListResources("lambda.function", core.ListResourcesContext{
			Logger:      logrus.NewEntry(logrus.New()),
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
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		resources, err := a.ListResources("lambda.function", core.ListResourcesContext{
			Integration: integrationCtx,
			Logger:      logrus.NewEntry(logrus.New()),
			HTTP:        httpContext,
			Parameters:  map[string]string{"region": "us-east-1"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "lambda.function", resources[0].Type)
		assert.Equal(t, "runFunction", resources[0].Name)
		assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:runFunction", resources[0].ID)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://lambda.us-east-1.amazonaws.com/2015-03-31/functions?MaxItems=50", httpContext.Requests[0].URL.String())
	})

	t.Run("ecr.repository returns repositories", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"repositories": [
								{
									"repositoryName": "backend",
									"repositoryArn": "arn:aws:ecr:us-east-1:123456789012:repository/backend"
								}
							]
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		resources, err := a.ListResources("ecr.repository", core.ListResourcesContext{
			Integration: integrationCtx,
			Logger:      logrus.NewEntry(logrus.New()),
			HTTP:        httpContext,
			Parameters:  map[string]string{"region": "us-east-1"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "ecr.repository", resources[0].Type)
		assert.Equal(t, "backend", resources[0].Name)
		assert.Equal(t, "arn:aws:ecr:us-east-1:123456789012:repository/backend", resources[0].ID)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.ecr.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
	})

	t.Run("ecs.cluster returns clusters", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"clusterArns": [
								"arn:aws:ecs:us-east-1:123456789012:cluster/demo"
							]
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		resources, err := a.ListResources("ecs.cluster", core.ListResourcesContext{
			Integration: integrationCtx,
			Logger:      logrus.NewEntry(logrus.New()),
			HTTP:        httpContext,
			Parameters:  map[string]string{"region": "us-east-1"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "ecs.cluster", resources[0].Type)
		assert.Equal(t, "demo", resources[0].Name)
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:cluster/demo", resources[0].ID)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://ecs.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
	})

	t.Run("ecs.service returns services", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"serviceArns": [
								"arn:aws:ecs:us-east-1:123456789012:service/demo/web"
							]
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		resources, err := a.ListResources("ecs.service", core.ListResourcesContext{
			Integration: integrationCtx,
			Logger:      logrus.NewEntry(logrus.New()),
			HTTP:        httpContext,
			Parameters: map[string]string{
				"region":  "us-east-1",
				"cluster": "demo",
			},
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "ecs.service", resources[0].Type)
		assert.Equal(t, "web", resources[0].Name)
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:service/demo/web", resources[0].ID)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://ecs.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
	})

	t.Run("ecs.taskDefinition returns task definitions", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"taskDefinitionArns": [
								"arn:aws:ecs:us-east-1:123456789012:task-definition/worker:7"
							]
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		resources, err := a.ListResources("ecs.taskDefinition", core.ListResourcesContext{
			Integration: integrationCtx,
			Logger:      logrus.NewEntry(logrus.New()),
			HTTP:        httpContext,
			Parameters:  map[string]string{"region": "us-east-1"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "ecs.taskDefinition", resources[0].Type)
		assert.Equal(t, "worker:7", resources[0].Name)
		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:task-definition/worker:7", resources[0].ID)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://ecs.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
	})

	t.Run("sns.topic returns topics", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<ListTopicsResponse>
						  <ListTopicsResult>
							<Topics>
							  <member>
								<TopicArn>arn:aws:sns:us-east-1:123456789012:orders-events</TopicArn>
							  </member>
							</Topics>
						  </ListTopicsResult>
						</ListTopicsResponse>
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		resources, err := a.ListResources("sns.topic", core.ListResourcesContext{
			Integration: integrationCtx,
			Logger:      logrus.NewEntry(logrus.New()),
			HTTP:        httpContext,
			Parameters:  map[string]string{"region": "us-east-1"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "sns.topic", resources[0].Type)
		assert.Equal(t, "orders-events", resources[0].Name)
		assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:orders-events", resources[0].ID)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://sns.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
	})
}

func stsResponse(token string, expiration string) string {
	return fmt.Sprintf(`
<AssumeRoleWithWebIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <AssumeRoleWithWebIdentityResult>
    <Credentials>
      <AccessKeyId>AKIA_TEST</AccessKeyId>
      <SecretAccessKey>secret</SecretAccessKey>
      <SessionToken>%s</SessionToken>
      <Expiration>%s</Expiration>
    </Credentials>
  </AssumeRoleWithWebIdentityResult>
</AssumeRoleWithWebIdentityResponse>
`, token, expiration)
}

func createRoleResponse() string {
	return `
<CreateRoleResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <CreateRoleResult>
    <Role>
      <Arn>arn:aws:iam::123456789012:role/superplane-destination-invoker-test</Arn>
    </Role>
  </CreateRoleResult>
</CreateRoleResponse>
`
}
