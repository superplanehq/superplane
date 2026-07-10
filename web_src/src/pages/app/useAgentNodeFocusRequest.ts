import { useEffect, type RefObject } from "react";

/** Focus payload AppPage stores and CanvasContent applies. */
export type CanvasFocusRequest = {
  nodeId: string;
  requestId: number;
  targetMode: "live" | "runs";
  tab?: "latest" | "settings";
};

type FocusRequestSetter = (request: CanvasFocusRequest) => void;

/**
 * Persists node-chip focus across canvas remounts (e.g. leaving Files mode, which
 * unmounts ReactFlow). Node chips dispatch `agent:focus-node`; AppPage owns that
 * event and converts it to `focusRequest`. CanvasContent applies `focusRequest`
 * only — it does not listen for the event directly.
 *
 * `targetModeRef` must track the canvas currently mounted (live vs run inspection),
 * matching CanvasContent's `isRunInspectionMode` gate.
 */
export function useAgentNodeFocusRequest(
  setFocusRequest: FocusRequestSetter,
  targetModeRef: RefObject<"live" | "runs">,
): void {
  useEffect(() => {
    const handler = (event: Event) => {
      const nodeId = (event as CustomEvent<{ nodeId?: string }>).detail?.nodeId;
      if (!nodeId) {
        return;
      }

      const targetMode = targetModeRef.current ?? "live";
      setFocusRequest({
        nodeId,
        requestId: Date.now(),
        targetMode,
        ...(targetMode === "live" ? { tab: "settings" as const } : {}),
      });
    };

    window.addEventListener("agent:focus-node", handler);
    return () => window.removeEventListener("agent:focus-node", handler);
  }, [setFocusRequest, targetModeRef]);
}
