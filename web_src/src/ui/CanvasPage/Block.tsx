import { Trigger, type TriggerProps } from "@/ui/trigger";
import { Composite, type CompositeProps } from "@/ui/composite";
import { Approval, type ApprovalProps } from "@/ui/approval";
import { Handle, Position } from "@xyflow/react";

type BlockState = "pending" | "working";
type BlockType = "trigger" | "composite" | "approval";

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
}

interface BlockProps {
  data: BlockData;
  onExpand?: (nodeId: string, nodeData: BlockData) => void;
  nodeId?: string;
}

export function Block({ data, onExpand, nodeId }: BlockProps) {
  return (
    <div className="relative w-fit">
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
    (data.type === "approval" && data.approval?.collapsed);

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
    (data.type === "trigger" && data.trigger?.collapsed);

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
    default:
      throw new Error(`Unknown block type: ${(data as BlockData).type}`);
  }
}
