// Package ec2provision maintains a pool of Ubuntu EC2 runners: cloud-init installs the statically linked
// runner binary (S3 via aws s3 cp, or HTTPS curl) and runs it under systemd so host-mode tasks run on Ubuntu.
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
	AMI              string
	InstanceType     string
	SubnetID         string
	SecurityGroupIDs []string
	// RunnerS3URI installs from s3://bucket/key (requires instance profile with s3:GetObject); mutually exclusive with RunnerBinaryURL.
	RunnerS3URI string
	// RunnerBinaryURL is an http(s) URL to curl when RunnerS3URI is empty.
	RunnerBinaryURL string
	// RunnerInstallAWSRegion is AWS_DEFAULT_REGION in user-data for aws s3 cp (same region as fleet-manager / bucket).
	RunnerInstallAWSRegion string
	FleetManagerURL        string // FLEET_MANAGER_URL for runners (reachable from VPC)
	RunnersAuthToken       string // optional runner AUTH_TOKEN
	KeyName                string // optional EC2 key pair name
	RunnersIAMProfName     string // optional IAM instance profile name for runners
	HotInstanceCount       int    // target pending+running managed instances (from EC2_PROVISION_HOT_INSTANCE_COUNT)
	// RunnerTerminateAfterEachTask sets RUNNER_TERMINATE_AFTER_EACH_TASK; fleet-manager terminates the EC2 instance after one task.
	RunnerTerminateAfterEachTask bool
}

// ErrDisabled means EC2 pool management is off (hot instance count env not set).
var ErrDisabled = errors.New("ec2 provisioning disabled: EC2_PROVISION_HOT_INSTANCE_COUNT is not set")

