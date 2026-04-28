import { createContext, useContext, type MouseEvent, type MutableRefObject, type ReactNode } from "react";

import type { BlockConnectionState, BlockEdgeState } from "./Block/types";

/** Callback bundle for custom nodes; lives on a ref so context value can stay stable. */
export type CanvasNodeRendererCallbacks = {
  handleNodeClick: (nodeId: string, event?: MouseEvent) => void;
  onNodeEdit: MutableRefObject<((nodeId: string) => void) | undefined>;
  onNodeDelete: MutableRefObject<((nodeId: string) => void) | undefined>;
  onRun: MutableRefObject<((nodeId?: string, initialData?: string) => void) | undefined>;
  onDuplicate: MutableRefObject<((nodeId: string) => void) | undefined>;
  onDeactivate: MutableRefObject<((nodeId: string) => void) | undefined>;
  onTogglePause: MutableRefObject<((nodeId: string) => void) | undefined>;
  onToggleView: MutableRefObject<((nodeId: string) => void) | undefined>;
  onAnnotationUpdate: MutableRefObject<
    | ((
        nodeId: string,
        updates: { text?: string; color?: string; width?: number; height?: number; x?: number; y?: number },
      ) => void)
    | undefined
  >;
  onAnnotationBlur: MutableRefObject<(() => void) | undefined>;
  runDisabled?: boolean;
  runDisabledTooltip?: string;
  showHeader: boolean;
  hasMultiSelection: boolean;
  canvasMode: "live" | "edit";
};

export type CanvasNodeOverlayValue = {
  callbacksRef: MutableRefObject<CanvasNodeRendererCallbacks>;
  hoveredEdge: BlockEdgeState | null;
  connectingFrom: BlockConnectionState | null;
  allEdges: BlockEdgeState[];
  highlightedNodeIds: ReadonlySet<string>;
  hasHighlightedNodes: boolean;
};

const CanvasNodeOverlayContext = createContext<CanvasNodeOverlayValue | null>(null);

export function CanvasNodeOverlayProvider({
  value,
  children,
}: {
  value: CanvasNodeOverlayValue;
  children: ReactNode;
}) {
  return <CanvasNodeOverlayContext.Provider value={value}>{children}</CanvasNodeOverlayContext.Provider>;
}

export function useCanvasNodeOverlay(): CanvasNodeOverlayValue | null {
  return useContext(CanvasNodeOverlayContext);
}
