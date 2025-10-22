import "reactflow/dist/style.css";

import { Handle, Position } from "reactflow";

type BlockState = "pending" | "working";
type BlockType = "trigger" | "composite";

interface BlockData {
  label: string;

  state: BlockState;
  type: BlockType;

  // last input event received by this block (for simulation display)
  lastEvent?: unknown;
}

interface BlockProps {
  data: BlockData;
}

export function Block({ data }: BlockProps) {
  const style =
    data.state === "working"
      ? { background: "lightblue" }
      : { background: "lightgray" };

  return (
    <div className="text-left" style={style}>
      <LeftHandle data={data} />

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

      <RightHandle data={data} />
    </div>
  );
}

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
