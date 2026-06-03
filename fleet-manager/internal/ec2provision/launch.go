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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	fmmetrics "github.com/superplane/runner/fleet-manager/internal/metrics"
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
	Metrics      *fmmetrics.PoolMetrics

	pendingMu sync.Mutex
	pending   map[string]time.Time // instance id -> RunInstances request time
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
	// Arch is the CPU architecture: "amd64" (default) or "arm64" (Graviton).
	// Controls AWS CLI and CloudWatch agent download URLs in runner user-data.
	Arch string
	// FleetID is a stable identifier for this fleet-manager deployment, used as the
	// superplane_fleet_id EC2 tag to partition runner instances across multiple
	// fleet-managers in the same AWS account. Defaults to hostname via ConfigFromEnv.
	// In the JSON config path, set equal to RunnerFleetID.
	FleetID string
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
	// TagKeyArch records the CPU architecture of the runner instance (amd64 or arm64).
	TagKeyArch = "superplane_runner_arch"
	// TagKeyFleetID partitions managed runners by their owning fleet. Multiple pools
	// (different VM arches / instance types) inside one fleet-manager process must filter
	// Describe results by this tag so they don't reconcile each other's instances.
	TagKeyFleetID = "superplane_fleet_id"

	maxLaunch = 50
)

// managedRunInstancesTags returns the tags applied at launch to every managed runner instance.
func managedRunInstancesTags(fleetID, arch string) []types.Tag {
	return []types.Tag{
		{Key: aws.String("Name"), Value: aws.String("superplane-runner")},
		{Key: aws.String(TagKeyManaged), Value: aws.String("true")},
		{Key: aws.String(TagKeyArch), Value: aws.String(configArch(arch))},
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
	return &Launcher{Client: cli, Config: cfg, Log: log, pending: make(map[string]time.Time)}, nil
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
	requestedAt := time.Now().UTC()
	userdata, err := userDataScript(l.Config, requestedAt.Unix())
	if err != nil {
		return nil, fmt.Errorf("user-data script: %w", err)
	}
	in := l.runInstancesInput(n, base64.StdEncoding.EncodeToString([]byte(userdata)))
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
	l.trackPendingLaunches(ids, requestedAt)
	if l.Log != nil {
		l.Log.Info("ec2 RunInstances launched", slog.Int("count", count), slog.Any("instance_ids", ids))
	}
	return ids, nil
}

// runInstancesInput builds the RunInstancesInput for launching count runner VMs.
func (l *Launcher) runInstancesInput(count int32, encodedUserData string) *ec2.RunInstancesInput {
	in := &ec2.RunInstancesInput{
		ImageId:          aws.String(l.Config.AMI),
		InstanceType:     types.InstanceType(l.Config.InstanceType),
		MinCount:         aws.Int32(count),
		MaxCount:         aws.Int32(count),
		UserData:         aws.String(encodedUserData),
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
				Tags:         managedRunInstancesTags(l.Config.FleetID, l.Config.Arch),
			},
		},
	}
	if l.Config.KeyName != "" {
		in.KeyName = aws.String(l.Config.KeyName)
	}
	if l.Config.RunnersIAMProfName != "" {
		in.IamInstanceProfile = &types.IamInstanceProfileSpecification{Name: aws.String(l.Config.RunnersIAMProfName)}
	}
	return in
}

// managedInstanceFilters returns DescribeInstances filters scoped to this launcher's fleet and arch.
func (l *Launcher) managedInstanceFilters(states []string) []types.Filter {
	arch := configArch(l.Config.Arch)
	filters := managedDescribeFilters(l.Config.FleetID, states)
	filters = append(filters, types.Filter{
		Name:   aws.String("tag:" + TagKeyArch),
		Values: []string{arch},
	})
	return filters
}

// configArch normalises an arch value, defaulting to amd64.
func configArch(arch string) string {
	arch = strings.ToLower(strings.TrimSpace(arch))
	if arch == "" {
		return defaultArch
	}
	return arch
}

// normalizeArch validates and normalises an arch string from env or config.
func normalizeArch(raw string) (string, error) {
	arch := strings.ToLower(strings.TrimSpace(raw))
	if arch == "" {
		return defaultArch, nil
	}
	switch arch {
	case "amd64", "arm64":
		return arch, nil
	default:
		return "", fmt.Errorf("%s must be amd64 or arm64", envArch)
	}
}

