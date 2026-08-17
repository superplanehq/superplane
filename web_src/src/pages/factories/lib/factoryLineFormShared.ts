import type { FactoryLineStep } from "@/api-client";

export const RUN_APP_TYPE = "runApp";

export type DraftStep = {
  name: string;
  appId: string;
  entrypoint: string;
};

export function emptyStep(): DraftStep {
  return { name: "", appId: "", entrypoint: "" };
}

export function draftStepsFromLine(steps: FactoryLineStep[] | undefined): DraftStep[] {
  if (!steps?.length) {
    return [emptyStep()];
  }

  return steps.map((step) => ({
    name: step.name ?? "",
    appId: step.app?.app ?? "",
    entrypoint: step.app?.entrypoint ?? "",
  }));
}

export function draftStepsToProto(steps: DraftStep[]): FactoryLineStep[] {
  return steps.map((step) => ({
    name: step.name.trim(),
    type: RUN_APP_TYPE,
    app: {
      app: step.appId,
      entrypoint: step.entrypoint.trim(),
    },
  }));
}
