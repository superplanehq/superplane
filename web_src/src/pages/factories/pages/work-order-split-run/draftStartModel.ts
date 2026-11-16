import { concreteClaudeModelId, displayModelName } from "@/lib/hostedLLMModels";
import { visibleThinkingLevelLabel } from "@/lib/thinkingLevel";
import { parseSelectableLLMModelKey, selectableLLMModelLabelFromKey } from "@/lib/selectableLLMModels";

export const DRAFT_START_MODEL_AUTO = "auto";

export type DraftStartSelection = { model?: string; thinkingLevel?: string };

function hasDraftStartOverrides(selection: DraftStartSelection): boolean {
  return selection.model !== undefined || selection.thinkingLevel !== undefined;
}

export function draftStartModelOptions(models: Array<{ id?: string; name?: string }> = [], defaultLabel = "Default") {
  const options = models
    .filter((item) => Boolean(item.id))
    .map((item) => ({
      value: item.id ?? "",
      label: selectableLLMModelLabelFromKey(item.name || item.id || ""),
      ...parseSelectableLLMModelKey(item.id ?? ""),
    }));
  return [
    { value: "", label: defaultLabel },
    ...options.map((option) => {
      const collisions = options.filter((other) => other.label === option.label);
      if (collisions.length === 1) return { value: option.value, label: option.label };
      const source = connectionLabel(option.source) || "Line runner";
      const needsProvider =
        option.provider && collisions.some((other) => other.value !== option.value && other.source === option.source);
      const qualifier = needsProvider ? `${source} · ${connectionLabel(option.provider)}` : source;
      return { value: option.value, label: `${option.label} (${qualifier})` };
    }),
  ];
}

function connectionLabel(id: string): string {
  const labels: Record<string, string> = {
    hosted: "Hosted",
    byok: "API key",
    subscription: "Subscription",
    openrouter: "OpenRouter",
    openai: "OpenAI",
    anthropic: "Anthropic",
  };
  return labels[id] ?? id;
}

function draftStartDefaultLabel(value?: string): string {
  return value ? `Default (${value})` : "Default";
}

export function draftStartDefaultLabels(
  defaults: DraftStartSelection | undefined,
  models: Array<{ id?: string; name?: string }> = [],
) {
  const resolved = defaults ? resolveDraftStartSelection({}, defaults, models) : undefined;
  return {
    model: draftStartDefaultLabel(resolved?.modelLabel),
    thinking: draftStartDefaultLabel(resolved?.thinkingLabel),
  };
}

export function resolveDraftStartSelection(
  selection: DraftStartSelection,
  defaults: DraftStartSelection,
  models: Array<{ id?: string; name?: string }> = [],
  defaultsStatus?: string,
) {
  const model = selection.model ?? defaults.model ?? "auto";
  const thinking = selection.thinkingLevel ?? defaults.thinkingLevel;
  const unresolvedModelStatus = selection.model === undefined ? defaultsStatus : undefined;
  const unresolvedThinkingStatus = selection.thinkingLevel === undefined ? defaultsStatus : undefined;
  const name = draftStartModelLabels(model, models, unresolvedModelStatus);
  const thinkingLabel =
    unresolvedThinkingStatus ?? visibleThinkingLevelLabel(thinking) ?? (thinking === "auto" ? "Auto" : "Default");
  const modelSource = selection.model === undefined ? " (Default)" : "";
  const thinkingSource = selection.thinkingLevel === undefined ? " (Default)" : "";
  const hasOnlyUnresolvedDefaults = Boolean(unresolvedModelStatus && unresolvedThinkingStatus);
  return {
    hasOnlyUnresolvedDefaults,
    hasOverrides: hasDraftStartOverrides(selection),
    isThinkingUnresolved: Boolean(unresolvedThinkingStatus),
    modelLabel: name,
    thinkingLabel,
    label: hasOnlyUnresolvedDefaults ? defaultsStatus : `${name}${modelSource} · ${thinkingLabel}${thinkingSource}`,
    accessibleLabel: hasOnlyUnresolvedDefaults
      ? defaultsStatus
      : `Model: ${name}${modelSource} · Thinking: ${thinkingLabel}${thinkingSource}`,
  };
}

