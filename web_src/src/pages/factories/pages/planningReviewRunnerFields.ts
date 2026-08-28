import type { ConfigurationField } from "@/api-client";

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

export const PLANNING_REVIEW_ENVIRONMENT_MODEL_FIELDS = ["machineType", "model"];

export const PLANNING_REVIEW_ADVANCED_FIELD_NAMES = [
  "workingDirectory",
  "credentials",
  "environmentFrom",
  "environment",
  "executionTimeoutSeconds",
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
