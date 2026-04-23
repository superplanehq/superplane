package integrations

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type setupTarget struct {
	name        *string
	integration *string
}

type setupInitCommand struct {
	setupTarget
	interactive *bool
}

type setupStatusCommand struct {
	setupTarget
}

type setupSubmitCommand struct {
	setupTarget
	stepName   *string
	stepInputs *string
}

func newSetupCommand(options core.BindOptions) *cobra.Command {
	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Manage integration setup flows",
	}

	var initName string
	var initIntegration string
	var initInteractive bool
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Create integration and output setup instructions",
		Args:  cobra.NoArgs,
	}
	initCmd.Flags().StringVar(&initName, "name", "", "connected integration name")
	initCmd.Flags().StringVar(&initIntegration, "integration", "", "integration definition name")
	initCmd.Flags().BoolVar(&initInteractive, "interactive", false, "complete the setup flow interactively after creating the integration")
	_ = initCmd.MarkFlagRequired("name")
	_ = initCmd.MarkFlagRequired("integration")
	core.Bind(
		initCmd,
		&setupInitCommand{
			setupTarget: setupTarget{name: &initName, integration: &initIntegration},
			interactive: &initInteractive,
		},
		options,
	)

	var statusName string
	var statusIntegration string
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show the current setup status for an existing integration",
		Args:  cobra.NoArgs,
	}
	statusCmd.Flags().StringVar(&statusName, "name", "", "connected integration name")
	statusCmd.Flags().StringVar(&statusIntegration, "integration", "", "integration definition name")
	_ = statusCmd.MarkFlagRequired("name")
	_ = statusCmd.MarkFlagRequired("integration")
	core.Bind(statusCmd, &setupStatusCommand{setupTarget: setupTarget{name: &statusName, integration: &statusIntegration}}, options)

	var submitName string
	var submitIntegration string
	var submitStepName string
	var submitStepInputs string
	submitCmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit the next setup step inputs for an integration",
		Args:  cobra.NoArgs,
	}
	submitCmd.Flags().StringVar(&submitName, "name", "", "connected integration name")
	submitCmd.Flags().StringVar(&submitIntegration, "integration", "", "integration definition name")
	submitCmd.Flags().StringVar(&submitStepName, "step-name", "", "step name returned in the previous next step")
	submitCmd.Flags().StringVar(&submitStepInputs, "step-inputs", "", "step inputs as JSON/YAML object or key=value,key2=value2")
	_ = submitCmd.MarkFlagRequired("name")
	_ = submitCmd.MarkFlagRequired("integration")
	_ = submitCmd.MarkFlagRequired("step-name")
	core.Bind(submitCmd, &setupSubmitCommand{setupTarget: setupTarget{name: &submitName, integration: &submitIntegration}, stepName: &submitStepName, stepInputs: &submitStepInputs}, options)

	setupCmd.AddCommand(initCmd)
	setupCmd.AddCommand(statusCmd)
	setupCmd.AddCommand(submitCmd)

	return setupCmd
}

func (c *setupInitCommand) Execute(ctx core.CommandContext) error {
	name, integrationName, err := c.setupTarget.values()
	if err != nil {
		return err
	}

	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	integration, err := createIntegrationForSetup(ctx, organizationID, name, integrationName)
	if err != nil {
		return err
	}

	isInteractive := c.interactive != nil && *c.interactive
	if isInteractive {
		if !ctx.Renderer.IsText() {
			return fmt.Errorf("--interactive requires text output")
		}

		return runInteractiveSetup(ctx, organizationID, integration)
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(integration)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderSetupStateText(stdout, integration)
	})
}

func (c *setupStatusCommand) Execute(ctx core.CommandContext) error {
	name, integrationName, err := c.setupTarget.values()
	if err != nil {
		return err
	}

	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	integration, err := findIntegrationForSetup(ctx, organizationID, name, integrationName)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(integration)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderSetupStateText(stdout, integration)
	})
}

