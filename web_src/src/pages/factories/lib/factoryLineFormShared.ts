import type { FactoryLineStep } from "@/api-client";

export const RUN_APP_TYPE = "runApp";

// How the step limits parallel runs. "" keeps the server default (10),
// "limited" uses the maxParallelism value, "unlimited" stores 0.
export type DraftParallelism = "" | "limited" | "unlimited";

export type DraftStep = {
  appId: string;
  entrypoint: string;
  parallelism: DraftParallelism;
  // Text form value; only meaningful while parallelism is "limited".
  maxParallelism: string;
};

export function emptyStep(): DraftStep {
  return { appId: "", entrypoint: "", parallelism: "", maxParallelism: "" };
}

export function draftStepsFromLine(steps: FactoryLineStep[] | undefined): DraftStep[] {
  if (!steps?.length) {
    return [emptyStep()];
  }

  return steps.map((step) => ({
    appId: step.app?.app ?? "",
    entrypoint: step.app?.entrypoint ?? "",
    parallelism: step.maxParallelism == null ? "" : step.maxParallelism === 0 ? "unlimited" : "limited",
    maxParallelism: step.maxParallelism != null && step.maxParallelism !== 0 ? String(step.maxParallelism) : "",
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

// 0 means unlimited on the wire; null keeps the server default.
function draftMaxParallelism(step: DraftStep): number | null {
  if (step.parallelism === "unlimited") {
    return 0;
  }

  if (step.parallelism !== "limited") {
    return null;
  }

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
