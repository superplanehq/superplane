import type {
  IntegrationSetupStepDefinition,
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
} from "@/api-client";

export function isCapabilityBasedIntegration(integration: OrganizationsIntegration) {
  if (!integration) return false;
  return integration.status?.legacySetup === false;
}

export function isCapabilityBasedIntegrationDefinition(integration: IntegrationsIntegrationDefinition) {
  return integration.legacySetupOnly === false;
}

export function usesHostedGitHubAppInstall(definition?: IntegrationsIntegrationDefinition): boolean {
  return definition?.name === "github" && definition.hostedAppInstall === true;
}

export function usesHostedJiraOAuth(definition?: IntegrationsIntegrationDefinition): boolean {
  return definition?.name === "jira" && definition.hostedAppInstall === true;
}

const HOSTED_JIRA_CREDENTIAL_FIELDS = ["clientId", "clientSecret"];

/** Hide Client ID and Client Secret when SuperPlane holds the Jira OAuth app. */
export function hiddenFieldsForHostedJira(
  definition: IntegrationsIntegrationDefinition | null | undefined,
  hiddenFieldNames: string[],
): string[] {
  if (!usesHostedJiraOAuth(definition ?? undefined)) {
    return hiddenFieldNames;
  }
  return [...hiddenFieldNames, ...HOSTED_JIRA_CREDENTIAL_FIELDS];
}

/** Show Create your own GitHub App beside hosted Connect. */
export function offersPrivateGitHubAppSetup(definition?: IntegrationsIntegrationDefinition): boolean {
  return usesHostedGitHubAppInstall(definition);
}

/** New setup wizard. Feature off uses the legacy Sync manifest instead. */
export function usesPrivateGitHubAppWizard(definition?: IntegrationsIntegrationDefinition): boolean {
  return offersPrivateGitHubAppSetup(definition) && isCapabilityBasedIntegrationDefinition(definition ?? {});
}

export function openRedirectPrompt(step: IntegrationSetupStepDefinition | null) {
  const redirectPrompt = step?.redirectPrompt;
  if (!redirectPrompt?.url) {
    return;
  }

  if (redirectPrompt.method?.toUpperCase() === "POST" && redirectPrompt.formFields) {
    const form = document.createElement("form");
    form.method = "POST";
    form.action = redirectPrompt.url;
    form.target = "_blank";
    form.style.display = "none";

    Object.entries(redirectPrompt.formFields).forEach(([key, value]) => {
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = key;
      input.value = String(value);
      form.appendChild(input);
    });

    document.body.appendChild(form);
    form.submit();
    document.body.removeChild(form);
    return;
  }

  window.open(redirectPrompt.url, "_blank");
}