func runInteractiveSetup(
	ctx core.CommandContext,
	organizationID string,
	integration openapi_client.OrganizationsIntegration,
) error {
	stdout := ctx.Cmd.OutOrStdout()
	reader := bufio.NewReader(ctx.Cmd.InOrStdin())
	metadata := integration.GetMetadata()

	_, _ = fmt.Fprintf(stdout, "New integration '%s' (%s) created\n\n", metadata.GetName(), metadata.GetId())

	for {
		step, hasNextStep := currentSetupStep(integration)
		if !hasNextStep {
			_, err := fmt.Fprintln(stdout, "Setup finished.")
			return err
		}

		if err := renderInteractiveSetupStep(stdout, step); err != nil {
			return err
		}

		stepInputs, err := promptSetupStepInputs(reader, stdout, step)
		if err != nil {
			return err
		}

		metadata := integration.GetMetadata()
		integration, err = submitSetupStep(ctx, organizationID, metadata.GetId(), step.GetName(), stepInputs)
		if err != nil {
			return err
		}

		_, _ = fmt.Fprintln(stdout)
	}
}

func renderInteractiveSetupStep(stdout io.Writer, step openapi_client.IntegrationSetupStepDefinition) error {
	stepTitle := strings.TrimSpace(step.GetLabel())
	if stepTitle == "" {
		stepTitle = strings.TrimSpace(step.GetName())
	}

	if stepTitle == "" {
		stepTitle = "Unknown step"
	}

	_, _ = fmt.Fprintf(stdout, "Next step: %s\n", stepTitle)

	inputSummary := formatInputsRequiredSummary(step)
	if inputSummary != "" {
		_, _ = fmt.Fprintf(stdout, "Inputs required: %s\n", inputSummary)
	}

	if step.HasInstructions() && strings.TrimSpace(step.GetInstructions()) != "" {
		renderedInstructions := core.RenderMarkdownForTerminal(step.GetInstructions())
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, renderedInstructions)
	}

	_, err := fmt.Fprintln(stdout)
	return err
}

func formatInputsRequiredSummary(step openapi_client.IntegrationSetupStepDefinition) string {
	inputs := step.GetInputs()
	if len(inputs) == 0 {
		return ""
	}

	formatted := make([]string, 0, len(inputs))
	for _, input := range inputs {
		label := strings.TrimSpace(input.GetLabel())
		if label == "" {
			label = strings.TrimSpace(input.GetName())
		}
		if label == "" {
			continue
		}

		// Some integrations use generic input labels like "API Token".
		// Use the step label to provide more context in interactive output.
		if strings.EqualFold(label, "API Token") {
			stepLabel := strings.TrimSpace(step.GetLabel())
			stepLabel = strings.TrimPrefix(stepLabel, "Enter ")
			stepLabel = strings.TrimPrefix(stepLabel, "enter ")
			stepLabel = strings.TrimSpace(stepLabel)
			if stepLabel != "" {
				label = stepLabel
			}
		}

		formatted = append(formatted, label)
	}

	return strings.Join(formatted, ", ")
}

func (c *setupSubmitCommand) Execute(ctx core.CommandContext) error {
	name, integrationName, err := c.setupTarget.values()
	if err != nil {
		return err
	}

	stepName := ""
	if c.stepName != nil {
		stepName = strings.TrimSpace(*c.stepName)
	}
	if stepName == "" {
		return fmt.Errorf("--step-name is required")
	}

	stepInputs, err := parseSetupStepInputs(c.stepInputs)
	if err != nil {
		return err
	}

	organizationID, err := core.ResolveOrganizationID(ctx)
	if err != nil {
		return err
	}

	integration, err := findIntegrationForSetup(ctx, organizationID, name, integrationName)
	if err != nil {
		return err
	}

	metadata := integration.GetMetadata()
	integration, err = submitSetupStep(ctx, organizationID, metadata.GetId(), stepName, stepInputs)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(integration)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderSetupStateText(stdout, integration)
	})
}

