package integrations

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type integrationResourceListResponse struct {
	Resources []openapi_client.OrganizationsIntegrationResourceRef `json:"resources"`
}

type listResourcesCommand struct {
	integrationID *string
	resourceType  *string
	parameters    *string
}

func (c *listResourcesCommand) Execute(ctx core.CommandContext) error {
	if c.integrationID == nil || strings.TrimSpace(*c.integrationID) == "" {
		return fmt.Errorf("--id is required")
	}
	if c.resourceType == nil || strings.TrimSpace(*c.resourceType) == "" {
		return fmt.Errorf("--type is required")
	}

	extraParameters, err := parseIntegrationResourceParametersFlag(*c.parameters)
	if err != nil {
		return err
	}
	extraParameters["type"] = *c.resourceType

	me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return err
	}
	if !me.HasOrganizationId() {
		return fmt.Errorf("organization id not found for authenticated user")
	}

	integrationResponse, _, err := ctx.API.OrganizationAPI.
		OrganizationsDescribeIntegration(ctx.Context, me.GetOrganizationId(), *c.integrationID).
		Execute()
	if err != nil {
		return err
	}

	integration := integrationResponse.GetIntegration()
	metadata := integration.GetMetadata()
	spec := integration.GetSpec()

	response, err := listIntegrationResourcesRequest(
		ctx,
		me.GetOrganizationId(),
		metadata.GetId(),
		extraParameters,
	)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response.Resources)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "INTEGRATION_ID\tINTEGRATION_NAME\tINTEGRATION\tTYPE\tNAME\tID")
		for _, resource := range response.Resources {
			_, _ = fmt.Fprintf(
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
		return writer.Flush()
	})
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
	ctx core.CommandContext,
	organizationID string,
	integrationID string,
	parameters map[string]string,
) (*integrationResourceListResponse, error) {
	config := ctx.API.GetConfig()
	if config == nil {
		return nil, fmt.Errorf("api client config is required")
	}

	baseURL, err := config.ServerURLWithContext(ctx.Context, "OrganizationAPIService.OrganizationsListIntegrationResources")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("api_url is required")
	}

	values := url.Values{}
	for key, value := range parameters {
		values.Set(key, value)
	}

	baseURL = strings.TrimRight(baseURL, "/")
	endpoint := fmt.Sprintf(
		"%s/api/v1/organizations/%s/integrations/%s/resources",
		baseURL,
		url.PathEscape(organizationID),
		url.PathEscape(integrationID),
	)
	if encoded := values.Encode(); encoded != "" {
		endpoint = endpoint + "?" + encoded
	}

	request, err := http.NewRequestWithContext(ctx.Context, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")

	if authorization := strings.TrimSpace(config.DefaultHeader["Authorization"]); authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	response, err := httpClient.Do(request)
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
			return nil, errors.New(errorPayload.Message)
		}
		return nil, fmt.Errorf("failed to list integration resources: %s", response.Status)
	}

	payload := integrationResourceListResponse{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	return &payload, nil
}
