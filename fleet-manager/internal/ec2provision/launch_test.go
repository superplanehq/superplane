package ec2provision

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestConfigFromEnvArchitecture(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want string
	}{
		{name: "defaults to amd64", env: "", want: "amd64"},
		{name: "arm64", env: "arm64", want: "arm64"},
		{name: "amd64", env: "amd64", want: "amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredProvisionEnv(t)
			t.Setenv(envArch, tt.env)

			cfg, err := ConfigFromEnv()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Arch != tt.want {
				t.Fatalf("arch: got %q want %q", cfg.Arch, tt.want)
			}
		})
	}
}

func TestConfigFromEnvRejectsInvalidArchitecture(t *testing.T) {
	setRequiredProvisionEnv(t)
	t.Setenv(envArch, "s390x")

	_, err := ConfigFromEnv()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "EC2_PROVISION_ARCH") {
		t.Fatalf("expected arch error, got %v", err)
	}
}

func TestUserDataScriptUsesArchitectureSpecificPackages(t *testing.T) {
	tests := []struct {
		name           string
		arch           string
		wantAWSCLI     string
		wantCloudWatch string
	}{
		{
			name:           "amd64",
			arch:           "amd64",
			wantAWSCLI:     "awscli-exe-linux-x86_64.zip",
			wantCloudWatch: "amazoncloudwatch-agent/ubuntu/amd64/latest/amazon-cloudwatch-agent.deb",
		},
		{
			name:           "arm64",
			arch:           "arm64",
			wantAWSCLI:     "awscli-exe-linux-aarch64.zip",
			wantCloudWatch: "amazoncloudwatch-agent/ubuntu/arm64/latest/amazon-cloudwatch-agent.deb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script, err := userDataScript(Config{
				Arch:                            tt.arch,
				RunnerS3URI:                     "s3://runner-binaries/release/runner-linux-" + tt.arch,
				RunnerInstallAWSRegion:          "us-east-1",
				TaskBrokerURL:                   "http://task-broker.example:8081",
				RunnerFleetID:                   "fleet-" + tt.arch,
				RunnerTerminateAfterEachTask:    true,
				RunnerCloudWatchLogGroup:        "/superplane/tasks",
				RunnerCloudWatchLogStreamPrefix: "tasks",
				RunnerProcessLogGroup:           "/superplane/runners",
			})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(script, tt.wantAWSCLI) {
				t.Fatalf("user-data missing AWS CLI package %q", tt.wantAWSCLI)
			}
			if !strings.Contains(script, tt.wantCloudWatch) {
				t.Fatalf("user-data missing CloudWatch agent package %q", tt.wantCloudWatch)
			}
			if !strings.Contains(script, "s3://runner-binaries/release/runner-linux-"+tt.arch) {
				t.Fatalf("user-data missing runner S3 URI for %s", tt.arch)
			}
		})
	}
}

func TestManagedInstanceFiltersCanScopeByArchitecture(t *testing.T) {
	l := &Launcher{Config: Config{Arch: "arm64"}}

	filters := l.managedInstanceFilters([]string{"pending", "running"})
	for _, filter := range filters {
		if aws.ToString(filter.Name) == "tag:"+TagKeyArch {
			if len(filter.Values) != 1 || filter.Values[0] != "arm64" {
				t.Fatalf("arch filter values: %#v", filter.Values)
			}
			return
		}
	}

	t.Fatal("missing architecture filter")
}

func TestManagedInstanceFiltersDefaultToAMD64Scope(t *testing.T) {
	l := &Launcher{}

	for _, filter := range l.managedInstanceFilters([]string{"pending", "running"}) {
		if aws.ToString(filter.Name) == "tag:"+TagKeyArch {
			if len(filter.Values) != 1 || filter.Values[0] != "amd64" {
				t.Fatalf("arch filter values: %#v", filter.Values)
			}
			return
		}
	}

	t.Fatal("missing architecture filter")
}

