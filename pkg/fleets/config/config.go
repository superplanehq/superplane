package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultReconcileIntervalSeconds = 15
	defaultRequestTimeoutSeconds    = 90
	defaultInstanceType             = "t3.micro"
	defaultAzureVMSize              = "Standard_D2ds_v4"
	defaultVolumeSizeGB             = 30

	fleetManagerConfigEnvironment = "FLEET_MANAGER_CONFIG"
	installationTokenEnvironment  = "INSTALLATION_ADMIN_TOKEN"

	ProviderAWS    = "aws"
	ProviderAzure  = "azure"
	ProviderDocker = "docker"
)

type Config struct {
	ID                       string  `json:"id"`
	SuperPlaneURL            string  `json:"superplaneUrl"`
	InstallationAdminToken   string  `json:"installationAdminToken"`
	RunnerReleaseBaseURL     string  `json:"runnerReleaseBaseUrl"`
	ReconcileIntervalSeconds int     `json:"reconcileIntervalSeconds"`
	RequestTimeoutSeconds    int     `json:"requestTimeoutSeconds"`
	Fleets                   []Fleet `json:"fleets"`
}

type Fleet struct {
	ID           string `json:"id"`
	WarmCapacity int    `json:"warmCapacity"`
	MaxCapacity  int    `json:"maxCapacity"`
	Provider     string `json:"provider"`
	AWS          AWS    `json:"aws"`
	Azure        Azure  `json:"azure"`
	Docker       Docker `json:"docker"`
}

type AWS struct {
	Region               string            `json:"region"`
	AMI                  string            `json:"ami"`
	InstanceType         string            `json:"instanceType"`
	Architecture         string            `json:"architecture"`
	SubnetIDs            []string          `json:"subnetIds"`
	SecurityGroupIDs     []string          `json:"securityGroupIds"`
	IAMInstanceProfile   string            `json:"iamInstanceProfile"`
	KeyName              string            `json:"keyName"`
	VolumeSizeGB         int32             `json:"volumeSizeGb"`
	VolumeIOPS           int32             `json:"volumeIops"`
	VolumeThroughputMBps int32             `json:"volumeThroughputMbps"`
	ResourceTags         map[string]string `json:"resourceTags"`
	CloudWatch           CloudWatch        `json:"cloudWatch"`
}

type CloudWatch struct {
	LogGroupName string `json:"logGroupName"`
}

type Azure struct {
	SubscriptionID         string            `json:"subscriptionId"`
	ResourceGroup          string            `json:"resourceGroup"`
	Location               string            `json:"location"`
	ImageID                string            `json:"imageId"`
	VMSize                 string            `json:"vmSize"`
	Architecture           string            `json:"architecture"`
	SubnetID               string            `json:"subnetId"`
	NetworkSecurityGroupID string            `json:"networkSecurityGroupId"`
	IdentityID             string            `json:"identityId"`
	Zones                  []string          `json:"zones"`
	DiskSizeGB             int32             `json:"diskSizeGb"`
	EphemeralOSDisk        bool              `json:"ephemeralOSDisk"`
	ResourceTags           map[string]string `json:"resourceTags"`
}

type Docker struct {
	Image        string   `json:"image"`
	Architecture string   `json:"architecture"`
	RunnerAPIURL string   `json:"runnerApiUrl"`
	Network      string   `json:"network"`
	Volumes      []string `json:"volumes"`
	ExtraHosts   []string `json:"extraHosts"`
}

func Load(path string) (*Config, error) {
	var config Config
	if body := strings.TrimSpace(os.Getenv(fleetManagerConfigEnvironment)); body != "" {
		if err := decode(strings.NewReader(body), ".yaml", &config); err != nil {
			return nil, fmt.Errorf("decode %s: %w", fleetManagerConfigEnvironment, err)
		}
	} else {
		path = strings.TrimSpace(path)
		if path == "" {
			return nil, fmt.Errorf("config path is required")
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open config: %w", err)
		}
		defer file.Close()

		if err := decode(file, filepath.Ext(path), &config); err != nil {
			return nil, fmt.Errorf("decode config: %w", err)
		}
	}
	if token := strings.TrimSpace(os.Getenv(installationTokenEnvironment)); token != "" {
		config.InstallationAdminToken = token
	}
	config.applyDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	return &config, nil
}

