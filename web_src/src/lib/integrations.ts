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
