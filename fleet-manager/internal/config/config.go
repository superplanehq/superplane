// Package config loads the fleet-manager JSON configuration file.
//
// The schema has a flat "global" section (broker / network / IAM / CloudWatch / defaults)
// plus a pools[] array — one entry per VM pool the fleet-manager owns. Each pool maps 1:1
// to a broker fleet identified by FleetID; everything in the global section is shared
// across all pools in the process.
//
// Load reads the file, applies defaults, validates, and returns *File. Callers turn each
// pool into an ec2provision.Config via (*File).ToPoolConfig.
//
// No package outside fleet-manager imports this. The runner and task-broker keep their
// env-var loaders.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/superplane/runner/fleet-manager/internal/ec2provision"
)

// File is the on-disk shape of the fleet-manager config.
type File struct {
	// --- global infra / broker / HTTP ---
	AWSRegion     string `json:"aws_region"`
	TaskBrokerURL string `json:"task_broker_url"`
	// TaskBrokerAuthToken is the shared secret used to authenticate against the task-broker.
	// FM uses it for task-counts polling; FM also injects it into runner user-data so runner
	// VMs use the same value when claiming tasks. Distinct from AuthToken below (which is
	// THIS fleet-manager's inbound /v1/* bearer).
	TaskBrokerAuthToken  string `json:"task_broker_auth_token"`
	ListenAddr           string `json:"listen_addr"`
	AuthToken            string `json:"auth_token"`
	DiagnosticsToken     string `json:"diagnostics_token"`
	ReconcileIntervalSec int    `json:"reconcile_interval_sec"`

	// --- global VM defaults (shared across pools for now; per-pool override is a future move) ---
	SubnetID                     string     `json:"subnet_id"`
	SecurityGroupIDs             []string   `json:"security_group_ids"`
	IAMInstanceProfile           string     `json:"iam_instance_profile"`
	KeyName                      string     `json:"key_name"`
	VolumeSizeGB                 int32      `json:"volume_size_gb"`
	BootGraceSec                 int        `json:"boot_grace_sec"`
	RunnerHealthPort             int        `json:"runner_health_port"`
	RunnerTerminateAfterEachTask *bool      `json:"runner_terminate_after_each_task,omitempty"`
	CloudWatch                   CloudWatch `json:"cloudwatch"`

	// --- pools ---
	Pools []Pool `json:"pools"`
}

// CloudWatch holds optional log-shipping settings forwarded into runner user-data.
type CloudWatch struct {
	LogGroup         string `json:"log_group"`
	StreamPrefix     string `json:"stream_prefix"`
	ProcessLogGroup  string `json:"process_log_group"`
	ProcessLogRegion string `json:"process_log_region"`
}

// Pool is one VM pool managed by the fleet-manager. Identified by FleetID, which is also
// the broker fleet primary key and the value of the superplane_fleet_id EC2 tag.
type Pool struct {
	FleetID      string `json:"fleet_id"`
	AMI          string `json:"ami"`
	InstanceType string `json:"instance_type"`
	// Arch is the CPU architecture for this pool: "amd64" (default) or "arm64" (Graviton).
	// Controls the AWS CLI and CloudWatch agent download URLs in runner user-data.
	Arch             string `json:"arch,omitempty"`
	RunnerS3URI      string `json:"runner_s3_uri"`
	HotInstanceCount int    `json:"hot_instance_count"`
	Headroom         int    `json:"headroom"`
}

// Defaults applied when fields are absent or zero.
const (
	defaultInstanceType         = "t3.micro"
	defaultListenAddr           = ":8080"
	defaultReconcileIntervalSec = 60
	minReconcileIntervalSec     = 15
	defaultVolumeSizeGB         = 30
	defaultBootGraceSec         = 300
	defaultRunnerHealthPort     = 9090
)

// Load reads path, parses JSON, applies defaults, and validates. Returns an error
// listing the first problem encountered (we do not aggregate; the operator fixes one,
// reruns, and sees the next).
func Load(path string) (*File, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("config path is empty")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var f File
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	f.applyDefaults()
	if err := f.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &f, nil
}

func (f *File) applyDefaults() {
	if strings.TrimSpace(f.ListenAddr) == "" {
		f.ListenAddr = defaultListenAddr
	}
	if f.ReconcileIntervalSec == 0 {
		f.ReconcileIntervalSec = defaultReconcileIntervalSec
	}
	if f.VolumeSizeGB == 0 {
		f.VolumeSizeGB = defaultVolumeSizeGB
	}
	if f.BootGraceSec == 0 {
		f.BootGraceSec = defaultBootGraceSec
	}
	if f.RunnerHealthPort == 0 {
		f.RunnerHealthPort = defaultRunnerHealthPort
	}
	if f.RunnerTerminateAfterEachTask == nil {
		t := true
		f.RunnerTerminateAfterEachTask = &t
	}
	for i := range f.Pools {
		if strings.TrimSpace(f.Pools[i].InstanceType) == "" {
			f.Pools[i].InstanceType = defaultInstanceType
		}
		if strings.TrimSpace(f.Pools[i].Arch) == "" {
			f.Pools[i].Arch = "amd64"
		}
	}
}