func (t setupTarget) values() (string, string, error) {
	if t.name == nil || strings.TrimSpace(*t.name) == "" {
		return "", "", fmt.Errorf("--name is required")
	}
	if t.integration == nil || strings.TrimSpace(*t.integration) == "" {
		return "", "", fmt.Errorf("--integration is required")
	}

	return strings.TrimSpace(*t.name), strings.TrimSpace(*t.integration), nil
}

func createIntegrationForSetup(
	ctx core.CommandContext,
	organizationID string,
	name string,
	integrationName string,
) (openapi_client.OrganizationsIntegration, error) {
	body := openapi_client.OrganizationsCreateIntegrationBody{}
	body.SetName(name)
	body.SetIntegrationName(integrationName)
	body.SetConfiguration(map[string]interface{}{})

	response, _, err := ctx.API.OrganizationAPI.
		OrganizationsCreateIntegration(ctx.Context, organizationID).
		Body(body).
		Execute()
	if err != nil {
		return openapi_client.OrganizationsIntegration{}, err
	}

	return response.GetIntegration(), nil
}

func findIntegrationForSetup(
	ctx core.CommandContext,
	organizationID string,
	name string,
	integrationName string,
) (openapi_client.OrganizationsIntegration, error) {
	response, _, err := ctx.API.OrganizationAPI.
		OrganizationsListIntegrations(ctx.Context, organizationID).
		Execute()
	if err != nil {
		return openapi_client.OrganizationsIntegration{}, err
	}

	for _, integration := range response.GetIntegrations() {
		metadata := integration.GetMetadata()
		if metadata.GetName() != name {
			continue
		}
		if metadata.GetIntegrationName() != integrationName {
			continue
		}

		detailedResponse, _, describeErr := ctx.API.OrganizationAPI.
			OrganizationsDescribeIntegration(ctx.Context, organizationID, metadata.GetId()).
			Execute()
		if describeErr != nil {
			return openapi_client.OrganizationsIntegration{}, describeErr
		}
		return detailedResponse.GetIntegration(), nil
	}

	return openapi_client.OrganizationsIntegration{}, fmt.Errorf("integration %q with definition %q not found", name, integrationName)
}

func submitSetupStep(
	ctx core.CommandContext,
	organizationID string,
	integrationID string,
	stepName string,
	stepInputs map[string]interface{},
) (openapi_client.OrganizationsIntegration, error) {
	if stepInputs == nil {
		stepInputs = map[string]interface{}{}
	}

	body := openapi_client.OrganizationsSubmitIntegrationSetupStepBody{}
	body.SetStepName(stepName)
	body.SetInputs(stepInputs)

	response, _, err := ctx.API.OrganizationAPI.
		OrganizationsSubmitIntegrationSetupStep(ctx.Context, organizationID, integrationID).
		Body(body).
		Execute()
	if err != nil {
		return openapi_client.OrganizationsIntegration{}, err
	}

	return response.GetIntegration(), nil
}

