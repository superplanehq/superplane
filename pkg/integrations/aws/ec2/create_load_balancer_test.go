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

func Test__CreateLoadBalancer__Setup(t *testing.T) {
	component := &CreateLoadBalancer{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": " ",
				"type":   LoadBalancerTypeApplication,
				"scheme": LoadBalancerSchemeInternetFacing,
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":   " ",
				"region": "us-east-1",
				"type":   LoadBalancerTypeApplication,
				"scheme": LoadBalancerSchemeInternetFacing,
			},
		})
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": "us-east-1",
				"type":   LoadBalancerTypeApplication,
				"scheme": LoadBalancerSchemeInternetFacing,
			},
			Metadata: metadata,
		})
		require.NoError(t, err)

		stored, ok := metadata.Get().(CreateLoadBalancerNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "my-lb", stored.Name)
		assert.Equal(t, LoadBalancerTypeApplication, stored.Type)
		assert.Equal(t, LoadBalancerSchemeInternetFacing, stored.Scheme)
	})
}

func Test__CreateLoadBalancer__Execute(t *testing.T) {
	component := &CreateLoadBalancer{}

	t.Run("create load balancer -> schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<CreateLoadBalancerResponse xmlns="https://elasticloadbalancing.amazonaws.com/doc/2015-12-01/">
							<CreateLoadBalancerResult>
								<LoadBalancers>
									<member>
										<LoadBalancerArn>arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/50dc6c495c0c9188</LoadBalancerArn>
										<LoadBalancerName>my-lb</LoadBalancerName>
										<DNSName>my-lb-123456789.us-east-1.elb.amazonaws.com</DNSName>
										<Scheme>internet-facing</Scheme>
										<Type>application</Type>
										<State><Code>provisioning</Code></State>
										<VpcId>vpc-12345678</VpcId>
									</member>
								</LoadBalancers>
							</CreateLoadBalancerResult>
							<ResponseMetadata>
								<RequestId>req-abc-123</RequestId>
							</ResponseMetadata>
						</CreateLoadBalancerResponse>
					`)),
				},
			},
		}

		requests := &contexts.RequestContext{}
		metaCtx := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":    "my-lb",
				"region":  "us-east-1",
				"type":    LoadBalancerTypeApplication,
				"scheme":  LoadBalancerSchemeInternetFacing,
				"subnets": []string{"subnet-abc123", "subnet-def456"},
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

		stored, ok := metaCtx.Get().(CreateLoadBalancerExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/50dc6c495c0c9188", stored.LoadBalancerARN)
	})
}

func Test__CreateLoadBalancer__Poll(t *testing.T) {
	component := &CreateLoadBalancer{}

	t.Run("active state -> emits output", func(t *testing.T) {
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

		executionState := &contexts.ExecutionStateContext{}
		metaCtx := &contexts.MetadataContext{
			Metadata: CreateLoadBalancerExecutionMetadata{LoadBalancerARN: lbARN},
		}

		err := component.HandleHook(core.ActionHookContext{
			Name: "poll",
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": "us-east-1",
				"type":   LoadBalancerTypeApplication,
				"scheme": LoadBalancerSchemeInternetFacing,
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
		assert.Equal(t, CreateLoadBalancerPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		data := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, lbARN, data["loadBalancerArn"])
		assert.Equal(t, LoadBalancerStateActive, data["state"])
	})

	t.Run("provisioning state -> schedules next poll", func(t *testing.T) {
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
										<State><Code>provisioning</Code></State>
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
			Metadata: CreateLoadBalancerExecutionMetadata{LoadBalancerARN: lbARN},
		}

		err := component.HandleHook(core.ActionHookContext{
			Name: "poll",
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": "us-east-1",
				"type":   LoadBalancerTypeApplication,
				"scheme": LoadBalancerSchemeInternetFacing,
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
