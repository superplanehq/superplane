package codebuild

import (
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	Source                     = "aws.codebuild"
	DetailTypeBuildStateChange = "CodeBuild Build State Change"
)

type Project struct {
	ProjectName string `json:"projectName" mapstructure:"projectName"`
	ProjectArn  string `json:"projectArn" mapstructure:"projectArn"`
}

func validateProject(ctx core.TriggerContext, region string, projectRef string, existing *Project) (*Project, error) {
	return validateProjectWithIntegration(ctx.HTTP, ctx.Integration, region, projectRef, existing)
}

func projectMatchesRef(project *Project, projectRef string) bool {
	if project == nil {
		return false
	}
	return projectRef == project.ProjectName || projectRef == project.ProjectArn
}

func projectNameFromRef(projectRef string) (string, error) {
	projectRef = strings.TrimSpace(projectRef)
	if projectRef == "" {
		return "", fmt.Errorf("project is required")
	}

	if !strings.HasPrefix(projectRef, "arn:") {
		return projectRef, nil
	}

	parts := strings.SplitN(projectRef, "project/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("invalid project ARN: %s", projectRef)
	}

	return strings.TrimSpace(parts[1]), nil
}

func validateProjectWithIntegration(
	httpCtx core.HTTPContext,
	integration core.IntegrationContext,
	region string,
	projectRef string,
	existing *Project,
) (*Project, error) {
	projectRef = strings.TrimSpace(projectRef)
	if projectRef == "" {
		return nil, fmt.Errorf("project is required")
	}

	if existing != nil && projectMatchesRef(existing, projectRef) {
		return existing, nil
	}

	credentials, err := common.CredentialsFromInstallation(integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	projectName, err := projectNameFromRef(projectRef)
	if err != nil {
		return nil, err
	}

	client := NewClient(httpCtx, credentials, region)
	project, err := client.DescribeProject(projectName)
	if err != nil {
		var awsErr *common.Error
		if errors.As(err, &awsErr) && awsErr.Code == "ResourceNotFoundException" {
			return nil, fmt.Errorf("project not found: %s", projectName)
		}
		return nil, err
	}

	return project, nil
}