// Env var constants and defaults used by ConfigFromEnv.
const (
	envAMI                 = "EC2_PROVISION_AMI_ID"
	envInstanceType        = "EC2_PROVISION_INSTANCE_TYPE"
	envArch                = "EC2_PROVISION_ARCH"
	envFleetID             = "EC2_PROVISION_FLEET_ID"
	envSubnet              = "EC2_PROVISION_SUBNET_ID"
	envSecurityGroups      = "EC2_PROVISION_SECURITY_GROUP_IDS"
	envRunnerS3URI         = "EC2_PROVISION_RUNNER_S3_URI"
	envTaskBrokerURL       = "EC2_PROVISION_TASK_BROKER_URL"
	envRunnerFleetID       = "EC2_PROVISION_RUNNER_FLEET_ID"
	envRunnersAuth         = "EC2_PROVISION_RUNNER_AUTH_TOKEN"
	envKeyName             = "EC2_PROVISION_KEY_NAME"
	envRunnerIAMProf       = "EC2_PROVISION_RUNNER_INSTANCE_PROFILE"
	envRunnerTerminateTask = "EC2_PROVISION_RUNNER_TERMINATE_AFTER_TASK"
	envHotCount            = "EC2_PROVISION_HOT_INSTANCE_COUNT"
	envRunnerCWGroup       = "EC2_PROVISION_RUNNER_CLOUDWATCH_LOG_GROUP"
	envRunnerCWPrefix      = "EC2_PROVISION_RUNNER_CLOUDWATCH_LOG_STREAM_PREFIX"
	envRunnerProcCWGroup   = "EC2_PROVISION_RUNNER_PROCESS_LOG_GROUP"
	envRunnerProcCWRegion  = "EC2_PROVISION_RUNNER_PROCESS_LOG_REGION"
	envVolumeSizeGB        = "EC2_PROVISION_VOLUME_SIZE_GB"
	envBootGraceSec        = "EC2_PROVISION_BOOT_GRACE_SEC"
	envRunnerHealthPort    = "EC2_PROVISION_RUNNER_HEALTH_PORT"
	envRunnerHeadroom      = "EC2_PROVISION_RUNNER_HEADROOM"

	defaultInstanceType     = "t3.micro"
	defaultArch             = "amd64"
	defaultVolumeSizeGB     = 30
	defaultBootGraceSec     = 300
	defaultRunnerHealthPort = 9090
)

// ErrDisabled means EC2 pool management is off (hot instance count env not set).
var ErrDisabled = errors.New("ec2 provisioning disabled: EC2_PROVISION_HOT_INSTANCE_COUNT is not set")

