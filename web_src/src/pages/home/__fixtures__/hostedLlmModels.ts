const STORYBOOK_HOSTED_MODELS: Record<string, string[]> = {
  anthropic: ["claude-sonnet-4-6"],
  openai: ["gpt-5"],
  openrouter: ["anthropic/claude-sonnet-4-6"],
};

const STORYBOOK_PROVIDER_NAMES: Record<string, string> = {
  anthropic: "Anthropic",
  openai: "OpenAI",
  openrouter: "OpenRouter",
};

/** SuperPlane-hosted allowlist used by setup and configuration-field stories. */
export function storybookHostedLlmModels(provider: string | null) {
  const ids = STORYBOOK_HOSTED_MODELS[provider ?? ""] ?? [];
  if (ids.length === 0) {
    return { enabled: false, models: [] as Array<{ id: string; name: string }> };
  }
  return { enabled: true, models: ids.map((id) => ({ id, name: id })) };
}

export function storybookSelectableLlmModels(byokByProvider?: Record<string, string[]>) {
  const models: Array<{
    source: { id: string; name: string };
    provider: { id: string; name: string };
    model: { id: string; name: string };
    key: string;
    label: string;
  }> = [];
  for (const [provider, ids] of Object.entries(STORYBOOK_HOSTED_MODELS)) {
    const providerName = STORYBOOK_PROVIDER_NAMES[provider] ?? provider;
    for (const id of ids) {
      models.push(selectableStoryModel("hosted", "SuperPlane", provider, providerName, id));
    }
    const byokIds = byokByProvider?.[provider] ?? ids;
    for (const id of byokIds) {
      models.push(selectableStoryModel("byok", "Your keys", provider, providerName, id));
    }
  }
  return models;
}

function selectableStoryModel(
  sourceId: string,
  sourceName: string,
  provider: string,
  providerName: string,
  id: string,
) {
  return {
    source: { id: sourceId, name: sourceName },
    provider: { id: provider, name: providerName },
    model: { id, name: id },
    key: `${sourceId}::${provider}::${id}`,
    label: provider === "openrouter" ? id : `${provider}/${id}`,
  };
}
