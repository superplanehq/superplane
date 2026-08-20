import type { FactoryLineStep } from "@/api-client";

export const RUN_APP_TYPE = "runApp";

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
