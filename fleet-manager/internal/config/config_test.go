package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validConfigJSON returns a JSON string representing a fully valid two-pool
// fleet-manager config. Negative tests start from this and mutate one thing.
func validConfigJSON() string {
	return `{
		"aws_region": "us-east-1",
		"task_broker_url": "http://broker.internal:8081",
		"runner_auth_token": "broker-bearer-token",
		"listen_addr": ":8080",
		"auth_token": "",
		"diagnostics_token": "fleet-admin-token",
		"reconcile_interval_sec": 60,
		"subnet_id": "subnet-abc",
		"security_group_ids": ["sg-abc"],
		"iam_instance_profile": "superplane-runner",
		"key_name": "",
		"volume_size_gb": 30,
		"boot_grace_sec": 300,
		"runner_health_port": 9090,
		"runner_terminate_after_each_task": true,
		"cloudwatch": {
			"log_group": "/superplane/runner",
			"stream_prefix": "runner-",
			"process_log_group": "/superplane/runner-process",
			"process_log_region": ""
		},
		"pools": [
			{
				"fleet_id": "aws-amd64",
				"ami": "ami-amd64-aaaa",
				"instance_type": "t3.micro",
				"runner_s3_uri": "s3://superplane-artifacts/runner-linux-amd64",
				"hot_instance_count": 3,
				"headroom": 2
			},
			{
				"fleet_id": "aws-arm64",
				"ami": "ami-arm64-bbbb",
				"instance_type": "t4g.micro",
				"runner_s3_uri": "s3://superplane-artifacts/runner-linux-arm64",
				"hot_instance_count": 2,
				"headroom": 1
			}
		]
	}`
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

func TestLoad_HappyPath(t *testing.T) {
	f, err := Load(writeConfig(t, validConfigJSON()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if f.AWSRegion != "us-east-1" {
		t.Errorf("AWSRegion = %q", f.AWSRegion)
	}
	if len(f.Pools) != 2 {
		t.Fatalf("expected 2 pools, got %d", len(f.Pools))
	}
	if f.Pools[0].FleetID != "aws-amd64" || f.Pools[1].FleetID != "aws-arm64" {
		t.Errorf("pool fleet ids = %q, %q", f.Pools[0].FleetID, f.Pools[1].FleetID)
	}
}

func TestLoad_EmptyPathErrors(t *testing.T) {
	if _, err := Load(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestLoad_MissingFileErrors(t *testing.T) {
	if _, err := Load("/nonexistent/path/config.json"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_MalformedJSONErrors(t *testing.T) {
	_, err := Load(writeConfig(t, `{ not json`))
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should mention parse: %v", err)
	}
}

func TestLoad_UnknownFieldRejected(t *testing.T) {
	body := strings.Replace(validConfigJSON(), `"aws_region": "us-east-1",`, `"aws_region": "us-east-1", "made_up_field": 42,`, 1)
	if _, err := Load(writeConfig(t, body)); err == nil {
		t.Fatal("expected error on unknown field")
	}
}

func TestLoad_RequiredGlobalsMissing(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(s string) string
		wantSub string
	}{
		{"aws_region", func(s string) string {
			return strings.Replace(s, `"aws_region": "us-east-1",`, `"aws_region": "",`, 1)
		}, "aws_region"},
		{"task_broker_url", func(s string) string {
			return strings.Replace(s, `"task_broker_url": "http://broker.internal:8081",`, `"task_broker_url": "",`, 1)
		}, "task_broker_url"},
		{"subnet_id", func(s string) string {
			return strings.Replace(s, `"subnet_id": "subnet-abc",`, `"subnet_id": "",`, 1)
		}, "subnet_id"},
		{"iam_instance_profile", func(s string) string {
			return strings.Replace(s, `"iam_instance_profile": "superplane-runner",`, `"iam_instance_profile": "",`, 1)
		}, "iam_instance_profile"},
		{"security_group_ids", func(s string) string {
			return strings.Replace(s, `"security_group_ids": ["sg-abc"],`, `"security_group_ids": [],`, 1)
		}, "security_group_ids"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(writeConfig(t, tc.mutate(validConfigJSON())))
			if err == nil {
				t.Fatalf("expected error mentioning %q", tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error %v should mention %q", err, tc.wantSub)
			}
		})
	}
}

func TestLoad_PoolValidation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(s string) string
		wantSub string
	}{
		{"no_pools", func(s string) string {
			return strings.Replace(s,
				`"pools": [
			{
				"fleet_id": "aws-amd64",
				"ami": "ami-amd64-aaaa",
				"instance_type": "t3.micro",
				"runner_s3_uri": "s3://superplane-artifacts/runner-linux-amd64",
				"hot_instance_count": 3,
				"headroom": 2
			},
			{
				"fleet_id": "aws-arm64",
				"ami": "ami-arm64-bbbb",
				"instance_type": "t4g.micro",
				"runner_s3_uri": "s3://superplane-artifacts/runner-linux-arm64",
				"hot_instance_count": 2,
				"headroom": 1
			}
		]`, `"pools": []`, 1)
		}, "pools[]"},
		{"missing_fleet_id", func(s string) string {
			return strings.Replace(s, `"fleet_id": "aws-amd64",`, `"fleet_id": "",`, 1)
		}, "fleet_id"},
		{"missing_ami", func(s string) string {
			return strings.Replace(s, `"ami": "ami-amd64-aaaa",`, `"ami": "",`, 1)
		}, "ami"},
		{"missing_runner_s3_uri", func(s string) string {
			return strings.Replace(s, `"runner_s3_uri": "s3://superplane-artifacts/runner-linux-amd64",`, `"runner_s3_uri": "",`, 1)
		}, "runner_s3_uri"},
		{"invalid_runner_s3_uri_scheme", func(s string) string {
			return strings.Replace(s, `"runner_s3_uri": "s3://superplane-artifacts/runner-linux-amd64",`, `"runner_s3_uri": "https://example.com/runner",`, 1)
		}, "s3://"},
		{"invalid_runner_s3_uri_no_key", func(s string) string {
			return strings.Replace(s, `"runner_s3_uri": "s3://superplane-artifacts/runner-linux-amd64",`, `"runner_s3_uri": "s3://just-bucket",`, 1)
		}, "bucket/key"},
		{"duplicate_fleet_id", func(s string) string {
			// Two pools with the same fleet_id.
			return strings.Replace(s, `"fleet_id": "aws-arm64",`, `"fleet_id": "aws-amd64",`, 1)
		}, "duplicated"},
		{"negative_hot_instance_count", func(s string) string {
			return strings.Replace(s, `"hot_instance_count": 3,`, `"hot_instance_count": -1,`, 1)
		}, "hot_instance_count"},
		{"negative_headroom", func(s string) string {
			return strings.Replace(s, `"headroom": 2`, `"headroom": -2`, 1)
		}, "headroom"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(writeConfig(t, tc.mutate(validConfigJSON())))
			if err == nil {
				t.Fatalf("expected error mentioning %q", tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error %v should mention %q", err, tc.wantSub)
			}
		})
	}
}

func TestLoad_GlobalBoundsValidation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(s string) string
		wantSub string
	}{
		{"volume_size_gb_zero_uses_default", nil, ""}, // tested elsewhere
		{"boot_grace_sec_negative", func(s string) string {
			return strings.Replace(s, `"boot_grace_sec": 300,`, `"boot_grace_sec": -1,`, 1)
		}, "boot_grace_sec"},
		{"runner_health_port_zero_uses_default", nil, ""}, // tested elsewhere
		{"runner_health_port_too_high", func(s string) string {
			return strings.Replace(s, `"runner_health_port": 9090,`, `"runner_health_port": 70000,`, 1)
		}, "runner_health_port"},
		{"reconcile_interval_below_min", func(s string) string {
			return strings.Replace(s, `"reconcile_interval_sec": 60,`, `"reconcile_interval_sec": 5,`, 1)
		}, "reconcile_interval_sec"},
	}
	for _, tc := range cases {
		if tc.mutate == nil {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(writeConfig(t, tc.mutate(validConfigJSON())))
			if err == nil {
				t.Fatalf("expected error mentioning %q", tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error %v should mention %q", err, tc.wantSub)
			}
		})
	}
}

func TestLoad_DefaultsApplied(t *testing.T) {
	// Strip every defaultable field; loader should re-fill them.
	minimal := `{
		"aws_region": "us-east-1",
		"task_broker_url": "http://b:1",
		"subnet_id": "subnet-x",
		"security_group_ids": ["sg-x"],
		"iam_instance_profile": "p",
		"pools": [
			{
				"fleet_id": "only",
				"ami": "ami-1",
				"runner_s3_uri": "s3://bkt/key",
				"hot_instance_count": 1
			}
		]
	}`
	f, err := Load(writeConfig(t, minimal))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if f.ListenAddr != defaultListenAddr {
		t.Errorf("ListenAddr default = %q, want %q", f.ListenAddr, defaultListenAddr)
	}
	if f.ReconcileIntervalSec != defaultReconcileIntervalSec {
		t.Errorf("ReconcileIntervalSec default = %d, want %d", f.ReconcileIntervalSec, defaultReconcileIntervalSec)
	}
	if f.VolumeSizeGB != defaultVolumeSizeGB {
		t.Errorf("VolumeSizeGB default = %d, want %d", f.VolumeSizeGB, defaultVolumeSizeGB)
	}
	if f.BootGraceSec != defaultBootGraceSec {
		t.Errorf("BootGraceSec default = %d, want %d", f.BootGraceSec, defaultBootGraceSec)
	}
	if f.RunnerHealthPort != defaultRunnerHealthPort {
		t.Errorf("RunnerHealthPort default = %d, want %d", f.RunnerHealthPort, defaultRunnerHealthPort)
	}
	if f.Pools[0].InstanceType != defaultInstanceType {
		t.Errorf("pool InstanceType default = %q, want %q", f.Pools[0].InstanceType, defaultInstanceType)
	}
	if f.RunnerTerminateAfterEachTask == nil || *f.RunnerTerminateAfterEachTask != true {
		t.Errorf("RunnerTerminateAfterEachTask default should be true, got %v", f.RunnerTerminateAfterEachTask)
	}
}

func TestLoad_TerminateAfterEachTaskFalseHonored(t *testing.T) {
	// Explicit false must NOT be overridden by the "default to true" rule.
	body := strings.Replace(validConfigJSON(),
		`"runner_terminate_after_each_task": true,`,
		`"runner_terminate_after_each_task": false,`, 1)
	f, err := Load(writeConfig(t, body))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if f.RunnerTerminateAfterEachTask == nil || *f.RunnerTerminateAfterEachTask != false {
		t.Errorf("explicit false should be honored, got %v", f.RunnerTerminateAfterEachTask)
	}
}

func TestToPoolConfig_MergesGlobalsIntoPerPool(t *testing.T) {
	f, err := Load(writeConfig(t, validConfigJSON()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	cfg := f.ToPoolConfig(f.Pools[1]) // arm64

	if cfg.AMI != "ami-arm64-bbbb" {
		t.Errorf("AMI = %q", cfg.AMI)
	}
	if cfg.InstanceType != "t4g.micro" {
		t.Errorf("InstanceType = %q", cfg.InstanceType)
	}
	if cfg.RunnerS3URI != "s3://superplane-artifacts/runner-linux-arm64" {
		t.Errorf("RunnerS3URI = %q", cfg.RunnerS3URI)
	}
	if cfg.RunnerFleetID != "aws-arm64" {
		t.Errorf("RunnerFleetID = %q", cfg.RunnerFleetID)
	}
	if cfg.HotInstanceCount != 2 {
		t.Errorf("HotInstanceCount = %d", cfg.HotInstanceCount)
	}
	if cfg.Headroom != 1 {
		t.Errorf("Headroom = %d", cfg.Headroom)
	}

	if cfg.RunnerInstallAWSRegion != "us-east-1" {
		t.Errorf("RunnerInstallAWSRegion = %q", cfg.RunnerInstallAWSRegion)
	}
	if cfg.TaskBrokerURL != "http://broker.internal:8081" {
		t.Errorf("TaskBrokerURL = %q", cfg.TaskBrokerURL)
	}
	if cfg.SubnetID != "subnet-abc" {
		t.Errorf("SubnetID = %q", cfg.SubnetID)
	}
	if len(cfg.SecurityGroupIDs) != 1 || cfg.SecurityGroupIDs[0] != "sg-abc" {
		t.Errorf("SecurityGroupIDs = %v", cfg.SecurityGroupIDs)
	}
	if cfg.RunnersIAMProfName != "superplane-runner" {
		t.Errorf("RunnersIAMProfName = %q", cfg.RunnersIAMProfName)
	}
	if cfg.RunnersAuthToken != "broker-bearer-token" {
		t.Errorf("RunnersAuthToken = %q", cfg.RunnersAuthToken)
	}
	if cfg.VolumeSizeGB != 30 {
		t.Errorf("VolumeSizeGB = %d", cfg.VolumeSizeGB)
	}
	if cfg.BootGraceSec != 300 {
		t.Errorf("BootGraceSec = %d", cfg.BootGraceSec)
	}
	if cfg.RunnerHealthPort != 9090 {
		t.Errorf("RunnerHealthPort = %d", cfg.RunnerHealthPort)
	}
	if !cfg.RunnerTerminateAfterEachTask {
		t.Errorf("RunnerTerminateAfterEachTask = false, want true")
	}
	if cfg.RunnerCloudWatchLogGroup != "/superplane/runner" {
		t.Errorf("RunnerCloudWatchLogGroup = %q", cfg.RunnerCloudWatchLogGroup)
	}
	if cfg.RunnerCloudWatchLogStreamPrefix != "runner-" {
		t.Errorf("RunnerCloudWatchLogStreamPrefix = %q", cfg.RunnerCloudWatchLogStreamPrefix)
	}
	if cfg.RunnerProcessLogGroup != "/superplane/runner-process" {
		t.Errorf("RunnerProcessLogGroup = %q", cfg.RunnerProcessLogGroup)
	}
}

func TestToPoolConfig_TerminateFalsePropagated(t *testing.T) {
	body := strings.Replace(validConfigJSON(),
		`"runner_terminate_after_each_task": true,`,
		`"runner_terminate_after_each_task": false,`, 1)
	f, err := Load(writeConfig(t, body))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg := f.ToPoolConfig(f.Pools[0])
	if cfg.RunnerTerminateAfterEachTask {
		t.Errorf("RunnerTerminateAfterEachTask should be false")
	}
}