func TestRunInstancesInputTagsRunnerArchitecture(t *testing.T) {
	l := &Launcher{Config: Config{
		AMI:                "ami-1234567890abcdef0",
		InstanceType:       "t4g.micro",
		Arch:               "arm64",
		SubnetID:           "subnet-1234567890abcdef0",
		SecurityGroupIDs:   []string{"sg-1234567890abcdef0"},
		RunnersIAMProfName: "superplane-runner-profile",
	}}

	in := l.runInstancesInput(2, "encoded-user-data")
	if aws.ToString(in.UserData) != "encoded-user-data" {
		t.Fatalf("user data: got %q", aws.ToString(in.UserData))
	}
	if aws.ToInt32(in.MinCount) != 2 || aws.ToInt32(in.MaxCount) != 2 {
		t.Fatalf("counts: min=%d max=%d", aws.ToInt32(in.MinCount), aws.ToInt32(in.MaxCount))
	}

	for _, spec := range in.TagSpecifications {
		for _, tag := range spec.Tags {
			if aws.ToString(tag.Key) == TagKeyArch {
				if aws.ToString(tag.Value) != "arm64" {
					t.Fatalf("arch tag: got %q want arm64", aws.ToString(tag.Value))
				}
				return
			}
		}
	}

	t.Fatal("missing runner architecture tag")
}

func TestConfigFromEnvUsesFleetID(t *testing.T) {
	setRequiredProvisionEnv(t)
	t.Setenv(envFleetID, "my-amd64-fleet")

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FleetID != "my-amd64-fleet" {
		t.Fatalf("fleet id: got %q want %q", cfg.FleetID, "my-amd64-fleet")
	}
}

func TestConfigFromEnvDefaultsFleetIDToHostname(t *testing.T) {
	setRequiredProvisionEnv(t)
	t.Setenv(envFleetID, "") // explicitly unset

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FleetID == "" {
		t.Fatal("expected fleet id to default to hostname, got empty string")
	}
}

func TestManagedInstanceFiltersIncludeFleetID(t *testing.T) {
	l := &Launcher{Config: Config{Arch: "amd64", FleetID: "fleet-a"}}

	for _, filter := range l.managedInstanceFilters([]string{"pending", "running"}) {
		if aws.ToString(filter.Name) == "tag:"+TagKeyFleetID {
			if len(filter.Values) != 1 || filter.Values[0] != "fleet-a" {
				t.Fatalf("fleet id filter values: %#v", filter.Values)
			}
			return
		}
	}
	t.Fatal("missing fleet id filter")
}

func TestRunInstancesInputTagsFleetID(t *testing.T) {
	l := &Launcher{Config: Config{
		AMI:                "ami-1234567890abcdef0",
		InstanceType:       "t4g.micro",
		Arch:               "arm64",
		FleetID:            "arm64-fleet-prod",
		SubnetID:           "subnet-1234567890abcdef0",
		SecurityGroupIDs:   []string{"sg-1234567890abcdef0"},
		RunnersIAMProfName: "superplane-runner-profile",
	}}

	in := l.runInstancesInput(1, "encoded-user-data")
	for _, spec := range in.TagSpecifications {
		for _, tag := range spec.Tags {
			if aws.ToString(tag.Key) == TagKeyFleetID {
				if aws.ToString(tag.Value) != "arm64-fleet-prod" {
					t.Fatalf("fleet id tag: got %q want arm64-fleet-prod", aws.ToString(tag.Value))
				}
				return
			}
		}
	}
	t.Fatal("missing fleet id tag in RunInstances input")
}

func TestConfigFromEnvRejectsDeprecatedBinaryURL(t *testing.T) {
	setRequiredProvisionEnv(t)
	t.Setenv("EC2_PROVISION_RUNNER_BINARY_URL", "https://example.com/runner")

	_, err := ConfigFromEnv()
	if err == nil {
		t.Fatal("expected error for deprecated EC2_PROVISION_RUNNER_BINARY_URL")
	}
	if !strings.Contains(err.Error(), "EC2_PROVISION_RUNNER_BINARY_URL") {
		t.Fatalf("expected deprecation error, got %v", err)
	}
}

func setRequiredProvisionEnv(t *testing.T) {
	t.Helper()
	t.Setenv(envHotCount, "1")
	t.Setenv(envAMI, "ami-1234567890abcdef0")
	t.Setenv(envSubnet, "subnet-1234567890abcdef0")
	t.Setenv(envSecurityGroups, "sg-1234567890abcdef0")
	t.Setenv(envTaskBrokerURL, "http://task-broker.example:8081")
	t.Setenv(envRunnerFleetID, "fleet-test")
	t.Setenv(envRunnerS3URI, "s3://runner-binaries/release/runner-linux-amd64")
	t.Setenv(envRunnerIAMProf, "superplane-runner-profile")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv(envFleetID, "test-fleet")
}
