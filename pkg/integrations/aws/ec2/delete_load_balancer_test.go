package ec2

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteLoadBalancer__Setup(t *testing.T) {
	component := &DeleteLoadBalancer{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       " ",
				"loadBalancer": "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/50dc6c495c0c9188",
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing load balancer ARN -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"loadBalancer": "",
			},
		})
		require.ErrorContains(t, err, "load balancer ARN is required")
	})

	t.Run("stores name in node metadata", func(t *testing.T) {
		lbARN := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/50dc6c495c0c9188"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DescribeLoadBalancersResponse xmlns="https://elasticloadbalancing.amazonaws.com/doc/2015-12-01/">
							<DescribeLoadBalancersResult>
								<LoadBalancers>
									<member>
										<LoadBalancerArn>` + lbARN + `</LoadBalancerArn>
										<LoadBalancerName>my-lb</LoadBalancerName>
										<DNSName>my-lb-123456789.us-east-1.elb.amazonaws.com</DNSName>
										<Scheme>internet-facing</Scheme>
										<Type>application</Type>
										<State><Code>active</Code></State>
										<VpcId>vpc-12345678</VpcId>
									</member>
								</LoadBalancers>
							</DescribeLoadBalancersResult>
							<ResponseMetadata>
								<RequestId>req-abc-123</RequestId>
							</ResponseMetadata>
						</DescribeLoadBalancersResponse>
					`)),
				},
			},
		}
		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"loadBalancer": lbARN,
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
			Metadata: metadata,
		})

		require.NoError(t, err)
		stored, ok := metadata.Get().(DeleteLoadBalancerNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "my-lb", stored.LoadBalancerName)
	})
}

func Test__DeleteLoadBalancer__Execute(t *testing.T) {
	component := &DeleteLoadBalancer{}

	t.Run("load balancer not found -> emits deleted (idempotent)", func(t *testing.T) {
		lbARN := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/50dc6c495c0c9188"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body: io.NopCloser(strings.NewReader(`
						<ErrorResponse>
							<Error>
								<Code>LoadBalancerNotFound</Code>
								<Message>Load balancer not found</Message>
							</Error>
						</ErrorResponse>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"loadBalancer": lbARN,
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, DeleteLoadBalancerPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		data := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, lbARN, data["loadBalancerArn"])
		assert.Equal(t, LoadBalancerStateDeleted, data["state"])
	})

	t.Run("delete load balancer -> schedules poll", func(t *testing.T) {
		lbARN := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/50dc6c495c0c9188"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DeleteLoadBalancerResponse xmlns="https://elasticloadbalancing.amazonaws.com/doc/2015-12-01/">
							<DeleteLoadBalancerResult/>
							<ResponseMetadata>
								<RequestId>req-abc-123</RequestId>
							</ResponseMetadata>
						</DeleteLoadBalancerResponse>
					`)),
				},
			},
		}

		requests := &contexts.RequestContext{}
		metaCtx := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":       "us-east-1",
				"loadBalancer": lbARN,
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
			Metadata: metaCtx,
			Requests: requests,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requests.Action)

		stored, ok := metaCtx.Get().(DeleteLoadBalancerExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, lbARN, stored.LoadBalancerARN)
	})
}

func Test__DeleteLoadBalancer__Poll(t *testing.T) {
	component := &DeleteLoadBalancer{}

	t.Run("not found -> emits deleted", func(t *testing.T) {
		lbARN := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/50dc6c495c0c9188"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body: io.NopCloser(strings.NewReader(`
						<ErrorResponse>
							<Error>
								<Code>LoadBalancerNotFound</Code>
								<Message>Load balancer not found</Message>
							</Error>
						</ErrorResponse>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		metaCtx := &contexts.MetadataContext{
			Metadata: DeleteLoadBalancerExecutionMetadata{LoadBalancerARN: lbARN},
		}

		err := component.HandleHook(core.ActionHookContext{
			Name: "poll",
			Configuration: map[string]any{
				"region":       "us-east-1",
				"loadBalancer": lbARN,
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
			Metadata:       metaCtx,
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, DeleteLoadBalancerPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		data := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, lbARN, data["loadBalancerArn"])
		assert.Equal(t, LoadBalancerStateDeleted, data["state"])
	})

	t.Run("still deleting -> schedules next poll", func(t *testing.T) {
		lbARN := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/50dc6c495c0c9188"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DescribeLoadBalancersResponse xmlns="https://elasticloadbalancing.amazonaws.com/doc/2015-12-01/">
							<DescribeLoadBalancersResult>
								<LoadBalancers>
									<member>
										<LoadBalancerArn>` + lbARN + `</LoadBalancerArn>
										<LoadBalancerName>my-lb</LoadBalancerName>
										<DNSName>my-lb-123456789.us-east-1.elb.amazonaws.com</DNSName>
										<Scheme>internet-facing</Scheme>
										<Type>application</Type>
										<State><Code>deleting</Code></State>
										<VpcId>vpc-12345678</VpcId>
									</member>
								</LoadBalancers>
							</DescribeLoadBalancersResult>
							<ResponseMetadata>
								<RequestId>req-abc-123</RequestId>
							</ResponseMetadata>
						</DescribeLoadBalancersResponse>
					`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		requests := &contexts.RequestContext{}
		metaCtx := &contexts.MetadataContext{
			Metadata: DeleteLoadBalancerExecutionMetadata{LoadBalancerARN: lbARN},
		}

		err := component.HandleHook(core.ActionHookContext{
			Name: "poll",
			Configuration: map[string]any{
				"region":       "us-east-1",
				"loadBalancer": lbARN,
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
			Metadata:       metaCtx,
			ExecutionState: executionState,
			Requests:       requests,
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requests.Action)
	})
}
