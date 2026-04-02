import type { MouseEvent } from "react";
import type { ComponentActionsProps } from "../../types/componentActions";
import type { AnnotationComponentProps } from "../../annotationComponent";
import type { ComponentBaseProps } from "../../componentBase";
import type { CompositeProps } from "../../composite";
import type { GroupNodeProps } from "../../groupNode";
import type { TriggerProps } from "../../trigger";

export type BlockState = "pending" | "working" | "success" | "failed" | "running";
export type BlockType = "trigger" | "component" | "composite" | "annotation" | "group";
export type BlockRenderFallbackSource = "mapper" | "additional-data" | "custom-field" | "node-render";

export interface BlockData {
  label: string;
  state: BlockState;
  type: BlockType;
  lastEvent?: unknown;
  outputChannels?: string[];
  renderFallback?: {
    source: BlockRenderFallbackSource;
    message: string;
  };
  trigger?: TriggerProps;
  component?: ComponentBaseProps;
  composite?: CompositeProps;
  annotation?: AnnotationComponentProps;
  group?: GroupNodeProps;
}

type BlockHandleType = "source" | "target";

export interface BlockConnectionState {
  nodeId?: string;
  handleId?: string | null;
  handleType?: BlockHandleType;
}

export interface BlockEdgeState {
  source?: string;
  sourceHandle?: string | null;
  target?: string;
}

export interface BlockInternalData {
  _hoveredEdge?: BlockEdgeState;
  _connectingFrom?: BlockConnectionState;
  _allEdges?: BlockEdgeState[];
  _isHighlighted?: boolean;
  _hasHighlightedNodes?: boolean;
  isTemplate?: boolean;
  isPendingConnection?: boolean;
}

export type CanvasBlockData = BlockData & BlockInternalData;

export interface BlockProps extends ComponentActionsProps {
  data: CanvasBlockData;
  nodeId?: string;
  selected?: boolean;
  showHeader?: boolean;
  onAnnotationUpdate?: (
    nodeId: string,
    updates: { text?: string; color?: string; width?: number; height?: number; x?: number; y?: number },
  ) => void;
  onAnnotationBlur?: () => void;
  onExpand?: (nodeId: string, nodeData: BlockData) => void;
  onClick?: (e: MouseEvent) => void;
}
