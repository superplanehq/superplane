const BYOK_PROVIDER_PRODUCT_NAMES: Record<string, string> = {
  anthropic: "Claude",
  openai: "OpenAI",
  openrouter: "OpenRouter",
};

export function byokProviderProductName(provider: string): string {
  return BYOK_PROVIDER_PRODUCT_NAMES[provider] ?? provider;
}

export function disconnectedProviderMessage(provider: string): string {
  return `Connect ${byokProviderProductName(provider)} on Integrations, then select models.`;
}

export const ORGANIZATION_LLM_MODELS_COPY = {
  pageTitle: "LLM Models",
  pageSubtitle: "Select models from connected providers. Selected models become available as Your keys in workspaces.",
  landingTitle: "Connect a model provider",
  landingDescription: "Connect Claude, OpenAI, or OpenRouter on Integrations first.",
  landingAction: "Open Integrations",
  loading: "Loading models...",
  listError: "Unable to list models from the connected key.",
  emptyCatalog: "No models are available from the connected key.",
  save: "Save models",
  saving: "Saving...",
  saveSuccess: "Selected models saved.",
  saveError: "Unable to save selected models.",
  noPermission: "You do not have permission to update organization models.",
  disconnectedLink: "Open Integrations",
} as const;

export function shouldShowLLMModelsLandingBanner(
  queries: Array<{ isLoading: boolean; error: unknown; data?: { connected?: boolean } }>,
): boolean {
  if (queries.some((query) => query.isLoading)) {
    return false;
  }
  if (queries.some((query) => query.error != null || query.data?.connected)) {
    return false;
  }
  return queries.length > 0 && queries.every((query) => query.data?.connected === false);
}
