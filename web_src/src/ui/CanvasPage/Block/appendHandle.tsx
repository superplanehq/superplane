import { Handle, Position } from "@xyflow/react";
import { Plus } from "lucide-react";
import type React from "react";
import { HANDLE_STYLE } from "./handleStyle";
import type { BlockProps } from "./types";

export const APPEND_CONNECTOR_COLOR = "#C9D5E1";
const APPEND_SOURCE_LINE_WIDTH = 42;
const APPEND_SOURCE_BUTTON_LEFT = 54;

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

const APPEND_PREVIEW_DOT_STYLE: React.CSSProperties = {
  width: 12,
  height: 12,
  borderRadius: 100,
  border: `3px solid ${APPEND_CONNECTOR_COLOR}`,
  background: "transparent",
};

export type AppendFromNodeHandler = NonNullable<BlockProps["onAppendFromNode"]>;

function getAppendSourceHandleClassName(isHighlighted: boolean | undefined) {
  return isHighlighted ? "sp-append-source-hitbox highlighted" : "sp-append-source-hitbox";
}

export function AppendSourceHandle({
  channel,
  label,
  isHighlighted,
  onAppend,
  style,
  lineWidth = APPEND_SOURCE_LINE_WIDTH,
  buttonLeft = APPEND_SOURCE_BUTTON_LEFT,
  buttonTop = -7,
}: {
  channel: string;
  label: string;
  isHighlighted: boolean | undefined;
  onAppend: () => void | Promise<void>;
  style: React.CSSProperties;
  lineWidth?: number;
  buttonLeft?: number;
  buttonTop?: number;
}) {
  const handleStyle: React.CSSProperties & { "--sp-append-source-hitbox-width": string } = {
    ...HANDLE_STYLE,
    "--sp-append-source-hitbox-width": `${buttonLeft + 24}px`,
    pointerEvents: "auto",
    ...style,
  };

  return (
    <Handle
      type="source"
      position={Position.Right}
      id={channel}
      className={getAppendSourceHandleClassName(isHighlighted)}
      onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
        void onAppend();
      }}
      style={handleStyle}
    >
      <span
        aria-hidden="true"
        style={{
          position: "absolute",
          left: 12,
          top: "50%",
          width: lineWidth,
          height: 3,
          transform: "translateY(-50%)",
          backgroundColor: APPEND_CONNECTOR_COLOR,
          pointerEvents: "none",
        }}
      />
      <AppendHandleButton
        label={label}
        onClick={onAppend}
        style={{
          left: buttonLeft,
          top: buttonTop,
          position: "absolute",
          pointerEvents: "none",
        }}
      />
    </Handle>
  );
}

export function AppendHandleButton({
  label,
  onClick,
  style,
}: {
  label: string;
  onClick: () => void | Promise<void>;
  style: React.CSSProperties;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      className="sp-append-source-handle hover:text-slate-500"
      onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
        void onClick();
      }}
      style={{
        ...APPEND_HANDLE_STYLE,
        ...style,
      }}
    >
      <Plus aria-hidden="true" className="h-4 w-4" strokeWidth={3} />
    </button>
  );
}

export function AppendHandlePreview({
  style,
  connectorTop = "50%",
  containerOffsetY = 0,
}: {
  style: React.CSSProperties;
  connectorTop?: number | string;
  containerOffsetY?: number;
}) {
  const previewStemLength = 44;
  const previewConnectorSize = 12;
  const previewContainerGap = 12;
  const previewContainerLeft = previewStemLength + previewConnectorSize / 2 + previewContainerGap;

  return (
    <div className="sp-append-source-preview" style={style}>
      <div
        style={{
          position: "absolute",
          left: 0,
          top: connectorTop,
          width: previewStemLength,
          height: 3,
          transform: "translateY(-50%)",
          backgroundColor: APPEND_CONNECTOR_COLOR,
          borderRadius: 999,
        }}
      />
      <div
        style={{
          ...APPEND_PREVIEW_DOT_STYLE,
          position: "absolute",
          left: previewStemLength + previewConnectorSize / 2,
          top: connectorTop,
          transform: "translate(-50%, -50%)",
          pointerEvents: "none",
        }}
      />
      <div
        style={{
          marginLeft: previewContainerLeft,
          marginTop: containerOffsetY,
          width: "23rem",
          height: 96,
          borderRadius: 8,
          background: "white",
          outline: "1px solid rgb(15 23 42 / 0.15)",
          opacity: 0.6,
        }}
      />
    </div>
  );
}
