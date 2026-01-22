import React from "react";
import { Composite, type CompositeProps } from "@/ui/composite";
import { SwitchComponent, type SwitchComponentProps } from "@/ui/switchComponent";
import { Trigger, type TriggerProps } from "@/ui/trigger";
import { Handle, Position } from "@xyflow/react";
import { SparklesIcon } from "lucide-react";
import { Button } from "../button";
import MergeComponent, { type MergeComponentProps } from "../merge";
import { ComponentActionsProps } from "../types/componentActions";
import { ComponentBase, ComponentBaseProps } from "../componentBase";
import { AnnotationComponent, type AnnotationComponentProps } from "../annotationComponent";

type BlockState = "pending" | "working" | "success" | "failed" | "running";
type BlockType = "trigger" | "component" | "composite" | "merge" | "switch" | "annotation";

interface BlockAi {
  show: boolean;
  suggestion: string | null;
  onApply: () => void;
  onDismiss: () => void;
}

export interface BlockData {
  label: string;

  state: BlockState;
  type: BlockType;

  // last input event received by this block (for simulation display)
  lastEvent?: unknown;

  // output channels for this block (e.g., ['default'], ['true', 'false'])
  outputChannels?: string[];

  // trigger node specific props
  trigger?: TriggerProps;

  // component base specific props
  component?: ComponentBaseProps;

  // composite node specific props
  composite?: CompositeProps;

  // switch node specific props
  switch?: SwitchComponentProps;

  // merge node specific props
  merge?: MergeComponentProps;

  // annotation node specific props
  annotation?: AnnotationComponentProps;
}

interface BlockProps extends ComponentActionsProps {
  data: BlockData;
  nodeId?: string;
  selected?: boolean;
  onAnnotationUpdate?: (nodeId: string, updates: { text?: string; color?: string }) => void;

  onExpand?: (nodeId: string, nodeData: BlockData) => void;
  onClick?: () => void;

  ai?: BlockAi;
}

export function Block(props: BlockProps) {
  const data = props.data;
  const ai = props.ai || {
    show: false,
    suggestion: null,
    onApply: () => {},
    onDismiss: () => {},
  };

  // Check if this node is highlighted (from execution chain)
  const isHighlighted = (data as any)._isHighlighted || false;
  const hasHighlightedNodes = (data as any)._hasHighlightedNodes || false;

  // Apply opacity to non-highlighted nodes when there are highlighted nodes
  const shouldDim = hasHighlightedNodes && !isHighlighted;

  return (
    <>
      <AiPopup {...ai} />

      <div className={`relative w-fit ${shouldDim ? "opacity-30" : ""}`} onClick={props.onClick}>
        <LeftHandle data={data} nodeId={props.nodeId} />
        <BlockContent {...props} />
        <RightHandle data={data} nodeId={props.nodeId} />
      </div>
    </>
  );
}

//
// Handles are small connection points on the sides of blocks
//

const HANDLE_STYLE = {
  width: 12,
  height: 12,
  borderRadius: 100,
  border: "3px solid #C9D5E1",
  background: "transparent",
};

function LeftHandle({ data, nodeId }: BlockProps) {
  if (data.type === "trigger" || data.type === "annotation") return null;

  // Check if this handle is part of the hovered edge (this is the target)
  const hoveredEdge = (data as any)._hoveredEdge;
  const connectingFrom = (data as any)._connectingFrom;
  const allEdges = (data as any)._allEdges || [];

  // Check if already connected to the source being dragged
  const isAlreadyConnected = connectingFrom
    ? allEdges.some(
        (edge: any) =>
          edge.source === connectingFrom.nodeId &&
          edge.sourceHandle === connectingFrom.handleId &&
          edge.target === nodeId,
      )
    : false;

  // Highlight if: 1) part of hovered edge, or 2) user is dragging from a source handle on a different node AND not already connected
  const isHighlighted =
    (hoveredEdge && hoveredEdge.target === nodeId) ||
    (connectingFrom &&
      connectingFrom.nodeId !== nodeId &&
      connectingFrom.handleType === "source" &&
      !isAlreadyConnected);

  const horizontalOffset = -15;

  return (
    <Handle
      type="target"
      position={Position.Left}
      style={{
        ...HANDLE_STYLE,
        left: horizontalOffset,
        top: 18,
      }}
      className={isHighlighted ? "highlighted" : undefined}
    />
  );
}

