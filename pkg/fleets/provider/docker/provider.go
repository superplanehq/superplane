package dockerprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/fleets/provider"
)

const (
	labelManaged        = "superplane.managed-runner"
	labelFleetManagerID = "superplane.fleet-manager-id"
	labelFleetID        = "superplane.fleet-id"
	labelRunnerID       = "superplane.runner-id"
	labelRunnerVersion  = "superplane.runner-version"
)

type Config struct {
	FleetManagerID string
	Image          string
	RunnerAPIURL   string
	Network        string
	Volumes        []string
	ExtraHosts     []string
	Init           bool
	User           string
	Entrypoint     []string
}

type commandRunner interface {
	Run(context.Context, ...string) ([]byte, error)
}

type dockerCLI struct{}

func (dockerCLI) Run(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
}

type Provider struct {
	config Config
	docker commandRunner
	log    *slog.Logger
}

func New(config Config, log *slog.Logger) (*Provider, error) {
	return newProvider(config, dockerCLI{}, log)
}

func newProvider(
	config Config,
	docker commandRunner,
	log *slog.Logger,
) (*Provider, error) {
	config.FleetManagerID = strings.TrimSpace(config.FleetManagerID)
	config.Image = strings.TrimSpace(config.Image)
	config.RunnerAPIURL = strings.TrimRight(
		strings.TrimSpace(config.RunnerAPIURL),
		"/",
	)
	config.Network = strings.TrimSpace(config.Network)
	config.Volumes = nonEmpty(config.Volumes)
	config.ExtraHosts = nonEmpty(config.ExtraHosts)
	config.User = strings.TrimSpace(config.User)
	if config.FleetManagerID == "" {
		return nil, fmt.Errorf("Fleet Manager ID is required")
	}
	if config.Image == "" {
		return nil, fmt.Errorf("Docker runner image is required")
	}
	if len(config.Entrypoint) > 0 && strings.TrimSpace(config.Entrypoint[0]) == "" {
		return nil, fmt.Errorf("Docker runner entrypoint executable is required")
	}
	if docker == nil {
		return nil, fmt.Errorf("Docker command runner is required")
	}
	if log == nil {
		log = slog.Default()
	}
	return &Provider{config: config, docker: docker, log: log}, nil
}

func (p *Provider) Name() string {
	return "docker"
}

func (p *Provider) List(
	ctx context.Context,
	fleetID string,
) ([]provider.Resource, error) {
	owned, err := p.listRunnerContainers(
		ctx,
		fleetID,
		"label="+labelFleetManagerID+"="+p.config.FleetManagerID,
	)
	if err != nil {
		return nil, err
	}
	if p.config.Network == "" {
		return owned, nil
	}
	onNetwork, err := p.listRunnerContainers(
		ctx,
		fleetID,
		"network="+p.config.Network,
	)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(owned))
	for _, resource := range owned {
		seen[resource.ID] = struct{}{}
	}
	for _, resource := range onNetwork {
		if _, exists := seen[resource.ID]; exists {
			continue
		}
		owned = append(owned, resource)
	}
	return owned, nil
}

func (p *Provider) listRunnerContainers(
	ctx context.Context,
	fleetID string,
	extraFilter string,
) ([]provider.Resource, error) {
	format := strings.Join([]string{
		"{{.ID}}",
		`{{.Label "` + labelRunnerID + `"}}`,
		`{{.Label "` + labelFleetID + `"}}`,
		"{{.State}}",
		`{{.Label "` + labelFleetManagerID + `"}}`,
	}, "\t")
	args := []string{
		"ps",
		"-a",
		"--filter", "label=" + labelManaged + "=true",
		"--filter", "label=" + labelFleetID + "=" + fleetID,
		"--filter", extraFilter,
		"--format", format,
	}
	output, err := p.docker.Run(ctx, args...)
	if err != nil {
		return nil, dockerError("list runner containers", output, err)
	}

	var resources []provider.Resource
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 5 || fields[0] == "" || fields[1] == "" {
			continue
		}
		managerID := strings.TrimSpace(fields[4])
		if managerID != "" && managerID != p.config.FleetManagerID {
			continue
		}
		if managerID == "" && !strings.HasPrefix(extraFilter, "network=") {
			continue
		}
		resources = append(resources, provider.Resource{
			ID:        fields[0],
			RunnerID:  fields[1],
			FleetID:   fields[2],
			State:     fields[3],
			CreatedAt: time.Time{},
		})
	}
	return resources, nil
}