func decode(reader io.Reader, extension string, config *Config) error {
	switch strings.ToLower(extension) {
	case ".json":
		return decodeJSON(reader, config)
	case ".yml", ".yaml":
		var document any
		if err := yaml.NewDecoder(reader).Decode(&document); err != nil {
			return err
		}
		raw, err := json.Marshal(document)
		if err != nil {
			return err
		}
		return decodeJSON(bytes.NewReader(raw), config)
	default:
		return fmt.Errorf(
			"unsupported file extension %q; use .json, .yml, or .yaml",
			extension,
		)
	}
}

func decodeJSON(reader io.Reader, config *Config) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	return decoder.Decode(config)
}

func (c *Config) ReconcileInterval() time.Duration {
	return time.Duration(c.ReconcileIntervalSeconds) * time.Second
}

func (c *Config) RequestTimeout() time.Duration {
	return time.Duration(c.RequestTimeoutSeconds) * time.Second
}

func (c *Config) applyDefaults() {
	c.ID = strings.TrimSpace(c.ID)
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
		if strings.TrimSpace(fleet.Azure.VMSize) == "" {
			fleet.Azure.VMSize = defaultAzureVMSize
		}
		if fleet.Azure.DiskSizeGB == 0 {
			fleet.Azure.DiskSizeGB = defaultVolumeSizeGB
		}
		fleet.ID = strings.TrimSpace(fleet.ID)
		fleet.AWS.Region = strings.TrimSpace(fleet.AWS.Region)
		fleet.AWS.Architecture = strings.ToLower(strings.TrimSpace(fleet.AWS.Architecture))
		fleet.AWS.SubnetIDs = nonEmpty(fleet.AWS.SubnetIDs)
		fleet.AWS.SecurityGroupIDs = nonEmpty(fleet.AWS.SecurityGroupIDs)
		fleet.AWS.CloudWatch.LogGroupName = strings.TrimSpace(
			fleet.AWS.CloudWatch.LogGroupName,
		)
		fleet.Azure.SubscriptionID = strings.TrimSpace(fleet.Azure.SubscriptionID)
		fleet.Azure.ResourceGroup = strings.TrimSpace(fleet.Azure.ResourceGroup)
		fleet.Azure.Location = strings.TrimSpace(fleet.Azure.Location)
		fleet.Azure.ImageID = strings.TrimSpace(fleet.Azure.ImageID)
		fleet.Azure.VMSize = strings.TrimSpace(fleet.Azure.VMSize)
		fleet.Azure.Architecture = strings.ToLower(strings.TrimSpace(fleet.Azure.Architecture))
		fleet.Azure.SubnetID = strings.TrimSpace(fleet.Azure.SubnetID)
		fleet.Azure.NetworkSecurityGroupID = strings.TrimSpace(
			fleet.Azure.NetworkSecurityGroupID,
		)
		fleet.Azure.IdentityID = strings.TrimSpace(fleet.Azure.IdentityID)
		fleet.Azure.Zones = nonEmpty(fleet.Azure.Zones)
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
		return fmt.Errorf("superplaneUrl is required")
	case c.ID == "":
		return fmt.Errorf("id is required")
	case strings.TrimSpace(c.InstallationAdminToken) == "":
		return fmt.Errorf("installationAdminToken is required")
	case c.ReconcileIntervalSeconds < 1:
		return fmt.Errorf("reconcileIntervalSeconds must be positive")
	case c.RequestTimeoutSeconds < 1:
		return fmt.Errorf("requestTimeoutSeconds must be positive")
	case len(c.Fleets) == 0:
		return fmt.Errorf("fleets must contain at least one fleet")
	}
	if c.hasProvider(ProviderAWS) || c.hasProvider(ProviderAzure) {
		switch {
		case c.RunnerReleaseBaseURL == "":
			return fmt.Errorf("runnerReleaseBaseUrl is required")
		case strings.Contains(
			strings.ToLower(c.RunnerReleaseBaseURL),
			"latest",
		):
			return fmt.Errorf("runnerReleaseBaseUrl must not use latest")
		}
	}

	seen := make(map[string]struct{}, len(c.Fleets))
	for index, fleet := range c.Fleets {
		prefix := fmt.Sprintf("fleets[%d]", index)
		switch {
		case fleet.ID == "":
			return fmt.Errorf("%s.id is required", prefix)
		case fleet.WarmCapacity < 0:
			return fmt.Errorf("%s.warmCapacity must not be negative", prefix)
		case fleet.MaxCapacity < 0:
			return fmt.Errorf("%s.maxCapacity must not be negative", prefix)
		case fleet.MaxCapacity > 0 && fleet.WarmCapacity > fleet.MaxCapacity:
			return fmt.Errorf(
				"%s.warmCapacity must not exceed maxCapacity",
				prefix,
			)
		case fleet.Provider != ProviderAWS &&
			fleet.Provider != ProviderAzure &&
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
		case fleet.Provider == ProviderAzure &&
			fleet.Azure.SubscriptionID == "":
			return fmt.Errorf("%s.azure.subscriptionId is required", prefix)
		case fleet.Provider == ProviderAzure &&
			fleet.Azure.ResourceGroup == "":
			return fmt.Errorf("%s.azure.resourceGroup is required", prefix)
		case fleet.Provider == ProviderAzure &&
			fleet.Azure.Location == "":
			return fmt.Errorf("%s.azure.location is required", prefix)
		case fleet.Provider == ProviderAzure &&
			fleet.Azure.ImageID == "":
			return fmt.Errorf("%s.azure.imageId is required", prefix)
		case fleet.Provider == ProviderAzure &&
			fleet.Azure.Architecture != "amd64" &&
			fleet.Azure.Architecture != "arm64":
			return fmt.Errorf(
				"%s.azure.architecture must be amd64 or arm64",
				prefix,
			)
		case fleet.Provider == ProviderAzure &&
			fleet.Azure.SubnetID == "":
			return fmt.Errorf("%s.azure.subnetId is required", prefix)
		case fleet.Provider == ProviderAzure &&
			fleet.Azure.NetworkSecurityGroupID == "":
			return fmt.Errorf(
				"%s.azure.networkSecurityGroupId is required",
				prefix,
			)
		case fleet.Provider == ProviderAzure &&
			fleet.Azure.IdentityID == "":
			return fmt.Errorf("%s.azure.identityId is required", prefix)
		case fleet.Provider == ProviderAzure &&
			len(fleet.Azure.Zones) == 0:
			return fmt.Errorf("%s.azure.zones must not be empty", prefix)
		case fleet.Provider == ProviderAzure &&
			fleet.Azure.DiskSizeGB < 1:
			return fmt.Errorf("%s.azure.diskSizeGb must be positive", prefix)
		case fleet.Provider == ProviderAzure:
			break
		case fleet.AWS.Region == "":
			return fmt.Errorf("%s.aws.region is required", prefix)
		case strings.TrimSpace(fleet.AWS.AMI) == "":
			return fmt.Errorf("%s.aws.ami is required", prefix)
		case fleet.AWS.Architecture != "amd64" && fleet.AWS.Architecture != "arm64":
			return fmt.Errorf("%s.aws.architecture must be amd64 or arm64", prefix)
		case len(fleet.AWS.SubnetIDs) == 0:
			return fmt.Errorf("%s.aws.subnetIds must not be empty", prefix)
		case len(fleet.AWS.SecurityGroupIDs) == 0:
			return fmt.Errorf("%s.aws.securityGroupIds must not be empty", prefix)
		case fleet.AWS.VolumeSizeGB < 1:
			return fmt.Errorf("%s.aws.volumeSizeGb must be positive", prefix)
		case fleet.AWS.VolumeIOPS < 0:
			return fmt.Errorf("%s.aws.volumeIops must not be negative", prefix)
		case fleet.AWS.VolumeThroughputMBps < 0:
			return fmt.Errorf("%s.aws.volumeThroughputMbps must not be negative", prefix)
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
