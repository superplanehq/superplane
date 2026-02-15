package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// Root list command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List SuperPlane resources",
	Long:    `List multiple SuperPlane resources.`,
	Aliases: []string{"ls"},
}

var listCanvasesCmd = &cobra.Command{
	Use:     "canvas",
	Short:   "List canvases",
	Aliases: []string{"canvases"},
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := DefaultClient()
		ctx := context.Background()
		response, _, err := client.CanvasAPI.CanvasesListCanvases(ctx).Execute()
		Check(err)

		writer := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(writer, "ID\tNAME\tCREATED_AT")
		for _, canvas := range response.GetCanvases() {
			metadata := canvas.GetMetadata()
			createdAt := ""
			if metadata.HasCreatedAt() {
				createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\n", metadata.GetId(), metadata.GetName(), createdAt)
		}
		_ = writer.Flush()
	},
}

var listIntegrationsCmd = &cobra.Command{
	Use:   "integrations",
	Short: "List integrations",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := DefaultClient()
		ctx := context.Background()
		writer := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

		if !listIntegrationsConnected {
			response, _, err := client.IntegrationAPI.IntegrationsListIntegrations(ctx).Execute()
			Check(err)

			fmt.Fprintln(writer, "NAME\tLABEL\tDESCRIPTION")
			for _, integration := range response.GetIntegrations() {
				fmt.Fprintf(writer, "%s\t%s\t%s\n", integration.GetName(), integration.GetLabel(), integration.GetDescription())
			}
			_ = writer.Flush()
			return
		}

		me, _, err := client.MeAPI.MeMe(ctx).Execute()
		Check(err)
		if !me.HasOrganizationId() {
			Fail("organization id not found for authenticated user")
		}

		connectedResponse, _, err := client.OrganizationAPI.OrganizationsListIntegrations(ctx, me.GetOrganizationId()).Execute()
		Check(err)

		availableResponse, _, err := client.IntegrationAPI.IntegrationsListIntegrations(ctx).Execute()
		Check(err)

		integrationsByName := make(map[string]openapi_client.IntegrationsIntegrationDefinition, len(availableResponse.GetIntegrations()))
		for _, integration := range availableResponse.GetIntegrations() {
			integrationsByName[integration.GetName()] = integration
		}

		fmt.Fprintln(writer, "ID\tNAME\tINTEGRATION\tLABEL\tDESCRIPTION\tSTATE")
		for _, integration := range connectedResponse.GetIntegrations() {
			metadata := integration.GetMetadata()
			spec := integration.GetSpec()
			status := integration.GetStatus()
			integrationName := spec.GetIntegrationName()

			definition, found := integrationsByName[integrationName]
			label := ""
			description := ""
			if found {
				label = definition.GetLabel()
				description = definition.GetDescription()
			}

			fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\t%s\t%s\n",
				metadata.GetId(),
				metadata.GetName(),
				integrationName,
				label,
				description,
				status.GetState(),
			)
		}
		_ = writer.Flush()
	},
}

var listComponentsFrom string
var listIntegrationsConnected bool

var listComponentsCmd = &cobra.Command{
	Use:   "components",
	Short: "List components",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := DefaultClient()
		ctx := context.Background()
		components := []openapi_client.ComponentsComponent{}

		if listComponentsFrom == "" {
			response, _, err := client.ComponentAPI.ComponentsListComponents(ctx).Execute()
			Check(err)
			components = response.GetComponents()
		} else {
			integration, err := findIntegrationDefinitionByName(ctx, client, listComponentsFrom)
			Check(err)
			components = integration.GetComponents()
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(writer, "NAME\tLABEL\tDESCRIPTION")
		for _, component := range components {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", component.GetName(), component.GetLabel(), component.GetDescription())
		}
		_ = writer.Flush()
	},
}

var listTriggersFrom string

var listTriggersCmd = &cobra.Command{
	Use:   "triggers",
	Short: "List triggers",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := DefaultClient()
		ctx := context.Background()
		triggers := []openapi_client.TriggersTrigger{}

		if listTriggersFrom == "" {
			response, _, err := client.TriggerAPI.TriggersListTriggers(ctx).Execute()
			Check(err)
			triggers = response.GetTriggers()
		} else {
			integration, err := findIntegrationDefinitionByName(ctx, client, listTriggersFrom)
			Check(err)
			triggers = integration.GetTriggers()
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(writer, "NAME\tLABEL\tDESCRIPTION")
		for _, trigger := range triggers {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", trigger.GetName(), trigger.GetLabel(), trigger.GetDescription())
		}
		_ = writer.Flush()
	},
}

type integrationResourceListResponse struct {
	Resources []openapi_client.OrganizationsIntegrationResourceRef `json:"resources"`
}

var listIntegrationResourcesType string
var listIntegrationResourcesParameters string
var listIntegrationResourcesIntegrationID string