type bootstrap struct {
	RunnerAPIURL      string            `json:"runner_api_url"`
	RegistrationToken string            `json:"registration_token"`
	Tags              map[string]string `json:"tags,omitempty"`
}

func (p *Provider) BuildBootstrap(
	request provider.RunnerBootstrap,
) ([]byte, error) {
	runnerAPIURL := request.RunnerAPIURL
	if p.config.RunnerAPIURL != "" {
		runnerAPIURL = p.config.RunnerAPIURL
	}
	return json.Marshal(bootstrap{
		RunnerAPIURL:      runnerAPIURL,
		RegistrationToken: request.RegistrationToken,
		Tags:              request.Tags,
	})
}

func (p *Provider) Create(
	ctx context.Context,
	request provider.CreateRequest,
) (provider.Resource, error) {
	if strings.TrimSpace(request.RunnerID) == "" {
		return provider.Resource{}, fmt.Errorf("runner ID is required")
	}
	if strings.TrimSpace(request.FleetID) == "" {
		return provider.Resource{}, fmt.Errorf("fleet ID is required")
	}
	var boot bootstrap
	if err := json.Unmarshal(request.Bootstrap, &boot); err != nil {
		return provider.Resource{}, fmt.Errorf("decode Docker runner bootstrap: %w", err)
	}
	if boot.RunnerAPIURL == "" || boot.RegistrationToken == "" {
		return provider.Resource{}, fmt.Errorf("Docker runner bootstrap is incomplete")
	}
	tags, err := provider.FormatRunnerTags(boot.Tags)
	if err != nil {
		return provider.Resource{}, fmt.Errorf("encode Docker runner tags: %w", err)
	}

	name := "superplane-runner-" + request.RunnerID
	args := []string{
		"run", "-d",
		"--name", name,
		"--label", labelManaged + "=true",
		"--label", labelFleetManagerID + "=" + p.config.FleetManagerID,
		"--label", labelFleetID + "=" + request.FleetID,
		"--label", labelRunnerID + "=" + request.RunnerID,
		"--label", labelRunnerVersion + "=" + request.RunnerVersion,
	}
	if p.config.Network != "" {
		args = append(args, "--network", p.config.Network)
	}
	for _, volume := range p.config.Volumes {
		args = append(args, "--volume", volume)
	}
	for _, host := range p.config.ExtraHosts {
		args = append(args, "--add-host", host)
	}
	if tags != "" {
		args = append(args, "--env", "RUNNER_TAGS="+tags)
	}
	if p.config.Init {
		args = append(args, "--init")
	}
	if p.config.User != "" {
		args = append(args, "--user", p.config.User)
	}
	if len(p.config.Entrypoint) > 0 {
		args = append(args, "--entrypoint", p.config.Entrypoint[0])
	}
	args = append(args, p.config.Image)
	if len(p.config.Entrypoint) > 1 {
		args = append(args, p.config.Entrypoint[1:]...)
	}
	args = append(
		args,
		"--url", boot.RunnerAPIURL,
		"--registration-token", boot.RegistrationToken,
	)
	output, err := p.docker.Run(ctx, args...)
	if err != nil {
		return provider.Resource{}, dockerError(
			"create runner container",
			output,
			err,
		)
	}
	containerID := strings.TrimSpace(string(output))
	if containerID == "" {
		return provider.Resource{}, fmt.Errorf("Docker returned an empty container ID")
	}
	p.log.Info(
		"created Docker runner",
		slog.String("runner_id", request.RunnerID),
		slog.String("fleet_id", request.FleetID),
		slog.String("container_id", containerID),
	)
	return provider.Resource{
		ID:        containerID,
		RunnerID:  request.RunnerID,
		FleetID:   request.FleetID,
		State:     "running",
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (p *Provider) Delete(
	ctx context.Context,
	resource provider.Resource,
) error {
	output, err := p.docker.Run(ctx, "rm", "-f", resource.ID)
	if err != nil {
		return dockerError("delete runner container", output, err)
	}
	return nil
}

func (p *Provider) DeletionIsSynchronous() bool {
	return true
}

func dockerError(operation string, output []byte, err error) error {
	message := strings.TrimSpace(string(output))
	if message == "" {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return fmt.Errorf("%s: %w: %s", operation, err, message)
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

var _ provider.SynchronousDeletionProvider = (*Provider)(nil)
