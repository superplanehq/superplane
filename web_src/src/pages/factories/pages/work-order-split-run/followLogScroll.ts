/** Hide Jump to latest until the user leaves this band. */
export const FOLLOW_BOTTOM_THRESHOLD_PX = 144;
/** Resume follow only when the user is back at the bottom. */
export const FOLLOW_RESUME_THRESHOLD_PX = 16;

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

export function distanceFromLogBottom(scrollTop: number, scrollHeight: number, clientHeight: number): number {
  return scrollHeight - scrollTop - clientHeight;
}

export function isNearLogBottom(
  scrollTop: number,
  scrollHeight: number,
  clientHeight: number,
  thresholdPx = FOLLOW_BOTTOM_THRESHOLD_PX,
): boolean {
  return distanceFromLogBottom(scrollTop, scrollHeight, clientHeight) <= thresholdPx;
}

/**
 * An upward scroll always stops follow so live lines do not snap the
 * user back. Resume waits until the user scrolls down to the bottom.
 */
export function nextFollowAfterScroll(input: {
  following: boolean;
  resumeOnBottom: boolean;
  distanceFromBottom: number;
  scrollingUp: boolean;
}): boolean {
  if (input.scrollingUp) {
    return false;
  }
  if (input.distanceFromBottom > FOLLOW_BOTTOM_THRESHOLD_PX) {
    return false;
  }
  if (input.resumeOnBottom && input.distanceFromBottom <= FOLLOW_RESUME_THRESHOLD_PX) {
    return true;
  }
  return input.following;
}

export function showJumpToLatest(following: boolean, distanceFromBottom: number): boolean {
  return !following && distanceFromBottom > FOLLOW_BOTTOM_THRESHOLD_PX;
}
