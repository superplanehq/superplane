package codepipeline

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListPipelines(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)

	pipelines, err := client.ListPipelines()
	if err != nil {
		return nil, fmt.Errorf("failed to list CodePipeline pipelines: %w", err)
	}

	// Pipeline name is used as ID because AWS ListPipelines does not return ARN.
	resources := make([]core.IntegrationResource, 0, len(pipelines))
	for _, pipeline := range pipelines {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: pipeline.Name,
			ID:   pipeline.Name,
		})
	}

	return resources, nil
}

func ListStages(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	pipeline := ctx.Parameters["pipeline"]
	if pipeline == "" {
		return nil, fmt.Errorf("pipeline is required")
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	definition, err := client.GetPipeline(pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline definition: %w", err)
	}

	stageRows, ok := definition.Pipeline["stages"].([]any)
	if !ok {
		return []core.IntegrationResource{}, nil
	}

	resources := make([]core.IntegrationResource, 0, len(stageRows))
	for _, row := range stageRows {
		stageMap, ok := row.(map[string]any)
		if !ok {
			continue
		}
		name, ok := stageMap["name"].(string)
		if !ok || name == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: name,
			ID:   name,
		})
	}

	return resources, nil
}

func ListPipelineExecutions(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	pipeline := ctx.Parameters["pipeline"]
	if pipeline == "" {
		return nil, fmt.Errorf("pipeline is required")
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	executions, err := client.ListPipelineExecutionSummaries(pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipeline executions: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(executions))
	for _, execution := range executions {
		label := execution.PipelineExecutionID
		if execution.Status != "" {
			label = fmt.Sprintf("%s (%s)", execution.PipelineExecutionID, execution.Status)
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: label,
			ID:   execution.PipelineExecutionID,
		})
	}

	return resources, nil
}
