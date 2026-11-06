import { concreteClaudeModelId, displayModelName } from "@/lib/hostedLLMModels";
import { thinkingOptions, type ThinkingOption } from "@/lib/thinkingLevel";

export const DRAFT_START_MODEL_AUTO = "auto";

const DRAFT_START_THINKING_AGENT_SETTING: ThinkingOption = { value: "", label: "Agent setting" };
const DRAFT_START_THINKING_DEFAULT: ThinkingOption = { value: "default", label: "Default" };

export function draftStartModelPayload(selected: string): string | undefined {
  const trimmed = selected.trim();
  if (trimmed === "" || trimmed === DRAFT_START_MODEL_AUTO) {
    return undefined;
  }
  return trimmed;
}

/**
 * Thinking levels for the Start model. Auto keeps the model of each agent,
 * so it has no single set of levels and Start sends no thinking override.
 */
export function draftStartThinkingLevels(
  models: ReadonlyArray<{ id?: string; thinkingLevels?: string[] }> | undefined,
  selected: string,
): string[] {
  if (selected === DRAFT_START_MODEL_AUTO) {
    return [];
  }
  return models?.find((model) => model.id === selected)?.thinkingLevels ?? [];
}

/**
 * Agent setting inherits saved efforts. Default clears them for this run,
 * including models without effort control. Auto keeps each agent's settings.
 */
export function draftStartThinkingOptions(levels: ReadonlyArray<string> | undefined, model: string): ThinkingOption[] {
  if (model === DRAFT_START_MODEL_AUTO) {
    return [];
  }
  return [
    DRAFT_START_THINKING_AGENT_SETTING,
    DRAFT_START_THINKING_DEFAULT,
    ...thinkingOptions(levels, { withDefault: false }),
  ];
}

/** Keep only available Start choices, including the run-only Default. */
export function supportedDraftStartThinkingLevel(value: string, options: ReadonlyArray<ThinkingOption>): string {
  const level = value.trim().toLowerCase();
  return options.some((option) => option.value === level) ? level : "";
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
