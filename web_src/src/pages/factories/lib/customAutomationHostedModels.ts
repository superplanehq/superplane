import { HOSTED_LLM_PROVIDERS, hostedLLMModelKey, parseHostedLLMModelKey } from "@/lib/hostedLLMModels";
import {
  SELECTABLE_LLM_SOURCE_BYOK,
  SELECTABLE_LLM_SOURCE_HOSTED,
  SELECTABLE_LLM_SOURCE_SUBSCRIPTION,
  type SelectableLLMModel,
} from "@/lib/selectableLLMModels";

export type AutomationHostedModels = {
  models: SelectableLLMModel[];
  savedKey?: string;
};

const namespaces = new Set<string>([
  ...HOSTED_LLM_PROVIDERS,
  SELECTABLE_LLM_SOURCE_HOSTED,
  SELECTABLE_LLM_SOURCE_BYOK,
  SELECTABLE_LLM_SOURCE_SUBSCRIPTION,
]);

function hasModelNamespace(value: string): boolean {
  return namespaces.has(parseHostedLLMModelKey(value).provider.trim().toLowerCase());
}

function qualifiedModel(value: string): string {
  const compact = value.trim().replace(/^hosted\s*::/i, "");
  const { provider, model } = parseHostedLLMModelKey(compact);
  return hostedLLMModelKey(provider.toLowerCase(), model) || value;
}

export function hostedModelForPicker(value: string | undefined, hosted?: AutomationHostedModels): string | undefined {
  if (!value || !hosted) return value;
  const id = value.trim();
  if (hasModelNamespace(id)) return qualifiedModel(id);

  // A saved bare ID keeps its original provider even if the workspace catalog changes.
  const saved = qualifiedModel(hosted.savedKey ?? "");
  if (saved && parseHostedLLMModelKey(saved).model === id) return saved;
  const matches = hosted.models.filter(({ model }) => model.id === id);
  return matches.length === 1 ? qualifiedModel(matches[0]!.key) : id;
}

export function hostedModelForYaml(value: unknown, hosted?: AutomationHostedModels): string | undefined {
  if (typeof value !== "string" || !value) return undefined;
  if (!hosted) return value;
  const qualified = qualifiedModel(value);
  const selected = hosted.models.find(({ key }) => key === value);
  if (!selected) return qualified;
  const id = selected.model.id;
  const shared = hosted.models.some((other) => other.model.id === id && other.key !== value);
  const saved = parseHostedLLMModelKey(qualifiedModel(hosted.savedKey ?? ""));
  const switchesProvider = saved.model === id && saved.provider !== selected.provider.id;
  return shared || switchesProvider || hasModelNamespace(id) ? qualified : id;
}