func (f *File) validate() error {
	if strings.TrimSpace(f.AWSRegion) == "" {
		return fmt.Errorf("aws_region is required")
	}
	if strings.TrimSpace(f.TaskBrokerURL) == "" {
		return fmt.Errorf("task_broker_url is required")
	}
	if strings.TrimSpace(f.SubnetID) == "" {
		return fmt.Errorf("subnet_id is required")
	}
	if strings.TrimSpace(f.IAMInstanceProfile) == "" {
		return fmt.Errorf("iam_instance_profile is required (runners need IAM credentials to read S3)")
	}
	cleanedSGs := make([]string, 0, len(f.SecurityGroupIDs))
	for _, sg := range f.SecurityGroupIDs {
		if s := strings.TrimSpace(sg); s != "" {
			cleanedSGs = append(cleanedSGs, s)
		}
	}
	if len(cleanedSGs) == 0 {
		return fmt.Errorf("security_group_ids must list at least one security group id")
	}
	f.SecurityGroupIDs = cleanedSGs

	if f.VolumeSizeGB < 1 {
		return fmt.Errorf("volume_size_gb must be a positive integer (GiB)")
	}
	if f.BootGraceSec < 0 {
		return fmt.Errorf("boot_grace_sec must be a non-negative integer")
	}
	if f.RunnerHealthPort < 1 || f.RunnerHealthPort > 65535 {
		return fmt.Errorf("runner_health_port must be a valid TCP port")
	}
	if f.ReconcileIntervalSec < minReconcileIntervalSec {
		return fmt.Errorf("reconcile_interval_sec must be >= %d", minReconcileIntervalSec)
	}

	if len(f.Pools) == 0 {
		return fmt.Errorf("pools[] must contain at least one entry")
	}
	seen := make(map[string]struct{}, len(f.Pools))
	for i := range f.Pools {
		p := &f.Pools[i]
		p.FleetID = strings.TrimSpace(p.FleetID)
		p.AMI = strings.TrimSpace(p.AMI)
		p.RunnerS3URI = strings.TrimSpace(p.RunnerS3URI)
		if p.FleetID == "" {
			return fmt.Errorf("pools[%d].fleet_id is required", i)
		}
		if _, dup := seen[p.FleetID]; dup {
			return fmt.Errorf("pools[%d].fleet_id %q is duplicated; fleet_ids must be unique", i, p.FleetID)
		}
		seen[p.FleetID] = struct{}{}
		if p.AMI == "" {
			return fmt.Errorf("pools[%d].ami is required", i)
		}
		if p.RunnerS3URI == "" {
			return fmt.Errorf("pools[%d].runner_s3_uri is required", i)
		}
		if err := ec2provision.ValidateRunnerS3URI(p.RunnerS3URI); err != nil {
			return fmt.Errorf("pools[%d].runner_s3_uri: %w", i, err)
		}
		if p.HotInstanceCount < 0 {
			return fmt.Errorf("pools[%d].hot_instance_count must be non-negative", i)
		}
		if p.Headroom < 0 {
			return fmt.Errorf("pools[%d].headroom must be non-negative", i)
		}
	}
	return nil
}

// ToPoolConfig produces the ec2provision.Config for one pool by merging globals into it.
// Field-by-field: pool-scoped values come from p, everything else from the global section.
func (f *File) ToPoolConfig(p Pool) ec2provision.Config {
	terminate := true
	if f.RunnerTerminateAfterEachTask != nil {
		terminate = *f.RunnerTerminateAfterEachTask
	}
	return ec2provision.Config{
		AMI:                             p.AMI,
		InstanceType:                    p.InstanceType,
		Arch:                            p.Arch,
		FleetID:                         p.FleetID,
		SubnetID:                        f.SubnetID,
		SecurityGroupIDs:                f.SecurityGroupIDs,
		RunnerS3URI:                     p.RunnerS3URI,
		RunnerInstallAWSRegion:          f.AWSRegion,
		TaskBrokerURL:                   f.TaskBrokerURL,
		RunnerFleetID:                   p.FleetID,
		RunnersAuthToken:                f.TaskBrokerAuthToken,
		KeyName:                         f.KeyName,
		RunnersIAMProfName:              f.IAMInstanceProfile,
		HotInstanceCount:                p.HotInstanceCount,
		Headroom:                        p.Headroom,
		RunnerTerminateAfterEachTask:    terminate,
		RunnerCloudWatchLogGroup:        f.CloudWatch.LogGroup,
		RunnerCloudWatchLogStreamPrefix: f.CloudWatch.StreamPrefix,
		RunnerProcessLogGroup:           f.CloudWatch.ProcessLogGroup,
		RunnerProcessLogRegion:          f.CloudWatch.ProcessLogRegion,
		VolumeSizeGB:                    f.VolumeSizeGB,
		BootGraceSec:                    f.BootGraceSec,
		RunnerHealthPort:                f.RunnerHealthPort,
	}
}
