package ec2

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateInstance__Setup(t *testing.T) {
	component := &CreateInstance{}

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":              "builder",
				"region":            " ",
				"imageOs":           "ubuntu",
				"image":             "ami-123",
				"instanceType":      "t3.micro",
				"subnet":            "subnet-123",
				"securityGroupMode": "existing",
				"securityGroup":     "sg-123",
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":              " ",
				"region":            "us-east-1",
				"imageOs":           "ubuntu",
				"image":             "ami-123",
				"instanceType":      "t3.micro",
				"subnet":            "subnet-123",
				"securityGroupMode": "existing",
				"securityGroup":     "sg-123",
			},
		})
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":              "builder",
				"region":            "us-east-1",
				"imageOs":           "ubuntu",
				"image":             "ami-123",
				"instanceType":      "t3.micro",
				"subnet":            "subnet-123",
				"securityGroupMode": "existing",
				"securityGroup":     "sg-123",
			},
			Metadata: metadata,
		})
		require.NoError(t, err)

		stored, ok := metadata.Get().(CreateInstanceNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "us-east-1", stored.Region)
		assert.Equal(t, "builder", stored.Name)
		assert.Equal(t, "ubuntu", stored.ImageOS)
		assert.Equal(t, "Ubuntu", stored.ImageOSLabel)
		assert.Equal(t, "t3.micro", stored.InstanceType)
	})
}

func Test__CreateInstance__Execute(t *testing.T) {
	component := &CreateInstance{}

	t.Run("run instance -> schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<DescribeImagesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<imagesSet>
								<item>
									<imageId>ami-123</imageId>
									<rootDeviceName>/dev/xvda</rootDeviceName>
									<imageState>available</imageState>
								</item>
							</imagesSet>
						</DescribeImagesResponse>
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<RunInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<requestId>req-123</requestId>
							<instancesSet>
								<item>
									<instanceId>i-abc123</instanceId>
									<instanceState><name>pending</name></instanceState>
								</item>
							</instancesSet>
						</RunInstancesResponse>
					`)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":                     "builder",
				"region":                   "us-east-1",
				"imageOs":                  "ubuntu",
				"image":                    "ami-123",
				"instanceType":             "t3.micro",
				"subnet":                   "subnet-123",
				"securityGroupMode":        "existing",
				"securityGroup":            "sg-123",
				"allowSshFromInternet":     false,
				"keyName":                  "my-key",
				"userData":                 "#!/bin/bash\necho hello",
				"configureRootVolume":      true,
				"volumeSizeGiB":            30,
				"volumeType":               "gp3",
				"associatePublicIpAddress": true,
			},
			HTTP:     httpContext,
			Metadata: metadata,
			Requests: requests,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		body, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		bodyString := string(body)
		assert.Contains(t, bodyString, "Action=RunInstances")
		assert.Contains(t, bodyString, "ImageId=ami-123")
		assert.Contains(t, bodyString, "InstanceType=t3.micro")
		assert.Contains(t, bodyString, "NetworkInterface.1.SubnetId=subnet-123")
		assert.Contains(t, bodyString, "NetworkInterface.1.SecurityGroupId.1=sg-123")
		assert.Contains(t, bodyString, "NetworkInterface.1.AssociatePublicIpAddress=true")
		assert.Contains(t, bodyString, "KeyName=my-key")
		assert.Contains(t, bodyString, "UserData="+url.QueryEscape(base64.StdEncoding.EncodeToString([]byte("#!/bin/bash\necho hello"))))
		assert.Contains(t, bodyString, "BlockDeviceMapping.1.DeviceName=%2Fdev%2Fxvda")
		assert.Contains(t, bodyString, "BlockDeviceMapping.1.Ebs.VolumeSize=30")
		assert.Contains(t, bodyString, "BlockDeviceMapping.1.Ebs.VolumeType=gp3")
		assert.Equal(t, "poll", requests.Action)
		assert.Equal(t, instancePollInterval, requests.Duration)

		stored, ok := metadata.Get().(CreateInstanceExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, "i-abc123", stored.InstanceID)
	})

	t.Run("associate public IP disabled -> sends false explicitly", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<RunInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<requestId>req-456</requestId>
							<instancesSet>
								<item>
									<instanceId>i-def456</instanceId>
									<instanceState><name>pending</name></instanceState>
								</item>
							</instancesSet>
						</RunInstancesResponse>
					`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":                     "builder",
				"region":                   "us-east-1",
				"image":                    "ami-123",
				"instanceType":             "t3.micro",
				"subnet":                   "subnet-123",
				"securityGroupMode":        "existing",
				"securityGroup":            "sg-123",
				"associatePublicIpAddress": false,
			},
			HTTP:     httpContext,
			Metadata: &contexts.MetadataContext{},
			Requests: &contexts.RequestContext{},
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "NetworkInterface.1.AssociatePublicIpAddress=false")
	})
}

