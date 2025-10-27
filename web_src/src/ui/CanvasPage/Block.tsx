import { Approval, type ApprovalProps } from "@/ui/approval";
import { Composite, type CompositeProps } from "@/ui/composite";
import { Trigger, type TriggerProps } from "@/ui/trigger";
import { Handle, Position } from "@xyflow/react";
import { OnApproveFn, OnRejectFn } from ".";

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
  nodeId?: string;

  onExpand?: (nodeId: string, nodeData: BlockData) => void;
  onApprove?: OnApproveFn;
  onReject?: OnRejectFn;
}

export function Block(props: BlockProps) {
  const data = props.data;

  return (
    <div className="relative w-fit">
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

function BlockContent({
  data,
  onExpand,
  onApprove,
  onReject,
  nodeId,
}: BlockProps) {
  const handleExpand = () => {
    if (onExpand && nodeId) {
      onExpand(nodeId, data);
    }
  };

  switch (data.type) {
    case "trigger":
      return <Trigger {...(data.trigger as TriggerProps)} />;
    case "composite":
      return (
        <Composite
          {...(data.composite as CompositeProps)}
          onExpandChildEvents={handleExpand}
        />
      );
    case "approval":
      return (
        <Approval
          {...prepApprovalData({ data, nodeId, onApprove, onReject })}
        />
      );
    default:
      throw new Error(`Unknown block type: ${(data as BlockData).type}`);
  }
}

function prepApprovalData({
  data,
  nodeId,
  onApprove,
  onReject,
}: BlockProps): ApprovalProps {
  const createApproveFn = (approveId: string) => {
    return (artifact?: Record<string, string>) => {
      if (onApprove && nodeId) {
        onApprove(nodeId, approveId, artifact);
      }
    };
  };

  const createRejectFn = (rejectId: string) => {
    return (comment?: string) => {
      if (onReject && nodeId) {
        onReject(nodeId, rejectId, comment);
      }
    };
  };

  const approvals = data.approval?.approvals || [];

  return {
    ...data.approval!,
    approvals: approvals.map((a) => ({
      ...a,
      onApprove: createApproveFn(a.id),
      onReject: createRejectFn(a.id),
    })),
  };
}
