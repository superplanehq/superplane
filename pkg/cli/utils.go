package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func getOneOrAnotherFlag(cmd *cobra.Command, flag1, flag2 string, required bool) string {
	flag1Value, _ := cmd.Flags().GetString(flag1)
	flag2Value, _ := cmd.Flags().GetString(flag2)

	if flag1Value != "" && flag2Value != "" {
		fmt.Fprintf(os.Stderr, "Error: cannot specify both --%s and --%s\n", flag1, flag2)
		os.Exit(1)
	}

	if flag1Value != "" {
		return flag1Value
	}

	if flag2Value != "" {
		return flag2Value
	}

	if required {
		fmt.Fprintf(os.Stderr, "Error: must specify either --%s or --%s\n", flag1, flag2)
		os.Exit(1)
	}

	return ""
}

func getDomainOrExit(client *openapi_client.APIClient, cmd *cobra.Command) (string, string) {
	response, _, err := client.MeAPI.MeMe(context.Background()).Execute()
	Check(err)

	return string(openapi_client.AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_ORGANIZATION), *response.OrganizationId
}

func findWorkflowIDByName(ctx context.Context, client *openapi_client.APIClient, name string) (string, error) {
	response, _, err := client.WorkflowAPI.WorkflowsListWorkflows(ctx).Execute()
	if err != nil {
		return "", err
	}

	var matches []openapi_client.WorkflowsWorkflow
	for _, workflow := range response.GetWorkflows() {
		if workflow.Metadata == nil || workflow.Metadata.Name == nil {
			continue
		}
		if *workflow.Metadata.Name == name {
			matches = append(matches, workflow)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("canvas %q not found", name)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("multiple canvases named %q found", name)
	}

	if matches[0].Metadata == nil || matches[0].Metadata.Id == nil {
		return "", fmt.Errorf("canvas %q is missing an id", name)
	}

	return *matches[0].Metadata.Id, nil
}
