package ssh

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

type HostMetadata struct{}

type HostMetadataSpec struct {
	Host string `json:"host"`
}

type HostMetadataExecutionMetadata struct {
	Metadata *HostMetadataResult `json:"metadata" mapstructure:"metadata"`
}

func (h *HostMetadata) Name() string {
	return "ssh.hostMetadata"
}

func (h *HostMetadata) Label() string {
	return "Host Metadata"
}

func (h *HostMetadata) Description() string {
	return "Retrieve metadata about a remote host (OS info, hostname, uptime, disk usage, memory)"
}

func (c *HostMetadata) Documentation() string {
	return `Retrieve system metadata and information from a remote host via SSH.

## Use Cases

- **Monitoring**: Gather system information for monitoring dashboards
- **Inventory**: Collect host details for infrastructure inventory
- **Health Checks**: Verify host availability and resource status
- **Troubleshooting**: Quickly gather diagnostic information from remote systems

## Configuration

- **Host**: Select a host resource from the SSH integration (format: user@host:port)

## Output

Returns comprehensive metadata about the remote host:

- **hostname**: The system hostname
- **os**: Full OS information from "uname -a"
- **kernel**: Kernel version from "uname -r"
- **architecture**: System architecture (e.g., x86_64, arm64) from "uname -m"
- **uptime**: System uptime information
- **diskUsage**: Disk usage statistics from "df -h"
- **memoryInfo**: Memory information (uses "free -m" on Linux, "vm_stat" on macOS)

This component always emits to the default channel upon successful execution.`
}

func (h *HostMetadata) Icon() string {
	return "server"
}

func (h *HostMetadata) Color() string {
	return "green"
}

func (h *HostMetadata) ExampleOutput() map[string]any {
	return map[string]any{
		"metadata": map[string]any{
			"hostname":     "example-server",
			"os":           "Linux example-server 5.4.0-74-generic #83-Ubuntu SMP",
			"kernel":       "5.4.0-74-generic",
			"architecture": "x86_64",
			"uptime":       "up 15 days, 3:45",
			"diskUsage":    "Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1        20G  8.5G   11G  45% /",
			"memoryInfo":   "              total        used        free      shared  buff/cache   available\nMem:           7982        3245        1023         234        3714        4201",
		},
	}
}

func (h *HostMetadata) OutputChannels(configuration any) []core.OutputChannel {
	// Host metadata always emits to default channel
	return []core.OutputChannel{}
}

func (h *HostMetadata) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "host",
			Label:    "Host",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "host",
					UseNameAsValue: true,
				},
			},
		},
	}
}

func (h *HostMetadata) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (h *HostMetadata) Setup(ctx core.SetupContext) error {
	spec := HostMetadataSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Host == "" {
		return fmt.Errorf("host is required")
	}

	return nil
}

func (h *HostMetadata) Execute(ctx core.ExecutionContext) error {
	spec := HostMetadataSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			fmt.Sprintf("error decoding configuration: %v", err),
		)
	}

	username, host, port, err := parseHostIdentifier(spec.Host)
	if err != nil {
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			fmt.Sprintf("invalid host format (expected user@host:port): %v", err),
		)
	}

	privateKey, err := ctx.Integration.GetConfig("privateKey")
	if err != nil || len(privateKey) == 0 {
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			"privateKey is required in SSH integration configuration",
		)
	}

	passphrase, _ := ctx.Integration.GetConfig("passphrase")

	sshCfg := Configuration{
		Host:       host,
		Port:       port,
		Username:   username,
		PrivateKey: string(privateKey),
		Passphrase: string(passphrase),
	}

	client, err := NewClientFromConfig(sshCfg)
	if err != nil {
		return ctx.ExecutionState.Fail(models.WorkflowNodeExecutionResultReasonError, fmt.Sprintf("failed to create SSH client: %v", err))
	}
	defer client.Close()

	ctx.Logger.Infof("Retrieving host metadata from SSH host")

	metadata, err := client.GetHostMetadata()
	if err != nil {
		return ctx.ExecutionState.Fail(models.WorkflowNodeExecutionResultReasonError, fmt.Sprintf("failed to retrieve host metadata: %v", err))
	}

	// Store metadata
	err = ctx.Metadata.Set(HostMetadataExecutionMetadata{
		Metadata: metadata,
	})
	if err != nil {
		return ctx.ExecutionState.Fail(models.WorkflowNodeExecutionResultReasonError, fmt.Sprintf("failed to set metadata: %v", err))
	}

	// Emit to default channel
	return ctx.ExecutionState.Emit("default", "ssh.host.metadata", []any{metadata})
}

func (h *HostMetadata) Cancel(ctx core.ExecutionContext) error {
	// Host metadata retrieval can't be cancelled
	return nil
}

func (h *HostMetadata) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// SSH doesn't handle webhooks
	return 404, fmt.Errorf("SSH hostMetadata does not handle webhooks")
}

func (h *HostMetadata) Actions() []core.Action {
	return []core.Action{}
}

func (h *HostMetadata) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for hostMetadata")
}
