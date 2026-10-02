package awsprovider

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/superplanehq/superplane/pkg/fleets/provider"
)

const (
	TagKeyManaged       = "superplane_managed_runner"
	TagKeyFleetID       = "superplane_fleet_id"
	TagKeyRunnerID      = "superplane_runner_id"
	TagKeyRunnerVersion = "superplane_runner_version"
	TagKeyArchitecture  = "superplane_runner_arch"

	minVolumeIOPS           int32 = 3000
	maxVolumeIOPS           int32 = 64000
	minVolumeThroughputMBps int32 = 125
	maxVolumeThroughputMBps int32 = 1000
)

type EC2API interface {
	DescribeInstances(
		context.Context,
		*ec2.DescribeInstancesInput,
		...func(*ec2.Options),
	) (*ec2.DescribeInstancesOutput, error)
	RunInstances(
		context.Context,
		*ec2.RunInstancesInput,
		...func(*ec2.Options),
	) (*ec2.RunInstancesOutput, error)
	TerminateInstances(
		context.Context,
		*ec2.TerminateInstancesInput,
		...func(*ec2.Options),
	) (*ec2.TerminateInstancesOutput, error)
}

type Config struct {
	AMI                  string
	InstanceType         string
	Architecture         string
	SubnetIDs            []string
	SecurityGroupIDs     []string
	IAMInstanceProfile   string
	KeyName              string
	VolumeSizeGB         int32
	VolumeIOPS           int32
	VolumeThroughputMBps int32
}

type Provider struct {
	client EC2API
	config Config
	log    *slog.Logger
}

func New(client EC2API, config Config, log *slog.Logger) (*Provider, error) {
	config.AMI = strings.TrimSpace(config.AMI)
	config.InstanceType = strings.TrimSpace(config.InstanceType)
	config.Architecture = strings.ToLower(strings.TrimSpace(config.Architecture))
	if config.AMI == "" {
		return nil, fmt.Errorf("AWS AMI is required")
	}
	if config.InstanceType == "" {
		return nil, fmt.Errorf("AWS instance type is required")
	}
	if config.Architecture != "amd64" && config.Architecture != "arm64" {
		return nil, fmt.Errorf("AWS architecture must be amd64 or arm64")
	}
	config.SubnetIDs = nonEmpty(config.SubnetIDs)
	if len(config.SubnetIDs) == 0 {
		return nil, fmt.Errorf("at least one AWS subnet is required")
	}
	config.SecurityGroupIDs = nonEmpty(config.SecurityGroupIDs)
	if len(config.SecurityGroupIDs) == 0 {
		return nil, fmt.Errorf("at least one AWS security group is required")
	}
	if config.VolumeSizeGB <= 0 {
		config.VolumeSizeGB = 30
	}
	if config.VolumeIOPS != 0 &&
		(config.VolumeIOPS < minVolumeIOPS || config.VolumeIOPS > maxVolumeIOPS) {
		return nil, fmt.Errorf(
			"AWS volume IOPS must be 0 or between %d and %d",
			minVolumeIOPS,
			maxVolumeIOPS,
		)
	}
	if config.VolumeThroughputMBps != 0 &&
		(config.VolumeThroughputMBps < minVolumeThroughputMBps ||
			config.VolumeThroughputMBps > maxVolumeThroughputMBps) {
		return nil, fmt.Errorf(
			"AWS volume throughput must be 0 or between %d and %d MiB/s",
			minVolumeThroughputMBps,
			maxVolumeThroughputMBps,
		)
	}
	if client == nil {
		return nil, fmt.Errorf("AWS EC2 client is required")
	}
	if log == nil {
		log = slog.Default()
	}
	return &Provider{client: client, config: config, log: log}, nil
}

func (p *Provider) Name() string {
	return "aws"
}

func (p *Provider) List(ctx context.Context, fleetID string) ([]provider.Resource, error) {
	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{Name: aws.String("tag:" + TagKeyManaged), Values: []string{"true"}},
			{Name: aws.String("tag:" + TagKeyFleetID), Values: []string{fleetID}},
			{
				Name: aws.String("instance-state-name"),
				Values: []string{
					"pending",
					"running",
					"stopping",
					"stopped",
					"shutting-down",
				},
			},
		},
	}

	var resources []provider.Resource
	for {
		output, err := p.client.DescribeInstances(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("describe EC2 runner instances: %w", err)
		}
		for _, reservation := range output.Reservations {
			for _, instance := range reservation.Instances {
				runnerID := tagValue(instance.Tags, TagKeyRunnerID)
				if runnerID == "" || instance.InstanceId == nil {
					continue
				}
				resource := provider.Resource{
					ID:       aws.ToString(instance.InstanceId),
					RunnerID: runnerID,
					FleetID:  tagValue(instance.Tags, TagKeyFleetID),
				}
				if instance.State != nil {
					resource.State = string(instance.State.Name)
				}
				if instance.LaunchTime != nil {
					resource.CreatedAt = *instance.LaunchTime
				}
				resources = append(resources, resource)
			}
		}
		if output.NextToken == nil || aws.ToString(output.NextToken) == "" {
			return resources, nil
		}
		input.NextToken = output.NextToken
	}
}

func (p *Provider) BuildBootstrap(request provider.RunnerBootstrap) ([]byte, error) {
	return buildUserData(request)
}

