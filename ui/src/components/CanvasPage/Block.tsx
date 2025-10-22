// No global CSS import here; keep node rendering isolated from library defaults.

import { Trigger, type TriggerProps } from "@/components/trigger";
import { Handle, Position } from "@xyflow/react";

type BlockState = "pending" | "working";
type BlockType = "trigger" | "composite";

interface BlockData {
  label: string;

  state: BlockState;
  type: BlockType;

  // last input event received by this block (for simulation display)
  lastEvent?: unknown;

  // trigger node specific props
  trigger?: TriggerProps;
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
  width: 4,
  height: 4,
  borderRadius: 100,
  border: "1px solid black",
  background: "transparent",
};

function LeftHandle({ data }: BlockProps) {
  if (data.type === "trigger") return null;

  return (
    <Handle
      type="target"
      position={Position.Left}
      style={{ ...HANDLE_STYLE, left: -10 }}
    />
  );
}

function RightHandle(_props: BlockProps) {
  return (
    <Handle
      type="source"
      position={Position.Right}
      style={{ ...HANDLE_STYLE, right: -10 }}
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
      return <Composite data={data} />;
    default:
      throw new Error(`Unknown block type: ${(data as any).type}`);
  }
}

function Composite({ data }: BlockProps) {
  const style =
    data.state === "working"
      ? { background: "lightblue" }
      : { background: "lightgray" };

  return (
    <div className="text-left" style={style}>
      <div className="font-medium text-gray-800">
        <div className="text-[10px]">{data?.label}</div>
      </div>

      <div className="text-[10px]">
        {data.state === "working" ? "Brrrrr...." : ""}
      </div>

      {data.lastEvent !== undefined && (
        <div className="text-[10px] text-gray-700 break-all px-1 py-0.5">
          evt:{" "}
          {typeof data.lastEvent === "string"
            ? data.lastEvent
            : JSON.stringify(data.lastEvent)}
        </div>
      )}
    </div>
  );
}
