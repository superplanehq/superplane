import { Trigger, type TriggerProps } from "@/ui/trigger";
import { Composite, type CompositeProps } from "@/ui/composite";
import { Approval, type ApprovalProps } from "@/ui/approval";
import { Filter, type FilterProps } from "@/ui/filter";
import { If, type IfProps } from "@/ui/if";
import { Noop, type NoopProps } from "@/ui/noop";
import { SwitchComponent, type SwitchComponentProps } from "@/ui/switchComponent";
import { Handle, Position } from "@xyflow/react";

type BlockState = "pending" | "working" | "success" | "failed" | "running";
type BlockType = "trigger" | "composite" | "approval" | "filter" | "if" | "noop" | "switch";

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

interface BlockProps {
  data: BlockData;
  onExpand?: (nodeId: string, nodeData: BlockData) => void;
  nodeId?: string;
}

export function Block({ data, onExpand, nodeId }: BlockProps) {
  return (
    <div>
      <LeftHandle data={data} />
      <BlockContent data={data} onExpand={onExpand} nodeId={nodeId} />
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
        left: -10,
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
        right: -10,
        top: isCollapsed ? "50%" : 30,
        transform: isCollapsed ? "translateY(-50%)" : undefined,
      }}
    />
  );
}

//
// Block content is the inner area of the block.
//

function BlockContent({ data, onExpand, nodeId }: BlockProps) {
  const handleExpand = () => {
    if (onExpand && nodeId) {
      onExpand(nodeId, data);
    }
  };

  switch (data.type) {
    case "trigger":
      return <Trigger {...(data.trigger as TriggerProps)} />;
    case "composite":
      return <Composite {...(data.composite as CompositeProps)} onExpandChildEvents={handleExpand} />;
    case "approval":
      return <Approval {...(data.approval as ApprovalProps)} />;
    case "filter":
      return <Filter {...(data.filter as FilterProps)} />;
    case "if":
      return <If {...(data.if as IfProps)} />;
    case "noop":
      return <Noop {...(data.noop as NoopProps)} />;
    case "switch":
      return <SwitchComponent {...(data.switch as SwitchComponentProps)} />;
    default:
      throw new Error(`Unknown block type: ${(data as BlockData).type}`);
  }
}
