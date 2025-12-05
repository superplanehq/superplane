import { Approval, type ApprovalProps } from "@/ui/approval";
import { Composite, type CompositeProps } from "@/ui/composite";
import { SwitchComponent, type SwitchComponentProps } from "@/ui/switchComponent";
import { Trigger, type TriggerProps } from "@/ui/trigger";
import { Handle, Position } from "@xyflow/react";
import { SparklesIcon } from "lucide-react";
import { Button } from "../button";
import { Filter, FilterProps } from "../filter";
import { If, IfProps } from "../if";
import MergeComponent, { type MergeComponentProps } from "../merge";
import { ComponentActionsProps } from "../types/componentActions";
import { Wait, WaitProps } from "../wait";
import { ComponentBase, ComponentBaseProps } from "../componentBase";

type BlockState = "pending" | "working" | "success" | "failed" | "running";
type BlockType = "trigger" | "component" | "composite" | "approval" | "filter" | "if" | "wait" | "merge" | "switch";

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

  // approval node specific props
  approval?: ApprovalProps;

  // filter node specific props
  filter?: FilterProps;

  // if node specific props
  if?: IfProps;

  // wait node specific props
  wait?: WaitProps;

  // switch node specific props
  switch?: SwitchComponentProps;

  // merge node specific props
  merge?: MergeComponentProps;
}

interface BlockProps extends ComponentActionsProps {
  data: BlockData;
  nodeId?: string;
  selected?: boolean;

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

  return (
    <>
      <AiPopup {...ai} />

      <div className="relative w-fit" onClick={props.onClick}>
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
  if (data.type === "trigger") return null;

  const isCollapsed =
    (data.type === "composite" && data.composite?.collapsed) ||
    (data.type === "approval" && data.approval?.collapsed) ||
    (data.type === "filter" && data.filter?.collapsed) ||
    (data.type === "if" && data.if?.collapsed) ||
    (data.type === "wait" && data.wait?.collapsed) ||
    (data.type === "switch" && data.switch?.collapsed);

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

  return (
    <Handle
      type="target"
      position={Position.Left}
      style={{
        ...HANDLE_STYLE,
        left: -15,
        top: isCollapsed ? "50%" : 30,
        transform: isCollapsed ? "translateY(-50%)" : undefined,
      }}
      className={isHighlighted ? "highlighted" : undefined}
    />
  );
}

function RightHandle({ data, nodeId }: BlockProps) {
  const isCollapsed =
    (data.type === "composite" && data.composite?.collapsed) ||
    (data.type === "approval" && data.approval?.collapsed) ||
    (data.type === "trigger" && data.trigger?.collapsed) ||
    (data.type === "filter" && data.filter?.collapsed) ||
    (data.type === "if" && data.if?.collapsed) ||
    (data.type === "wait" && data.wait?.collapsed) ||
    (data.type === "switch" && data.switch?.collapsed);

  let channels = data.outputChannels || ["default"];

  const isIfOrSwitch = data.type === "if" || data.type === "switch";

  if (isIfOrSwitch && !isCollapsed) {
    return null;
  }

  if (data.type === "if") {
    channels = ["true", "false"];
  }

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

    return (
      <Handle
        type="source"
        position={Position.Right}
        id={channels[0]}
        style={{
          ...HANDLE_STYLE,
          right: -15,
          top: isCollapsed ? "50%" : 30,
          transform: isCollapsed ? "translateY(-50%)" : undefined,
        }}
        className={isHighlighted ? "highlighted" : undefined}
      />
    );
  }

  const baseTop = isCollapsed ? 30 : 80; // Adjust starting position based on collapsed state
  const spacing = 40; // Space between handles

  return (
    <>
      {channels.map((channel, index) => {
        // Check if already connected to the target being dragged
        const isAlreadyConnected = connectingFrom
          ? allEdges.some(
              (edge: any) =>
                edge.source === nodeId && edge.sourceHandle === channel && edge.target === connectingFrom.nodeId,
            )
          : false;

        const isHighlighted =
          (hoveredEdge && hoveredEdge.source === nodeId && hoveredEdge.sourceHandle === channel) ||
          (connectingFrom && connectingFrom.nodeId === nodeId && connectingFrom.handleId === channel) ||
          (connectingFrom &&
            connectingFrom.nodeId !== nodeId &&
            connectingFrom.handleType === "target" &&
            !isAlreadyConnected);

        return (
          <div
            key={channel}
            className="absolute"
            style={{
              left: "100%",
              top: baseTop + index * spacing,
              transform: "translateY(-50%)",
              paddingLeft: 4,
            }}
          >
            <div className="relative flex items-center">
              {/* Small line from node */}
              <div
                style={{
                  width: 20,
                  height: 3,
                  backgroundColor: "#C9D5E1",
                  pointerEvents: "none",
                  marginRight: 4,
                }}
              />
              {/* Label text */}
              <span
                className="text-xs font-medium whitespace-nowrap"
                style={{
                  color: "#8B9AAC",
                  pointerEvents: "none",
                  paddingLeft: 2,
                  paddingRight: 2,
                }}
              >
                {channel}
              </span>
              {/* Small line to handle */}
              <div
                style={{
                  width: 16,
                  height: 3,
                  backgroundColor: "#C9D5E1",
                  pointerEvents: "none",
                  marginLeft: 4,
                }}
              />
              {/* Handle (connection point) */}
              <Handle
                type="source"
                position={Position.Right}
                id={channel}
                style={{
                  ...HANDLE_STYLE,
                  position: "relative",
                  pointerEvents: "auto",
                  marginLeft: -6,
                  top: 5,
                }}
                className={isHighlighted ? "highlighted" : undefined}
              />
            </div>
          </div>
        );
      })}
    </>
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
    isCompactView,
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
    case "approval":
      return <Approval {...(data.approval as ApprovalProps)} selected={selected} {...actionProps} />;
    case "filter":
      return <Filter {...(data.filter as FilterProps)} selected={selected} {...actionProps} />;
    case "if":
      return <If {...(data.if as IfProps)} selected={selected} {...actionProps} />;
    case "wait":
      return <Wait {...(data.wait as WaitProps)} selected={selected} {...actionProps} />;
    case "switch":
      return <SwitchComponent {...(data.switch as SwitchComponentProps)} selected={selected} {...actionProps} />;
    case "merge":
      return <MergeComponent {...(data.merge as MergeComponentProps)} selected={selected} {...actionProps} />;
    default:
      throw new Error(`Unknown block type: ${(data as BlockData).type}`);
  }
}
