const STORYBOOK_HOSTED_MODELS: Record<string, string[]> = {
  anthropic: ["claude-sonnet-4-6"],
  openai: ["gpt-5"],
  openrouter: ["anthropic/claude-sonnet-4-6"],
};

/** SuperPlane-hosted allowlist used by setup and configuration-field stories. */
export function storybookHostedLlmModels(provider: string | null) {
  const ids = STORYBOOK_HOSTED_MODELS[provider ?? ""] ?? [];
  if (ids.length === 0) {
    return { enabled: false, models: [] as Array<{ id: string; name: string }> };
  }
  return { enabled: true, models: ids.map((id) => ({ id, name: id })) };
}
