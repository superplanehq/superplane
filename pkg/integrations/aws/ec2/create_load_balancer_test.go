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

	t.Run("too few subnets -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":    "my-lb",
				"region":  "us-east-1",
				"type":    LoadBalancerTypeApplication,
				"scheme":  LoadBalancerSchemeInternetFacing,
				"subnets": []string{"subnet-abc123"},
			},
		})
		require.ErrorContains(t, err, "at least 2 subnet(s)")
	})

	t.Run("listener protocol set but invalid port -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":                "my-lb",
				"region":              "us-east-1",
				"type":                LoadBalancerTypeApplication,
				"scheme":              LoadBalancerSchemeInternetFacing,
				"subnets":             []string{"subnet-abc123", "subnet-def456"},
				"listenerProtocol":    ListenerProtocolHTTP,
				"listenerTargetGroup": "arn:aws:elasticloadbalancing:us-east-1:123:targetgroup/tg/abc",
				"listenerPort":        0,
			},
		})
		require.ErrorContains(t, err, "listener port must be between 1 and 65535")
	})

	t.Run("listener protocol does not match lb type -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":                "my-lb",
				"region":              "us-east-1",
				"type":                LoadBalancerTypeApplication,
				"scheme":              LoadBalancerSchemeInternetFacing,
				"subnets":             []string{"subnet-abc123", "subnet-def456"},
				"listenerProtocol":    ListenerProtocolTCP,
				"listenerTargetGroup": "arn:aws:elasticloadbalancing:us-east-1:123:targetgroup/tg/abc",
				"listenerPort":        80,
			},
		})
		require.ErrorContains(t, err, "not valid for application load balancers")
	})

	t.Run("HTTPS listener without certificate -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":                "my-lb",
				"region":              "us-east-1",
				"type":                LoadBalancerTypeApplication,
				"scheme":              LoadBalancerSchemeInternetFacing,
				"subnets":             []string{"subnet-abc123", "subnet-def456"},
				"listenerProtocol":    ListenerProtocolHTTPS,
				"listenerTargetGroup": "arn:aws:elasticloadbalancing:us-east-1:123:targetgroup/tg/abc",
				"listenerPort":        443,
			},
		})
		require.ErrorContains(t, err, "listenerCertificateArn is required for HTTPS listeners")
	})

	t.Run("listener protocol set without target group -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":             "my-lb",
				"region":           "us-east-1",
				"type":             LoadBalancerTypeApplication,
				"scheme":           LoadBalancerSchemeInternetFacing,
				"subnets":          []string{"subnet-abc123", "subnet-def456"},
				"listenerProtocol": ListenerProtocolHTTP,
				"listenerPort":     80,
			},
		})
		require.ErrorContains(t, err, "listenerTargetGroup is required when listenerProtocol is specified")
	})

	t.Run("target group set without listener protocol -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":                "my-lb",
				"region":              "us-east-1",
				"type":                LoadBalancerTypeApplication,
				"scheme":              LoadBalancerSchemeInternetFacing,
				"subnets":             []string{"subnet-abc123", "subnet-def456"},
				"listenerTargetGroup": "arn:aws:elasticloadbalancing:us-east-1:123:targetgroup/tg/abc",
				"listenerPort":        80,
			},
		})
		require.ErrorContains(t, err, "listenerProtocol is required when listenerTargetGroup is specified")
	})

	t.Run("subnets with blank entries below minimum -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":    "my-lb",
				"region":  "us-east-1",
				"type":    LoadBalancerTypeApplication,
				"scheme":  LoadBalancerSchemeInternetFacing,
				"subnets": []string{"subnet-abc123", " ", ""},
			},
		})
		require.ErrorContains(t, err, "at least 2 subnet(s)")
	})

	t.Run("subnets in same availability zone -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DescribeSubnetsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<subnetSet>
								<item>
									<subnetId>subnet-abc123</subnetId>
									<vpcId>vpc-123</vpcId>
									<availabilityZone>us-east-1a</availabilityZone>
								</item>
							</subnetSet>
						</DescribeSubnetsResponse>
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DescribeSubnetsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<subnetSet>
								<item>
									<subnetId>subnet-def456</subnetId>
									<vpcId>vpc-123</vpcId>
									<availabilityZone>us-east-1a</availabilityZone>
								</item>
							</subnetSet>
						</DescribeSubnetsResponse>
					`)),
				},
			},
		}

		err := component.Setup(core.SetupContext{
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
		})
		require.ErrorContains(t, err, "subnets must be in different Availability Zones")
		require.ErrorContains(t, err, "us-east-1a")
	})

	t.Run("subnets in different availability zones -> valid configuration", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DescribeSubnetsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<subnetSet>
								<item>
									<subnetId>subnet-abc123</subnetId>
									<vpcId>vpc-123</vpcId>
									<availabilityZone>us-east-1a</availabilityZone>
								</item>
							</subnetSet>
						</DescribeSubnetsResponse>
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DescribeSubnetsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<subnetSet>
								<item>
									<subnetId>subnet-def456</subnetId>
									<vpcId>vpc-123</vpcId>
									<availabilityZone>us-east-1b</availabilityZone>
								</item>
							</subnetSet>
						</DescribeSubnetsResponse>
					`)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":    "my-lb",
				"region":  "us-east-1",
				"type":    LoadBalancerTypeApplication,
				"scheme":  LoadBalancerSchemeInternetFacing,
				"subnets": []string{"subnet-abc123", "subnet-def456"},
			},
			HTTP:     httpContext,
			Metadata: metadata,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})
		require.NoError(t, err)
	})

	t.Run("NLB with TCP listener and target group -> valid configuration", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":                "my-nlb",
				"region":              "us-east-1",
				"type":                LoadBalancerTypeNetwork,
				"scheme":              LoadBalancerSchemeInternetFacing,
				"subnets":             []string{"subnet-abc123", "subnet-def456"},
				"listenerProtocol":    ListenerProtocolTCP,
				"listenerPort":        80,
				"listenerTargetGroup": "arn:aws:elasticloadbalancing:us-east-1:123:targetgroup/tg/abc",
			},
			Metadata: metadata,
		})
		require.NoError(t, err)
	})

	t.Run("NLB with TLS listener without certificate -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":                "my-nlb",
				"region":              "us-east-1",
				"type":                LoadBalancerTypeNetwork,
				"scheme":              LoadBalancerSchemeInternetFacing,
				"subnets":             []string{"subnet-abc123", "subnet-def456"},
				"listenerProtocol":    ListenerProtocolTLS,
				"listenerPort":        443,
				"listenerTargetGroup": "arn:aws:elasticloadbalancing:us-east-1:123:targetgroup/tg/abc",
			},
		})
		require.ErrorContains(t, err, "listenerCertificateArn is required for TLS listeners")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":    "my-lb",
				"region":  "us-east-1",
				"type":    LoadBalancerTypeApplication,
				"scheme":  LoadBalancerSchemeInternetFacing,
				"subnets": []string{"subnet-abc123", "subnet-def456"},
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

	t.Run("blank subnet entries -> consecutive member indices in request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<CreateLoadBalancerResponse xmlns="https://elasticloadbalancing.amazonaws.com/doc/2015-12-01/">
							<CreateLoadBalancerResult>
								<LoadBalancers>
									<member>
										<LoadBalancerArn>arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/abc</LoadBalancerArn>
										<LoadBalancerName>my-lb</LoadBalancerName>
										<DNSName>my-lb.us-east-1.elb.amazonaws.com</DNSName>
										<Scheme>internet-facing</Scheme>
										<Type>application</Type>
										<State><Code>provisioning</Code></State>
										<VpcId>vpc-12345678</VpcId>
									</member>
								</LoadBalancers>
							</CreateLoadBalancerResult>
							<ResponseMetadata><RequestId>req-1</RequestId></ResponseMetadata>
						</CreateLoadBalancerResponse>
					`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":    "my-lb",
				"region":  "us-east-1",
				"type":    LoadBalancerTypeApplication,
				"scheme":  LoadBalancerSchemeInternetFacing,
				"subnets": []string{"subnet-abc123", " ", "subnet-def456"},
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
			Metadata: &contexts.MetadataContext{},
			Requests: &contexts.RequestContext{},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		bodyStr := string(body)
		assert.Contains(t, bodyStr, "Subnets.member.1=subnet-abc123")
		assert.Contains(t, bodyStr, "Subnets.member.2=subnet-def456")
		assert.NotContains(t, bodyStr, "Subnets.member.3")
	})

	t.Run("gateway load balancer -> no security groups in request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<CreateLoadBalancerResponse xmlns="https://elasticloadbalancing.amazonaws.com/doc/2015-12-01/">
							<CreateLoadBalancerResult>
								<LoadBalancers>
									<member>
										<LoadBalancerArn>arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/gwy/my-gwlb/1234567890</LoadBalancerArn>
										<LoadBalancerName>my-gwlb</LoadBalancerName>
										<DNSName>my-gwlb.us-east-1.elb.amazonaws.com</DNSName>
										<Type>gateway</Type>
										<State><Code>provisioning</Code></State>
										<VpcId>vpc-12345678</VpcId>
									</member>
								</LoadBalancers>
							</CreateLoadBalancerResult>
							<ResponseMetadata><RequestId>req-gwy-1</RequestId></ResponseMetadata>
						</CreateLoadBalancerResponse>
					`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":           "my-gwlb",
				"region":         "us-east-1",
				"type":           LoadBalancerTypeGateway,
				"subnets":        []string{"subnet-abc123"},
				"securityGroups": []string{"sg-should-be-ignored"},
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
			Metadata: &contexts.MetadataContext{},
			Requests: &contexts.RequestContext{},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.NotContains(t, string(body), "SecurityGroups")
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

	t.Run("active_impaired state -> emits output", func(t *testing.T) {
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
										<State><Code>active_impaired</Code></State>
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
	})

	t.Run("active state with NLB TCP listener -> creates listener then emits", func(t *testing.T) {
		lbARN := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/my-nlb/50dc6c495c0c9188"
		tgARN := "arn:aws:elasticloadbalancing:us-east-1:123:targetgroup/my-tg/abc123"
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
										<LoadBalancerName>my-nlb</LoadBalancerName>
										<DNSName>my-nlb.us-east-1.elb.amazonaws.com</DNSName>
										<Scheme>internet-facing</Scheme>
										<Type>network</Type>
										<State><Code>active</Code></State>
										<VpcId>vpc-12345678</VpcId>
									</member>
								</LoadBalancers>
							</DescribeLoadBalancersResult>
							<ResponseMetadata><RequestId>req-1</RequestId></ResponseMetadata>
						</DescribeLoadBalancersResponse>
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<CreateListenerResponse xmlns="https://elasticloadbalancing.amazonaws.com/doc/2015-12-01/">
							<CreateListenerResult>
								<Listeners>
									<member>
										<ListenerArn>arn:aws:elasticloadbalancing:us-east-1:123:listener/net/my-nlb/abc/def</ListenerArn>
										<Protocol>TCP</Protocol>
										<Port>80</Port>
									</member>
								</Listeners>
							</CreateListenerResult>
							<ResponseMetadata><RequestId>req-2</RequestId></ResponseMetadata>
						</CreateListenerResponse>
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
				"name":                "my-nlb",
				"region":              "us-east-1",
				"type":                LoadBalancerTypeNetwork,
				"scheme":              LoadBalancerSchemeInternetFacing,
				"listenerProtocol":    ListenerProtocolTCP,
				"listenerPort":        80,
				"listenerTargetGroup": tgARN,
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
		assert.Len(t, httpContext.Requests, 2)

		stored, ok := metaCtx.Get().(CreateLoadBalancerExecutionMetadata)
		require.True(t, ok)
		assert.True(t, stored.ListenerCreated)
		assert.Equal(t, 0, stored.ListenerErrors)
	})

	t.Run("active state with listener creation failure -> schedules retry not immediate failure", func(t *testing.T) {
		lbARN := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/50dc6c495c0c9188"
		tgARN := "arn:aws:elasticloadbalancing:us-east-1:123:targetgroup/tg/abc"
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
										<DNSName>my-lb.us-east-1.elb.amazonaws.com</DNSName>
										<Scheme>internet-facing</Scheme>
										<Type>application</Type>
										<State><Code>active</Code></State>
										<VpcId>vpc-12345678</VpcId>
									</member>
								</LoadBalancers>
							</DescribeLoadBalancersResult>
							<ResponseMetadata><RequestId>req-1</RequestId></ResponseMetadata>
						</DescribeLoadBalancersResponse>
					`)),
				},
				{
					StatusCode: http.StatusBadRequest,
					Body: io.NopCloser(strings.NewReader(`
						<ErrorResponse>
							<Error>
								<Code>TargetGroupAssociationLimit</Code>
								<Message>The target group is already associated with another load balancer.</Message>
							</Error>
							<RequestId>req-err-1</RequestId>
						</ErrorResponse>
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
				"name":                "my-lb",
				"region":              "us-east-1",
				"type":                LoadBalancerTypeApplication,
				"scheme":              LoadBalancerSchemeInternetFacing,
				"listenerProtocol":    ListenerProtocolHTTP,
				"listenerPort":        80,
				"listenerTargetGroup": tgARN,
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

		stored, ok := metaCtx.Get().(CreateLoadBalancerExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, 1, stored.ListenerErrors)
		assert.False(t, stored.ListenerCreated)
	})

	t.Run("listener creation failure exceeds limit -> fails execution", func(t *testing.T) {
		lbARN := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/50dc6c495c0c9188"
		tgARN := "arn:aws:elasticloadbalancing:us-east-1:123:targetgroup/tg/abc"

		describeResponse := func() *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DescribeLoadBalancersResponse xmlns="https://elasticloadbalancing.amazonaws.com/doc/2015-12-01/">
						<DescribeLoadBalancersResult>
							<LoadBalancers>
								<member>
									<LoadBalancerArn>` + lbARN + `</LoadBalancerArn>
									<LoadBalancerName>my-lb</LoadBalancerName>
									<DNSName>my-lb.us-east-1.elb.amazonaws.com</DNSName>
									<Scheme>internet-facing</Scheme>
									<Type>application</Type>
									<State><Code>active</Code></State>
									<VpcId>vpc-12345678</VpcId>
								</member>
							</LoadBalancers>
						</DescribeLoadBalancersResult>
						<ResponseMetadata><RequestId>req-1</RequestId></ResponseMetadata>
					</DescribeLoadBalancersResponse>
				`)),
			}
		}
		listenerErrorResponse := func() *http.Response {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body: io.NopCloser(strings.NewReader(`
					<ErrorResponse>
						<Error>
							<Code>TargetGroupAssociationLimit</Code>
							<Message>Target group already in use.</Message>
						</Error>
						<RequestId>req-err</RequestId>
					</ErrorResponse>
				`)),
			}
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				describeResponse(), listenerErrorResponse(),
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.HandleHook(core.ActionHookContext{
			Name: "poll",
			Configuration: map[string]any{
				"name":                "my-lb",
				"region":              "us-east-1",
				"type":                LoadBalancerTypeApplication,
				"scheme":              LoadBalancerSchemeInternetFacing,
				"listenerProtocol":    ListenerProtocolHTTP,
				"listenerPort":        80,
				"listenerTargetGroup": tgARN,
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
			Metadata: &contexts.MetadataContext{
				Metadata: CreateLoadBalancerExecutionMetadata{
					LoadBalancerARN: lbARN,
					ListenerErrors:  maxLoadBalancerListenerErrors - 1,
				},
			},
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.True(t, executionState.Finished)
		assert.False(t, executionState.Passed)
		assert.Contains(t, executionState.FailureMessage, "giving up creating listener")
	})

	t.Run("active state with listener already created -> does not create listener again", func(t *testing.T) {
		lbARN := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/50dc6c495c0c9188"
		describeResponse := func() *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DescribeLoadBalancersResponse xmlns="https://elasticloadbalancing.amazonaws.com/doc/2015-12-01/">
						<DescribeLoadBalancersResult>
							<LoadBalancers>
								<member>
									<LoadBalancerArn>` + lbARN + `</LoadBalancerArn>
									<LoadBalancerName>my-lb</LoadBalancerName>
									<DNSName>my-lb.us-east-1.elb.amazonaws.com</DNSName>
									<Scheme>internet-facing</Scheme>
									<Type>application</Type>
									<State><Code>active</Code></State>
									<VpcId>vpc-12345678</VpcId>
								</member>
							</LoadBalancers>
						</DescribeLoadBalancersResult>
						<ResponseMetadata><RequestId>req-2</RequestId></ResponseMetadata>
					</DescribeLoadBalancersResponse>
				`)),
			}
		}

		httpContext := &contexts.HTTPContext{
			// Only one response: describe. If listener creation is attempted a second
			// time, Do() would return "no response mocked" and the test would fail.
			Responses: []*http.Response{describeResponse()},
		}

		integration := &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		}

		// Metadata with ListenerCreated already set
		metaCtx := &contexts.MetadataContext{
			Metadata: CreateLoadBalancerExecutionMetadata{
				LoadBalancerARN: lbARN,
				ListenerCreated: true,
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.HandleHook(core.ActionHookContext{
			Name: "poll",
			Configuration: map[string]any{
				"name":                "my-lb",
				"region":              "us-east-1",
				"type":                LoadBalancerTypeApplication,
				"scheme":              LoadBalancerSchemeInternetFacing,
				"listenerProtocol":    ListenerProtocolHTTP,
				"listenerPort":        80,
				"listenerTargetGroup": "arn:aws:elasticloadbalancing:us-east-1:123:targetgroup/tg/abc",
			},
			HTTP:           httpContext,
			Integration:    integration,
			Metadata:       metaCtx,
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		// Only the describe call was made; no CreateListener call
		assert.Len(t, httpContext.Requests, 1)
		assert.True(t, executionState.Passed)
	})
}
