package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	defaultReconcileIntervalSeconds = 15
	defaultRequestTimeoutSeconds    = 90
	defaultInstanceType             = "t3.micro"
	defaultVolumeSizeGB             = 30

	ProviderAWS    = "aws"
	ProviderDocker = "docker"
)

type Config struct {
	SuperPlaneURL            string  `json:"superplane_url"`
	InstallationAdminToken   string  `json:"installation_admin_token"`
	RunnerReleaseBaseURL     string  `json:"runner_release_base_url"`
	AWSRegion                string  `json:"aws_region"`
	ReconcileIntervalSeconds int     `json:"reconcile_interval_seconds"`
	RequestTimeoutSeconds    int     `json:"request_timeout_seconds"`
	Fleets                   []Fleet `json:"fleets"`
}

type Fleet struct {
	ID           string `json:"id"`
	WarmCapacity int    `json:"warm_capacity"`
	Provider     string `json:"provider"`
	AWS          AWS    `json:"aws"`
	Docker       Docker `json:"docker"`
}

type AWS struct {
	AMI                  string   `json:"ami"`
	InstanceType         string   `json:"instance_type"`
	Architecture         string   `json:"architecture"`
	SubnetIDs            []string `json:"subnet_ids"`
	SecurityGroupIDs     []string `json:"security_group_ids"`
	IAMInstanceProfile   string   `json:"iam_instance_profile"`
	KeyName              string   `json:"key_name"`
	VolumeSizeGB         int32    `json:"volume_size_gb"`
	VolumeIOPS           int32    `json:"volume_iops"`
	VolumeThroughputMBps int32    `json:"volume_throughput_mbps"`
}

type Docker struct {
	Image        string   `json:"image"`
	Architecture string   `json:"architecture"`
	RunnerAPIURL string   `json:"runner_api_url"`
	Network      string   `json:"network"`
	Volumes      []string `json:"volumes"`
	ExtraHosts   []string `json:"extra_hosts"`
}

func Load(path string) (*Config, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("config path is required")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if token := strings.TrimSpace(os.Getenv("INSTALLATION_ADMIN_TOKEN")); token != "" {
		config.InstallationAdminToken = token
	}
	config.applyDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Config) ReconcileInterval() time.Duration {
	return time.Duration(c.ReconcileIntervalSeconds) * time.Second
}

func (c *Config) RequestTimeout() time.Duration {
	return time.Duration(c.RequestTimeoutSeconds) * time.Second
}

func (c *Config) applyDefaults() {
	c.RunnerReleaseBaseURL = strings.TrimRight(
		strings.TrimSpace(c.RunnerReleaseBaseURL),
		"/",
	)
	if c.ReconcileIntervalSeconds == 0 {
		c.ReconcileIntervalSeconds = defaultReconcileIntervalSeconds
	}
	if c.RequestTimeoutSeconds == 0 {
		c.RequestTimeoutSeconds = defaultRequestTimeoutSeconds
	}
	for index := range c.Fleets {
		fleet := &c.Fleets[index]
		fleet.Provider = strings.ToLower(strings.TrimSpace(fleet.Provider))
		if fleet.Provider == "" {
			fleet.Provider = ProviderAWS
		}
		if strings.TrimSpace(fleet.AWS.InstanceType) == "" {
			fleet.AWS.InstanceType = defaultInstanceType
		}
		if fleet.AWS.VolumeSizeGB == 0 {
			fleet.AWS.VolumeSizeGB = defaultVolumeSizeGB
		}
		fleet.ID = strings.TrimSpace(fleet.ID)
		fleet.AWS.Architecture = strings.ToLower(strings.TrimSpace(fleet.AWS.Architecture))
		fleet.AWS.SubnetIDs = nonEmpty(fleet.AWS.SubnetIDs)
		fleet.AWS.SecurityGroupIDs = nonEmpty(fleet.AWS.SecurityGroupIDs)
		fleet.Docker.Image = strings.TrimSpace(fleet.Docker.Image)
		fleet.Docker.Architecture = strings.ToLower(
			strings.TrimSpace(fleet.Docker.Architecture),
		)
		fleet.Docker.RunnerAPIURL = strings.TrimRight(
			strings.TrimSpace(fleet.Docker.RunnerAPIURL),
			"/",
		)
		fleet.Docker.Network = strings.TrimSpace(fleet.Docker.Network)
		fleet.Docker.Volumes = nonEmpty(fleet.Docker.Volumes)
		fleet.Docker.ExtraHosts = nonEmpty(fleet.Docker.ExtraHosts)
	}
}

