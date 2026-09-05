import { compareModelLabels, hostedLLMTechnicalName, parseHostedLLMModelKey } from "./hostedLLMModels";

export const SELECTABLE_LLM_SOURCE_HOSTED = "hosted";
export const SELECTABLE_LLM_SOURCE_BYOK = "byok";

export type SelectableLLMSourceID = typeof SELECTABLE_LLM_SOURCE_HOSTED | typeof SELECTABLE_LLM_SOURCE_BYOK;

export type SelectableLLMNamedID = {
  id: string;
  name: string;
};

export type SelectableLLMModel = {
  source: SelectableLLMNamedID;
  provider: SelectableLLMNamedID;
  model: SelectableLLMNamedID;
  key: string;
  label: string;
};

export function selectableLLMModelKey(source: string, provider: string, model: string): string {
  const trimmedSource = source.trim();
  const trimmedProvider = provider.trim();
  const trimmedModel = model.trim();
  if (trimmedSource === "" || trimmedProvider === "" || trimmedModel === "") {
    return "";
  }
  return `${trimmedSource}::${trimmedProvider}::${trimmedModel}`;
}

export function parseSelectableLLMModelKey(value: string): {
  source: string;
  provider: string;
  model: string;
} {
  const parts = value.split("::");
  if (parts.length < 3) {
    return { source: "", provider: "", model: "" };
  }
  return { source: parts[0] ?? "", provider: parts[1] ?? "", model: parts.slice(2).join("::") };
}

export function selectableLLMModelLabel(provider: string, model: string): string {
  return hostedLLMTechnicalName(provider, model);
}

export function selectableLLMModelLabelFromKey(value: string): string {
  const parsed = parseSelectableLLMModelKey(value);
  if (parsed.provider === "") {
    return value.trim();
  }
  return selectableLLMModelLabel(parsed.provider, parsed.model);
}

export function sortSelectableLLMModels(models: SelectableLLMModel[]): SelectableLLMModel[] {
  return models.slice().sort((left, right) => {
    const byLabel = compareModelLabels(left.label, right.label);
    if (byLabel !== 0) {
      return byLabel;
    }
    if (left.source.id !== right.source.id) {
      return left.source.id.localeCompare(right.source.id);
    }
    return left.key.localeCompare(right.key);
  });
}

export function selectableLLMModelsFromResponse(
  items: Array<{
    source?: { id?: string; name?: string };
    provider?: { id?: string; name?: string };
    model?: { id?: string; name?: string };
    key?: string;
    label?: string;
  }>,
  sources?: SelectableLLMSourceID[],
): SelectableLLMModel[] {
  const listed = items.flatMap((item) => {
    const mapped = selectableLLMModelFromResponse(item);
    return mapped ? [mapped] : [];
  });
  const filtered = sources
    ? listed.filter((item) => sources.includes(item.source.id as SelectableLLMSourceID))
    : listed;
  return sortSelectableLLMModels(filtered);
}

export function hostedSelectableLLMModelKey(provider: string, model: string): string {
  return selectableLLMModelKey(SELECTABLE_LLM_SOURCE_HOSTED, provider, model);
}

function selectableNamedID(value: { id?: string; name?: string } | undefined): SelectableLLMNamedID | undefined {
  const id = value?.id?.trim() ?? "";
  if (id === "") {
    return undefined;
  }
  return { id, name: value?.name?.trim() || id };
}

function selectableLLMModelFromResponse(item: {
  source?: { id?: string; name?: string };
  provider?: { id?: string; name?: string };
  model?: { id?: string; name?: string };
  key?: string;
  label?: string;
}): SelectableLLMModel | undefined {
  const key = item.key?.trim() ?? "";
  const source = selectableNamedID(item.source);
  const provider = selectableNamedID(item.provider);
  const model = selectableNamedID(item.model);
  if (key === "" || !source || !provider || !model) {
    return undefined;
  }
  return {
    source,
    provider,
    model,
    key,
    label: item.label?.trim() || model.id,
  };
}

export function resolveSelectableLLMModelKey(
  sessionKey: string,
  models: SelectableLLMModel[],
  instanceHostedKey = "",
): string {
  const session = sessionKey.trim();
  if (session !== "") {
    return session;
  }
  const instance = instanceHostedKey.trim();
  if (instance !== "" && models.some((model) => model.key === instance)) {
    return instance;
  }
  return "";
}

export function normalizeSuperPlaneModelValue(value: string): string {
  const trimmed = value.trim();
  if (trimmed === "") {
    return "";
  }
  const selectable = parseSelectableLLMModelKey(trimmed);
  if (selectable.source === SELECTABLE_LLM_SOURCE_HOSTED && selectable.provider !== "" && selectable.model !== "") {
    return selectableLLMModelKey(selectable.source, selectable.provider, selectable.model);
  }
  const hosted = parseHostedLLMModelKey(trimmed);
  if (hosted.provider === SELECTABLE_LLM_SOURCE_HOSTED || hosted.provider === SELECTABLE_LLM_SOURCE_BYOK) {
    return trimmed;
  }
  if (hosted.provider !== "" && hosted.model !== "") {
    return hostedSelectableLLMModelKey(hosted.provider, hosted.model);
  }
  return trimmed;
}
