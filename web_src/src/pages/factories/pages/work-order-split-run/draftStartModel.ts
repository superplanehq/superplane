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

export function runnerModelsFromCanvasNodes(
  nodes?: Array<{ configuration?: { model?: unknown } | Record<string, unknown> }>,
): string {
  return joinRunnerModels(
    (nodes ?? []).map((node) => {
      const model = node.configuration && "model" in node.configuration ? node.configuration.model : undefined;
      return typeof model === "string" ? model : undefined;
    }),
  );
}

export function phaseWithRunnerModel<T extends { model?: string }>(
  phase: T,
  nodes?: Array<{ configuration?: { model?: unknown } | Record<string, unknown> }>,
): T {
  if (phase.model?.trim()) {
    return phase;
  }
  const fromCanvas = runnerModelsFromCanvasNodes(nodes);
  return fromCanvas ? { ...phase, model: fromCanvas } : phase;
}
