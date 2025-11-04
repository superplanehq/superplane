import { Approval, type ApprovalProps } from "@/ui/approval";
import { Composite, type CompositeProps } from "@/ui/composite";
import {
  SwitchComponent,
  type SwitchComponentProps,
} from "@/ui/switchComponent";
import { Trigger, type TriggerProps } from "@/ui/trigger";
import { Handle, Position } from "@xyflow/react";
import { SparklesIcon } from "lucide-react";
import { Button } from "../button";
import { Filter, FilterProps } from "../filter";
import { If, IfProps } from "../if";
import { Noop, NoopProps } from "../noop";
import { Http, HttpProps } from "../http";
import { Wait, WaitProps } from "../wait";
import { TimeGate, TimeGateProps } from "../timeGate";
import { ComponentActionsProps } from "../types/componentActions";

type BlockState = "pending" | "working" | "success" | "failed" | "running";
type BlockType =
  | "trigger"
  | "composite"
  | "approval"
  | "filter"
  | "if"
  | "noop"
  | "http"
  | "wait"
  | "time_gate"
  | "switch";

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

  // composite node specific props
  composite?: CompositeProps;

  // approval node specific props
  approval?: ApprovalProps;

  // filter node specific props
  filter?: FilterProps;

  // if node specific props
  if?: IfProps;

  // noop node specific props
  noop?: NoopProps;

  // http node specific props
  http?: HttpProps;

  // wait node specific props
  wait?: WaitProps;

  // time_gate node specific props
  time_gate?: TimeGateProps;

  // switch node specific props
  switch?: SwitchComponentProps;
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
    onApply: () => { },
    onDismiss: () => { },
  };

  return (
    <>
      <AiPopup {...ai} />

      <div className="relative w-fit" onClick={props.onClick}>
        <LeftHandle data={data} />
        <BlockContent {...props} />
        <RightHandle data={data} />
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

function LeftHandle({ data }: BlockProps) {
  if (data.type === "trigger") return null;

  const isCollapsed =
    (data.type === "composite" && data.composite?.collapsed) ||
    (data.type === "approval" && data.approval?.collapsed) ||
    (data.type === "filter" && data.filter?.collapsed) ||
    (data.type === "if" && data.if?.collapsed) ||
    (data.type === "noop" && data.noop?.collapsed) ||
    (data.type === "http" && data.http?.collapsed) ||
    (data.type === "wait" && data.wait?.collapsed) ||
    (data.type === "time_gate" && data.time_gate?.collapsed) ||
    (data.type === "switch" && data.switch?.collapsed);

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
    />
  );
}

function RightHandle({ data }: BlockProps) {
  const isCollapsed =
    (data.type === "composite" && data.composite?.collapsed) ||
    (data.type === "approval" && data.approval?.collapsed) ||
    (data.type === "trigger" && data.trigger?.collapsed) ||
    (data.type === "filter" && data.filter?.collapsed) ||
    (data.type === "if" && data.if?.collapsed) ||
    (data.type === "noop" && data.noop?.collapsed) ||
    (data.type === "http" && data.http?.collapsed) ||
    (data.type === "wait" && data.wait?.collapsed) ||
    (data.type === "time_gate" && data.time_gate?.collapsed) ||
    (data.type === "switch" && data.switch?.collapsed);

  let channels = data.outputChannels || ["default"];

  const isIfOrSwitch = data.type === "if" || data.type === "switch";

  if (isIfOrSwitch && !isCollapsed) {
    return null;
  }

  if (data.type === "if") {
    channels = ["true", "false"];
  }

  // Single channel: render one handle that respects collapsed state
  if (channels.length === 1) {
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
      />
    );
  }


  const baseTop = isCollapsed ? 30 : 80; // Adjust starting position based on collapsed state
  const spacing = 40; // Space between handles

  return <>
    {channels.map((channel, index) => (
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
          />
        </div>
      </div>
    ))}
  </>
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
          <Button
            size="sm"
            variant="default"
            className="mt-2"
            onClick={handleApply}
          >
            Apply
          </Button>

          <Button
            size="sm"
            variant="secondary"
            className="mt-2"
            onClick={handleDismiss}
          >
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
    onConfigure,
    onDuplicate,
    onDeactivate,
    onToggleView,
    onDelete,
    isCompactView,
  };

  switch (data.type) {
    case "trigger":
      return <Trigger {...(data.trigger as TriggerProps)} selected={selected} {...actionProps} />;
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
    case "noop":
      return <Noop {...(data.noop as NoopProps)} selected={selected} {...actionProps} />;
    case "http":
      return <Http {...(data.http as HttpProps)} selected={selected} {...actionProps} />;
    case "wait":
      return <Wait {...(data.wait as WaitProps)} selected={selected} {...actionProps} />;
    case "time_gate":
      return <TimeGate {...(data.time_gate as TimeGateProps)} selected={selected} {...actionProps} />;
    case "switch":
      return (
        <SwitchComponent
          {...(data.switch as SwitchComponentProps)}
          selected={selected}
          {...actionProps}
        />
      );
    default:
      throw new Error(`Unknown block type: ${(data as BlockData).type}`);
  }
}
