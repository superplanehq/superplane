// Package ec2provision maintains pools of Ubuntu EC2 runners. Each pool is one
// Launcher: cloud-init installs the statically linked runner binary from S3 and runs
// it under systemd. Multiple Launchers may run inside one fleet-manager process
// (multi-arch / multi-size); they are partitioned in EC2 by the superplane_fleet_id
// tag (see managedRunInstancesTags + managedDescribeFilters).
//
// The Launcher's Config struct is source-of-loader-agnostic. The fleet-manager process
// builds one per pool from the JSON config file (see fleet-manager/internal/config).
package ec2provision

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/superplane/runner/shared/api"
)

type TaskCountsClient interface {
	FleetTaskCounts(ctx context.Context, fleetID string) (api.FleetTaskCountsResponse, error)
}

// Launcher calls EC2 RunInstances with a deterministic cloud-init user-data starter.
type Launcher struct {
	Client       *ec2.Client
	Config       Config
	Log          *slog.Logger
	BrokerClient TaskCountsClient
}

// FleetID returns the broker fleet id this launcher manages (the partition key used in
// the superplane_fleet_id EC2 tag). Stable accessor so callers don't reach into Config.
func (l *Launcher) FleetID() string {
	return l.Config.RunnerFleetID
}

// Config is the per-pool launcher configuration. Source-of-loader-agnostic: callers
// (today: fleet-manager/internal/config from a JSON file) fill it in field-by-field.
type Config struct {
	// AMI is the EC2 image id this pool boots. Per-arch (different AMI for arm64 vs amd64).
	AMI string
	// InstanceType is the EC2 instance type (t3.micro, t4g.micro, …). Per-arch / per-size.
	InstanceType string
	// SubnetID is the VPC subnet runner VMs launch into.
	SubnetID string
	// SecurityGroupIDs are attached to runner VMs.
	SecurityGroupIDs []string
	// RunnerS3URI is s3://bucket/key for the runner binary (requires instance profile with s3:GetObject).
	// Per-arch (runner-linux-amd64 vs runner-linux-arm64).
	RunnerS3URI string
	// RunnerInstallAWSRegion is AWS_DEFAULT_REGION in user-data for aws s3 cp (same region as fleet-manager / bucket).
	RunnerInstallAWSRegion string
	// TaskBrokerURL is the broker URL runner VMs talk to (must be reachable from the runner VPC).
	TaskBrokerURL string
	// RunnerFleetID is this pool's broker fleet id; also the partition value in the superplane_fleet_id EC2 tag.
	RunnerFleetID string
	// RunnersAuthToken is the bearer token runner VMs (and this Launcher's broker client) use against task-broker.
	RunnersAuthToken string
	// KeyName is an optional EC2 key pair name attached to runner VMs.
	KeyName string
	// RunnersIAMProfName is the IAM instance profile attached to runner VMs (lets them read S3, ship logs).
	RunnersIAMProfName string
	// HotInstanceCount is the target pending+running managed instances for this pool when Headroom is 0.
	// Also the fallback target when Headroom > 0 and the broker task-counts call fails.
	HotInstanceCount int
	// Headroom > 0 enables dynamic scaling: want = queued + claimed + Headroom each tick.
	Headroom int
	// RunnerTerminateAfterEachTask sets RUNNER_TERMINATE_AFTER_EACH_TASK in runner user-data;
	// fleet-manager then terminates the EC2 instance after one task.
	RunnerTerminateAfterEachTask bool
	// RunnerCloudWatchLogGroup sets RUNNER_CLOUDWATCH_LOG_GROUP in runner user-data (optional).
	RunnerCloudWatchLogGroup string
	// RunnerCloudWatchLogStreamPrefix sets RUNNER_CLOUDWATCH_LOG_STREAM_PREFIX (optional;
	// must match TASK_CLOUDWATCH_LOG_STREAM_PREFIX on the task-broker side).
	RunnerCloudWatchLogStreamPrefix string
	// RunnerProcessLogGroup is the CloudWatch log group for runner process (systemd service) logs.
	// When set, the CloudWatch agent is installed and configured to ship journald output for
	// superplane-runner.service to this group under stream name <instance-id>/runner-process.
	RunnerProcessLogGroup string
	// VolumeSizeGB is the root EBS volume size in GiB.
	VolumeSizeGB int32
	// RunnerProcessLogRegion is the AWS region for runner process logs (defaults to RunnerInstallAWSRegion).
	RunnerProcessLogRegion string
	// BootGraceSec skips health probes for newly launched instances during this grace window.
	BootGraceSec int
	// RunnerHealthPort is the TCP port for GET /healthz on runner private IP (default 9090).
	RunnerHealthPort int
}

