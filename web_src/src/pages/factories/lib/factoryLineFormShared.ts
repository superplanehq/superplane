import type { FactoryLineStep } from "@/api-client";

export const RUN_APP_TYPE = "runApp";

export type DraftStep = {
  name: string;
  appId: string;
  entrypoint: string;
  // Text form value; empty means the server default (10), "0" means unlimited.
  maxParallelism: string;
};

export function emptyStep(): DraftStep {
  return { name: "", appId: "", entrypoint: "", maxParallelism: "" };
}

export function draftStepsFromLine(steps: FactoryLineStep[] | undefined): DraftStep[] {
  if (!steps?.length) {
    return [emptyStep()];
  }

  return steps.map((step) => ({
    name: step.name ?? "",
    appId: step.app?.app ?? "",
    entrypoint: step.app?.entrypoint ?? "",
    maxParallelism: step.maxParallelism != null ? String(step.maxParallelism) : "",
  }));
}

export function draftStepsToProto(steps: DraftStep[]): FactoryLineStep[] {
  return steps.map((step) => {
    const proto: FactoryLineStep = {
      name: step.name.trim(),
      type: RUN_APP_TYPE,
      app: {
        app: step.appId,
        entrypoint: step.entrypoint.trim(),
      },
    };

    const maxParallelism = parseMaxParallelism(step.maxParallelism);
    if (maxParallelism != null) {
      proto.maxParallelism = maxParallelism;
    }

    return proto;
  });
}

function parseMaxParallelism(value: string): number | null {
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }

  const parsed = Number.parseInt(trimmed, 10);
  if (Number.isNaN(parsed) || parsed < 0) {
    return null;
  }

  return parsed;
}
