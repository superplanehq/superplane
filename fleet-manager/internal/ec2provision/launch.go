// Package ec2provision maintains a pool of Ubuntu EC2 runners (user-data starts the Docker agent).
//
// Enabled when EC2_PROVISION_HOT_INSTANCE_COUNT is set together with AMI, subnet, security groups,
// and fleet-manager URL; see ConfigFromEnv and ErrDisabled.
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
	AMI                string
	InstanceType       string
	SubnetID           string
	SecurityGroupIDs   []string
	RunnerImage        string
	FleetManagerURL    string // FLEET_MANAGER_URL for new runner containers (reachable from VPC)
	RunnersAuthToken   string // optional, passed into runner AUTH_TOKEN (-e)
	KeyName            string // optional EC2 key pair name
	RunnersIAMProfName string // optional IAM instance profile name for runners
	HotInstanceCount   int    // target pending+running managed instances (from EC2_PROVISION_HOT_INSTANCE_COUNT)
	// RunnerTerminateAfterEachTask sets RUNNER_TERMINATE_AFTER_EACH_TASK on new runner containers (EC2 terminate after one task).
	RunnerTerminateAfterEachTask bool
}

// ErrDisabled means EC2 pool management is off (hot instance count env not set).
var ErrDisabled = errors.New("ec2 provisioning disabled: EC2_PROVISION_HOT_INSTANCE_COUNT is not set")

const (
	envAMI             = "EC2_PROVISION_AMI_ID"
	envInstanceType    = "EC2_PROVISION_INSTANCE_TYPE"
	envSubnet          = "EC2_PROVISION_SUBNET_ID"
	envSecurityGroups  = "EC2_PROVISION_SECURITY_GROUP_IDS"
	envRunnerImage     = "EC2_PROVISION_RUNNER_IMAGE"
	envFleetManagerURL = "EC2_PROVISION_FLEET_MANAGER_URL"
	envRunnersAuth     = "EC2_PROVISION_RUNNER_AUTH_TOKEN"
	envKeyName         = "EC2_PROVISION_KEY_NAME"
	envRunnerIAMProf       = "EC2_PROVISION_RUNNER_INSTANCE_PROFILE"
	envRunnerTerminateTask = "EC2_PROVISION_RUNNER_TERMINATE_AFTER_TASK"
	envHotCount            = "EC2_PROVISION_HOT_INSTANCE_COUNT"

	defaultInstanceType = "t3.micro"
	defaultRunnerImage  = "ghcr.io/superplanehq/runner/runner:latest"

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
	url := strings.TrimSpace(os.Getenv(envFleetManagerURL))
	if ami == "" || sub == "" || sgs == "" || url == "" {
		return Config{}, fmt.Errorf("set %s, %s, %s, and %s", envAMI, envSubnet, envSecurityGroups, envFleetManagerURL)
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
	itype := strings.TrimSpace(os.Getenv(envInstanceType))
	if itype == "" {
		itype = defaultInstanceType
	}
	rimg := strings.TrimSpace(os.Getenv(envRunnerImage))
	if rimg == "" {
		rimg = defaultRunnerImage
	}
	terminateAfterTask := true
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envRunnerTerminateTask))) {
	case "0", "false", "no", "off":
		terminateAfterTask = false
	}
	return Config{
		AMI:                ami,
		InstanceType:       itype,
		SubnetID:           sub,
		SecurityGroupIDs:   sgIDs,
		RunnerImage:        rimg,
		FleetManagerURL:    url,
		RunnersAuthToken:   strings.TrimSpace(os.Getenv(envRunnersAuth)),
		KeyName:            strings.TrimSpace(os.Getenv(envKeyName)),
		RunnersIAMProfName: strings.TrimSpace(os.Getenv(envRunnerIAMProf)),
		HotInstanceCount:   hot,
		RunnerTerminateAfterEachTask: terminateAfterTask,
	}, nil
}

