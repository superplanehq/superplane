package ssh

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("ssh", &SSH{})
}

type SSH struct{}

type Configuration struct {
	Host       string `json:"host" mapstructure:"host"`
	Port       int    `json:"port" mapstructure:"port"`
	Username   string `json:"username" mapstructure:"username"`
	PrivateKey string `json:"privateKey" mapstructure:"privateKey"`
	Passphrase string `json:"passphrase,omitempty" mapstructure:"passphrase"`
}

type Metadata struct {
	Hosts []HostInfo `json:"hosts"`
}

type HostInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
}

func (s *SSH) Name() string  { return "ssh" }
func (s *SSH) Label() string { return "SSH" }
func (s *SSH) Icon() string  { return "server" }

func (s *SSH) Description() string {
	return "Connect to remote hosts via SSH and execute commands, scripts, and retrieve host metadata"
}

func (s *SSH) Instructions() string {
	return "To set up SSH integration:\n\n1. Generate an SSH key pair if you don't have one: `ssh-keygen -t rsa -b 4096`\n2. Copy your public key to the remote host: `ssh-copy-id user@host`\n3. Provide your private key and connection details in the configuration below"
}

func (s *SSH) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "host",
			Label:       "Host",
			Type:        configuration.FieldTypeString,
			Description: "SSH hostname or IP address",
			Placeholder: "e.g. example.com or 192.168.1.100",
			Required:    true,
		},
		{
			Name:        "port",
			Label:       "Port",
			Type:        configuration.FieldTypeNumber,
			Description: "SSH port number",
			Placeholder: "22",
			Default:     22,
			Required:    false,
		},
		{
			Name:        "username",
			Label:       "Username",
			Type:        configuration.FieldTypeString,
			Description: "SSH username",
			Placeholder: "e.g. root, ubuntu, admin",
			Required:    true,
		},
		{
			Name:        "privateKey",
			Label:       "Private Key",
			Type:        configuration.FieldTypeText,
			Sensitive:   true,
			Description: "SSH private key (PEM/OpenSSH format)",
			Required:    true,
		},
		{
			Name:        "passphrase",
			Label:       "Passphrase",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Optional passphrase for encrypted private key",
			Required:    false,
		},
	}
}

func (s *SSH) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.Host == "" {
		return fmt.Errorf("host is required")
	}
	if config.Username == "" {
		return fmt.Errorf("username is required")
	}
	if config.Port == 0 {
		config.Port = 22
	}

	// Sensitive fields are stored encrypted - use GetConfig to decrypt them
	privateKey, err := ctx.Integration.GetConfig("privateKey")
	if err != nil || len(privateKey) == 0 {
		return fmt.Errorf("privateKey is required")
	}
	config.PrivateKey = string(privateKey)

	// Passphrase is optional
	passphrase, _ := ctx.Integration.GetConfig("passphrase")
	config.Passphrase = string(passphrase)

	client, err := NewClientFromConfig(config)
	if err != nil {
		return fmt.Errorf("error creating SSH client: %v", err)
	}
	defer client.Close()

	conn, err := client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to SSH host: %v", err)
	}
	defer conn.Close()

	metadata := Metadata{
		Hosts: []HostInfo{
			{
				Host:     config.Host,
				Port:     config.Port,
				Username: config.Username,
			},
		},
	}

	ctx.Integration.SetMetadata(metadata)
	ctx.Integration.SetState("ready", "")
	return nil
}

func (s *SSH) HandleRequest(ctx core.HTTPRequestContext) {
	// SSH doesn't handle HTTP requests
}

func (s *SSH) CompareWebhookConfig(a, b any) (bool, error) {
	// SSH doesn't use webhooks
	return false, nil
}

func (s *SSH) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	// SSH doesn't use webhooks
	return nil, fmt.Errorf("SSH integration does not support webhooks")
}

func (s *SSH) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	// SSH doesn't use webhooks
	return nil
}

func (s *SSH) Components() []core.Component {
	return []core.Component{
		&ExecuteCommand{},
		&ExecuteScript{},
		&HostMetadata{},
	}
}

func (s *SSH) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (s *SSH) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "host" {
		return []core.IntegrationResource{}, nil
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(metadata.Hosts))
	for _, host := range metadata.Hosts {
		hostIdentifier := fmt.Sprintf("%s@%s:%d", host.Username, host.Host, host.Port)
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: hostIdentifier,
			ID:   hostIdentifier,
		})
	}

	return resources, nil
}
