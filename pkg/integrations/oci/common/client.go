package common

import (
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"
	ocicommon "github.com/oracle/oci-go-sdk/v65/common"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/superplanehq/superplane/pkg/core"
)

type Config struct {
	TenancyOCID string `mapstructure:"tenancyOcid"`
	UserOCID    string `mapstructure:"userOcid"`
	Fingerprint string `mapstructure:"fingerprint"`
	Region      string `mapstructure:"region"`
	PrivateKey  string `mapstructure:"privateKey"`
}

func NewConfigurationProvider(ctx core.IntegrationContext) (ocicommon.ConfigurationProvider, error) {
	var config Config
	if err := mapstructure.Decode(ctx.GetMetadata(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode integration configuration: %w", err)
	}

	return ocicommon.NewRawConfigurationProvider(
		config.TenancyOCID,
		config.UserOCID,
		config.Region,
		config.Fingerprint,
		config.PrivateKey,
		nil,
	), nil
}

type ComputeClientWrapper struct {
	client ocicore.ComputeClient
}

func NewComputeClientWrapper(ctx core.IntegrationContext) (*ComputeClientWrapper, error) {
	provider, err := NewConfigurationProvider(ctx)
	if err != nil {
		return nil, err
	}
	client, err := ocicore.NewComputeClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}
	return &ComputeClientWrapper{client: client}, nil
}

func (w *ComputeClientWrapper) LaunchInstance(ctx context.Context, request ocicore.LaunchInstanceRequest) (ocicore.LaunchInstanceResponse, error) {
	return w.client.LaunchInstance(ctx, request)
}

func (w *ComputeClientWrapper) GetInstance(ctx context.Context, request ocicore.GetInstanceRequest) (ocicore.GetInstanceResponse, error) {
	return w.client.GetInstance(ctx, request)
}

func (w *ComputeClientWrapper) UpdateInstance(ctx context.Context, request ocicore.UpdateInstanceRequest) (ocicore.UpdateInstanceResponse, error) {
	return w.client.UpdateInstance(ctx, request)
}

func (w *ComputeClientWrapper) TerminateInstance(ctx context.Context, request ocicore.TerminateInstanceRequest) (ocicore.TerminateInstanceResponse, error) {
	return w.client.TerminateInstance(ctx, request)
}

func (w *ComputeClientWrapper) InstanceAction(ctx context.Context, request ocicore.InstanceActionRequest) (ocicore.InstanceActionResponse, error) {
	return w.client.InstanceAction(ctx, request)
}