func Test__CreateInstance__PollEmitsWhenRunning(t *testing.T) {
	component := &CreateInstance{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-456</requestId>
						<reservationSet>
							<item>
								<instancesSet>
									<item>
										<instanceId>i-abc123</instanceId>
										<instanceType>t3.micro</instanceType>
										<imageId>ami-123</imageId>
										<instanceState><name>running</name></instanceState>
										<privateIpAddress>10.0.1.25</privateIpAddress>
										<ipAddress>54.198.10.42</ipAddress>
										<dnsName>ec2-54-198-10-42.compute-1.amazonaws.com</dnsName>
										<privateDnsName>ip-10-0-1-25.ec2.internal</privateDnsName>
										<subnetId>subnet-123</subnetId>
										<vpcId>vpc-123</vpcId>
										<tagSet>
											<item><key>Name</key><value>builder</value></item>
										</tagSet>
									</item>
								</instancesSet>
							</item>
						</reservationSet>
					</DescribeInstancesResponse>
				`)),
			},
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.HandleHook(core.ActionHookContext{
		Name: "poll",
		Configuration: map[string]any{
			"region": "us-east-1",
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
			Metadata: CreateInstanceExecutionMetadata{InstanceID: "i-abc123"},
		},
		Requests:       &contexts.RequestContext{},
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, CreateInstancePayloadType, executionState.Type)
	require.Len(t, executionState.Payloads, 1)
}

func Test__CreateInstance__PollReschedulesWhenInstanceShuttingDown(t *testing.T) {
	component := &CreateInstance{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<reservationSet>
							<item>
								<instancesSet>
									<item>
										<instanceId>i-abc123</instanceId>
										<instanceState><name>shutting-down</name></instanceState>
									</item>
								</instancesSet>
							</item>
						</reservationSet>
					</DescribeInstancesResponse>
				`)),
			},
		},
	}
	requests := &contexts.RequestContext{}

	err := component.HandleHook(core.ActionHookContext{
		Name: "poll",
		Configuration: map[string]any{
			"region": "us-east-1",
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
			Metadata: CreateInstanceExecutionMetadata{InstanceID: "i-abc123"},
		},
		Requests:       requests,
		ExecutionState: &contexts.ExecutionStateContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	assert.Equal(t, "poll", requests.Action, "shutting-down is transient; poll should be rescheduled")
}

func Test__CreateInstance__PollFailsImmediatelyOnNonRecoverableState(t *testing.T) {
	for _, state := range []string{"terminated", "stopped", "stopping"} {
		t.Run(state, func(t *testing.T) {
			component := &CreateInstance{}
			httpContext := &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`
							<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
								<reservationSet>
									<item>
										<instancesSet>
											<item>
												<instanceId>i-abc123</instanceId>
												<instanceState><name>` + state + `</name></instanceState>
											</item>
										</instancesSet>
									</item>
								</reservationSet>
							</DescribeInstancesResponse>
						`)),
					},
				},
			}

			err := component.HandleHook(core.ActionHookContext{
				Name: "poll",
				Configuration: map[string]any{
					"region": "us-east-1",
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
					Metadata: CreateInstanceExecutionMetadata{InstanceID: "i-abc123"},
				},
				Requests:       &contexts.RequestContext{},
				ExecutionState: &contexts.ExecutionStateContext{},
				Logger:         logrus.NewEntry(logrus.New()),
			})

			require.ErrorContains(t, err, "will not reach running without intervention")
		})
	}
}

func Test__CreateInstance__PollErrorsWhenMetadataMissingInstanceID(t *testing.T) {
	component := &CreateInstance{}

	err := component.HandleHook(core.ActionHookContext{
		Name: "poll",
		Configuration: map[string]any{
			"region": "us-east-1",
		},
		Metadata:       &contexts.MetadataContext{Metadata: CreateInstanceExecutionMetadata{}},
		Requests:       &contexts.RequestContext{},
		ExecutionState: &contexts.ExecutionStateContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	})

	require.ErrorContains(t, err, "poll metadata is missing instanceId")
}

