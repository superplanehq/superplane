export type CanvasViewport = {
  x: number;
  y: number;
  zoom: number;
};

type MutableRef<T> = {
  current: T;
};

type RunsViewportKey = "runs" | null;

export function syncRunInspectionViewportTransition({
  isRunInspectionMode,
  liveViewportRef,
  runsViewportRef,
  liveHasFitToViewRef,
  runsHasFitToViewRef,
  lastRunsViewportKeyRef,
}: {
  isRunInspectionMode: boolean;
  liveViewportRef: MutableRef<CanvasViewport | undefined>;
  runsViewportRef: MutableRef<CanvasViewport | undefined>;
  liveHasFitToViewRef: MutableRef<boolean>;
  runsHasFitToViewRef: MutableRef<boolean>;
  lastRunsViewportKeyRef: MutableRef<RunsViewportKey>;
}) {
  const nextKey: RunsViewportKey = isRunInspectionMode ? "runs" : null;
  const previousKey = lastRunsViewportKeyRef.current;
  if (previousKey === nextKey) {
    return;
  }

  if (nextKey === "runs") {
    const liveViewport = liveViewportRef.current;
    runsHasFitToViewRef.current = Boolean(liveViewport);
    runsViewportRef.current = liveViewport ? { ...liveViewport } : undefined;
    lastRunsViewportKeyRef.current = nextKey;
    return;
  }

  if (previousKey === "runs" && runsViewportRef.current) {
    liveViewportRef.current = { ...runsViewportRef.current };
    liveHasFitToViewRef.current = true;
  }

  runsHasFitToViewRef.current = false;
  runsViewportRef.current = undefined;
  lastRunsViewportKeyRef.current = nextKey;
}
