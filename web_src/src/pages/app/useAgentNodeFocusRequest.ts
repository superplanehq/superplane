import { useEffect } from "react";

export type AgentNodeFocusRequest = {
  nodeId: string;
  requestId: number;
  targetMode: "live";
  tab: "settings";
};

type FocusRequestSetter = (request: {
  nodeId: string;
  requestId: number;
  targetMode: "live" | "runs";
  tab?: "latest" | "settings";
}) => void;

/**
 * Persists node-chip focus across canvas remounts (e.g. leaving Files mode, which
 * unmounts ReactFlow). Node chips dispatch `agent:focus-node`; AppPage owns that
 * event and converts it to `focusRequest`. CanvasContent applies `focusRequest`
 * only — it does not listen for the event directly.
 */
export function useAgentNodeFocusRequest(setFocusRequest: FocusRequestSetter): void {
  useEffect(() => {
    const handler = (event: Event) => {
      const nodeId = (event as CustomEvent<{ nodeId?: string }>).detail?.nodeId;
      if (!nodeId) {
        return;
      }

      setFocusRequest({
        nodeId,
        requestId: Date.now(),
        targetMode: "live",
        tab: "settings",
      });
    };

    window.addEventListener("agent:focus-node", handler);
    return () => window.removeEventListener("agent:focus-node", handler);
  }, [setFocusRequest]);
}