func renderSetupStateText(stdout io.Writer, integration openapi_client.OrganizationsIntegration) error {
	metadata := integration.GetMetadata()
	status := integration.GetStatus()

	_, _ = fmt.Fprintf(stdout, "Integration ID: %s\n", metadata.GetId())
	_, _ = fmt.Fprintf(stdout, "Name: %s\n", metadata.GetName())
	_, _ = fmt.Fprintf(stdout, "Integration: %s\n", metadata.GetIntegrationName())
	_, _ = fmt.Fprintf(stdout, "State: %s\n", status.GetState())
	if status.HasStateDescription() && strings.TrimSpace(status.GetStateDescription()) != "" {
		_, _ = fmt.Fprintf(stdout, "State Description: %s\n", status.GetStateDescription())
	}

	step, hasStep := currentSetupStep(integration)
	if !hasStep {
		_, err := fmt.Fprintln(stdout, "Next Step: none")
		return err
	}

	_, _ = fmt.Fprintf(stdout, "Next Step: %s\n", step.GetName())
	if step.HasLabel() && strings.TrimSpace(step.GetLabel()) != "" {
		_, _ = fmt.Fprintf(stdout, "Step Label: %s\n", step.GetLabel())
	}
	if step.HasType() {
		_, _ = fmt.Fprintf(stdout, "Step Type: %s\n", step.GetType())
	}

	if step.HasInstructions() && strings.TrimSpace(step.GetInstructions()) != "" {
		renderedInstructions := core.RenderMarkdownForTerminal(step.GetInstructions())

		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "Instructions:")
		_, _ = fmt.Fprintln(stdout, renderedInstructions)
	}

	if step.HasRedirectPrompt() {
		redirectPrompt := step.GetRedirectPrompt()
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "Redirect Prompt:")
		_, _ = fmt.Fprintf(stdout, "  Method: %s\n", redirectPrompt.GetMethod())
		_, _ = fmt.Fprintf(stdout, "  URL: %s\n", redirectPrompt.GetUrl())
		if redirectPrompt.HasFormFields() && len(redirectPrompt.GetFormFields()) > 0 {
			_, _ = fmt.Fprintln(stdout, "  Form Fields:")
			for key, value := range redirectPrompt.GetFormFields() {
				_, _ = fmt.Fprintf(stdout, "    %s=%s\n", key, value)
			}
		}
	}

	inputs := step.GetInputs()
	if len(inputs) == 0 {
		return nil
	}

	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Inputs:")
	for _, field := range inputs {
		requiredSuffix := ""
		if field.GetRequired() {
			requiredSuffix = ", required"
		}

		_, _ = fmt.Fprintf(stdout, "  %s (%s%s)\n", field.GetName(), field.GetType(), requiredSuffix)
		if field.HasLabel() && strings.TrimSpace(field.GetLabel()) != "" {
			_, _ = fmt.Fprintf(stdout, "    Label: %s\n", field.GetLabel())
		}
		if field.HasDescription() && strings.TrimSpace(field.GetDescription()) != "" {
			_, _ = fmt.Fprintf(stdout, "    Description: %s\n", field.GetDescription())
		}
		if field.HasDefaultValue() && strings.TrimSpace(field.GetDefaultValue()) != "" {
			_, _ = fmt.Fprintf(stdout, "    Default: %s\n", field.GetDefaultValue())
		}
		if options, hasOptions := getSelectOptions(field); field.GetType() == "select" && hasOptions {
			for _, option := range options {
				_, _ = fmt.Fprintf(stdout, "    Option: %s (%s)\n", option.GetLabel(), option.GetValue())
			}
		}
		if options, hasOptions := getMultiSelectOptions(field); field.GetType() == "multi-select" && hasOptions {
			for _, option := range options {
				_, _ = fmt.Fprintf(stdout, "    Option: %s (%s)\n", option.GetLabel(), option.GetValue())
			}
		}
	}

	return nil
}

func currentSetupStep(integration openapi_client.OrganizationsIntegration) (openapi_client.IntegrationSetupStepDefinition, bool) {
	status := integration.GetStatus()
	step, hasNextStep := status.GetNextStepOk()
	if !hasNextStep || step == nil {
		return openapi_client.IntegrationSetupStepDefinition{}, false
	}
	if strings.TrimSpace(step.GetName()) == "" {
		return openapi_client.IntegrationSetupStepDefinition{}, false
	}
	return *step, true
}

func promptSetupStepInputs(
	reader *bufio.Reader,
	stdout io.Writer,
	step openapi_client.IntegrationSetupStepDefinition,
) (map[string]interface{}, error) {
	if step.GetType() == openapi_client.INTEGRATIONSETUPSTEPDEFINITIONTYPE_REDIRECT_PROMPT {
		_, _ = fmt.Fprint(stdout, "Press Enter after completing the redirect step: ")
		if _, err := reader.ReadString('\n'); err != nil {
			return nil, fmt.Errorf("failed to read setup confirmation: %w", err)
		}
		return map[string]interface{}{}, nil
	}

	inputs := map[string]interface{}{}
	for _, field := range step.GetInputs() {
		name := strings.TrimSpace(field.GetName())
		if name == "" {
			continue
		}

		value, include, err := promptSetupFieldValue(reader, stdout, field)
		if err != nil {
			return nil, err
		}
		if include {
			inputs[name] = value
		}
	}

	return inputs, nil
}

