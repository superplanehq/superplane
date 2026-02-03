package ecr

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type ScanImage struct{}

type ScanImageConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	Repository  string `json:"repository" mapstructure:"repository"`
	ImageDigest string `json:"imageDigest" mapstructure:"imageDigest"`
	ImageTag    string `json:"imageTag" mapstructure:"imageTag"`
}

type ScanImageMetadata struct {
	Region      string `json:"region" mapstructure:"region"`
	Repository  string `json:"repository" mapstructure:"repository"`
	ImageDigest string `json:"imageDigest" mapstructure:"imageDigest"`
}

func (c *ScanImage) Name() string {
	return "aws.ecr.scanImage"
}

func (c *ScanImage) Label() string {
	return "ECR • Scan Image"
}

func (c *ScanImage) Description() string {
	return "Scan an ECR image for vulnerabilities"
}

func (c *ScanImage) Documentation() string {
	return `The Scan Image component scans an ECR image for vulnerabilities.

## Use Cases

- **Security automation**: Scan images for vulnerabilities
- **Compliance checks**: Validate images against severity thresholds
- **Reporting**: Capture scan summaries and findings for audits

## Configuration

- **Region**: AWS region of the ECR repository
- **Repository**: ECR repository name or ARN
- **Image Digest**: Digest of the image (optional)
- **Image Tag**: Tag of the image (optional)

At least one of **Image Digest** or **Image Tag** is required. If both are provided, the request includes both.`
}

func (c *ScanImage) Icon() string {
	return "aws"
}

func (c *ScanImage) Color() string {
	return "gray"
}

func (c *ScanImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ScanImage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeString,
			Required: true,
			Default:  "us-east-1",
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "ECR repository name or ARN",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ecr.repository",
				},
			},
		},
		{
			Name:        "imageDigest",
			Label:       "Image Digest",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "sha256:...",
		},
		{
			Name:        "imageTag",
			Label:       "Image Tag",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "latest",
		},
	}
}

func (c *ScanImage) Setup(ctx core.SetupContext) error {
	var config GetImageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region := strings.TrimSpace(config.Region)
	if region == "" {
		return fmt.Errorf("region is required")
	}

	if strings.TrimSpace(config.Repository) == "" {
		return fmt.Errorf("repository is required")
	}

	if strings.TrimSpace(config.ImageDigest) == "" && strings.TrimSpace(config.ImageTag) == "" {
		return fmt.Errorf("image digest or image tag is required")
	}

	return nil
}

func (c *ScanImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ScanImage) Execute(ctx core.ExecutionContext) error {
	var config GetImageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	response, err := client.ScanImage(config.Repository, config.ImageDigest, config.ImageTag)
	if err != nil {
		return fmt.Errorf("failed to scan image: %w", err)
	}

	//
	// If the scan is not complete, poll for findings every 10 seconds.
	//
	if response.ScanStatus.Status != "COMPLETE" {
		err = ctx.Metadata.Set(ScanImageMetadata{
			Region:      config.Region,
			Repository:  config.Repository,
			ImageDigest: config.ImageDigest,
		})

		if err != nil {
			return fmt.Errorf("failed to set metadata: %w", err)
		}

		return ctx.Requests.ScheduleActionCall(
			"pollFindings",
			map[string]any{},
			10*time.Second,
		)
	}

	findings, err := client.DescribeImageScanFindings(config.Repository, config.ImageDigest, config.ImageTag)
	if err != nil {
		return fmt.Errorf("failed to describe image scan findings: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ecr.image.scanFindings",
		[]any{findings},
	)
}

func (c *ScanImage) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "pollFindings",
			Description: "Poll for scan findings",
		},
	}
}

func (c *ScanImage) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "pollFindings":
		return c.pollFindings(ctx)

	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *ScanImage) pollFindings(ctx core.ActionContext) error {
	metadata := ScanImageMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, metadata.Region)
	findings, err := client.DescribeImageScanFindings(metadata.Repository, metadata.ImageDigest, "")
	if err != nil {
		return fmt.Errorf("failed to describe image scan findings: %w", err)
	}

	if findings.ImageScanStatus.Status != "COMPLETE" {
		return ctx.Requests.ScheduleActionCall(
			"pollFindings",
			map[string]any{},
			10*time.Second,
		)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ecr.image.scanFindings",
		[]any{findings},
	)
}

func (c *ScanImage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *ScanImage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ScanImage) Cleanup(ctx core.SetupContext) error {
	return nil
}
