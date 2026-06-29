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
	"github.com/superplanehq/superplane/pkg/configuration"
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

func Test__CreateInstance__ConfigurationIncludesProductionLaunchFields(t *testing.T) {
	component := &CreateInstance{}
	fields := component.Configuration()

	securityGroup := findConfigurationField(t, fields, "securityGroup")
	require.NotNil(t, securityGroup.TypeOptions)
	require.NotNil(t, securityGroup.TypeOptions.Resource)
	assert.Equal(t, "ec2.securityGroup", securityGroup.TypeOptions.Resource.Type)
	assert.True(t, securityGroup.TypeOptions.Resource.Multi)

	iamProfile := findConfigurationField(t, fields, "iamInstanceProfile")
	require.NotNil(t, iamProfile.TypeOptions)
	require.NotNil(t, iamProfile.TypeOptions.Resource)
	assert.Equal(t, ResourceTypeIAMInstanceProfile, iamProfile.TypeOptions.Resource.Type)

	tags := findConfigurationField(t, fields, "tags")
	require.NotNil(t, tags.TypeOptions)
	require.NotNil(t, tags.TypeOptions.List)
	assert.ElementsMatch(t, []string{"key", "value"}, configurationFieldNames(tags.TypeOptions.List.ItemDefinition.Schema))

	blockDevices := findConfigurationField(t, fields, "additionalBlockDevices")
	require.NotNil(t, blockDevices.TypeOptions)
	require.NotNil(t, blockDevices.TypeOptions.List)
	assert.Contains(t, configurationFieldNames(blockDevices.TypeOptions.List.ItemDefinition.Schema), "deviceName")
	assert.Contains(t, configurationFieldNames(blockDevices.TypeOptions.List.ItemDefinition.Schema), "kmsKeyId")

	timeout := findConfigurationField(t, fields, "waitForRunningTimeoutSeconds")
	require.NotNil(t, timeout.TypeOptions)
	require.NotNil(t, timeout.TypeOptions.Number)
	require.NotNil(t, timeout.TypeOptions.Number.Max)
	assert.Equal(t, maxWaitTimeoutSeconds, *timeout.TypeOptions.Number.Max)

	assert.ElementsMatch(t, []string{createInstanceCreated, createInstanceFailed}, outputChannelNames(component.OutputChannels(nil)))

	output := component.ExampleOutput()
	data, ok := output["data"].(map[string]any)
	require.True(t, ok)
	for _, key := range []string{"instanceId", "publicDnsName", "publicIpAddress", "privateDnsName", "privateIpAddress", "name", "state", "availabilityZone", "instanceType", "imageId", "tags"} {
		assert.Contains(t, data, key)
	}
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

	t.Run("IAM profile, tags, and multiple security groups are sent", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						<RunInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
							<requestId>req-789</requestId>
							<instancesSet>
								<item>
									<instanceId>i-ghi789</instanceId>
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
				"securityGroup":            []any{"sg-base", "sg-service"},
				"iamInstanceProfile":       "superplane-release-canary-ec2",
				"associatePublicIpAddress": true,
				"tags": []any{
					map[string]any{"key": "purpose", "value": "release-canary"},
					map[string]any{"key": "target", "value": "demo"},
				},
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
		bodyString := string(body)
		assert.Contains(t, bodyString, "NetworkInterface.1.SecurityGroupId.1=sg-base")
		assert.Contains(t, bodyString, "NetworkInterface.1.SecurityGroupId.2=sg-service")
		assert.Contains(t, bodyString, "IamInstanceProfile.Name=superplane-release-canary-ec2")
		assert.Contains(t, bodyString, "TagSpecification.1.ResourceType=instance")
		assert.Contains(t, bodyString, "TagSpecification.1.Tag.1.Key=Name")
		assert.Contains(t, bodyString, "TagSpecification.1.Tag.1.Value=builder")
		assert.Contains(t, bodyString, "TagSpecification.1.Tag.2.Key=purpose")
		assert.Contains(t, bodyString, "TagSpecification.1.Tag.2.Value=release-canary")
		assert.Contains(t, bodyString, "TagSpecification.1.Tag.3.Key=target")
		assert.Contains(t, bodyString, "TagSpecification.2.ResourceType=volume")
		assert.Contains(t, bodyString, "TagSpecification.2.Tag.1.Key=Name")
		assert.Contains(t, bodyString, "TagSpecification.2.Tag.2.Key=purpose")
	})

	t.Run("RunInstances API error emits failed output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body: io.NopCloser(strings.NewReader(`
						<ErrorResponse>
							<Errors>
								<Error>
									<Code>InsufficientInstanceCapacity</Code>
									<Message>We currently do not have sufficient capacity.</Message>
								</Error>
							</Errors>
						</ErrorResponse>
					`)),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":                     "builder",
				"region":                   "us-east-1",
				"image":                    "ami-123",
				"instanceType":             "t3.micro",
				"subnet":                   "subnet-123",
				"securityGroupMode":        "existing",
				"securityGroup":            "sg-123",
				"associatePublicIpAddress": true,
			},
			HTTP:           httpContext,
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
			ExecutionState: executionState,
			Integration: &contexts.IntegrationContext{
				CurrentSecrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, createInstanceFailed, executionState.Channel)
		payload := emittedPayloadData(t, executionState)
		assert.Contains(t, payload["error"], "failed to run instance")
		assert.Equal(t, "InsufficientInstanceCapacity", payload["awsErrorCode"])
		assert.Equal(t, "", payload["instanceId"])
		assert.Equal(t, "", payload["lastObservedState"])
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
										<placement><availabilityZone>us-east-1a</availabilityZone></placement>
										<tagSet>
											<item><key>Name</key><value>builder</value></item>
											<item><key>purpose</key><value>release-canary</value></item>
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
	assert.Equal(t, createInstanceCreated, executionState.Channel)
	assert.Equal(t, CreateInstancePayloadType, executionState.Type)
	require.Len(t, executionState.Payloads, 1)
	payload := emittedPayloadData(t, executionState)
	assert.Equal(t, "i-abc123", payload["instanceId"])
	assert.Equal(t, "ec2-54-198-10-42.compute-1.amazonaws.com", payload["publicDnsName"])
	assert.Equal(t, "54.198.10.42", payload["publicIpAddress"])
	assert.Equal(t, "ip-10-0-1-25.ec2.internal", payload["privateDnsName"])
	assert.Equal(t, "10.0.1.25", payload["privateIpAddress"])
	assert.Equal(t, "builder", payload["name"])
	assert.Equal(t, "running", payload["state"])
	assert.Equal(t, "us-east-1a", payload["availabilityZone"])
	assert.Equal(t, "t3.micro", payload["instanceType"])
	assert.Equal(t, "ami-123", payload["imageId"])
	assert.Len(t, payload["tags"], 2)
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

func Test__CreateInstance__PollTimeoutEmitsFailed(t *testing.T) {
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
										<instanceState><name>pending</name></instanceState>
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
			"region":                       "us-east-1",
			"waitForRunningTimeoutSeconds": 1,
			"associatePublicIpAddress":     true,
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
	assert.Equal(t, createInstanceFailed, executionState.Channel)
	payload := emittedPayloadData(t, executionState)
	assert.Contains(t, payload["error"], "timed out waiting for instance i-abc123")
	assert.Equal(t, "i-abc123", payload["instanceId"])
	assert.Equal(t, "pending", payload["lastObservedState"])
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
			assert.Equal(t, createInstanceFailed, executionState.Channel)
			payload := emittedPayloadData(t, executionState)
			assert.Contains(t, payload["error"], "will not reach running without intervention")
			assert.Equal(t, "i-abc123", payload["instanceId"])
			assert.Equal(t, state, payload["lastObservedState"])
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

func Test__ListInstanceProfiles(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					<ListInstanceProfilesResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
						<ListInstanceProfilesResult>
							<InstanceProfiles>
								<member>
									<InstanceProfileName>superplane-release-canary-ec2</InstanceProfileName>
									<Arn>arn:aws:iam::123456789012:instance-profile/superplane-release-canary-ec2</Arn>
								</member>
							</InstanceProfiles>
							<IsTruncated>false</IsTruncated>
						</ListInstanceProfilesResult>
					</ListInstanceProfilesResponse>
				`)),
			},
		},
	}

	resources, err := ListInstanceProfiles(core.ListResourcesContext{
		HTTP: httpContext,
		Parameters: map[string]string{
			"region": "us-east-1",
		},
		Integration: &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
				"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
				"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
			},
		},
	}, ResourceTypeIAMInstanceProfile)

	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, ResourceTypeIAMInstanceProfile, resources[0].Type)
	assert.Equal(t, "superplane-release-canary-ec2", resources[0].Name)
	assert.Equal(t, "superplane-release-canary-ec2", resources[0].ID)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "iam.amazonaws.com", httpContext.Requests[0].URL.Host)
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

func emittedPayloadData(t *testing.T, executionState *contexts.ExecutionStateContext) map[string]any {
	t.Helper()

	require.Len(t, executionState.Payloads, 1)
	wrapped, ok := executionState.Payloads[0].(map[string]any)
	require.True(t, ok)
	payload, ok := wrapped["data"].(map[string]any)
	require.True(t, ok)
	return payload
}

func findConfigurationField(t *testing.T, fields []configuration.Field, name string) configuration.Field {
	t.Helper()

	for _, field := range fields {
		if field.Name == name {
			return field
		}
	}

	require.Failf(t, "missing field", "field %s not found", name)
	return configuration.Field{}
}

func configurationFieldNames(fields []configuration.Field) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}

func outputChannelNames(channels []core.OutputChannel) []string {
	names := make([]string, 0, len(channels))
	for _, channel := range channels {
		names = append(names, channel.Name)
	}
	return names
}
