import type { FactoryLineStep } from "@/api-client";

export const RUN_APP_TYPE = "runApp";

export type DraftStep = {
  appId: string;
  entrypoint: string;
};

export function emptyStep(): DraftStep {
  return { appId: "", entrypoint: "" };
}

export function draftStepsFromLine(steps: FactoryLineStep[] | undefined): DraftStep[] {
  if (!steps?.length) {
    return [emptyStep()];
  }

  return steps.map((step) => ({
    appId: step.app?.app ?? "",
    entrypoint: step.app?.entrypoint ?? "",
  }));
}

export function draftStepsToProto(steps: DraftStep[]): FactoryLineStep[] {
  return steps.map((step) => ({
    type: RUN_APP_TYPE,
    app: {
      app: step.appId,
      entrypoint: step.entrypoint.trim(),
    },
  }));
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
