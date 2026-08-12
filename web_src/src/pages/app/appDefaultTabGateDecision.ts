import type { AppTabId } from "@/lib/lastVisitedAppTab";
import type { DefaultTabResolution } from "./defaultAppTab";

export type AppDefaultTabGateDecision =
  | { kind: "commit" }
  | { kind: "skeleton" }
  | { kind: "factory-canvas" }
  | { kind: "stored-tab"; tab: AppTabId }
  | { kind: "resolution"; resolution: DefaultTabResolution };

type DecideArgs = {
  alreadyCommitted: boolean;
  canvasId: string;
  pinned: boolean;
  viewParam: string;
  isNonCanvasView: boolean;
  canvasQueryEnabled: boolean;
  canvasLoading: boolean;
  canvasUndefined: boolean;
  factoryOwnedApp: boolean;
  storedTab: AppTabId | null;
  resolution: DefaultTabResolution;
};

export function decideAppDefaultTabGate({
  alreadyCommitted,
  canvasId,
  pinned,
  isNonCanvasView,
  canvasQueryEnabled,
  canvasLoading,
  canvasUndefined,
  factoryOwnedApp,
  storedTab,
  resolution,
}: DecideArgs): AppDefaultTabGateDecision {
  if (alreadyCommitted || !canvasId) {
    return { kind: "commit" };
  }

  // Deep links that are already on the canvas surface (run/edit/version/…) can
  // commit immediately. Console/Memory/Files pins must wait for the factory check.
  if (pinned && !isNonCanvasView) {
    return { kind: "commit" };
  }

  if (canvasQueryEnabled && canvasLoading && canvasUndefined) {
    return { kind: "skeleton" };
  }

  if (factoryOwnedApp) {
    if (isNonCanvasView) {
      return { kind: "factory-canvas" };
    }
    return { kind: "commit" };
  }

  if (pinned) {
    return { kind: "commit" };
  }

  if (storedTab !== null) {
    return { kind: "stored-tab", tab: storedTab };
  }

  return { kind: "resolution", resolution };
}
