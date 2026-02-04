package codeartifact

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func validateRepository(ctx core.IntegrationContext, http core.HTTPContext, region string, repository string) (*Repository, error) {
	credentials, err := common.CredentialsFromInstallation(ctx)
	if err != nil {
		return nil, err
	}

	client := NewClient(http, credentials, region)
	repositories, err := client.ListRepositories()
	if err != nil {
		return nil, err
	}

	for _, r := range repositories {
		if r.Name == repository || r.Arn == repository {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("repository not found: %s", repository)
}
