import { useEffect, useRef } from "react";

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
 * Pass the same run-inspection flag CanvasContent uses so chips focus the
 * currently mounted canvas (live vs runs).
 */
export function useAgentNodeFocusRequest(
  setFocusRequest: FocusRequestSetter,
  isRunInspectionMode: boolean,
): void {
  const targetModeRef = useRef<"live" | "runs">("live");
  targetModeRef.current = isRunInspectionMode ? "runs" : "live";

  useEffect(() => {
    const handler = (event: Event) => {
      const nodeId = (event as CustomEvent<{ nodeId?: string }>).detail?.nodeId;
      if (!nodeId) {
        return;
      }

      const targetMode = targetModeRef.current;
      setFocusRequest({
        nodeId,
        requestId: Date.now(),
        targetMode,
        ...(targetMode === "live" ? { tab: "settings" as const } : {}),
      });
    };

    window.addEventListener("agent:focus-node", handler);
    return () => window.removeEventListener("agent:focus-node", handler);
  }, [setFocusRequest]);
}
