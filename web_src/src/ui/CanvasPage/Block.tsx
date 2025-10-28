import { Approval, type ApprovalProps } from "@/ui/approval";
import { Composite, type CompositeProps } from "@/ui/composite";
import {
  SwitchComponent,
  type SwitchComponentProps,
} from "@/ui/switchComponent";
import { Trigger, type TriggerProps } from "@/ui/trigger";
import { Handle, Position } from "@xyflow/react";
import { Filter, FilterProps } from "../filter";
import { If, IfProps } from "../if";
import { Noop, NoopProps } from "../noop";
import { ComponentActionsProps } from "../types/componentActions";

type BlockState = "pending" | "working" | "success" | "failed" | "running";
type BlockType =
  | "trigger"
  | "composite"
  | "approval"
  | "filter"
  | "if"
  | "noop"
  | "switch";

export interface BlockData {
  label: string;

  state: BlockState;
  type: BlockType;

  // last input event received by this block (for simulation display)
  lastEvent?: unknown;

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

  // switch node specific props
  switch?: SwitchComponentProps;
}

interface BlockProps extends ComponentActionsProps {
  data: BlockData;
  nodeId?: string;
  selected?: boolean;

  onExpand?: (nodeId: string, nodeData: BlockData) => void;
  onClick?: () => void;
}

export function Block(props: BlockProps) {
  const data = props.data;

  return (
    <div className="relative w-fit" onClick={props.onClick}>
      <LeftHandle data={data} />
      <BlockContent {...props} />
      <RightHandle data={data} />
    </div>
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
    (data.type === "switch" && data.switch?.collapsed);

  return (
    <Handle
      type="source"
      position={Position.Right}
      style={{
        ...HANDLE_STYLE,
        right: -15,
        top: isCollapsed ? "50%" : 30,
        transform: isCollapsed ? "translateY(-50%)" : undefined,
      }}
    />
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
      return <Filter {...(data.filter as FilterProps)} selected={selected} />;
    case "if":
      return <If {...(data.if as IfProps)} selected={selected} />;
    case "noop":
      return <Noop {...(data.noop as NoopProps)} selected={selected} />;
    case "switch":
      return <SwitchComponent {...(data.switch as SwitchComponentProps)} selected={selected} />;
    default:
      throw new Error(`Unknown block type: ${(data as BlockData).type}`);
  }
}

