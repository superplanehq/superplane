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

export function InputBlock({ data }: { data: { label?: string } }) {
  return <Block data={data} rightHandle />;
}

export function OutputBlock({ data }: { data: { label?: string } }) {
  return <Block data={data} leftHandle />;
}

export function DefaultBlock({ data }: { data: { label?: string } }) {
  return <Block data={data} leftHandle rightHandle />;
}

interface BlockProps {
  data: { label?: string };
  leftHandle?: boolean;
  rightHandle?: boolean;
}

export function Block({ data, leftHandle, rightHandle }: BlockProps) {
  return (
    <div>
      {leftHandle && (
        <Handle
          type="target"
          position={Position.Left}
          style={LEFT_HANDLE_STYLE}
        />
      )}

      <div className="font-medium text-gray-800">{data?.label}</div>

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