function RightHandle({ data, nodeId }: BlockProps) {
  // Hide right handle for template nodes, pending connection nodes, and annotation nodes
  const isTemplate = (data as any).isTemplate;
  const isPendingConnection = (data as any).isPendingConnection;
  if (isTemplate || isPendingConnection || data.type === "annotation") return null;

  const channels = data.outputChannels || ["default"];

  // Get hovered edge info and connecting state
  const hoveredEdge = (data as any)._hoveredEdge;
  const connectingFrom = (data as any)._connectingFrom;
  const allEdges = (data as any)._allEdges || [];

  // Single channel: render one handle that respects collapsed state
  if (channels.length === 1) {
    // Check if already connected to the target being dragged
    const isAlreadyConnected = connectingFrom
      ? allEdges.some(
          (edge: any) =>
            edge.source === nodeId && edge.sourceHandle === channels[0] && edge.target === connectingFrom.nodeId,
        )
      : false;

    const isHighlighted =
      (hoveredEdge && hoveredEdge.source === nodeId && hoveredEdge.sourceHandle === channels[0]) ||
      (connectingFrom && connectingFrom.nodeId === nodeId && connectingFrom.handleId === channels[0]) ||
      (connectingFrom &&
        connectingFrom.nodeId !== nodeId &&
        connectingFrom.handleType === "target" &&
        !isAlreadyConnected);

    const horizontalOffset = -15;

    return (
      <Handle
        type="source"
        position={Position.Right}
        id={channels[0]}
        style={{
          ...HANDLE_STYLE,
          right: horizontalOffset,
          top: 18,
        }}
        className={isHighlighted ? "highlighted" : undefined}
      />
    );
  }

  // Helper function to check if a channel handle should be highlighted
  const getChannelHighlight = (channel: string) => {
    const isAlreadyConnected = connectingFrom
      ? allEdges.some(
          (edge: any) =>
            edge.source === nodeId && edge.sourceHandle === channel && edge.target === connectingFrom.nodeId,
        )
      : false;

    return (
      (hoveredEdge && hoveredEdge.source === nodeId && hoveredEdge.sourceHandle === channel) ||
      (connectingFrom && connectingFrom.nodeId === nodeId && connectingFrom.handleId === channel) ||
      (connectingFrom &&
        connectingFrom.nodeId !== nodeId &&
        connectingFrom.handleType === "target" &&
        !isAlreadyConnected)
    );
  };

  // Multiple channels: tree layout with trunk, branches, labels, and handles
  // Visual: [node]──┬── label ─○
  //                 ├── label ─○
  //                 └── label ─○

  // Layout constants
  const channelSpacing = 24;
  const handleSize = 12;
  const edgeColor = "#C9D5E1";

  // Horizontal positions (left to right)
  const trunkLength = 16;
  const branchEndX = trunkLength + 24; // Where diagonal branches end
  const labelStartX = branchEndX + 4; // Label position with padding
  const lineEndX = branchEndX + 62; // Where horizontal line ends (reserves space for labels)
  const handleGap = 4;
  const handleLeftX = lineEndX + handleGap; // Handle's left edge position

  // SVG dimensions
  const totalHeight = (channels.length - 1) * channelSpacing;
  const svgHeight = totalHeight + 40;
  const svgCenterY = svgHeight / 2;

  // Pre-calculate channel positions
  const channelPositions = channels.map((channel, index) => ({
    channel,
    offsetY: (index - (channels.length - 1) / 2) * channelSpacing,
    isHighlighted: getChannelHighlight(channel),
  }));

  return (
    <div
      className="absolute"
      style={{
        left: "100%",
        top: 0,
        bottom: 0,
        pointerEvents: "none",
      }}
    >
      {/* SVG for trunk and branch lines */}
      <svg
        width={lineEndX}
        height={svgHeight}
        style={{
          position: "absolute",
          left: 0,
          top: `calc(50% - ${svgCenterY}px)`,
          overflow: "visible",
        }}
      >
        {/* Trunk line from node */}
        <line x1={0} y1={svgCenterY} x2={trunkLength} y2={svgCenterY} stroke={edgeColor} strokeWidth={3} />

        {/* Branch lines for each channel */}
        {channelPositions.map(({ channel, offsetY }) => {
          const y = svgCenterY + offsetY;
          return (
            <g key={channel}>
              {/* Diagonal from trunk to channel Y */}
              <line x1={trunkLength} y1={svgCenterY} x2={branchEndX} y2={y} stroke={edgeColor} strokeWidth={3} />
              {/* Horizontal line through label area to before handle */}
              <line x1={branchEndX} y1={y} x2={lineEndX} y2={y} stroke={edgeColor} strokeWidth={3} />
            </g>
          );
        })}
      </svg>

      {/* Labels and handles for each channel */}
      {channelPositions.map(({ channel, offsetY, isHighlighted }) => (
        <React.Fragment key={channel}>
          {/* Label with background to cover the line */}
          <span
            className="text-xs font-medium whitespace-nowrap absolute"
            style={{
              left: labelStartX,
              top: `calc(50% + ${offsetY}px)`,
              transform: "translateY(-50%)",
              color: "#8B9AAC",
              lineHeight: `${handleSize}px`,
              background: "#F8FAFC",
              paddingLeft: 4,
              paddingRight: 4,
            }}
          >
            {channel}
          </span>

          {/* Handle */}
          <Handle
            type="source"
            position={Position.Right}
            id={channel}
            style={{
              ...HANDLE_STYLE,
              left: handleLeftX,
              top: `calc(50% + ${offsetY}px)`,
              transform: "translateY(-50%)",
              pointerEvents: "auto",
            }}
            className={isHighlighted ? "highlighted" : undefined}
          />
        </React.Fragment>
      ))}
    </div>
  );
}

