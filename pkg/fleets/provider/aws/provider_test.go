package awsprovider

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/superplanehq/superplane/pkg/fleets/artifact"
	"github.com/superplanehq/superplane/pkg/fleets/provider"
)

type fakeEC2 struct {
	describeOutput *ec2.DescribeInstancesOutput
	describeInput  *ec2.DescribeInstancesInput
	runOutputs     []*ec2.RunInstancesOutput
	runErrors      []error
	runInputs      []*ec2.RunInstancesInput
	terminateInput *ec2.TerminateInstancesInput
	terminateError error
}

func (f *fakeEC2) DescribeInstances(
	_ context.Context,
	input *ec2.DescribeInstancesInput,
	_ ...func(*ec2.Options),
) (*ec2.DescribeInstancesOutput, error) {
	f.describeInput = input
	return f.describeOutput, nil
}

func (f *fakeEC2) RunInstances(
	_ context.Context,
	input *ec2.RunInstancesInput,
	_ ...func(*ec2.Options),
) (*ec2.RunInstancesOutput, error) {
	f.runInputs = append(f.runInputs, input)
	index := len(f.runInputs) - 1
	if index < len(f.runErrors) && f.runErrors[index] != nil {
		return nil, f.runErrors[index]
	}
	if index < len(f.runOutputs) {
		return f.runOutputs[index], nil
	}
	return nil, errors.New("unexpected RunInstances call")
}

func (f *fakeEC2) TerminateInstances(
	_ context.Context,
	input *ec2.TerminateInstancesInput,
	_ ...func(*ec2.Options),
) (*ec2.TerminateInstancesOutput, error) {
	f.terminateInput = input
	return &ec2.TerminateInstancesOutput{}, f.terminateError
}

