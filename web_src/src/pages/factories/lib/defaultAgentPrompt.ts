import type { CanvasesCanvas } from "@/api-client";

import type { PlanningReviewDraft, PlanningReviewStep } from "../pages/planningReviewMockup";

export const RESTORE_DEFAULT_PROMPT_COPY = {
  error: "Could not restore the default prompt.",
  confirmTitle: "Restore the default prompt?",
  confirmDescription:
    "This replaces the prompt text on this agent with the factory default. The change applies only after you click Save Agent.",
  confirmAction: "Restore default prompt",
} as const;

type CanvasSpec = CanvasesCanvas["spec"];

export function defaultAgentSteps(
  spec: CanvasSpec | null | undefined,
  agentNodeId: string,
): PlanningReviewStep[] | null {
  if (!agentNodeId) {
    return null;
  }
  const node = spec?.nodes?.find((entry) => entry.id === agentNodeId);
  const steps = node?.configuration?.steps;
  if (!node || !Array.isArray(steps)) {
    return null;
  }

  const extracted: PlanningReviewStep[] = [];
  for (const value of steps) {
    if (isPromptStep(value)) {
      const step: PlanningReviewStep = { name: value.name, type: "prompt", prompt: value.prompt };
      if (typeof value.workingDirectory === "string") {
        step.workingDirectory = value.workingDirectory;
      }
      extracted.push(step);
      continue;
    }
    if (!isBashStep(value)) {
      continue;
    }
    const step: PlanningReviewStep = { name: value.name, type: "bash" };
    if (typeof value.command === "string") {
      step.command = value.command;
    }
    if (typeof value.workingDirectory === "string") {
      step.workingDirectory = value.workingDirectory;
    }
    extracted.push(step);
  }
  return extracted;
}

export function applyDefaultAgentPrompts(
  draft: PlanningReviewDraft,
  defaultSteps: PlanningReviewStep[],
): PlanningReviewDraft {
  const index = draft.components.findIndex((component) => component.expanded);
  const componentIndex = index >= 0 ? index : 0;
  const component = draft.components[componentIndex];
  if (!component) {
    return draft;
  }

  const steps = componentSteps(component.configuration.steps);
  const nextSteps = stepsWithDefaultPrompts(steps, defaultSteps);
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

export function agentPromptsMatchDefault(draft: PlanningReviewDraft, defaultSteps: PlanningReviewStep[]): boolean {
  return applyDefaultAgentPrompts(draft, defaultSteps) === draft;
}

function stepsWithDefaultPrompts(
  steps: PlanningReviewStep[],
  defaultSteps: PlanningReviewStep[],
): PlanningReviewStep[] {
  const defaults = defaultSteps.filter((step) => step.type === "prompt" && typeof step.prompt === "string");
  if (defaults.length === 0) {
    return steps;
  }

  const current = steps.flatMap((step, index) => (step.type === "prompt" ? [{ index, name: step.name }] : []));
  if (current.length === 0) {
    return stepsWithRestoredPrompts(steps, defaultSteps, defaults);
  }

  const usedDefaults = new Set<number>();
  const assigned = new Map<number, string>();
  for (const entry of current) {
    const defaultIndex = defaults.findIndex((step, index) => !usedDefaults.has(index) && step.name === entry.name);
    if (defaultIndex < 0) {
      continue;
    }
    usedDefaults.add(defaultIndex);
    assigned.set(entry.index, defaults[defaultIndex].prompt ?? "");
  }

  const unmatchedCurrent = current.filter((entry) => !assigned.has(entry.index));
  const unmatchedDefaults = defaults.filter((_, index) => !usedDefaults.has(index));
  if (unmatchedCurrent.length === unmatchedDefaults.length) {
    unmatchedCurrent.forEach((entry, order) => {
      assigned.set(entry.index, unmatchedDefaults[order].prompt ?? "");
    });
  }

  return replaceAssignedPrompts(steps, assigned);
}

function stepsWithRestoredPrompts(
  steps: PlanningReviewStep[],
  defaultSteps: PlanningReviewStep[],
  prompts: PlanningReviewStep[],
): PlanningReviewStep[] {
  const insertions = prompts.map((prompt, order) => ({
    index: promptInsertionIndex(steps, defaultSteps, prompt),
    order,
    step: factoryPromptStep(prompt),
  }));
  insertions.sort((left, right) => left.index - right.index || left.order - right.order);
  const next = [...steps];
  let offset = 0;
  for (const insertion of insertions) {
    next.splice(insertion.index + offset, 0, insertion.step);
    offset += 1;
  }
  return next;
}

function promptInsertionIndex(
  steps: PlanningReviewStep[],
  defaultSteps: PlanningReviewStep[],
  prompt: PlanningReviewStep,
): number {
  const promptIndex = defaultSteps.indexOf(prompt);
  if (promptIndex < 0) {
    return steps.length;
  }

  for (let index = promptIndex - 1; index >= 0; index -= 1) {
    if (defaultSteps[index].type === "prompt") {
      continue;
    }
    const found = lastMatchingStepIndex(steps, defaultSteps[index]);
    if (found >= 0) {
      return found + 1;
    }
  }

  for (let index = promptIndex + 1; index < defaultSteps.length; index += 1) {
    if (defaultSteps[index].type === "prompt") {
      continue;
    }
    const found = firstMatchingStepIndex(steps, defaultSteps[index]);
    if (found >= 0) {
      return found;
    }
  }

  return steps.length;
}

function firstMatchingStepIndex(steps: PlanningReviewStep[], factory: PlanningReviewStep): number {
  return steps.findIndex((step) => sameStepNameAndType(step, factory));
}

function lastMatchingStepIndex(steps: PlanningReviewStep[], factory: PlanningReviewStep): number {
  for (let index = steps.length - 1; index >= 0; index -= 1) {
    if (sameStepNameAndType(steps[index], factory)) {
      return index;
    }
  }
  return -1;
}

function sameStepNameAndType(left: PlanningReviewStep, right: PlanningReviewStep): boolean {
  return left.type === right.type && left.name === right.name;
}

function replaceAssignedPrompts(steps: PlanningReviewStep[], assigned: Map<number, string>): PlanningReviewStep[] {
  let changed = false;
  const next = steps.map((step, index) => {
    const prompt = assigned.get(index);
    if (prompt === undefined || (step.prompt ?? "") === prompt) {
      return step;
    }
    changed = true;
    return { ...step, prompt };
  });
  return changed ? next : steps;
}

function factoryPromptStep(step: PlanningReviewStep): PlanningReviewStep {
  return {
    name: step.name,
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

function isPromptStep(value: unknown): value is { name: string; prompt: string; workingDirectory?: unknown } {
  if (!isRecord(value) || typeof value.name !== "string") {
    return false;
  }
  return value.type === "prompt" && typeof value.prompt === "string";
}

function isBashStep(value: unknown): value is { name: string; command?: unknown; workingDirectory?: unknown } {
  if (!isRecord(value) || typeof value.name !== "string") {
    return false;
  }
  return value.type === "bash";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