// ConfigFromEnv builds a Config from EC2_PROVISION_* environment variables.
// Retained for local/testing use; production uses the JSON config file.
func ConfigFromEnv() (Config, error) {
	hotRaw := strings.TrimSpace(os.Getenv(envHotCount))
	if hotRaw == "" {
		return Config{}, ErrDisabled
	}
	hot, err := strconv.Atoi(hotRaw)
	if err != nil || hot < 0 {
		return Config{}, fmt.Errorf("%s must be a non-negative integer", envHotCount)
	}
	ami := strings.TrimSpace(os.Getenv(envAMI))
	sub := strings.TrimSpace(os.Getenv(envSubnet))
	sgs := strings.TrimSpace(os.Getenv(envSecurityGroups))
	url := strings.TrimSpace(os.Getenv(envTaskBrokerURL))
	runnerFleetID := strings.TrimSpace(os.Getenv(envRunnerFleetID))
	if ami == "" || sub == "" || sgs == "" || url == "" || runnerFleetID == "" {
		return Config{}, fmt.Errorf("set %s, %s, %s, %s, and %s", envAMI, envSubnet, envSecurityGroups, envTaskBrokerURL, envRunnerFleetID)
	}
	var sgIDs []string
	for _, p := range strings.Split(sgs, ",") {
		if p = strings.TrimSpace(p); p != "" {
			sgIDs = append(sgIDs, p)
		}
	}
	if len(sgIDs) == 0 {
		return Config{}, fmt.Errorf("%s must list at least one security group id", envSecurityGroups)
	}
	region := strings.TrimSpace(os.Getenv("AWS_REGION"))
	if region == "" {
		region = strings.TrimSpace(os.Getenv("AWS_DEFAULT_REGION"))
	}
	if region == "" {
		return Config{}, fmt.Errorf("set AWS_REGION (or AWS_DEFAULT_REGION) for EC2 provisioning")
	}
	runnerS3 := strings.TrimSpace(os.Getenv(envRunnerS3URI))
	if strings.TrimSpace(os.Getenv("EC2_PROVISION_RUNNER_BINARY_URL")) != "" {
		return Config{}, fmt.Errorf("EC2_PROVISION_RUNNER_BINARY_URL is no longer supported; use %s instead", envRunnerS3URI)
	}
	if runnerS3 == "" {
		return Config{}, fmt.Errorf("set %s", envRunnerS3URI)
	}
	if err := ValidateRunnerS3URI(runnerS3); err != nil {
		return Config{}, err
	}
	prof := strings.TrimSpace(os.Getenv(envRunnerIAMProf))
	if prof == "" {
		return Config{}, fmt.Errorf("set %s (runners need IAM credentials to read S3)", envRunnerIAMProf)
	}
	itype := strings.TrimSpace(os.Getenv(envInstanceType))
	if itype == "" {
		itype = defaultInstanceType
	}
	arch, err := normalizeArch(os.Getenv(envArch))
	if err != nil {
		return Config{}, err
	}
	fleetID := strings.TrimSpace(os.Getenv(envFleetID))
	if fleetID == "" {
		hn, herr := os.Hostname()
		if herr != nil || strings.TrimSpace(hn) == "" {
			return Config{}, fmt.Errorf("set %s (could not derive from hostname: %v)", envFleetID, herr)
		}
		fleetID = strings.TrimSpace(hn)
	}
	terminateAfterTask := true
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envRunnerTerminateTask))) {
	case "0", "false", "no", "off":
		terminateAfterTask = false
	}
	volumeSizeGB := int32(defaultVolumeSizeGB)
	if v := strings.TrimSpace(os.Getenv(envVolumeSizeGB)); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return Config{}, fmt.Errorf("%s must be a positive integer (GiB)", envVolumeSizeGB)
		}
		volumeSizeGB = int32(n)
	}
	bootGrace := defaultBootGraceSec
	if v := strings.TrimSpace(os.Getenv(envBootGraceSec)); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return Config{}, fmt.Errorf("%s must be a non-negative integer", envBootGraceSec)
		}
		bootGrace = n
	}
	healthPort := defaultRunnerHealthPort
	if v := strings.TrimSpace(os.Getenv(envRunnerHealthPort)); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 65535 {
			return Config{}, fmt.Errorf("%s must be a valid TCP port", envRunnerHealthPort)
		}
		healthPort = n
	}
	headroom := 0
	if v := strings.TrimSpace(os.Getenv(envRunnerHeadroom)); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return Config{}, fmt.Errorf("%s must be a non-negative integer", envRunnerHeadroom)
		}
		headroom = n
	}
	return Config{
		AMI:                             ami,
		InstanceType:                    itype,
		Arch:                            arch,
		FleetID:                         fleetID,
		RunnerFleetID:                   runnerFleetID,
		SubnetID:                        sub,
		SecurityGroupIDs:                sgIDs,
		RunnerS3URI:                     runnerS3,
		RunnerInstallAWSRegion:          region,
		TaskBrokerURL:                   url,
		RunnersAuthToken:                strings.TrimSpace(os.Getenv(envRunnersAuth)),
		KeyName:                         strings.TrimSpace(os.Getenv(envKeyName)),
		RunnersIAMProfName:              prof,
		HotInstanceCount:                hot,
		RunnerTerminateAfterEachTask:    terminateAfterTask,
		RunnerCloudWatchLogGroup:        strings.TrimSpace(os.Getenv(envRunnerCWGroup)),
		RunnerCloudWatchLogStreamPrefix: strings.TrimSpace(os.Getenv(envRunnerCWPrefix)),
		RunnerProcessLogGroup:           strings.TrimSpace(os.Getenv(envRunnerProcCWGroup)),
		RunnerProcessLogRegion:          strings.TrimSpace(os.Getenv(envRunnerProcCWRegion)),
		VolumeSizeGB:                    volumeSizeGB,
		BootGraceSec:                    bootGrace,
		RunnerHealthPort:                healthPort,
		Headroom:                        headroom,
	}, nil
}
