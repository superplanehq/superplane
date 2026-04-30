import { Plus } from "lucide-react";
import type React from "react";

export const APPEND_CONNECTOR_COLOR = "#C9D5E1";

const APPEND_HANDLE_STYLE: React.CSSProperties = {
  width: 24,
  height: 24,
  borderRadius: 100,
  border: `3px solid ${APPEND_CONNECTOR_COLOR}`,
  background: "white",
  pointerEvents: "auto",
  cursor: "pointer",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  color: "#94A3B8",
};

export function AppendHandleButton({ label, style }: { label: string; style: React.CSSProperties }) {
  return (
    <button
      type="button"
      aria-label={label}
      className="sp-append-source-handle hover:text-slate-500"
      style={{
        ...APPEND_HANDLE_STYLE,
        ...style,
      }}
    >
      <Plus aria-hidden="true" className="h-4 w-4" strokeWidth={3} />
    </button>
  );
}
