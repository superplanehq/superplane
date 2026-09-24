export const HOSTED_MODEL_ALL_PROVIDERS = "all";
export const HOSTED_LLM_PROVIDERS = ["anthropic", "openai", "openrouter"] as const;

export function hostedLLMModelKey(provider: string, model: string): string {
  const trimmedProvider = provider.trim();
  const trimmedModel = model.trim();
  if (trimmedProvider === "" || trimmedModel === "") {
    return "";
  }
  return `${trimmedProvider}::${trimmedModel}`;
}

export function parseHostedLLMModelKey(value: string): { provider: string; model: string } {
  const separator = value.indexOf("::");
  if (separator < 0) {
    return { provider: "", model: "" };
  }
  return { provider: value.slice(0, separator), model: value.slice(separator + 2) };
}

export function hostedLLMTechnicalName(provider: string, model: string): string {
  const trimmedProvider = provider.trim();
  const trimmedModel = model.trim();
  if (trimmedProvider === "" || trimmedModel === "") {
    return trimmedModel;
  }
  if (trimmedProvider === "openrouter") {
    return trimmedModel;
  }
  return `${trimmedProvider}/${trimmedModel}`;
}

export function hostedLLMTechnicalNameFromKey(value: string): string {
  const parsed = parseHostedLLMModelKey(value);
  if (parsed.provider === "") {
    return value.trim();
  }
  return hostedLLMTechnicalName(parsed.provider, parsed.model);
}

export function compareModelLabels(a: string, b: string): number {
  return a.localeCompare(b, undefined, { numeric: true, sensitivity: "base" });
}

export function uniqueSortedModelIds(ids: string[]): string[] {
  const unique = Array.from(new Set(ids.map((id) => id.trim()).filter((id) => id !== "")));
  unique.sort(compareModelLabels);
  return unique;
}

export function filterModelIds(ids: string[], query: string): string[] {
  const needle = query.trim().toLowerCase();
  if (needle === "") {
    return ids;
  }

  return ids.filter((id) => id.toLowerCase().includes(needle));
}

const PREFERRED_MODEL_SUBSTRINGS: Record<string, readonly string[]> = {
  // Claude Opus 5.5 is published as claude-opus-5-5, and sometimes as claude-opus-5.5.
  anthropic: ["opus-5-5", "opus-5.5", "sonnet"],
  openai: ["gpt-5"],
  // Grok 4.7 is published as x-ai/grok-4.7.
  openrouter: ["grok-4.7", "sonnet"],
};

/** Prefer a known default from the provider allowlist; otherwise use the first id. */
export function pickHostedModel(provider: string, modelIds: string[]): string | undefined {
  const ids = uniqueSortedModelIds(modelIds);
  for (const preferred of PREFERRED_MODEL_SUBSTRINGS[provider] ?? []) {
    const match = ids.find((id) => id.toLowerCase().includes(preferred));
    if (match) return match;
  }
  return ids[0];
}

/**
 * Find the first allowlisted id that contains the hint. Returns undefined when
 * none does, so a caller can keep the model it already resolved rather than
 * fall back to an unrelated one.
 */
export function pickModelMatching(modelIds: string[], hint: string): string | undefined {
  const needle = hint.trim().toLowerCase();
  if (needle === "") return undefined;
  return uniqueSortedModelIds(modelIds).find((id) => id.toLowerCase().includes(needle));
}

/** Prefer Claude Opus 5.5, then a Sonnet id; otherwise use the first id. */
export function pickHostedAnthropicModel(modelIds: string[]): string | undefined {
  return pickHostedModel("anthropic", modelIds);
}

export function hostedModelIds(models: { id?: string | null }[] | undefined): string[] {
  return (models ?? []).map((model) => model.id ?? "").filter((id) => id !== "");
}