func (p *Provider) Create(
	ctx context.Context,
	request provider.CreateRequest,
) (provider.Resource, error) {
	if strings.TrimSpace(request.RunnerID) == "" {
		return provider.Resource{}, fmt.Errorf("runner ID is required")
	}
	if strings.TrimSpace(request.FleetID) == "" {
		return provider.Resource{}, fmt.Errorf("fleet ID is required")
	}
	if len(request.Bootstrap) == 0 {
		return provider.Resource{}, fmt.Errorf("runner bootstrap data is required")
	}

	var lastErr error
	for _, subnetID := range p.config.SubnetIDs {
		output, err := p.client.RunInstances(ctx, p.runInstancesInput(request, subnetID))
		if err == nil {
			if len(output.Instances) != 1 || output.Instances[0].InstanceId == nil {
				return provider.Resource{}, fmt.Errorf("EC2 RunInstances returned no instance")
			}
			instance := output.Instances[0]
			resource := provider.Resource{
				ID:        aws.ToString(instance.InstanceId),
				RunnerID:  request.RunnerID,
				FleetID:   request.FleetID,
				State:     string(types.InstanceStateNamePending),
				CreatedAt: time.Now().UTC(),
			}
			if instance.State != nil {
				resource.State = string(instance.State.Name)
			}
			if instance.LaunchTime != nil {
				resource.CreatedAt = *instance.LaunchTime
			}
			p.log.Info(
				"created AWS runner",
				slog.String("runner_id", request.RunnerID),
				slog.String("fleet_id", request.FleetID),
				slog.String("instance_id", resource.ID),
			)
			return resource, nil
		}
		lastErr = err
		if !isInsufficientCapacity(err) {
			return provider.Resource{}, fmt.Errorf("create EC2 runner: %w", err)
		}
		p.log.Warn(
			"AWS subnet has insufficient capacity",
			slog.String("subnet_id", subnetID),
			slog.String("runner_id", request.RunnerID),
		)
	}
	return provider.Resource{}, fmt.Errorf("create EC2 runner in configured subnets: %w", lastErr)
}

func (p *Provider) Delete(ctx context.Context, resource provider.Resource) error {
	instanceID := strings.TrimSpace(resource.ID)
	if instanceID == "" {
		return fmt.Errorf("AWS instance ID is required")
	}
	_, err := p.client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if isInstanceNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("terminate EC2 runner %s: %w", instanceID, err)
	}
	p.log.Info(
		"deleted AWS runner",
		slog.String("runner_id", resource.RunnerID),
		slog.String("instance_id", instanceID),
	)
	return nil
}

func (p *Provider) runInstancesInput(
	request provider.CreateRequest,
	subnetID string,
) *ec2.RunInstancesInput {
	tags := []types.Tag{
		{Key: aws.String("Name"), Value: aws.String("superplane-runner")},
		{Key: aws.String(TagKeyManaged), Value: aws.String("true")},
		{Key: aws.String(TagKeyFleetID), Value: aws.String(request.FleetID)},
		{Key: aws.String(TagKeyRunnerID), Value: aws.String(request.RunnerID)},
		{Key: aws.String(TagKeyRunnerVersion), Value: aws.String(request.RunnerVersion)},
		{Key: aws.String(TagKeyArchitecture), Value: aws.String(p.config.Architecture)},
	}
	input := &ec2.RunInstancesInput{
		ImageId:                           aws.String(p.config.AMI),
		InstanceType:                      types.InstanceType(p.config.InstanceType),
		MinCount:                          aws.Int32(1),
		MaxCount:                          aws.Int32(1),
		ClientToken:                       aws.String(clientToken(request.RunnerID, subnetID)),
		UserData:                          aws.String(base64.StdEncoding.EncodeToString(request.Bootstrap)),
		SubnetId:                          aws.String(subnetID),
		SecurityGroupIds:                  p.config.SecurityGroupIDs,
		InstanceInitiatedShutdownBehavior: types.ShutdownBehaviorTerminate,
		BlockDeviceMappings: []types.BlockDeviceMapping{{
			DeviceName: aws.String("/dev/sda1"),
			Ebs:        p.rootVolume(),
		}},
		TagSpecifications: []types.TagSpecification{
			{ResourceType: types.ResourceTypeInstance, Tags: tags},
			{ResourceType: types.ResourceTypeVolume, Tags: tags},
		},
	}
	if p.config.KeyName != "" {
		input.KeyName = aws.String(p.config.KeyName)
	}
	if p.config.IAMInstanceProfile != "" {
		input.IamInstanceProfile = &types.IamInstanceProfileSpecification{
			Name: aws.String(p.config.IAMInstanceProfile),
		}
	}
	return input
}

func (p *Provider) rootVolume() *types.EbsBlockDevice {
	volume := &types.EbsBlockDevice{
		DeleteOnTermination: aws.Bool(true),
		VolumeSize:          aws.Int32(p.config.VolumeSizeGB),
		VolumeType:          types.VolumeTypeGp3,
	}
	if p.config.VolumeIOPS > 0 {
		volume.Iops = aws.Int32(p.config.VolumeIOPS)
	}
	if p.config.VolumeThroughputMBps > 0 {
		volume.Throughput = aws.Int32(p.config.VolumeThroughputMBps)
	}
	return volume
}

func tagValue(tags []types.Tag, key string) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == key {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func isInsufficientCapacity(err error) bool {
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == "InsufficientInstanceCapacity"
}

func isInstanceNotFound(err error) bool {
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == "InvalidInstanceID.NotFound"
}

func nonEmpty(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func clientToken(runnerID, subnetID string) string {
	const maxRunnerIDLength = 47
	if len(runnerID) > maxRunnerIDLength {
		runnerID = runnerID[:maxRunnerIDLength]
	}
	sum := sha256.Sum256([]byte(runnerID + "\x00" + subnetID))
	return runnerID + "-" + hex.EncodeToString(sum[:8])
}
