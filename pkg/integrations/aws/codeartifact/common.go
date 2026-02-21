package codeartifact

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

/*
 * Package formats supported by AWS CodeArtifact.
 * See: https://docs.aws.amazon.com/codeartifact/latest/APIReference/API_DescribePackageVersion.html
 */
var PackageFormatOptions = []configuration.FieldOption{
	{Label: "npm", Value: "npm"},
	{Label: "pypi", Value: "pypi"},
	{Label: "maven", Value: "maven"},
	{Label: "nuget", Value: "nuget"},
	{Label: "generic", Value: "generic"},
	{Label: "ruby", Value: "ruby"},
	{Label: "swift", Value: "swift"},
	{Label: "cargo", Value: "cargo"},
}

/*
 * Package version statuses supported by AWS CodeArtifact.
 */
var PackageVersionStatusOptions = []configuration.FieldOption{
	{Value: "Published", Label: "Published"},
	{Value: "Unfinished", Label: "Unfinished"},
	{Value: "Unlisted", Label: "Unlisted"},
	{Value: "Archived", Label: "Archived"},
	{Value: "Disposed", Label: "Disposed"},
}

/*
 * CodeArtifact is only available in the following regions.
 * See: https://docs.aws.amazon.com/general/latest/gr/codeartifact.html
 */
var RegionsForCodeArtifact = []configuration.FieldOption{
	{
		Label: "us-east-1",
		Value: "us-east-1",
	},
	{
		Label: "us-east-2",
		Value: "us-east-2",
	},
	{
		Label: "us-west-2",
		Value: "us-west-2",
	},
	{
		Label: "ap-south-1",
		Value: "ap-south-1",
	},
	{
		Label: "ap-southeast-1",
		Value: "ap-southeast-1",
	},
	{
		Label: "ap-southeast-2",
		Value: "ap-southeast-2",
	},
	{
		Label: "ap-northeast-1",
		Value: "ap-northeast-1",
	},
	{
		Label: "eu-central-1",
		Value: "eu-central-1",
	},
	{
		Label: "eu-west-1",
		Value: "eu-west-1",
	},
	{
		Label: "eu-west-2",
		Value: "eu-west-2",
	},
	{
		Label: "eu-south-1",
		Value: "eu-south-1",
	},
	{
		Label: "eu-west-3",
		Value: "eu-west-3",
	},
	{
		Label: "eu-north-1",
		Value: "eu-north-1",
	},
}

func validateRepository(ctx core.IntegrationContext, http core.HTTPContext, region string, domain string, repository string) (*Repository, error) {
	credentials, err := common.CredentialsFromInstallation(ctx)
	if err != nil {
		return nil, err
	}

	client := NewClient(http, credentials, region)
	repositories, err := client.ListRepositories(domain)
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

/*
 * Parse a comma separated list of versions into a slice of strings.
 */
func parseVersionsList(s string) []string {
	versions := []string{}
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			versions = append(versions, v)
		}
	}

	return versions
}
