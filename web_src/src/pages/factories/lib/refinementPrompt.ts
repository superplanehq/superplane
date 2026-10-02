import type { CanvasesCanvas } from "@/api-client";

import type { PlanningReviewDraft, PlanningReviewStep } from "../pages/planningReviewMockup";

export const REFINE_TASK_NODE_ID = "refine-task";
export const REFINE_TASK_STEP_NAME = "Refine Task";

type CanvasSpec = CanvasesCanvas["spec"];

export function defaultRefineTaskStep(spec: CanvasSpec | null | undefined): PlanningReviewStep | null {
  const node = spec?.nodes?.find((entry) => entry.id === REFINE_TASK_NODE_ID);
  const steps = node?.configuration?.steps;
  if (!Array.isArray(steps)) {
    return null;
  }

  const match = steps.find(isRefineTaskPromptStep);
  if (!match) {
    return null;
  }

  const step: PlanningReviewStep = {
    name: REFINE_TASK_STEP_NAME,
    type: "prompt",
    prompt: match.prompt,
  };
  if (typeof match.workingDirectory === "string") {
    step.workingDirectory = match.workingDirectory;
  }
  return step;
}

export function applyDefaultRefinementPrompt(
  draft: PlanningReviewDraft,
  defaultStep: PlanningReviewStep,
): PlanningReviewDraft {
  const index = draft.components.findIndex((component) => component.expanded);
  const componentIndex = index >= 0 ? index : 0;
  const component = draft.components[componentIndex];
  if (!component) {
    return draft;
  }

  const steps = componentSteps(component.configuration.steps);
  const nextSteps = stepsWithDefaultPrompt(steps, defaultStep);
  if (nextSteps === steps) {
    return draft;
  }

  const nextComponent = {
    ...component,
    configuration: { ...component.configuration, steps: nextSteps },
  };
  return {
    ...draft,
    components: draft.components.map((entry, position) => (position === componentIndex ? nextComponent : entry)),
  };
}

export function refinementPromptMatchesDefault(draft: PlanningReviewDraft, defaultStep: PlanningReviewStep): boolean {
  return applyDefaultRefinementPrompt(draft, defaultStep) === draft;
}

function stepsWithDefaultPrompt(steps: PlanningReviewStep[], defaultStep: PlanningReviewStep): PlanningReviewStep[] {
  const namedIndex = steps.findIndex((step) => step.name === REFINE_TASK_STEP_NAME && step.type === "prompt");
  if (namedIndex >= 0) {
    return replacePrompt(steps, namedIndex, defaultStep.prompt);
  }

  const promptIndexes = steps.flatMap((step, index) => (step.type === "prompt" ? [index] : []));
  if (promptIndexes.length === 1) {
    return replacePrompt(steps, promptIndexes[0], defaultStep.prompt);
  }

  return [...steps, factoryRefineTaskStep(defaultStep)];
}

function replacePrompt(steps: PlanningReviewStep[], index: number, prompt: string | undefined): PlanningReviewStep[] {
  if ((steps[index]?.prompt ?? "") === (prompt ?? "")) {
    return steps;
  }
  return steps.map((step, position) => (position === index ? { ...step, prompt } : step));
}

function factoryRefineTaskStep(step: PlanningReviewStep): PlanningReviewStep {
  return {
    name: step.name || REFINE_TASK_STEP_NAME,
    type: "prompt",
    prompt: step.prompt,
    ...(typeof step.workingDirectory === "string" ? { workingDirectory: step.workingDirectory } : {}),
  };
}

function componentSteps(value: unknown): PlanningReviewStep[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter(isPlanningReviewStep);
}

function isPlanningReviewStep(value: unknown): value is PlanningReviewStep {
  if (!isRecord(value) || typeof value.name !== "string") {
    return false;
  }
  return value.type === "bash" || value.type === "prompt";
}

function isRefineTaskPromptStep(value: unknown): value is { prompt: string; workingDirectory?: unknown } {
  if (!isRecord(value)) {
    return false;
  }
  return value.name === REFINE_TASK_STEP_NAME && value.type === "prompt" && typeof value.prompt === "string";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
