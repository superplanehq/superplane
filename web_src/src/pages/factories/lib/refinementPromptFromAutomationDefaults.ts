import { parseCanvasYamlToSpec } from "@/pages/app/lib/canvas-yaml-staging";

export const REFINE_TASK_NODE_ID = "refine-task";
export const REFINE_TASK_STEP_NAME = "Refine Task";

export function refinementPromptFromAutomationDefaults(canvasYaml: string): string {
  const node = parseCanvasYamlToSpec(canvasYaml)?.nodes?.find((entry) => entry.id === REFINE_TASK_NODE_ID);
  if (!node) {
    return "";
  }
  return promptFromRefineTaskStep(node.configuration?.steps);
}

function promptFromRefineTaskStep(steps: unknown): string {
  if (!Array.isArray(steps)) {
    return "";
  }
  const step = steps.find((entry) => isRefineTaskPromptStep(entry));
  if (!step || typeof step !== "object") {
    return "";
  }
  const prompt = (step as { prompt?: unknown }).prompt;
  return typeof prompt === "string" ? prompt : "";
}

function isRefineTaskPromptStep(step: unknown): boolean {
  if (!step || typeof step !== "object") {
    return false;
  }
  const record = step as { name?: unknown; type?: unknown };
  return record.name === REFINE_TASK_STEP_NAME && record.type === "prompt";
}
