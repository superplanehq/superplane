import { useRef } from "react";

import type { CreateWithAgentMachineStatus } from "../createWithAgentTypes";

export type PlanChipStatus = "ready" | "updated";

export type PlanChipState = { body: string; status: PlanChipStatus | undefined };

export function nextPlanChipStatus(
  previous: PlanChipState | null,
  body: string | undefined,
  opened = false,
): PlanChipState | null {
  if (!body) {
    return null;
  }
  if (!previous) {
    return { body, status: opened ? undefined : "ready" };
  }
  if (previous.body !== body) {
    return { body, status: opened ? undefined : "updated" };
  }
  if (opened && previous.status) {
    return { body, status: undefined };
  }
  return previous;
}

export function usePlanChipStatus(body: string | undefined, opened = false): PlanChipStatus | undefined {
  const seen = useRef<PlanChipState | null>(null);
  seen.current = nextPlanChipStatus(seen.current, body, opened);
  return seen.current?.status;
}

export function composerChipsWorking(input: {
  isAnalyzing?: boolean;
  score?: number;
  machineStatus?: CreateWithAgentMachineStatus;
}): boolean {
  if (input.machineStatus === "starting" || input.machineStatus === "running") {
    return true;
  }
  return Boolean(input.isAnalyzing) && input.score == null;
}