func TestCreateTagsAWSResourcesWithRunnerIdentity(t *testing.T) {
	client := &fakeEC2{
		runOutputs: []*ec2.RunInstancesOutput{{
			Instances: []types.Instance{{InstanceId: aws.String("i-runner")}},
		}},
	}
	awsProvider := newTestProvider(t, client)
	resource, err := awsProvider.Create(context.Background(), provider.CreateRequest{
		RunnerID:      "09fe835e-9f8d-4ea4-9f78-962494c90e21",
		FleetID:       "linux-amd64",
		RunnerVersion: "1.2.3",
		Bootstrap:     []byte("#!/bin/bash\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resource.ID != "i-runner" || resource.RunnerID != "09fe835e-9f8d-4ea4-9f78-962494c90e21" {
		t.Fatalf("resource = %#v", resource)
	}

	input := client.runInputs[0]
	if !strings.HasPrefix(aws.ToString(input.ClientToken), resource.RunnerID+"-") ||
		len(aws.ToString(input.ClientToken)) > 64 {
		t.Fatalf("client token = %q", aws.ToString(input.ClientToken))
	}
	if aws.ToString(input.SubnetId) != "subnet-a" {
		t.Fatalf("subnet = %q", aws.ToString(input.SubnetId))
	}
	if input.MetadataOptions == nil ||
		input.MetadataOptions.HttpTokens != types.HttpTokensStateRequired {
		t.Fatalf("metadata options = %#v", input.MetadataOptions)
	}
	if got, err := base64.StdEncoding.DecodeString(aws.ToString(input.UserData)); err != nil || string(got) != "#!/bin/bash\n" {
		t.Fatalf("user data = %q, err = %v", got, err)
	}
	tags := tagMap(input.TagSpecifications[0].Tags)
	if tags[TagKeyRunnerID] != resource.RunnerID ||
		tags[TagKeyFleetID] != "linux-amd64" ||
		tags[TagKeyRunnerVersion] != "1.2.3" ||
		tags[TagKeyFleetManagerID] != "fleet-manager-production" ||
		tags["Environment"] != "production" {
		t.Fatalf("tags = %#v", tags)
	}
	if _, exists := tags["superplane_managed_runner"]; exists {
		t.Fatalf("legacy managed tag is present: %#v", tags)
	}
	if len(input.TagSpecifications) != 2 {
		t.Fatalf("tag specifications = %d, want instance and volume", len(input.TagSpecifications))
	}
	volumeTags := tagMap(input.TagSpecifications[1].Tags)
	if volumeTags[TagKeyFleetManagerID] != "fleet-manager-production" ||
		volumeTags["Environment"] != "production" {
		t.Fatalf("volume tags = %#v", volumeTags)
	}
}

func TestListFiltersAWSResourcesByFleetManagerAndFleet(t *testing.T) {
	client := &fakeEC2{describeOutput: &ec2.DescribeInstancesOutput{}}
	awsProvider := newTestProvider(t, client)

	if _, err := awsProvider.List(context.Background(), "linux-amd64"); err != nil {
		t.Fatal(err)
	}
	filters := filterMap(client.describeInput.Filters)
	if filters["tag:"+TagKeyFleetManagerID] != "fleet-manager-production" ||
		filters["tag:"+TagKeyFleetID] != "linux-amd64" {
		t.Fatalf("filters = %#v", filters)
	}
}

func TestNewRejectsResourceTagsThatOverrideReservedTags(t *testing.T) {
	for _, key := range []string{TagKeyFleetID, tagKeyLegacyManaged} {
		t.Run(key, func(t *testing.T) {
			_, err := New(&fakeEC2{}, Config{
				FleetManagerID:   "fleet-manager-production",
				AMI:              "ami-123",
				InstanceType:     "t3.micro",
				Architecture:     "amd64",
				SubnetIDs:        []string{"subnet-a"},
				SecurityGroupIDs: []string{"sg-a"},
				ResourceTags:     map[string]string{key: "overridden"},
			}, nil)
			if err == nil || !strings.Contains(err.Error(), "reserved AWS resource tag") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestNewRejectsTooManyResourceTags(t *testing.T) {
	resourceTags := make(map[string]string, maxCustomResourceTags+1)
	for index := range maxCustomResourceTags + 1 {
		resourceTags[fmt.Sprintf("CustomTag%d", index)] = "value"
	}

	_, err := New(&fakeEC2{}, Config{
		FleetManagerID:   "fleet-manager-production",
		AMI:              "ami-123",
		InstanceType:     "t3.micro",
		Architecture:     "amd64",
		SubnetIDs:        []string{"subnet-a"},
		SecurityGroupIDs: []string{"sg-a"},
		ResourceTags:     resourceTags,
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "at most 44 entries") {
		t.Fatalf("error = %v", err)
	}
}

func TestCreateRetriesNextSubnetWhenAWSCapacityIsUnavailable(t *testing.T) {
	client := &fakeEC2{
		runErrors: []error{
			&smithy.GenericAPIError{Code: "InsufficientInstanceCapacity"},
			nil,
		},
		runOutputs: []*ec2.RunInstancesOutput{
			nil,
			{Instances: []types.Instance{{InstanceId: aws.String("i-second")}}},
		},
	}
	awsProvider := newTestProvider(t, client)
	if _, err := awsProvider.Create(context.Background(), provider.CreateRequest{
		RunnerID:      "runner-1",
		FleetID:       "fleet-a",
		RunnerVersion: "1.0.0",
		Bootstrap:     []byte("bootstrap"),
	}); err != nil {
		t.Fatal(err)
	}
	if len(client.runInputs) != 2 ||
		aws.ToString(client.runInputs[0].SubnetId) != "subnet-a" ||
		aws.ToString(client.runInputs[1].SubnetId) != "subnet-b" {
		t.Fatalf("subnet attempts = %#v", client.runInputs)
	}
}

func TestBuildBootstrapUsesPublicArtifactAndExactVersion(t *testing.T) {
	awsProvider := newTestProvider(t, &fakeEC2{})
	script, err := awsProvider.BuildBootstrap(provider.RunnerBootstrap{
		RunnerID:          "runner-1",
		FleetID:           "fleet-a",
		RunnerAPIURL:      "https://superplane.example",
		RegistrationToken: "short-lived-registration-token",
		Artifact: artifact.Artifact{
			URL:    "https://downloads.example/runner/v1.2.3/runner-linux-amd64.tar.gz",
			SHA256: strings.Repeat("a", 64),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := string(script)
	for _, expected := range []string{
		"https://downloads.example/runner/v1.2.3/runner-linux-amd64.tar.gz",
		"sha256sum --check --strict",
		"--extract",
		"--gzip",
		"set +x",
		`RUNNER_API_URL="https://superplane.example" \`,
		`RUNNER_REGISTRATION_TOKEN="short-lived-registration-token" \`,
		`"$bundle_dir/install.sh"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("bootstrap does not contain %q:\n%s", expected, body)
		}
	}
}

func TestDeleteTreatsMissingAWSInstanceAsSuccess(t *testing.T) {
	client := &fakeEC2{
		terminateError: &smithy.GenericAPIError{Code: "InvalidInstanceID.NotFound"},
	}
	awsProvider := newTestProvider(t, client)
	if err := awsProvider.Delete(context.Background(), provider.Resource{ID: "i-gone"}); err != nil {
		t.Fatal(err)
	}
}

func newTestProvider(t *testing.T, client EC2API) *Provider {
	t.Helper()
	awsProvider, err := New(client, Config{
		FleetManagerID:     "fleet-manager-production",
		AMI:                "ami-123",
		InstanceType:       "t3.micro",
		Architecture:       "amd64",
		SubnetIDs:          []string{"subnet-a", "subnet-b"},
		SecurityGroupIDs:   []string{"sg-a"},
		IAMInstanceProfile: "runner-profile",
		ResourceTags:       map[string]string{"Environment": "production"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return awsProvider
}

func tagMap(tags []types.Tag) map[string]string {
	values := make(map[string]string, len(tags))
	for _, tag := range tags {
		values[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return values
}

func filterMap(filters []types.Filter) map[string]string {
	values := make(map[string]string, len(filters))
	for _, filter := range filters {
		if len(filter.Values) == 1 {
			values[aws.ToString(filter.Name)] = filter.Values[0]
		}
	}
	return values
}
