package terraform

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/go-tfe"
	"github.com/mitchellh/mapstructure"
)

type Client struct {
	TFE *tfe.Client
}

func NewClient(configuration map[string]any) (*Client, error) {
	var config Configuration
	err := mapstructure.Decode(configuration, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	if config.APIToken == "" {
		return nil, fmt.Errorf("apiToken is required")
	}

	address := config.Address
	if address == "" {
		address = "https://app.terraform.io"
	}

	tfeConfig := &tfe.Config{
		Token:   config.APIToken,
		Address: address,
	}

	tfeClient, err := tfe.NewClient(tfeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create tfe client: %w", err)
	}

	return &Client{
		TFE: tfeClient,
	}, nil
}

func (c *Client) Validate() error {
	_, err := c.TFE.Users.ReadCurrent(context.TODO())
	if err != nil {
		if isUnauthorized(err) {
			return fmt.Errorf("invalid or expired API Token (Unauthorized): %w", err)
		}
		return fmt.Errorf("failed to validate api token: %w", err)
	}
	return nil
}

func isUnauthorized(err error) bool {
	return strings.Contains(err.Error(), "unauthorized") || errors.Is(err, tfe.ErrResourceNotFound)
}

func (c *Client) ResolveWorkspaceID(ctx context.Context, identifier string) (string, error) {
	if strings.HasPrefix(identifier, "ws-") {
		return identifier, nil
	}
	parts := strings.Split(identifier, "/")
	if len(parts) == 2 {
		ws, err := c.TFE.Workspaces.Read(ctx, parts[0], parts[1])
		if err != nil {
			return "", fmt.Errorf("failed to lookup workspace %s: %w", identifier, err)
		}
		return ws.ID, nil
	}
	return "", fmt.Errorf("invalid workspace identifier format. Expected 'ws-xxx' or 'org_name/workspace_name'")
}
