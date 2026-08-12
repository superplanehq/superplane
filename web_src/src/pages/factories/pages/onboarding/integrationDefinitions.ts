import type { IntegrationsIntegrationDefinition } from "@/api-client";

import type { IntegrationId } from "./onboardingFixtures";

type ConfigField = NonNullable<IntegrationsIntegrationDefinition["configuration"]>[number];

function field(partial: Partial<ConfigField> & Pick<ConfigField, "name" | "label" | "type">): ConfigField {
  return {
    description: "",
    required: false,
    visibilityConditions: [],
    requiredConditions: [],
    sensitive: false,
    togglable: false,
    ...partial,
  };
}

function definition(
  name: string,
  label: string,
  description: string,
  configuration: ConfigField[],
): IntegrationsIntegrationDefinition {
  return {
    name,
    label,
    icon: name,
    description,
    configuration,
    instructions: "",
    // Keep create dialog self-contained (no capability wizard hand-off) in Storybook.
    legacySetupOnly: true,
  };
}

const DEFINITIONS: Record<IntegrationId, IntegrationsIntegrationDefinition> = {
  github: definition("github", "GitHub", "GitHub repositories, issues, and pull requests", [
    field({
      name: "organization",
      label: "Organization",
      type: "string",
      description: "Optional org to install into. Leave blank for your personal account.",
    }),
  ]),
  gitlab: definition("gitlab", "GitLab", "GitLab repositories, issues, and merge requests", [
    field({
      name: "token",
      label: "Access token",
      type: "string",
      required: true,
      sensitive: true,
      description: "Personal or group access token with api scope.",
    }),
  ]),
  claude: definition("claude", "Claude", "Use Claude / Claude Code in workflows", [
    field({
      name: "apiKey",
      label: "API Key",
      type: "string",
      required: true,
      sensitive: true,
      description: "Anthropic API key",
    }),
  ]),
  cursor: definition("cursor", "Cursor", "Use Cursor as a coding agent", [
    field({
      name: "apiKey",
      label: "API Key",
      type: "string",
      required: true,
      sensitive: true,
    }),
  ]),
  openai: definition("openai", "OpenAI", "Use OpenAI / Codex models", [
    field({
      name: "apiKey",
      label: "API Key",
      type: "string",
      required: true,
      sensitive: true,
    }),
  ]),
  linear: definition("linear", "Linear", "Import issues from Linear", [
    field({
      name: "apiKey",
      label: "API Key",
      type: "string",
      required: true,
      sensitive: true,
    }),
  ]),
  jira: definition("jira", "Jira", "Import issues from Jira", [
    field({
      name: "baseUrl",
      label: "Site URL",
      type: "string",
      required: true,
      description: "e.g. https://acme.atlassian.net",
    }),
    field({
      name: "email",
      label: "Email",
      type: "string",
      required: true,
    }),
    field({
      name: "apiToken",
      label: "API token",
      type: "string",
      required: true,
      sensitive: true,
    }),
  ]),
};

export function integrationDefinitionFor(id: IntegrationId): IntegrationsIntegrationDefinition {
  return DEFINITIONS[id];
}

export const STORYBOOK_ORG_ID = "3ee1aa47-3a60-4c1f-b645-0b9859ab91f8";
