const UNMOUNT_END_MS = 100;
const pendingUnmountEnds = new Map<string, number>();

export function cancelScheduledPlanningSessionEnd(key: string) {
  const timer = pendingUnmountEnds.get(key);
  if (timer === undefined) {
    return;
  }
  window.clearTimeout(timer);
  pendingUnmountEnds.delete(key);
}

export function schedulePlanningSessionEnd(key: string, end: () => void) {
  cancelScheduledPlanningSessionEnd(key);
  pendingUnmountEnds.set(
    key,
    window.setTimeout(() => {
      pendingUnmountEnds.delete(key);
      end();
    }, UNMOUNT_END_MS),
  );
}

export function resetCreateWithAgentUnmountEnds() {
  for (const timer of pendingUnmountEnds.values()) {
    window.clearTimeout(timer);
  }
  pendingUnmountEnds.clear();
}