const (
	// TagKeyManaged is applied to fleet-manager-managed runner instances for Describe/Reconcile filtering.
	TagKeyManaged = "superplane_managed_runner"
	// TagKeyFleetID partitions managed runners by their owning fleet. Multiple pools
	// (different VM arches / instance types) inside one fleet-manager process must filter
	// Describe results by this tag so they don't reconcile each other's instances.
	TagKeyFleetID = "superplane_fleet_id"

	maxLaunch = 50
)

// managedRunInstancesTags returns the tags applied at launch to every managed runner instance.
// Includes the per-fleet partition key (TagKeyFleetID) so multiple pools managed by one
// fleet-manager process do not reconcile each other's instances.
func managedRunInstancesTags(fleetID string) []types.Tag {
	return []types.Tag{
		{Key: aws.String("Name"), Value: aws.String("superplane-runner")},
		{Key: aws.String(TagKeyManaged), Value: aws.String("true")},
		{Key: aws.String(TagKeyFleetID), Value: aws.String(fleetID)},
	}
}

// managedDescribeFilters returns DescribeInstances filters that scope results to the
// fleet-manager-managed runners owned by this pool (fleetID), restricted to the given
// instance-state-name values.
func managedDescribeFilters(fleetID string, states []string) []types.Filter {
	return []types.Filter{
		{Name: aws.String("tag:" + TagKeyManaged), Values: []string{"true"}},
		{Name: aws.String("tag:" + TagKeyFleetID), Values: []string{fleetID}},
		{Name: aws.String("instance-state-name"), Values: states},
	}
}

// ValidateRunnerS3URI checks that s is a well-formed s3://bucket/key object URI.
// Callers wrap the returned error with their own field-name (env var name, JSON key, etc.).
func ValidateRunnerS3URI(s string) error {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "s3://") {
		return fmt.Errorf("must start with s3://")
	}
	key := strings.TrimPrefix(s, "s3://")
	if key == "" || !strings.Contains(key, "/") {
		return fmt.Errorf("must be s3://bucket/key (object path required)")
	}
	return nil
}

// New returns an EC2 client + config (uses AWS default credential chain + region).
func New(ctx context.Context, cfg Config, log *slog.Logger) (*Launcher, error) {
	region := strings.TrimSpace(cfg.RunnerInstallAWSRegion)
	if region == "" {
		return nil, fmt.Errorf("runner install region is empty")
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	cli := ec2.NewFromConfig(awsCfg)
	if log == nil {
		log = slog.Default()
	}
	return &Launcher{Client: cli, Config: cfg, Log: log}, nil
}

// Launch creates `count` on-demand Ubuntu hosts that install the runner from S3 and connect to TaskBrokerURL.
func (l *Launcher) Launch(ctx context.Context, count int) ([]string, error) {
	if count < 1 {
		return nil, fmt.Errorf("count must be at least 1")
	}
	if count > maxLaunch {
		return nil, fmt.Errorf("count exceeds maximum of %d", maxLaunch)
	}
	n := int32(count)
	userdata, err := userDataScript(l.Config)
	if err != nil {
		return nil, fmt.Errorf("user-data script: %w", err)
	}
	in := &ec2.RunInstancesInput{
		ImageId:          aws.String(l.Config.AMI),
		InstanceType:     types.InstanceType(l.Config.InstanceType),
		MinCount:         aws.Int32(n),
		MaxCount:         aws.Int32(n),
		UserData:         aws.String(base64.StdEncoding.EncodeToString([]byte(userdata))),
		SubnetId:         aws.String(l.Config.SubnetID),
		SecurityGroupIds: l.Config.SecurityGroupIDs,
		BlockDeviceMappings: []types.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &types.EbsBlockDevice{
					VolumeSize:          aws.Int32(l.Config.VolumeSizeGB),
					VolumeType:          types.VolumeTypeGp3,
					DeleteOnTermination: aws.Bool(true),
				},
			},
		},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         managedRunInstancesTags(l.Config.RunnerFleetID),
			},
		},
	}
	if l.Config.KeyName != "" {
		in.KeyName = aws.String(l.Config.KeyName)
	}
	if l.Config.RunnersIAMProfName != "" {
		in.IamInstanceProfile = &types.IamInstanceProfileSpecification{Name: aws.String(l.Config.RunnersIAMProfName)}
	}
	out, err := l.Client.RunInstances(ctx, in)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, inst := range out.Instances {
		if inst.InstanceId != nil {
			ids = append(ids, *inst.InstanceId)
		}
	}
	if l.Log != nil {
		l.Log.Info("ec2 RunInstances launched", slog.Int("count", count), slog.Any("instance_ids", ids))
	}
	return ids, nil
}
