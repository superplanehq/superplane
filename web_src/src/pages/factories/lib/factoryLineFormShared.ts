import type { FactoryLineStep } from "@/api-client";

export const RUN_APP_TYPE = "runApp";

export const DEFAULT_LINE_STEP_PARALLELISM = 10;
export const LINE_STEP_PARALLELISM_MIN = 1;
export const LINE_STEP_PARALLELISM_MAX = 100;

export type DraftStep = {
  appId: string;
  entrypoint: string;
  // Text form value; empty keeps the server default (10).
  maxParallelism: string;
};

export function emptyStep(): DraftStep {
  return { appId: "", entrypoint: "", maxParallelism: "" };
}

export function draftStepsFromLine(steps: FactoryLineStep[] | undefined): DraftStep[] {
  if (!steps?.length) {
    return [emptyStep()];
  }

  return steps.map((step) => ({
    appId: step.app?.app ?? "",
    entrypoint: step.app?.entrypoint ?? "",
    maxParallelism: step.maxParallelism != null && step.maxParallelism > 0 ? String(step.maxParallelism) : "",
  }));
}

export function draftStepsToProto(steps: DraftStep[]): FactoryLineStep[] {
  return steps.map((step) => {
    const proto: FactoryLineStep = {
      type: RUN_APP_TYPE,
      app: {
        app: step.appId,
        entrypoint: step.entrypoint.trim(),
      },
    };

    const maxParallelism = draftMaxParallelism(step);
    if (maxParallelism != null) {
      proto.maxParallelism = maxParallelism;
    }

    return proto;
  });
}

export function lineStepParallelism(step: { maxParallelism?: number | null } | undefined): number {
  const value = step?.maxParallelism;
  if (value == null || value < LINE_STEP_PARALLELISM_MIN || value > LINE_STEP_PARALLELISM_MAX) {
    return DEFAULT_LINE_STEP_PARALLELISM;
  }
  return value;
}

export function clampLineStepParallelism(value: number): number {
  return Math.min(LINE_STEP_PARALLELISM_MAX, Math.max(LINE_STEP_PARALLELISM_MIN, Math.round(value)));
}

export function setParallelismLabel(parallelism: number): string {
  return `Set parallelism (${parallelism})`;
}

export function replaceLineStepParallelism(
  steps: FactoryLineStep[] | undefined,
  stepIndex: number,
  parallelism: number,
): FactoryLineStep[] {
  const drafts = draftStepsFromLine(steps);
  const current = drafts[stepIndex];
  if (!current) {
    return draftStepsToProto(drafts);
  }
  drafts[stepIndex] = {
    ...current,
    maxParallelism: String(clampLineStepParallelism(parallelism)),
  };
  return draftStepsToProto(drafts);
}

export function removeLineStep(steps: FactoryLineStep[] | undefined, stepIndex: number): FactoryLineStep[] {
  if (!steps?.length || stepIndex < 0 || stepIndex >= steps.length) {
    return draftStepsToProto(draftStepsFromLine(steps));
  }
  return draftStepsToProto(draftStepsFromLine(steps).filter((_, index) => index !== stepIndex));
}

// Null keeps the server default (10).
function draftMaxParallelism(step: DraftStep): number | null {
  const parsed = Number.parseInt(step.maxParallelism.trim(), 10);
  if (Number.isNaN(parsed) || parsed < 1) {
    return null;
  }

  return parsed;
}

/** Display label for a line step: the automation (canvas) name. */
export function automationNameForLineStep(
  step: { app?: { app?: string } } | undefined,
  apps: Array<{ id?: string; name?: string }> | undefined,
  fallbackIndex: number,
): string {
  const appId = step?.app?.app?.trim();
  const name = apps?.find((app) => app.id === appId)?.name?.trim();
  if (name) {
    return name;
  }
  return `Phase ${fallbackIndex + 1}`;
}
