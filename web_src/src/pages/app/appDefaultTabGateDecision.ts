import type { AppTabId } from "@/lib/lastVisitedAppTab";
import type { ClassicAppRouteRedirect } from "./classicAppRouteRedirect";
import type { DefaultTabResolution } from "./defaultAppTab";

export type AppDefaultTabGateDecision =
  | { kind: "commit" }
  | { kind: "skeleton" }
  | { kind: "redirect"; to: string }
  | { kind: "stored-tab"; tab: AppTabId }
  | { kind: "resolution"; resolution: DefaultTabResolution };

type DecideArgs = {
  alreadyCommitted: boolean;
  canvasId: string;
  pinned: boolean;
  isNonCanvasView: boolean;
  canvasQueryEnabled: boolean;
  canvasLoading: boolean;
  canvasUndefined: boolean;
  storedTab: AppTabId | null;
  resolution: DefaultTabResolution;
  classicSurface: ClassicAppRouteRedirect;
};

export function decideAppDefaultTabGate({
  alreadyCommitted,
  canvasId,
  pinned,
  isNonCanvasView,
  canvasQueryEnabled,
  canvasLoading,
  canvasUndefined,
  storedTab,
  resolution,
  classicSurface,
}: DecideArgs): AppDefaultTabGateDecision {
  if (alreadyCommitted || !canvasId) {
    return { kind: "commit" };
  }

  if (canvasQueryEnabled && canvasLoading && canvasUndefined) {
    return { kind: "skeleton" };
  }

  if (classicSurface.kind === "wait") {
    return { kind: "skeleton" };
  }

  if (classicSurface.kind === "redirect") {
    return { kind: "redirect", to: classicSurface.to };
  }

  // Deep links that are already on the canvas surface (run/edit/version/…) can
  // commit immediately. Console/Memory/Files pins wait for the factory check
  // above so factory-owned apps never stay on the classic URL.
  if (pinned && !isNonCanvasView) {
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