function AiPopup({ show, suggestion, onApply, onDismiss }: BlockAi) {
  if (!show) return null;
  if (!suggestion) return null;

  const handleApply = (e: React.MouseEvent) => {
    e.stopPropagation();
    onApply();
  };

  const handleDismiss = (e: React.MouseEvent) => {
    e.stopPropagation();
    onDismiss();
  };

  return (
    <div className="absolute left-0 -translate-y-[100%] text-left text-base">
      <div className="bg-white rounded-lg shadow p-3 relative mb-2 border-blue-500 border-2">
        <div className="flex items-center gap-1 mb-2">
          <SparklesIcon className="inline-block text-blue-500" size={14} />
          <div className="text-gray-800 font-bold">Improvements</div>
        </div>

        <div className="text-sm">{suggestion}</div>

        <div className="flex gap-2 mt-2">
          <Button size="sm" variant="default" className="mt-2" onClick={handleApply}>
            Apply
          </Button>

          <Button size="sm" variant="secondary" className="mt-2" onClick={handleDismiss}>
            Dismiss
          </Button>
        </div>
      </div>
    </div>
  );
}

//
// Block content is the inner area of the block.
//

function BlockContent({
  data,
  onExpand,
  nodeId,
  selected = false,
  onAnnotationUpdate,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onEdit,
  onConfigure,
  onDuplicate,
  onDeactivate,
  onToggleCollapse,
  onToggleView,
  onDelete,
  isCompactView,
}: BlockProps) {
  const compactView =
    isCompactView ??
    (() => {
      switch (data.type) {
        case "composite":
          return !!data.composite?.collapsed;
        case "trigger":
          return !!data.trigger?.collapsed;
        case "switch":
          return !!data.switch?.collapsed;
        case "component":
          return !!data.component?.collapsed;
        case "merge":
          return !!data.merge?.collapsed;
        default:
          return false;
      }
    })();
  const handleExpand = () => {
    if (onExpand && nodeId) {
      onExpand(nodeId, data);
    }
  };

  const actionProps = {
    onRun,
    runDisabled,
    runDisabledTooltip,
    onEdit,
    onDuplicate,
    onDeactivate,
    onToggleCollapse,
    onToggleView,
    onDelete,
    isCompactView: compactView,
    onConfigure: data.type === "composite" ? onConfigure : undefined,
  };

  switch (data.type) {
    case "trigger":
      return <Trigger {...(data.trigger as TriggerProps)} selected={selected} {...actionProps} />;
    case "component":
      return <ComponentBase {...(data.component as ComponentBaseProps)} selected={selected} {...actionProps} />;
    case "composite":
      return (
        <Composite
          {...(data.composite as CompositeProps)}
          onExpandChildEvents={handleExpand}
          selected={selected}
          {...actionProps}
        />
      );
    case "switch":
      return <SwitchComponent {...(data.switch as SwitchComponentProps)} selected={selected} {...actionProps} />;
    case "merge":
      return <MergeComponent {...(data.merge as MergeComponentProps)} selected={selected} {...actionProps} />;
    case "annotation": {
      const handleAnnotationUpdate = (updates: { text?: string; color?: string }) => {
        if (nodeId && onAnnotationUpdate) {
          onAnnotationUpdate(nodeId, updates);
        }
      };
      return (
        <AnnotationComponent
          {...(data.annotation as AnnotationComponentProps)}
          noteId={nodeId}
          selected={selected}
          onAnnotationUpdate={handleAnnotationUpdate}
          {...actionProps}
        />
      );
    }
    default:
      throw new Error(`Unknown block type: ${(data as BlockData).type}`);
  }
}