function draftStartModelLabels(
  model: string,
  models: Array<{ id?: string; name?: string }>,
  unresolvedStatus?: string,
) {
  if (unresolvedStatus) return unresolvedStatus;
  if (model === "auto" || !model) return "Auto";
  return (
    draftStartModelOptions(models).find((item) => item.value === model)?.label || selectableLLMModelLabelFromKey(model)
  );
}

export function draftStartModelPayload(selected?: string): string | undefined {
  const trimmed = selected?.trim() ?? "";
  if (trimmed === "" || trimmed === DRAFT_START_MODEL_AUTO) {
    return undefined;
  }
  return trimmed;
}

/** Short label for a stored runner model id. Aliases use a versioned id from modelIds. */
export function displayRunnerModel(id: string, modelIds: readonly string[] = []): string {
  const trimmed = id.trim();
  if (trimmed === "") {
    return "";
  }
  if (trimmed.includes(" · ")) {
    return joinRunnerModels(trimmed.split(" · ").map((part) => displayRunnerModel(part, modelIds)));
  }
  return displayModelName(concreteClaudeModelId(trimmed, modelIds));
}

export function joinRunnerModels(ids: Array<string | undefined>): string {
  const unique = [...new Set(ids.map((id) => id?.trim() ?? "").filter(Boolean))];
  return unique.join(" · ");
}

type RunnerCanvasNode = {
  id?: string;
  component?: string;
  configuration?: { model?: unknown } | Record<string, unknown>;
};

const RUNNER_PROVIDER: Record<string, string> = {
  runnerClaudeCode: "anthropic",
  runnerCodex: "openai",
  runnerOpenRouter: "openrouter",
  runnerSuperPlane: "hosted",
};

export function runnerModelsFromCanvasNodes(nodes?: RunnerCanvasNode[]): string {
  return joinRunnerModels((nodes ?? []).map(runnerModelFromCanvasNode));
}

function runnerModelFromCanvasNode(node: RunnerCanvasNode): string | undefined {
  const model = node.configuration && "model" in node.configuration ? node.configuration.model : undefined;
  return typeof model === "string" ? model : undefined;
}

export function canvasNodesForRunnerModel(
  nodes?: RunnerCanvasNode[],
  statuses?: Record<string, string>,
): RunnerCanvasNode[] | undefined {
  if (!nodes || !statuses) {
    return nodes;
  }
  const ran = nodes.filter((node) => {
    const status = node.id ? statuses[node.id] : undefined;
    return Boolean(status && status !== "did_not_run");
  });
  if (ran.some((node) => runnerModelFromCanvasNode(node)?.trim())) {
    return ran;
  }
  return nodes;
}

function runnerAcceptsStartModel(component: string | undefined, model: string): boolean {
  const trimmed = model.trim();
  if (!trimmed) {
    return false;
  }
  const runner = RUNNER_PROVIDER[component ?? ""];
  if (!runner) {
    return false;
  }
  if (runner === "hosted" || runner === "openrouter") {
    return true;
  }
  const hint = startModelProvider(trimmed);
  return hint == null || hint === runner;
}

export function phaseWithRunnerModel<T extends { model?: string }>(phase: T, nodes?: RunnerCanvasNode[]): T {
  const start = phase.model?.trim();
  const fromCanvas = runnerModelsFromCanvasNodes(nodes);
  if (start && canvasRunnersAcceptStartModel(nodes, start)) {
    return phase;
  }
  if (fromCanvas) {
    return { ...phase, model: fromCanvas };
  }
  if (start) {
    return { ...phase, model: undefined };
  }
  return phase;
}

function canvasRunnersAcceptStartModel(nodes: RunnerCanvasNode[] | undefined, model: string): boolean {
  const runners = (nodes ?? []).filter((node) => RUNNER_PROVIDER[node.component ?? ""]);
  if (runners.length === 0) {
    return true;
  }
  return runners.some((node) => runnerAcceptsStartModel(node.component, model));
}

function startModelProvider(model: string): string | undefined {
  const id = model.toLowerCase();
  if (id.startsWith("anthropic/") || id.includes("claude")) {
    return "anthropic";
  }
  if (id.startsWith("openai/") || id.includes("codex") || /(^|\/)gpt-/.test(id) || /(^|\/)o[1-9]/.test(id)) {
    return "openai";
  }
  if (id.includes("/")) {
    return "openrouter";
  }
  return undefined;
}