// New returns an EC2 client + config (uses AWS default credential chain + region).
func New(ctx context.Context, cfg Config, log *slog.Logger) (*Launcher, error) {
	region := strings.TrimSpace(os.Getenv("AWS_REGION"))
	if region == "" {
		region = strings.TrimSpace(os.Getenv("AWS_DEFAULT_REGION"))
	}
	if region == "" {
		return nil, fmt.Errorf("set AWS_REGION (or AWS_DEFAULT_REGION)")
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

// Launch creates `count` on-demand Ubuntu hosts that pull the runner image and connect to FleetManagerURL.
func (l *Launcher) Launch(ctx context.Context, count int) ([]string, error) {
	if count < 1 {
		return nil, fmt.Errorf("count must be at least 1")
	}
	if count > maxLaunch {
		return nil, fmt.Errorf("count exceeds maximum of %d", maxLaunch)
	}
	n := int32(count)
	userdata := userDataScript(l.Config)
	in := &ec2.RunInstancesInput{
		ImageId:          aws.String(l.Config.AMI),
		InstanceType:     types.InstanceType(l.Config.InstanceType),
		MinCount:         aws.Int32(n),
		MaxCount:         aws.Int32(n),
		UserData:         aws.String(base64.StdEncoding.EncodeToString([]byte(userdata))),
		SubnetId:         aws.String(l.Config.SubnetID),
		SecurityGroupIds: l.Config.SecurityGroupIDs,
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

func userDataScript(c Config) string {
	var b strings.Builder
	b.WriteString("#!/bin/bash\n")
	b.WriteString("set -euxo pipefail\n")
	b.WriteString("export DEBIAN_FRONTEND=noninteractive\n")
	b.WriteString("# Instance id for fleet-manager to terminate after each task\n")
	b.WriteString("IMDS=http://169.254.169.254\n")
	b.WriteString("TOKEN=$(curl -sf --max-time 3 \"$IMDS/latest/api/token\" -X PUT -H \"X-aws-ec2-metadata-token-ttl-seconds: 21600\" || true)\n")
	b.WriteString("IID=\"\"\n")
	b.WriteString("if [ -n \"$TOKEN\" ]; then IID=$(curl -sf --max-time 3 -H \"X-aws-ec2-metadata-token: $TOKEN\" \"$IMDS/latest/meta-data/instance-id\") || IID=\"\"; fi\n")
	b.WriteString("if [ -z \"$IID\" ]; then IID=$(curl -sf --max-time 3 \"$IMDS/latest/meta-data/instance-id\") || IID=\"unknown-host\"; fi\n")
	b.WriteString("apt-get update -qy\n")
	b.WriteString("apt-get install -qy docker.io\n")
	b.WriteString("systemctl enable --now docker\n")
	fmt.Fprintf(&b, "docker pull %s\n", strconv.Quote(c.RunnerImage))
	b.WriteString("docker rm -f superplane-runner 2>/dev/null || true\n")
	restart := "unless-stopped"
	if c.RunnerTerminateAfterEachTask {
		// Disposable worker: exits after RUNNER_TERMINATE_AFTER_EACH_TASK; avoid restart loops before fleet-manager terminates the VM.
		restart = "no"
	}
	fmt.Fprintf(&b, "docker run -d --name superplane-runner --restart %s \\\n", restart)
	fmt.Fprintf(&b, "  -e FLEET_MANAGER_URL=%s \\\n", strconv.Quote(c.FleetManagerURL))
	if c.RunnersAuthToken != "" {
		fmt.Fprintf(&b, "  -e AUTH_TOKEN=%s \\\n", strconv.Quote(c.RunnersAuthToken))
	}
	b.WriteString("  -e RUNNER_ID=\"$IID\" \\\n")
	if c.RunnerTerminateAfterEachTask {
		b.WriteString("  -e RUNNER_TERMINATE_AFTER_EACH_TASK=true \\\n")
	}
	fmt.Fprintf(&b, "  %s\n", strconv.Quote(c.RunnerImage))
	return b.String()
}

