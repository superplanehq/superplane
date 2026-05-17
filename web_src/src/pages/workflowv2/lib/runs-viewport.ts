export type CanvasViewport = {
  x: number;
  y: number;
  zoom: number;
};

type WorkflowViewMode = "runs" | null;

type SyncViewportOnModeSwitchInput = {
  previousMode: WorkflowViewMode;
  nextMode: WorkflowViewMode;
  canvasViewport?: CanvasViewport;
  runsViewport?: CanvasViewport;
};

type SyncViewportOnModeSwitchResult = {
  canvasViewport?: CanvasViewport;
  runsViewport?: CanvasViewport;
  canvasHasFitToView?: boolean;
  runsHasFitToView?: boolean;
  skipNextRunsFitAll: boolean;
};

export function syncViewportOnModeSwitch(input: SyncViewportOnModeSwitchInput): SyncViewportOnModeSwitchResult | null {
  if (input.previousMode === input.nextMode) {
    return null;
  }

  if (input.nextMode === "runs") {
    if (!input.canvasViewport) {
      return {
        runsViewport: undefined,
        runsHasFitToView: false,
        skipNextRunsFitAll: false,
      };
    }

    return {
      runsViewport: input.canvasViewport,
      runsHasFitToView: true,
      skipNextRunsFitAll: true,
    };
  }

  if (input.previousMode === "runs") {
    if (!input.runsViewport) {
      return {
        skipNextRunsFitAll: false,
      };
    }

    return {
      canvasViewport: input.runsViewport,
      canvasHasFitToView: true,
      skipNextRunsFitAll: false,
    };
  }

  return {
    skipNextRunsFitAll: false,
  };
}
