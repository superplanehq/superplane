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

const PREFERRED_MODEL_SUBSTRING: Record<string, string> = {
  anthropic: "sonnet",
  openai: "gpt-5",
  openrouter: "sonnet",
};

/** Prefer a known default from the provider allowlist; otherwise use the first id. */
export function pickHostedModel(provider: string, modelIds: string[]): string | undefined {
  const ids = uniqueSortedModelIds(modelIds);
  const preferred = PREFERRED_MODEL_SUBSTRING[provider];
  if (preferred) {
    const match = ids.find((id) => id.toLowerCase().includes(preferred));
    if (match) return match;
  }
  return ids[0];
}

/** Prefer a Sonnet id from the Anthropic hosted allowlist; otherwise use the first id. */
export function pickHostedAnthropicModel(modelIds: string[]): string | undefined {
  return pickHostedModel("anthropic", modelIds);
}

export function hostedModelIds(models: { id?: string | null }[] | undefined): string[] {
  return (models ?? []).map((model) => model.id ?? "").filter((id) => id !== "");
}