func promptSetupFieldValue(
	reader *bufio.Reader,
	stdout io.Writer,
	field openapi_client.ConfigurationField,
) (interface{}, bool, error) {
	fieldName := strings.TrimSpace(field.GetName())
	if fieldName == "" {
		return nil, false, nil
	}

	promptLabel := strings.TrimSpace(field.GetLabel())
	if promptLabel == "" {
		promptLabel = fieldName
	}

	for {
		prompt := promptLabel
		if field.GetRequired() {
			prompt += " (required)"
		}
		if field.HasDefaultValue() && strings.TrimSpace(field.GetDefaultValue()) != "" {
			prompt += fmt.Sprintf(" [default: %s]", field.GetDefaultValue())
		}
		if field.GetSensitive() {
			prompt += " [input visible]"
		}

		if options, hasOptions := getSelectOptions(field); field.GetType() == "select" && hasOptions {
			_, _ = fmt.Fprintln(stdout, "Options:")
			for index, option := range options {
				_, _ = fmt.Fprintf(stdout, "  %d. %s (%s)\n", index+1, option.GetLabel(), option.GetValue())
			}
		}

		if options, hasOptions := getMultiSelectOptions(field); field.GetType() == "multi-select" && hasOptions {
			_, _ = fmt.Fprintln(stdout, "Options:")
			for index, option := range options {
				_, _ = fmt.Fprintf(stdout, "  %d. %s (%s)\n", index+1, option.GetLabel(), option.GetValue())
			}
			_, _ = fmt.Fprintln(stdout, "Enter comma-separated option values or indexes.")
		}

		_, _ = fmt.Fprintf(stdout, "%s: ", prompt)
		rawValue, err := reader.ReadString('\n')
		if err != nil {
			return nil, false, fmt.Errorf("failed to read input for %q: %w", fieldName, err)
		}

		rawValue = strings.TrimSpace(rawValue)
		if rawValue == "" {
			if field.HasDefaultValue() {
				parsedDefault, parseErr := parseInputByFieldType(field.GetType(), field.GetDefaultValue())
				if parseErr != nil {
					return nil, false, fmt.Errorf("invalid default value for %q: %w", fieldName, parseErr)
				}
				return parsedDefault, true, nil
			}

			if field.GetRequired() {
				_, _ = fmt.Fprintf(stdout, "%s is required\n", promptLabel)
				continue
			}

			return nil, false, nil
		}

		parsedValue, parseErr := parseInputByField(field, rawValue)
		if parseErr != nil {
			_, _ = fmt.Fprintf(stdout, "%s\n", parseErr.Error())
			continue
		}

		return parsedValue, true, nil
	}
}

func parseInputByField(field openapi_client.ConfigurationField, value string) (interface{}, error) {
	fieldType := field.GetType()
	if options, hasOptions := getSelectOptions(field); fieldType == "select" && hasOptions {
		return resolveSelectValue(options, value)
	}
	if options, hasOptions := getMultiSelectOptions(field); fieldType == "multi-select" && hasOptions {
		return resolveMultiSelectValues(options, value)
	}

	return parseInputByFieldType(fieldType, value)
}

func getSelectOptions(field openapi_client.ConfigurationField) ([]openapi_client.ConfigurationSelectOption, bool) {
	typeOptions, hasTypeOptions := field.GetTypeOptionsOk()
	if !hasTypeOptions || typeOptions == nil {
		return nil, false
	}

	selectOptions, hasSelect := typeOptions.GetSelectOk()
	if !hasSelect || selectOptions == nil {
		return nil, false
	}

	return selectOptions.GetOptions(), true
}

func getMultiSelectOptions(field openapi_client.ConfigurationField) ([]openapi_client.ConfigurationSelectOption, bool) {
	typeOptions, hasTypeOptions := field.GetTypeOptionsOk()
	if !hasTypeOptions || typeOptions == nil {
		return nil, false
	}

	multiSelectOptions, hasMultiSelect := typeOptions.GetMultiSelectOk()
	if !hasMultiSelect || multiSelectOptions == nil {
		return nil, false
	}

	return multiSelectOptions.GetOptions(), true
}

