import { describe, expect, it } from "vitest";
import { syncRunInspectionViewportTransition, type CanvasViewport } from "./run-inspection-viewport";

function ref<T>(current: T): { current: T } {
  return { current };
}

describe("syncRunInspectionViewportTransition", () => {
  it("seeds the runs viewport from the current live viewport when entering run inspection", () => {
    const liveViewportRef = ref<CanvasViewport | undefined>({ x: -100, y: -50, zoom: 0.8 });
    const runsViewportRef = ref<CanvasViewport | undefined>(undefined);
    const liveHasFitToViewRef = ref(true);
    const runsHasFitToViewRef = ref(false);
    const lastRunsViewportKeyRef = ref<"runs" | null>(null);

    syncRunInspectionViewportTransition({
      isRunInspectionMode: true,
      liveViewportRef,
      runsViewportRef,
      liveHasFitToViewRef,
      runsHasFitToViewRef,
      lastRunsViewportKeyRef,
    });

    expect(runsViewportRef.current).toEqual({ x: -100, y: -50, zoom: 0.8 });
    expect(runsViewportRef.current).not.toBe(liveViewportRef.current);
    expect(runsHasFitToViewRef.current).toBe(true);
    expect(lastRunsViewportKeyRef.current).toBe("runs");
  });

  it("preserves the current runs viewport as the live viewport when leaving run inspection", () => {
    const liveViewportRef = ref<CanvasViewport | undefined>({ x: -100, y: -50, zoom: 0.8 });
    const runsViewportRef = ref<CanvasViewport | undefined>({ x: -320, y: -180, zoom: 1.1 });
    const liveHasFitToViewRef = ref(false);
    const runsHasFitToViewRef = ref(true);
    const lastRunsViewportKeyRef = ref<"runs" | null>("runs");

    syncRunInspectionViewportTransition({
      isRunInspectionMode: false,
      liveViewportRef,
      runsViewportRef,
      liveHasFitToViewRef,
      runsHasFitToViewRef,
      lastRunsViewportKeyRef,
    });

    expect(liveViewportRef.current).toEqual({ x: -320, y: -180, zoom: 1.1 });
    expect(runsViewportRef.current).toBeUndefined();
    expect(liveHasFitToViewRef.current).toBe(true);
    expect(runsHasFitToViewRef.current).toBe(false);
    expect(lastRunsViewportKeyRef.current).toBeNull();
  });
});
