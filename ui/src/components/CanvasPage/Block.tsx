import "reactflow/dist/style.css";

import { Handle, Position } from "reactflow";

const HANDLE_STYLE = {
  width: 4,
  height: 4,
  borderRadius: 100,
  border: "1px solid black",
  background: "transparent",
};

const LEFT_HANDLE_STYLE = {
  ...HANDLE_STYLE,
  left: -10,
};

const RIGHT_HANDLE_STYLE = {
  ...HANDLE_STYLE,
  right: -10,
};

type NodeState = "pending" | "working";

interface BlockData {
  label?: string;
  state?: NodeState;
}

export function InputBlock({ data }: { data: BlockData }) {
  return <Block data={data} rightHandle />;
}

export function OutputBlock({ data }: { data: BlockData }) {
  return <Block data={data} leftHandle />;
}

export function DefaultBlock({ data }: { data: BlockData }) {
  return <Block data={data} leftHandle rightHandle />;
}

interface BlockProps {
  data: BlockData;
  leftHandle?: boolean;
  rightHandle?: boolean;
}

export function Block({ data, leftHandle, rightHandle }: BlockProps) {
  const style =
    data.state === "working"
      ? { background: "lightblue" }
      : { background: "lightgray" };

  return (
    <div className="text-left p-0" style={style}>
      {leftHandle && (
        <Handle
          type="target"
          position={Position.Left}
          style={LEFT_HANDLE_STYLE}
        />
      )}

      <div className="font-medium text-gray-800">
        <div className="text-[10px]">{data?.label}</div>
      </div>

      <div className="text-[10px]">
        {data.state === "working" ? "Brrrrr...." : ""}
      </div>

      {rightHandle && (
        <Handle
          type="source"
          position={Position.Right}
          style={RIGHT_HANDLE_STYLE}
        />
      )}
    </div>
  );
}