func Test__CreateInstance__ExecuteCreatesSecurityGroup(t *testing.T) {
	component := &CreateInstance{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<DescribeSubnetsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<subnetSet>
							<item>
								<subnetId>subnet-123</subnetId>
								<vpcId>vpc-123</vpcId>
								<cidrBlock>10.0.1.0/24</cidrBlock>
								<availabilityZone>us-east-1a</availabilityZone>
							</item>
						</subnetSet>
					</DescribeSubnetsResponse>
				`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<CreateSecurityGroupResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-sg</requestId>
						<groupId>sg-new123</groupId>
					</CreateSecurityGroupResponse>
				`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<AuthorizeSecurityGroupIngressResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-auth</requestId>
						<return>true</return>
					</AuthorizeSecurityGroupIngressResponse>
				`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<RunInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
						<requestId>req-123</requestId>
						<instancesSet>
							<item>
								<instanceId>i-abc123</instanceId>
								<instanceState><name>pending</name></instanceState>
							</item>
						</instancesSet>
					</RunInstancesResponse>
				`)),
			},
		},
	}

	metadata := &contexts.MetadataContext{}
	requests := &contexts.RequestContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"name":                     "builder",
			"region":                   "us-east-1",
			"imageOs":                  "ubuntu",
			"image":                    "ami-123",
			"instanceType":             "t3.micro",
			"subnet":                   "subnet-123",
			"securityGroupMode":        "create",
			"allowSshFromInternet":     true,
			"associatePublicIpAddress": true,
		},
		HTTP:     httpContext,
		Metadata: metadata,
		Requests: requests,
		Integration: &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 4)

	createBody, err := io.ReadAll(httpContext.Requests[1].Body)
	require.NoError(t, err)
	createBodyString := string(createBody)
	assert.Contains(t, createBodyString, "Action=CreateSecurityGroup")
	assert.Contains(t, createBodyString, "GroupName=launch-wizard-builder")
	assert.Contains(t, createBodyString, "VpcId=vpc-123")

	authBody, err := io.ReadAll(httpContext.Requests[2].Body)
	require.NoError(t, err)
	authBodyString := string(authBody)
	assert.Contains(t, authBodyString, "Action=AuthorizeSecurityGroupIngress")
	assert.Contains(t, authBodyString, "GroupId=sg-new123")
	assert.Contains(t, authBodyString, "FromPort=22")
	assert.Contains(t, authBodyString, "CidrIp=0.0.0.0%2F0")

	runBody, err := io.ReadAll(httpContext.Requests[3].Body)
	require.NoError(t, err)
	runBodyString := string(runBody)
	assert.Contains(t, runBodyString, "NetworkInterface.1.SecurityGroupId.1=sg-new123")
}

func Test__CreateInstance__Cancel(t *testing.T) {
	component := &CreateInstance{}
	config := map[string]any{
		"name":              "builder",
		"region":            "us-east-1",
		"image":             "ami-123",
		"instanceType":      "t3.micro",
		"subnet":            "subnet-123",
		"securityGroupMode": "existing",
		"securityGroup":     "sg-123",
	}

	t.Run("no instance launched -> no-op", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		err := component.Cancel(core.ExecutionContext{
			Configuration: config,
			HTTP:          httpContext,
			Metadata:      &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 0, "no API calls expected when no instance launched")
	})

	t.Run("instance launched -> terminates it", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<TerminateInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<requestId>req-term</requestId>
							<instancesSet>
								<item>
									<instanceId>i-abc123</instanceId>
									<previousState><name>running</name><code>16</code></previousState>
									<currentState><name>shutting-down</name><code>32</code></currentState>
								</item>
							</instancesSet>
						</TerminateInstancesResponse>
					`)),
				},
			},
		}
		err := component.Cancel(core.ExecutionContext{
			Configuration: config,
			HTTP:          httpContext,
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{"instanceId": "i-abc123"},
			},
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Action=TerminateInstances")
		assert.Contains(t, string(body), "InstanceId.1=i-abc123")
	})

	t.Run("instance already gone -> no error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body: io.NopCloser(strings.NewReader(`
						<ErrorResponse>
							<Errors>
								<Error>
									<Code>InvalidInstanceID.NotFound</Code>
									<Message>The instance ID 'i-abc123' does not exist</Message>
								</Error>
							</Errors>
						</ErrorResponse>
					`)),
				},
			},
		}
		err := component.Cancel(core.ExecutionContext{
			Configuration: config,
			HTTP:          httpContext,
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{"instanceId": "i-abc123"},
			},
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
}
