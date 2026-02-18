package bitbucket

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	issuePayloadType        = "bitbucket.issue"
	issueCommentPayloadType = "bitbucket.issueComment"
)

func metadataContextForExecution(ctx core.ExecutionContext) core.MetadataContext {
	if ctx.NodeMetadata != nil {
		return ctx.NodeMetadata
	}

	return ctx.Metadata
}

func newClientFromIntegration(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, *Metadata, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return nil, nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	if metadata.Workspace == nil {
		return nil, nil, fmt.Errorf("integration workspace metadata is required")
	}

	client, err := NewClient(metadata.AuthType, httpCtx, integration)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, &metadata, nil
}