func (c *Config) validate() error {
	switch {
	case strings.TrimSpace(c.SuperPlaneURL) == "":
		return fmt.Errorf("superplane_url is required")
	case strings.TrimSpace(c.InstallationAdminToken) == "":
		return fmt.Errorf("installation_admin_token is required")
	case c.ReconcileIntervalSeconds < 1:
		return fmt.Errorf("reconcile_interval_seconds must be positive")
	case c.RequestTimeoutSeconds < 1:
		return fmt.Errorf("request_timeout_seconds must be positive")
	case len(c.Fleets) == 0:
		return fmt.Errorf("fleets must contain at least one fleet")
	}
	if c.hasProvider(ProviderAWS) {
		switch {
		case c.RunnerReleaseBaseURL == "":
			return fmt.Errorf("runner_release_base_url is required")
		case strings.Contains(
			strings.ToLower(c.RunnerReleaseBaseURL),
			"latest",
		):
			return fmt.Errorf("runner_release_base_url must not use latest")
		case strings.TrimSpace(c.AWSRegion) == "":
			return fmt.Errorf("aws_region is required")
		}
	}

	seen := make(map[string]struct{}, len(c.Fleets))
	for index, fleet := range c.Fleets {
		prefix := fmt.Sprintf("fleets[%d]", index)
		switch {
		case fleet.ID == "":
			return fmt.Errorf("%s.id is required", prefix)
		case fleet.WarmCapacity < 0:
			return fmt.Errorf("%s.warm_capacity must not be negative", prefix)
		case fleet.Provider != ProviderAWS &&
			fleet.Provider != ProviderDocker:
			return fmt.Errorf("%s.provider is invalid", prefix)
		case fleet.Provider == ProviderDocker &&
			fleet.Docker.Image == "":
			return fmt.Errorf("%s.docker.image is required", prefix)
		case fleet.Provider == ProviderDocker &&
			fleet.Docker.Architecture != "amd64" &&
			fleet.Docker.Architecture != "arm64":
			return fmt.Errorf(
				"%s.docker.architecture must be amd64 or arm64",
				prefix,
			)
		case fleet.Provider == ProviderDocker:
			break
		case strings.TrimSpace(fleet.AWS.AMI) == "":
			return fmt.Errorf("%s.aws.ami is required", prefix)
		case fleet.AWS.Architecture != "amd64" && fleet.AWS.Architecture != "arm64":
			return fmt.Errorf("%s.aws.architecture must be amd64 or arm64", prefix)
		case len(fleet.AWS.SubnetIDs) == 0:
			return fmt.Errorf("%s.aws.subnet_ids must not be empty", prefix)
		case len(fleet.AWS.SecurityGroupIDs) == 0:
			return fmt.Errorf("%s.aws.security_group_ids must not be empty", prefix)
		case fleet.AWS.VolumeSizeGB < 1:
			return fmt.Errorf("%s.aws.volume_size_gb must be positive", prefix)
		case fleet.AWS.VolumeIOPS < 0:
			return fmt.Errorf("%s.aws.volume_iops must not be negative", prefix)
		case fleet.AWS.VolumeThroughputMBps < 0:
			return fmt.Errorf("%s.aws.volume_throughput_mbps must not be negative", prefix)
		}
		if _, exists := seen[fleet.ID]; exists {
			return fmt.Errorf("fleet ID %q is duplicated", fleet.ID)
		}
		seen[fleet.ID] = struct{}{}
	}
	return nil
}

func (c *Config) hasProvider(provider string) bool {
	for _, fleet := range c.Fleets {
		if fleet.Provider == provider {
			return true
		}
	}
	return false
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
