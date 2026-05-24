// Package ec2provision maintains a pool of Ubuntu EC2 runners: cloud-init installs the statically linked
// runner binary from S3 and runs it under systemd so host-mode tasks run on Ubuntu.
//
// Enabled when EC2_PROVISION_HOT_INSTANCE_COUNT is set together with AMI, subnet, security groups,
// task-broker URL, and runner fleet id; see ConfigFromEnv and ErrDisabled.
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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Launcher calls EC2 RunInstances with a deterministic cloud-init user-data starter.
type Launcher struct {
	Client *ec2.Client
	Config Config
	Log    *slog.Logger
}

// Config is filled from EC2_PROVISION_* environment variables.
type Config struct {
	AMI              string
	InstanceType     string
	SubnetID         string
	SecurityGroupIDs []string
	// RunnerS3URI is s3://bucket/key (requires instance profile with s3:GetObject).
	RunnerS3URI string
	// RunnerInstallAWSRegion is AWS_DEFAULT_REGION in user-data for aws s3 cp (same region as fleet-manager / bucket).
	RunnerInstallAWSRegion string
	TaskBrokerURL          string // EC2_PROVISION_TASK_BROKER_URL for runners (reachable from VPC)
	RunnerFleetID          string // EC2_PROVISION_RUNNER_FLEET_ID registered on task-broker
	RunnersAuthToken       string // optional runner AUTH_TOKEN
	KeyName                string // optional EC2 key pair name
	RunnersIAMProfName     string // optional IAM instance profile name for runners
	HotInstanceCount       int    // target pending+running managed instances (from EC2_PROVISION_HOT_INSTANCE_COUNT)
	// RunnerTerminateAfterEachTask sets RUNNER_TERMINATE_AFTER_EACH_TASK; fleet-manager terminates the EC2 instance after one task.
	RunnerTerminateAfterEachTask bool
	// RunnerCloudWatchLogGroup sets RUNNER_CLOUDWATCH_LOG_GROUP in EC2 user-data (optional).
	RunnerCloudWatchLogGroup string
	// RunnerCloudWatchLogStreamPrefix sets RUNNER_CLOUDWATCH_LOG_STREAM_PREFIX (optional; must match TASK_CLOUDWATCH_LOG_STREAM_PREFIX on fleet-manager).
	RunnerCloudWatchLogStreamPrefix string
	// RunnerProcessLogGroup is the CloudWatch log group for runner process (systemd service) logs.
	// When set, the CloudWatch agent is installed and configured to ship journald output for
	// superplane-runner.service to this group under stream name <instance-id>/runner-process.
	RunnerProcessLogGroup string
	// RunnerProcessLogRegion is the AWS region for runner process logs (defaults to RunnerCloudWatchRegion or us-east-1).
	VolumeSizeGB           int32
	RunnerProcessLogRegion string
	// BootGraceSec skips health probes for newly launched instances.
	BootGraceSec int
	// RunnerHealthPort is the TCP port for GET /healthz on runner private IP (default 9090).
	RunnerHealthPort int
}

// ErrDisabled means EC2 pool management is off (hot instance count env not set).
var ErrDisabled = errors.New("ec2 provisioning disabled: EC2_PROVISION_HOT_INSTANCE_COUNT is not set")

const (
	envAMI                 = "EC2_PROVISION_AMI_ID"
	envInstanceType        = "EC2_PROVISION_INSTANCE_TYPE"
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

	defaultInstanceType     = "t3.micro"
	defaultVolumeSizeGB     = 30
	defaultBootGraceSec     = 300
	defaultRunnerHealthPort = 9090

	// TagKeyManaged is applied to fleet-manager-managed runner instances for Describe/Reconcile filtering.
	TagKeyManaged = "superplane_managed_runner"

	maxLaunch = 50
)

// ConfigFromEnv validates required provisioning settings.
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
	fleetID := strings.TrimSpace(os.Getenv(envRunnerFleetID))
	if ami == "" || sub == "" || sgs == "" || url == "" || fleetID == "" {
		return Config{}, fmt.Errorf("set %s, %s, %s, %s, and %s", envAMI, envSubnet, envSecurityGroups, envTaskBrokerURL, envRunnerFleetID)
	}
	var sgIDs []string
	for _, p := range strings.Split(sgs, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
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
	if runnerS3 == "" {
		return Config{}, fmt.Errorf("set %s", envRunnerS3URI)
	}
	if err := validateRunnerS3URI(runnerS3); err != nil {
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
	return Config{
		AMI:                             ami,
		InstanceType:                    itype,
		SubnetID:                        sub,
		SecurityGroupIDs:                sgIDs,
		RunnerS3URI:                     runnerS3,
		RunnerInstallAWSRegion:          region,
		TaskBrokerURL:                   url,
		RunnerFleetID:                   fleetID,
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
	}, nil
}

func validateRunnerS3URI(s string) error {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "s3://") {
		return fmt.Errorf("%s must start with s3://", envRunnerS3URI)
	}
	key := strings.TrimPrefix(s, "s3://")
	if key == "" || !strings.Contains(key, "/") {
		return fmt.Errorf("%s must be s3://bucket/key (object path required)", envRunnerS3URI)
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
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String("superplane-runner")},
					{Key: aws.String(TagKeyManaged), Value: aws.String("true")},
				},
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
