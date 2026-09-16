import type { ConfigurationField } from "@/api-client";
import { HOSTED_MODEL_ALL_PROVIDERS } from "@/lib/hostedLLMModels";

const SUPERPLANE_AGENT_COMPONENT = "runnerSuperPlane";

const select = (options: Array<{ label: string; value: string }>) => ({
  select: { options },
});

const sourceField = (label: string): ConfigurationField => ({
  name: "source",
  label: "Source",
  type: "select",
  required: true,
  description: label,
  typeOptions: select([
    { label: "Integration", value: "integration" },
    { label: "Secret", value: "secret" },
  ]),
});

const integrationField = (description: string): ConfigurationField => ({
  name: "integration",
  label: "Integration",
  type: "select",
  required: false,
  description,
  visibilityConditions: [{ field: "source", values: ["integration"] }],
  requiredConditions: [{ field: "source", values: ["integration"] }],
  typeOptions: select([
    { label: "claude-superplane-apps", value: "claude-superplane-apps" },
    { label: "github-superplanehq", value: "github-superplanehq" },
  ]),
});

const environmentVariableSchema: ConfigurationField[] = [
  {
    name: "name",
    label: "Name",
    type: "string",
    required: true,
    description: "Environment variable name (letters, numbers, underscore)",
  },
  {
    name: "valueSource",
    label: "Value source",
    type: "select",
    required: true,
    description: "Where this variable value comes from",
    typeOptions: select([
      { label: "Literal value", value: "literal" },
      { label: "Secret key", value: "secret" },
    ]),
  },
  {
    name: "value",
    label: "Value",
    type: "string",
    required: false,
    description: "Literal value. Supports expressions such as {{ previous().data.author.email }}",
    visibilityConditions: [{ field: "valueSource", values: ["literal"] }],
    requiredConditions: [{ field: "valueSource", values: ["literal"] }],
  },
  {
    name: "secret",
    label: "Secret key",
    type: "string",
    required: false,
    description: "Stored credential key to use as the variable value",
    visibilityConditions: [{ field: "valueSource", values: ["secret"] }],
    requiredConditions: [{ field: "valueSource", values: ["secret"] }],
  },
];

export const PLANNING_REVIEW_RUNNER_FIELDS: ConfigurationField[] = [
  {
    name: "machineType",
    label: "Environment",
    type: "select",
    required: true,
    typeOptions: select([
      { label: "e1-large-amd64", value: "e1-large-amd64" },
      { label: "e1-large-arm64", value: "e1-large-arm64" },
      { label: "e1-tiny-amd64", value: "e1-tiny-amd64" },
      { label: "e1-tiny-arm64", value: "e1-tiny-arm64" },
    ]),
  },
  {
    name: "credentials",
    label: "Credentials",
    type: "object",
    required: true,
    description: "Anthropic API key, Claude integration, or SuperPlane-hosted credentials.",
    typeOptions: {
      object: {
        schema: [sourceField("Where the credentials come from"), integrationField("Name of the Claude integration")],
      },
    },
  },
  {
    name: "model",
    label: "Model",
    type: "select",
    required: false,
    description: "Claude model used for prompt steps.",
    typeOptions: select([
      { label: "Claude Sonnet", value: "sonnet" },
      { label: "Claude Opus", value: "opus" },
      { label: "Claude Haiku", value: "haiku" },
    ]),
  },
  {
    name: "workingDirectory",
    label: "Working directory",
    type: "string",
    required: false,
    description: "Optional starting directory.",
    placeholder: "/tmp/repo",
  },
  {
    name: "environmentFrom",
    label: "Environment from",
    type: "list",
    required: false,
    description: "Import environment variables from connected integrations or organization secrets",
    typeOptions: {
      list: {
        itemLabel: "Source",
        itemDefinition: {
          type: "object",
          schema: [
            sourceField("Where imported environment variables come from"),
            integrationField("Name of the integration"),
          ],
        },
      },
    },
  },
  {
    name: "environment",
    label: "Environment variables",
    type: "list",
    required: false,
    description: "Optional key/value pairs passed into the agent environment (in addition to ANTHROPIC_API_KEY)",
    typeOptions: {
      list: {
        itemLabel: "Variable",
        itemDefinition: { type: "object", schema: environmentVariableSchema },
      },
    },
  },
  {
    name: "executionTimeoutSeconds",
    label: "Execution timeout (seconds)",
    type: "number",
    required: false,
    description: "Hard time limit for the whole task, including all steps. Defaults to 3600 seconds (1 hour).",
    typeOptions: { number: { min: 0, max: 86_400 } },
  },
];

const CLAUDE_MODEL_FIELD = PLANNING_REVIEW_RUNNER_FIELDS.find((field) => field.name === "model");

/** Same field Run SuperPlane Agent uses on the automation canvas. */
const SUPERPLANE_AGENT_MODEL_FIELD: ConfigurationField = {
  name: "model",
  label: "Model",
  type: "hosted-model",
  required: false,
  description:
    "Select a SuperPlane-hosted model. The instance SuperPlane agent model is used when you do not select one.",
  placeholder: "Instance SuperPlane agent model",
  typeOptions: { hostedModel: { provider: HOSTED_MODEL_ALL_PROVIDERS } },
};

function withModelUsedLabel(field: ConfigurationField): ConfigurationField {
  return { ...field, label: "Model used", description: "" };
}

/** Model picker for the agent editor. SuperPlane agents use the hosted allowlist. */
export function planningReviewModelUsedField(
  componentName: string | undefined,
  catalogFields?: ConfigurationField[],
): ConfigurationField | undefined {
  const catalogModel = catalogFields?.find((field) => field.name === "model");
  if (catalogModel) {
    return withModelUsedLabel(catalogModel);
  }
  if (componentName === SUPERPLANE_AGENT_COMPONENT) {
    return withModelUsedLabel(SUPERPLANE_AGENT_MODEL_FIELD);
  }
  return CLAUDE_MODEL_FIELD ? withModelUsedLabel(CLAUDE_MODEL_FIELD) : undefined;
}
