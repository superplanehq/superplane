export const FOLLOW_BOTTOM_THRESHOLD_PX = 16;

export function runningSplitRunPhaseId(phases: ReadonlyArray<{ id: string; status: string }>): string | null {
  return phases.find((phase) => phase.status === "running")?.id ?? null;
}

export function followAfterRunningPhaseChange(
  wasFollowing: boolean,
  previousRunningPhaseId: string | null,
  runningPhaseId: string | null,
): boolean {
  if (runningPhaseId && runningPhaseId !== previousRunningPhaseId) {
    return true;
  }
  return wasFollowing;
}

export function isNearLogBottom(
  scrollTop: number,
  scrollHeight: number,
  clientHeight: number,
  thresholdPx = FOLLOW_BOTTOM_THRESHOLD_PX,
): boolean {
  return scrollHeight - scrollTop - clientHeight <= thresholdPx;
}
