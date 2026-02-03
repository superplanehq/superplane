package ecr

import (
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	Source                   = "aws.ecr"
	DetailTypeECRImageScan   = "ECR Image Scan"
	DetailTypeECRImageAction = "ECR Image Action"
)

type Repository struct {
	RepositoryName string `json:"repositoryName" mapstructure:"repositoryName"`
	RepositoryArn  string `json:"repositoryArn" mapstructure:"repositoryArn"`
}

func validateRepository(ctx core.TriggerContext, region string, repositoryRef string, existing *Repository) (*Repository, error) {
	return validateRepositoryWithIntegration(ctx.HTTP, ctx.Integration, region, repositoryRef, existing)
}

func repositoryMatchesRef(repository *Repository, repositoryRef string) bool {
	return repositoryRef == repository.RepositoryName || repositoryRef == repository.RepositoryArn
}

func repositoryNameFromRef(repositoryRef string) (string, error) {
	if !strings.HasPrefix(repositoryRef, "arn:") {
		return repositoryRef, nil
	}

	parts := strings.SplitN(repositoryRef, "repository/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("invalid repository ARN: %s", repositoryRef)
	}

	return parts[1], nil
}

func validateRepositoryWithIntegration(httpCtx core.HTTPContext, integration core.IntegrationContext, region string, repositoryRef string, existing *Repository) (*Repository, error) {
	repositoryRef = strings.TrimSpace(repositoryRef)
	if repositoryRef == "" {
		return nil, fmt.Errorf("repository is required")
	}

	if existing != nil && repositoryMatchesRef(existing, repositoryRef) {
		return existing, nil
	}

	credentials, err := common.CredentialsFromInstallation(integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(httpCtx, credentials, region)
	repositoryName, err := repositoryNameFromRef(repositoryRef)
	if err != nil {
		return nil, err
	}

	repository, err := client.DescribeRepository(repositoryName)
	if err != nil {
		var awsErr *common.Error
		if errors.As(err, &awsErr) && awsErr.Code == "RepositoryNotFoundException" {
			return nil, fmt.Errorf("repository not found: %s", repositoryName)
		}
		return nil, err
	}

	return repository, nil
}
