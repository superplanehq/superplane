export const DRAFT_START_MODEL_AUTO = "auto";

export const DRAFT_START_MODEL_HELP = [
  "Auto uses the default model on each automation.",
  "Pick a model to overwrite that default for this start.",
] as const;

export function draftStartModelPayload(selected: string): string | undefined {
  const trimmed = selected.trim();
  if (trimmed === "" || trimmed === DRAFT_START_MODEL_AUTO) {
    return undefined;
  }
  return trimmed;
}

/** Short label for a stored runner model id. */
export function displayRunnerModel(id: string): string {
  const trimmed = id.trim();
  if (trimmed === "") {
    return "";
  }
  const slash = trimmed.lastIndexOf("/");
  if (slash >= 0 && slash < trimmed.length - 1) {
    return trimmed.slice(slash + 1);
  }
  return trimmed;
}

export function joinRunnerModels(ids: Array<string | undefined>): string {
  const unique = [...new Set(ids.map((id) => id?.trim() ?? "").filter(Boolean))];
  return unique.join(" · ");
}

type RunnerCanvasNode = {
  component?: string;
  configuration?: { model?: unknown } | Record<string, unknown>;
};

const RUNNER_PROVIDER: Record<string, string> = {
  runnerClaudeCode: "anthropic",
  runnerCodex: "openai",
  runnerOpenRouter: "openrouter",
};

export function runnerModelsFromCanvasNodes(nodes?: RunnerCanvasNode[]): string {
  return joinRunnerModels(
    (nodes ?? []).map((node) => {
      const model = node.configuration && "model" in node.configuration ? node.configuration.model : undefined;
      return typeof model === "string" ? model : undefined;
    }),
  );
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
  if (runner === "openrouter") {
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