const (
	envAMI                 = "EC2_PROVISION_AMI_ID"
	envInstanceType        = "EC2_PROVISION_INSTANCE_TYPE"
	envSubnet              = "EC2_PROVISION_SUBNET_ID"
	envSecurityGroups      = "EC2_PROVISION_SECURITY_GROUP_IDS"
	envRunnerS3URI         = "EC2_PROVISION_RUNNER_S3_URI"
	envRunnerBinaryURL     = "EC2_PROVISION_RUNNER_BINARY_URL"
	envFleetManagerURL     = "EC2_PROVISION_FLEET_MANAGER_URL"
	envRunnersAuth         = "EC2_PROVISION_RUNNER_AUTH_TOKEN"
	envKeyName             = "EC2_PROVISION_KEY_NAME"
	envRunnerIAMProf       = "EC2_PROVISION_RUNNER_INSTANCE_PROFILE"
	envRunnerTerminateTask = "EC2_PROVISION_RUNNER_TERMINATE_AFTER_TASK"
	envHotCount            = "EC2_PROVISION_HOT_INSTANCE_COUNT"

	defaultInstanceType = "t3.micro"

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
	region := strings.TrimSpace(os.Getenv("AWS_REGION"))
	if region == "" {
		region = strings.TrimSpace(os.Getenv("AWS_DEFAULT_REGION"))
	}
	if region == "" {
		return Config{}, fmt.Errorf("set AWS_REGION (or AWS_DEFAULT_REGION) for EC2 provisioning")
	}
	runnerS3 := strings.TrimSpace(os.Getenv(envRunnerS3URI))
	runnerBinURL := strings.TrimSpace(os.Getenv(envRunnerBinaryURL))
	switch {
	case runnerS3 != "" && runnerBinURL != "":
		return Config{}, fmt.Errorf("set only one of %s or %s", envRunnerS3URI, envRunnerBinaryURL)
	case runnerS3 != "":
		if err := validateRunnerS3URI(runnerS3); err != nil {
			return Config{}, err
		}
		prof := strings.TrimSpace(os.Getenv(envRunnerIAMProf))
		if prof == "" {
			return Config{}, fmt.Errorf("set %s when using %s (runners need IAM credentials to read S3)", envRunnerIAMProf, envRunnerS3URI)
		}
	case runnerBinURL != "":
		if !strings.HasPrefix(runnerBinURL, "http://") && !strings.HasPrefix(runnerBinURL, "https://") {
			return Config{}, fmt.Errorf("%s must start with http:// or https://", envRunnerBinaryURL)
		}
	default:
		return Config{}, fmt.Errorf("set %s (recommended for private builds) or %s (public http(s) URL)", envRunnerS3URI, envRunnerBinaryURL)
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
	return Config{
		AMI:                          ami,
		InstanceType:                 itype,
		SubnetID:                     sub,
		SecurityGroupIDs:             sgIDs,
		RunnerS3URI:                  runnerS3,
		RunnerBinaryURL:              runnerBinURL,
		RunnerInstallAWSRegion:       region,
		FleetManagerURL:              url,
		RunnersAuthToken:             strings.TrimSpace(os.Getenv(envRunnersAuth)),
		KeyName:                      strings.TrimSpace(os.Getenv(envKeyName)),
		RunnersIAMProfName:           strings.TrimSpace(os.Getenv(envRunnerIAMProf)),
		HotInstanceCount:             hot,
		RunnerTerminateAfterEachTask: terminateAfterTask,
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

// Launch creates `count` on-demand Ubuntu hosts that install the runner from S3 or RunnerBinaryURL and connect to FleetManagerURL.
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
	if strings.TrimSpace(c.RunnerS3URI) != "" {
		b.WriteString("apt-get install -qy ca-certificates curl docker.io awscli\n")
	} else {
		b.WriteString("apt-get install -qy ca-certificates curl docker.io\n")
	}
	b.WriteString("systemctl enable --now docker\n")
	if strings.TrimSpace(c.RunnerS3URI) != "" {
		fmt.Fprintf(&b, "export AWS_DEFAULT_REGION=%s\n", strconv.Quote(strings.TrimSpace(c.RunnerInstallAWSRegion)))
		fmt.Fprintf(&b, "RUNNER_S3=%s\n", strconv.Quote(strings.TrimSpace(c.RunnerS3URI)))
		b.WriteString("aws s3 cp \"$RUNNER_S3\" /usr/local/bin/runner\n")
	} else {
		fmt.Fprintf(&b, "BINURL=%s\n", strconv.Quote(strings.TrimSpace(c.RunnerBinaryURL)))
		b.WriteString("curl -fsSL \"$BINURL\" -o /usr/local/bin/runner\n")
	}
	b.WriteString("chmod 755 /usr/local/bin/runner\n")
	fmt.Fprintf(&b, "FMTMP=%s\n", strconv.Quote(c.FleetManagerURL))
	fmt.Fprintf(&b, "RUNTOK=%s\n", strconv.Quote(c.RunnersAuthToken))
	b.WriteString("umask 022\n")
	b.WriteString("{\n")
	b.WriteString("  printf 'FLEET_MANAGER_URL=%s\\n' \"$FMTMP\"\n")
	b.WriteString("  printf 'RUNNER_ID=%s\\n' \"$IID\"\n")
	b.WriteString("  if [ -n \"$RUNTOK\" ]; then printf 'AUTH_TOKEN=%s\\n' \"$RUNTOK\"; fi\n")
	if c.RunnerTerminateAfterEachTask {
		b.WriteString("  printf 'RUNNER_TERMINATE_AFTER_EACH_TASK=true\\n'\n")
	}
	b.WriteString("} > /etc/default/superplane-runner\n")
	b.WriteString("chmod 644 /etc/default/superplane-runner\n")
	restartPolicy := "always"
	if c.RunnerTerminateAfterEachTask {
		restartPolicy = "no"
	}
	b.WriteString("cat > /etc/systemd/system/superplane-runner.service <<'UNITEOF'\n")
	b.WriteString("[Unit]\n")
	b.WriteString("Description=Superplane runner (host process; host-mode tasks run on Ubuntu)\n")
	b.WriteString("After=network-online.target docker.service\n")
	b.WriteString("Wants=docker.service\n")
	b.WriteString("\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	b.WriteString("EnvironmentFile=/etc/default/superplane-runner\n")
	fmt.Fprintf(&b, "Restart=%s\n", restartPolicy)
	b.WriteString("ExecStart=/usr/local/bin/runner\n")
	b.WriteString("\n")
	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=multi-user.target\n")
	b.WriteString("UNITEOF\n")
	b.WriteString("systemctl daemon-reload\n")
	b.WriteString("systemctl enable --now superplane-runner.service\n")
	return b.String()
}
