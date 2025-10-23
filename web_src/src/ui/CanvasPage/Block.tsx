import { Trigger, type TriggerProps } from "@/ui/trigger";
import { Composite, type CompositeProps } from "@/ui/composite";
import { Approval, type ApprovalProps } from "@/ui/approval";
import { Handle, Position } from "@xyflow/react";

type BlockState = "pending" | "working";
type BlockType = "trigger" | "composite" | "approval";

interface BlockData {
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
}

export function Block({ data }: BlockProps) {
  return (
    <div>
      <LeftHandle data={data} />
      <BlockContent data={data} />
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

  return (
    <Handle
      type="target"
      position={Position.Left}
      style={{
        ...HANDLE_STYLE,
        left: -10,
        top: 30,
      }}
    />
  );
}

function RightHandle(_props: BlockProps) {
  return (
    <Handle
      type="source"
      position={Position.Right}
      style={{
        ...HANDLE_STYLE,
        right: -10,
        top: 30,
      }}
    />
  );
}

//
// Block content is the inner area of the block.
//

function BlockContent({ data }: BlockProps) {
  switch (data.type) {
    case "trigger":
      return <Trigger {...(data.trigger as TriggerProps)} />;
    case "composite":
      return <Composite {...(data.composite as CompositeProps)} />;
    case "approval":
      return <Approval {...(data.approval as ApprovalProps)} />;
    default:
      throw new Error(`Unknown block type: ${(data as any).type}`);
  }
}