func parseInputByFieldType(fieldType string, value string) (interface{}, error) {
	switch fieldType {
	case "boolean":
		boolValue, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(value)))
		if err != nil {
			return nil, fmt.Errorf("expected a boolean value (true/false)")
		}
		return boolValue, nil
	case "number":
		if integerValue, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
			return integerValue, nil
		}

		numberValue, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return nil, fmt.Errorf("expected a numeric value")
		}
		return numberValue, nil
	case "multi-select":
		if strings.TrimSpace(value) == "" {
			return []string{}, nil
		}

		rawItems := strings.Split(value, ",")
		items := make([]string, 0, len(rawItems))
		for _, rawItem := range rawItems {
			item := strings.TrimSpace(rawItem)
			if item != "" {
				items = append(items, item)
			}
		}
		return items, nil
	default:
		return value, nil
	}
}

func resolveSelectValue(options []openapi_client.ConfigurationSelectOption, input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", nil
	}

	if index, err := strconv.Atoi(input); err == nil {
		if index < 1 || index > len(options) {
			return "", fmt.Errorf("option must be between 1 and %d", len(options))
		}
		return options[index-1].GetValue(), nil
	}

	for _, option := range options {
		if option.GetValue() == input || option.GetLabel() == input {
			return option.GetValue(), nil
		}
	}

	return "", fmt.Errorf("invalid option %q", input)
}

func resolveMultiSelectValues(options []openapi_client.ConfigurationSelectOption, input string) ([]string, error) {
	rawItems := strings.Split(input, ",")
	values := make([]string, 0, len(rawItems))

	for _, rawItem := range rawItems {
		item := strings.TrimSpace(rawItem)
		if item == "" {
			continue
		}

		value, err := resolveSelectValue(options, item)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}

	return values, nil
}

func parseSetupStepInputs(raw *string) (map[string]interface{}, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return map[string]interface{}{}, nil
	}

	trimmed := strings.TrimSpace(*raw)

	parsedJSON := map[string]interface{}{}
	if err := json.Unmarshal([]byte(trimmed), &parsedJSON); err == nil {
		return parsedJSON, nil
	}

	parsedYAML := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(trimmed), &parsedYAML); err == nil {
		return parsedYAML, nil
	}

	if strings.Contains(trimmed, "=") {
		parsedKV, err := parseStepInputsKeyValue(trimmed)
		if err == nil {
			return parsedKV, nil
		}
	}

	return nil, fmt.Errorf("invalid --step-inputs, expected JSON/YAML object or key=value,key2=value2")
}

func parseStepInputsKeyValue(raw string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		trimmedPair := strings.TrimSpace(pair)
		if trimmedPair == "" {
			continue
		}

		key, value, found := strings.Cut(trimmedPair, "=")
		if !found {
			return nil, fmt.Errorf("invalid step input %q, expected key=value", trimmedPair)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid empty key in step inputs")
		}

		value = strings.TrimSpace(value)
		parsed, err := parseLooseInputValue(value)
		if err != nil {
			return nil, fmt.Errorf("invalid value for %q: %w", key, err)
		}
		result[key] = parsed
	}

	return result, nil
}

func parseLooseInputValue(raw string) (interface{}, error) {
	if raw == "" {
		return "", nil
	}

	if strings.EqualFold(raw, "null") {
		return nil, nil
	}

	if strings.EqualFold(raw, "true") || strings.EqualFold(raw, "false") {
		boolValue, err := strconv.ParseBool(strings.ToLower(raw))
		if err != nil {
			return nil, err
		}
		return boolValue, nil
	}

	if integerValue, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return integerValue, nil
	}

	if floatValue, err := strconv.ParseFloat(raw, 64); err == nil {
		return floatValue, nil
	}

	if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "\"") {
		var parsed interface{}
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			return parsed, nil
		}
	}

	return raw, nil
}
