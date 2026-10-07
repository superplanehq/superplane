package gcpprovider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/fleets/provider"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

const (
	LabelFleetManagerID = "superplane_fleet_manager_id"
	LabelFleetID        = "superplane_fleet_id"
	LabelArchitecture   = "superplane_runner_arch"

	// Labels cannot hold uppercase letters, dots, or colons, so the exact
	// runner ID and version live in metadata.
	MetadataKeyRunnerID      = "superplane-runner-id"
	MetadataKeyRunnerVersion = "superplane-runner-version"
	metadataKeyUserData      = "user-data"
	metadataKeyBlockSSHKeys  = "block-project-ssh-keys"

	instanceNamePrefix = "superplane-runner-"
	maxInstanceName    = 63

	gceLabelLimit      = 64
	reservedLabelCount = 3
	maxCustomLabels    = gceLabelLimit - reservedLabelCount
)

var (
	labelKeyPattern   = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,62}$`)
	labelValuePattern = regexp.MustCompile(`^[a-z0-9_-]{0,63}$`)
	invalidNameChars  = regexp.MustCompile(`[^a-z0-9-]`)

	capacityErrorCodes = []string{
		"ZONE_RESOURCE_POOL_EXHAUSTED",
		"ZONE_RESOURCE_POOL_EXHAUSTED_WITH_DETAILS",
	}
)

type Config struct {
	FleetManagerID      string
	ProjectID           string
	Zones               []string
	MachineType         string
	Image               string
	Architecture        string
	Subnetwork          string
	ServiceAccountEmail string
	NetworkTags         []string
	DiskSizeGB          int64
	DiskType            string
	Labels              map[string]string
}

type Provider struct {
	client ComputeAPI
	config Config
	log    *slog.Logger
}

func New(client ComputeAPI, config Config, log *slog.Logger) (*Provider, error) {
	config.FleetManagerID = strings.TrimSpace(config.FleetManagerID)
	config.ProjectID = strings.TrimSpace(config.ProjectID)
	config.Zones = nonEmpty(config.Zones)
	config.MachineType = strings.TrimSpace(config.MachineType)
	config.Image = strings.TrimSpace(config.Image)
	config.Architecture = strings.ToLower(strings.TrimSpace(config.Architecture))
	config.Subnetwork = strings.TrimSpace(config.Subnetwork)
	config.ServiceAccountEmail = strings.TrimSpace(config.ServiceAccountEmail)
	config.NetworkTags = nonEmpty(config.NetworkTags)
	config.DiskType = strings.TrimSpace(config.DiskType)
	switch {
	case config.FleetManagerID == "" || !labelValuePattern.MatchString(config.FleetManagerID):
		return nil, fmt.Errorf("Fleet Manager ID must be a valid GCP label value")
	case config.ProjectID == "":
		return nil, fmt.Errorf("GCP project ID is required")
	case len(config.Zones) == 0:
		return nil, fmt.Errorf("at least one GCP zone is required")
	case config.MachineType == "":
		return nil, fmt.Errorf("GCP machine type is required")
	case config.Image == "":
		return nil, fmt.Errorf("GCP image is required")
	case config.Architecture != "amd64" && config.Architecture != "arm64":
		return nil, fmt.Errorf("GCP architecture must be amd64 or arm64")
	case config.Subnetwork == "":
		return nil, fmt.Errorf("GCP subnetwork is required")
	case client == nil:
		return nil, fmt.Errorf("Compute Engine client is required")
	}
	if config.DiskSizeGB <= 0 {
		config.DiskSizeGB = 30
	}
	if config.DiskType == "" {
		config.DiskType = "pd-balanced"
	}
	if err := validateLabels(config.Labels); err != nil {
		return nil, err
	}
	if log == nil {
		log = slog.Default()
	}
	return &Provider{client: client, config: config, log: log}, nil
}

func (p *Provider) Name() string {
	return "gcp"
}

func (p *Provider) List(ctx context.Context, fleetID string) ([]provider.Resource, error) {
	filter := fmt.Sprintf(
		`(labels.%s = "%s") AND (labels.%s = "%s")`,
		LabelFleetManagerID,
		p.config.FleetManagerID,
		LabelFleetID,
		fleetID,
	)
	instances, err := p.client.ListInstances(ctx, p.config.ProjectID, filter)
	if err != nil {
		return nil, fmt.Errorf("list GCP runner instances: %w", err)
	}

	var resources []provider.Resource
	for _, instance := range instances {
		runnerID := metadataValue(instance.Metadata, MetadataKeyRunnerID)
		zone := lastPathSegment(instance.Zone)
		if runnerID == "" || zone == "" || instance.Name == "" {
			continue
		}
		resource := provider.Resource{
			ID:       resourceID(zone, instance.Name),
			RunnerID: runnerID,
			FleetID:  instance.Labels[LabelFleetID],
			State:    instance.Status,
		}
		if created, err := time.Parse(time.RFC3339, instance.CreationTimestamp); err == nil {
			resource.CreatedAt = created.UTC()
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

func (p *Provider) BuildBootstrap(request provider.RunnerBootstrap) ([]byte, error) {
	return buildUserData(request)
}

func (p *Provider) Create(
	ctx context.Context,
	request provider.CreateRequest,
) (provider.Resource, error) {
	switch {
	case strings.TrimSpace(request.RunnerID) == "":
		return provider.Resource{}, fmt.Errorf("runner ID is required")
	case !labelValuePattern.MatchString(request.FleetID) || request.FleetID == "":
		return provider.Resource{}, fmt.Errorf("fleet ID must be a valid GCP label value")
	case len(request.Bootstrap) == 0:
		return provider.Resource{}, fmt.Errorf("runner bootstrap data is required")
	}

	name := instanceName(request.RunnerID)
	var lastErr error
	for _, zone := range p.config.Zones {
		err := p.client.InsertInstance(ctx, p.config.ProjectID, zone, p.instance(request, name, zone))
		if err == nil || isStatus(err, http.StatusConflict) {
			p.log.Info(
				"created GCP runner",
				slog.String("runner_id", request.RunnerID),
				slog.String("fleet_id", request.FleetID),
				slog.String("instance", name),
				slog.String("zone", zone),
			)
			return provider.Resource{
				ID:        resourceID(zone, name),
				RunnerID:  request.RunnerID,
				FleetID:   request.FleetID,
				State:     "PROVISIONING",
				CreatedAt: time.Now().UTC(),
			}, nil
		}
		lastErr = err
		if !isInsufficientCapacity(err) {
			return provider.Resource{}, fmt.Errorf("create GCP runner: %w", err)
		}
		p.log.Warn(
			"GCP zone has insufficient capacity",
			slog.String("zone", zone),
			slog.String("runner_id", request.RunnerID),
		)
	}
	return provider.Resource{}, fmt.Errorf("create GCP runner in configured zones: %w", lastErr)
}

func (p *Provider) Delete(ctx context.Context, resource provider.Resource) error {
	zone, name, found := strings.Cut(strings.TrimSpace(resource.ID), "/")
	if !found || zone == "" || name == "" {
		return fmt.Errorf("GCP resource ID must have the form zone/instance: %q", resource.ID)
	}
	err := p.client.DeleteInstance(ctx, p.config.ProjectID, zone, name)
	if err != nil && !isStatus(err, http.StatusNotFound) && !isDeleting(err) {
		return fmt.Errorf("delete GCP runner %s: %w", resource.ID, err)
	}
	p.log.Info(
		"deleted GCP runner",
		slog.String("runner_id", resource.RunnerID),
		slog.String("instance", name),
		slog.String("zone", zone),
	)
	return nil
}

func (p *Provider) instance(
	request provider.CreateRequest,
	name, zone string,
) *compute.Instance {
	labels := map[string]string{
		LabelFleetManagerID: p.config.FleetManagerID,
		LabelFleetID:        request.FleetID,
		LabelArchitecture:   p.config.Architecture,
	}
	for key, value := range p.config.Labels {
		labels[key] = value
	}

	instance := &compute.Instance{
		Name:        name,
		MachineType: fmt.Sprintf("zones/%s/machineTypes/%s", zone, p.config.MachineType),
		Labels:      labels,
		Metadata: &compute.Metadata{Items: []*compute.MetadataItems{
			metadataItem(metadataKeyUserData, string(request.Bootstrap)),
			metadataItem(MetadataKeyRunnerID, request.RunnerID),
			metadataItem(MetadataKeyRunnerVersion, request.RunnerVersion),
			metadataItem(metadataKeyBlockSSHKeys, "true"),
		}},
		Disks: []*compute.AttachedDisk{{
			Boot:       true,
			AutoDelete: true,
			InitializeParams: &compute.AttachedDiskInitializeParams{
				SourceImage: p.config.Image,
				DiskSizeGb:  p.config.DiskSizeGB,
				DiskType:    fmt.Sprintf("zones/%s/diskTypes/%s", zone, p.config.DiskType),
				Labels:      labels,
			},
		}},
		NetworkInterfaces: []*compute.NetworkInterface{{
			Subnetwork: p.config.Subnetwork,
		}},
		ShieldedInstanceConfig: &compute.ShieldedInstanceConfig{
			EnableSecureBoot:          true,
			EnableVtpm:                true,
			EnableIntegrityMonitoring: true,
		},
	}
	if len(p.config.NetworkTags) > 0 {
		instance.Tags = &compute.Tags{Items: p.config.NetworkTags}
	}
	if p.config.ServiceAccountEmail != "" {
		instance.ServiceAccounts = []*compute.ServiceAccount{{
			Email:  p.config.ServiceAccountEmail,
			Scopes: []string{compute.CloudPlatformScope},
		}}
	}
	return instance
}

func validateLabels(labels map[string]string) error {
	if len(labels) > maxCustomLabels {
		return fmt.Errorf(
			"GCP labels must contain at most %d entries; Fleet Manager applies %d reserved labels",
			maxCustomLabels,
			reservedLabelCount,
		)
	}
	for key, value := range labels {
		switch {
		case key == LabelFleetManagerID || key == LabelFleetID || key == LabelArchitecture:
			return fmt.Errorf("reserved GCP label %q cannot be overridden", key)
		case !labelKeyPattern.MatchString(key):
			return fmt.Errorf("GCP label key %q is invalid", key)
		case !labelValuePattern.MatchString(value):
			return fmt.Errorf("GCP label %q has an invalid value", key)
		}
	}
	return nil
}

func instanceName(runnerID string) string {
	cleaned := invalidNameChars.ReplaceAllString(strings.ToLower(strings.TrimSpace(runnerID)), "-")
	name := instanceNamePrefix + cleaned
	if len(name) > maxInstanceName {
		name = name[:maxInstanceName]
	}
	return strings.TrimRight(name, "-")
}

func resourceID(zone, name string) string {
	return zone + "/" + name
}

func metadataItem(key, value string) *compute.MetadataItems {
	return &compute.MetadataItems{Key: key, Value: &value}
}

func metadataValue(metadata *compute.Metadata, key string) string {
	if metadata == nil {
		return ""
	}
	for _, item := range metadata.Items {
		if item != nil && item.Key == key && item.Value != nil {
			return *item.Value
		}
	}
	return ""
}

func lastPathSegment(url string) string {
	return url[strings.LastIndex(url, "/")+1:]
}

func isStatus(err error, status int) bool {
	var apiErr *googleapi.Error
	return errors.As(err, &apiErr) && apiErr.Code == status
}

func isDeleting(err error) bool {
	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) || apiErr.Code != http.StatusBadRequest {
		return false
	}
	return slices.ContainsFunc(apiErr.Errors, func(item googleapi.ErrorItem) bool {
		return item.Reason == "resourceNotReady"
	})
}

func isInsufficientCapacity(err error) bool {
	var operationErr *operationError
	if errors.As(err, &operationErr) {
		return slices.ContainsFunc(operationErr.codes, func(code string) bool {
			return slices.Contains(capacityErrorCodes, code)
		})
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return slices.ContainsFunc(apiErr.Errors, func(item googleapi.ErrorItem) bool {
			return slices.Contains(capacityErrorCodes, item.Reason)
		})
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
