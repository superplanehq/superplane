package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/eventarc"
)

const (
	webhookMetadataPipelineID   = "pipelineId"
	webhookMetadataEnrollmentID = "enrollmentId"
	webhookMetadataBusID        = "busId"
	webhookMetadataSourceID     = "sourceId"

	sharedBusPrefix    = "sp-bus-"
	sharedSourcePrefix = "sp-src-"
	pipelinePrefix     = "sp-pipe-"
	enrollmentPrefix   = "sp-enrl-"

	operationTimeout = 5 * time.Minute
)

type webhookConfig struct {
	ProjectID string `mapstructure:"projectId"`
	Region    string `mapstructure:"region"`
	CelFilter string `mapstructure:"celFilter"`
}

type WebhookHandler struct{}

func (h *WebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	var config webhookConfig
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("decode webhook configuration: %w", err)
	}

	projectID := strings.TrimSpace(config.ProjectID)
	if projectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}
	region := strings.TrimSpace(config.Region)
	if region == "" {
		region = "us-central1"
	}
	celFilter := strings.TrimSpace(config.CelFilter)
	if celFilter == "" {
		return nil, fmt.Errorf("CEL filter is required")
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("create GCP client: %w", err)
	}

	meta := ctx.Integration.GetMetadata()
	var m gcpcommon.Metadata
	if err := mapstructure.Decode(meta, &m); err != nil {
		return nil, fmt.Errorf("decode integration metadata: %w", err)
	}
	serviceAccountEmail := strings.TrimSpace(m.ClientEmail)

	reqCtx := context.Background()
	sanitizedProject := sanitizeID(projectID)
	busID := sharedBusPrefix + sanitizedProject
	sourceID := sharedSourcePrefix + sanitizedProject

	webhookID := ctx.Webhook.GetID()
	pipelineID := pipelinePrefix + strings.ReplaceAll(webhookID, "-", "")
	enrollmentID := enrollmentPrefix + strings.ReplaceAll(webhookID, "-", "")

	busFullName := eventarc.MessageBusFullName(projectID, region, busID)
	pipelineFullName := eventarc.PipelineFullName(projectID, region, pipelineID)

	if err := h.ensureSharedResources(reqCtx, client, projectID, region, busID, sourceID, busFullName); err != nil {
		return nil, err
	}

	webhookURL := ctx.Webhook.GetURL()
	pipelineOp, err := eventarc.CreatePipeline(reqCtx, client, projectID, region, pipelineID, webhookURL, serviceAccountEmail)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: %w", err)
	}
	if err := eventarc.PollOperation(reqCtx, client, pipelineOp, operationTimeout); err != nil {
		return nil, fmt.Errorf("wait for pipeline: %w", err)
	}

	enrollmentOp, err := eventarc.CreateEnrollment(reqCtx, client, projectID, region, enrollmentID, busFullName, pipelineFullName, celFilter)
	if err != nil {
		_ = eventarc.DeletePipeline(reqCtx, client, projectID, region, pipelineID)
		return nil, fmt.Errorf("create enrollment: %w", err)
	}
	if err := eventarc.PollOperation(reqCtx, client, enrollmentOp, operationTimeout); err != nil {
		_ = eventarc.DeletePipeline(reqCtx, client, projectID, region, pipelineID)
		return nil, fmt.Errorf("wait for enrollment: %w", err)
	}

	return map[string]string{
		webhookMetadataPipelineID:   pipelineID,
		webhookMetadataEnrollmentID: enrollmentID,
		webhookMetadataBusID:        busID,
		webhookMetadataSourceID:     sourceID,
	}, nil
}

func (h *WebhookHandler) ensureSharedResources(ctx context.Context, client *gcpcommon.Client, projectID, region, busID, sourceID, busFullName string) error {
	if err := eventarc.GetMessageBus(ctx, client, projectID, region, busID); err != nil {
		if !eventarc.IsNotFoundError(err) {
			return fmt.Errorf("check message bus: %w", err)
		}

		busOp, err := eventarc.CreateMessageBus(ctx, client, projectID, region, busID)
		if err != nil {
			if !eventarc.IsAlreadyExistsError(err) {
				return fmt.Errorf("create message bus: %w", err)
			}
		} else {
			if err := eventarc.PollOperation(ctx, client, busOp, operationTimeout); err != nil {
				return fmt.Errorf("wait for message bus: %w", err)
			}
		}
	}

	if err := eventarc.GetGoogleAPISource(ctx, client, projectID, region, sourceID); err != nil {
		if !eventarc.IsNotFoundError(err) {
			return fmt.Errorf("check google api source: %w", err)
		}

		sourceOp, err := eventarc.CreateGoogleAPISource(ctx, client, projectID, region, sourceID, busFullName)
		if err != nil {
			if !eventarc.IsAlreadyExistsError(err) {
				return fmt.Errorf("create google api source: %w", err)
			}
		} else {
			if err := eventarc.PollOperation(ctx, client, sourceOp, operationTimeout); err != nil {
				return fmt.Errorf("wait for google api source: %w", err)
			}
		}
	}

	return nil
}

func (h *WebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	var config webhookConfig
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return fmt.Errorf("decode webhook configuration: %w", err)
	}
	projectID := strings.TrimSpace(config.ProjectID)
	if projectID == "" {
		return nil
	}
	region := strings.TrimSpace(config.Region)
	if region == "" {
		region = "us-central1"
	}

	meta := ctx.Webhook.GetMetadata()
	metaMap, _ := meta.(map[string]any)
	if metaMap == nil {
		return nil
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("create GCP client for cleanup: %w", err)
	}
	reqCtx := context.Background()

	if id := getMetaString(metaMap, webhookMetadataEnrollmentID); id != "" {
		_ = eventarc.DeleteEnrollment(reqCtx, client, projectID, region, id)
	}
	if id := getMetaString(metaMap, webhookMetadataPipelineID); id != "" {
		_ = eventarc.DeletePipeline(reqCtx, client, projectID, region, id)
	}

	return nil
}

func (h *WebhookHandler) CompareConfig(a, b any) (bool, error) {
	var ca, cb webhookConfig
	if err := mapstructure.Decode(a, &ca); err != nil {
		return false, err
	}
	if err := mapstructure.Decode(b, &cb); err != nil {
		return false, err
	}
	return strings.TrimSpace(ca.ProjectID) == strings.TrimSpace(cb.ProjectID) &&
		strings.TrimSpace(ca.Region) == strings.TrimSpace(cb.Region) &&
		strings.TrimSpace(ca.CelFilter) == strings.TrimSpace(cb.CelFilter), nil
}

func (h *WebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func getMetaString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func sanitizeID(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		}
	}
	result := b.String()
	if len(result) > 40 {
		result = result[:40]
	}
	return result
}
