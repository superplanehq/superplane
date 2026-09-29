const BYOK_PROVIDER_PRODUCT_NAMES: Record<string, string> = {
  anthropic: "Claude",
  openai: "OpenAI",
  openrouter: "OpenRouter",
};

export function byokProviderProductName(provider: string): string {
  return BYOK_PROVIDER_PRODUCT_NAMES[provider] ?? provider;
}

export function providerKeyHeading(provider: string): string {
  return `Your ${byokProviderProductName(provider)} key`;
}

export function providerKeyIntegrationLink(provider: string): string {
  return `Open ${byokProviderProductName(provider)} integration`;
}

export const ORGANIZATION_LLM_MODELS_COPY = {
  pageTitle: "LLM Models",
  pageSubtitle: "Select the models that agents in this workspace can use.",
  modelsTitle: "Models",
  hostedModelsHelper: "An installation admin selects the SuperPlane-hosted models.",
  hostedModelsEmpty: "No SuperPlane-hosted models are available.",
  ownKeyHelper: "Agents in this workspace run with this key. Your provider bills these runs.",
  hostedBadge: "SuperPlane",
  ownKeyBadge: "Your key",
  noProviderTitle: "Connect a model provider",
  noProviderDescription: "Connect Claude, OpenAI, or OpenRouter on Integrations. All models from the key are enabled.",
  noProviderAction: "Open Integrations",
  loading: "Loading models...",
  listError: "Unable to list models from the connected key.",
  emptyCatalog: "No models are available from the connected key.",
  save: "Save models",
  saving: "Saving...",
  saveSuccess: "Models saved.",
  saveError: "Unable to save models.",
  noPermission: "You do not have permission to update organization models.",
} as const;