var listIntegrationResourcesCmd = &cobra.Command{
	Use:   "integration-resources",
	Short: "List integration resources",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if strings.TrimSpace(listIntegrationResourcesType) == "" {
			Fail("--type is required")
		}

		extraParameters, err := parseIntegrationResourceParametersFlag(listIntegrationResourcesParameters)
		Check(err)
		extraParameters["type"] = listIntegrationResourcesType

		client := DefaultClient()
		ctx := context.Background()
		me, _, err := client.MeAPI.MeMe(ctx).Execute()
		Check(err)
		if !me.HasOrganizationId() {
			Fail("organization id not found for authenticated user")
		}

		if strings.TrimSpace(listIntegrationResourcesIntegrationID) == "" {
			Fail("--integration-id is required")
		}

		integrationResponse, _, err := client.OrganizationAPI.
			OrganizationsDescribeIntegration(ctx, me.GetOrganizationId(), listIntegrationResourcesIntegrationID).
			Execute()
		Check(err)

		integration := integrationResponse.GetIntegration()
		metadata := integration.GetMetadata()
		spec := integration.GetSpec()

		config := NewClientConfig()
		writer := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(writer, "INTEGRATION_ID\tINTEGRATION_NAME\tINTEGRATION\tTYPE\tNAME\tID")

		response, err := listIntegrationResourcesRequest(
			ctx,
			config,
			me.GetOrganizationId(),
			metadata.GetId(),
			extraParameters,
		)
		Check(err)

		for _, resource := range response.Resources {
			fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\t%s\t%s\n",
				metadata.GetId(),
				metadata.GetName(),
				spec.GetIntegrationName(),
				resource.GetType(),
				resource.GetName(),
				resource.GetId(),
			)
		}

		_ = writer.Flush()
	},
}

func parseIntegrationResourceParametersFlag(raw string) (map[string]string, error) {
	parameters := map[string]string{}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return parameters, nil
	}

	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		trimmedPair := strings.TrimSpace(pair)
		if trimmedPair == "" {
			return nil, fmt.Errorf("invalid empty parameter in --parameters")
		}

		key, value, found := strings.Cut(trimmedPair, "=")
		if !found {
			return nil, fmt.Errorf("invalid parameter %q, expected key=value", trimmedPair)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			return nil, fmt.Errorf("invalid parameter %q, expected non-empty key and value", trimmedPair)
		}

		parameters[key] = value
	}

	return parameters, nil
}

func listIntegrationResourcesRequest(
	ctx context.Context,
	config *ClientConfig,
	organizationID string,
	integrationID string,
	parameters map[string]string,
) (*integrationResourceListResponse, error) {
	values := url.Values{}
	for key, value := range parameters {
		values.Set(key, value)
	}

	baseURL := strings.TrimRight(config.BaseURL, "/")
	endpoint := fmt.Sprintf(
		"%s/api/v1/organizations/%s/integrations/%s/resources",
		baseURL,
		url.PathEscape(organizationID),
		url.PathEscape(integrationID),
	)
	if encoded := values.Encode(); encoded != "" {
		endpoint = endpoint + "?" + encoded
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	if config.APIToken != "" {
		request.Header.Set("Authorization", "Bearer "+config.APIToken)
	}

	response, err := config.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= http.StatusMultipleChoices {
		errorPayload := struct {
			Message string `json:"message"`
		}{}
		_ = json.Unmarshal(body, &errorPayload)
		if errorPayload.Message != "" {
			return nil, fmt.Errorf(errorPayload.Message)
		}
		return nil, fmt.Errorf("failed to list integration resources: %s", response.Status)
	}

	payload := integrationResourceListResponse{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	return &payload, nil
}

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listCanvasesCmd)
	listCmd.AddCommand(listIntegrationsCmd)
	listCmd.AddCommand(listComponentsCmd)
	listCmd.AddCommand(listTriggersCmd)
	listCmd.AddCommand(listIntegrationResourcesCmd)

	listIntegrationsCmd.Flags().BoolVar(&listIntegrationsConnected, "connected", false, "list connected integrations for the authenticated organization")
	listComponentsCmd.Flags().StringVar(&listComponentsFrom, "from", "", "integration name")
	listTriggersCmd.Flags().StringVar(&listTriggersFrom, "from", "", "integration name")
	listIntegrationResourcesCmd.Flags().StringVar(&listIntegrationResourcesType, "type", "", "integration resource type")
	listIntegrationResourcesCmd.Flags().StringVar(&listIntegrationResourcesParameters, "parameters", "", "additional comma-separated query parameters (key=value,key2=value2)")
	listIntegrationResourcesCmd.Flags().StringVar(&listIntegrationResourcesIntegrationID, "integration-id", "", "connected integration id")
	_ = listIntegrationResourcesCmd.MarkFlagRequired("type")
	_ = listIntegrationResourcesCmd.MarkFlagRequired("integration-id")
}
