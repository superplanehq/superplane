import { Handle, Position } from "@xyflow/react";
import { Plus } from "lucide-react";
import type React from "react";
import { useState } from "react";
import { CANVAS_CONNECTOR_COLOR } from "@/lib/canvasEdgeColors";
import { FACTORY_NODE_CARD_HEIGHT, FACTORY_NODE_CARD_WIDTH } from "@/lib/factoryCanvasChrome";
import { resolveHandleStyle } from "./handleStyle";
import type { BlockProps } from "./types";

const APPEND_SOURCE_LINE_WIDTH = 42;
const APPEND_SOURCE_BUTTON_LEFT = 54;
const APPEND_HORIZONTAL_NUDGE = -4;

const APPEND_HANDLE_STYLE: React.CSSProperties = {
  width: 24,
  height: 24,
  borderRadius: 100,
  border: "3px solid var(--sp-handle-border, #C9D5E1)",
  pointerEvents: "auto",
  cursor: "pointer",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
};

const APPEND_PREVIEW_DOT_BASE_STYLE: React.CSSProperties = {
  width: 12,
  height: 12,
  borderRadius: 100,
  background: "transparent",
};

export type AppendFromNodeHandler = NonNullable<BlockProps["onAppendFromNode"]>;
export type AppendHandleOrientation = "horizontal" | "vertical";

function getAppendSourceHandleClassName(isHighlighted: boolean | undefined, plusHovered: boolean) {
  if (isHighlighted && plusHovered) {
    return "sp-append-source-hitbox highlighted plus-hovered";
  }
  if (isHighlighted) {
    return "sp-append-source-hitbox highlighted";
  }
  if (plusHovered) {
    return "sp-append-source-hitbox plus-hovered";
  }
  return "sp-append-source-hitbox";
}

export function AppendSourceHandle({
  channel,
  label,
  isHighlighted,
  onAppend,
  style,
  orientation = "horizontal",
  lineWidth = APPEND_SOURCE_LINE_WIDTH,
  buttonLeft = APPEND_SOURCE_BUTTON_LEFT,
  buttonTop = -9,
}: {
  channel: string;
  label: string;
  isHighlighted: boolean | undefined;
  onAppend: () => void | Promise<void>;
  style: React.CSSProperties;
  orientation?: AppendHandleOrientation;
  lineWidth?: number;
  buttonLeft?: number;
  buttonTop?: number;
}) {
  const [plusHovered, setPlusHovered] = useState(false);
  const isVertical = orientation === "vertical";

  const handleStyle: React.CSSProperties = {
    ...resolveHandleStyle(isVertical),
    pointerEvents: "auto",
    zIndex: 12,
    ...style,
  };

  return (
    <Handle
      type="source"
      position={isVertical ? Position.Bottom : Position.Right}
      id={channel}
      className={getAppendSourceHandleClassName(isHighlighted, plusHovered)}
      onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
        void onAppend();
      }}
      style={handleStyle}
    >
      <span
        aria-hidden="true"
        data-testid="append-connector-stem"
        style={
          isVertical
            ? {
                position: "absolute",
                left: "50%",
                top: 12,
                width: 3,
                height: lineWidth,
                transform: "translateX(-50%)",
                backgroundColor: CANVAS_CONNECTOR_COLOR,
                pointerEvents: "none",
                zIndex: -1,
              }
            : {
                position: "absolute",
                left: 12 + APPEND_HORIZONTAL_NUDGE,
                top: "50%",
                width: lineWidth,
                height: 3,
                transform: "translateY(-50%)",
                backgroundColor: CANVAS_CONNECTOR_COLOR,
                pointerEvents: "none",
                zIndex: -1,
              }
        }
      />
      <AppendHandleButton
        label={label}
        onClick={onAppend}
        onHoverChange={setPlusHovered}
        style={
          isVertical
            ? {
                left: "50%",
                top: buttonTop,
                position: "absolute",
                transform: "translateX(-50%)",
              }
            : {
                left: buttonLeft + APPEND_HORIZONTAL_NUDGE,
                top: buttonTop,
                position: "absolute",
              }
        }
      />
    </Handle>
  );
}

export function AppendHandleButton({
  label,
  onClick,
  onHoverChange,
  style,
}: {
  label: string;
  onClick: () => void | Promise<void>;
  onHoverChange?: (hovered: boolean) => void;
  style: React.CSSProperties;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      className="sp-append-source-handle"
      onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
        void onClick();
      }}
      onMouseEnter={() => onHoverChange?.(true)}
      onMouseLeave={() => onHoverChange?.(false)}
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
  orientation = "horizontal",
  connectorTop = "50%",
  containerOffsetY = 0,
}: {
  style: React.CSSProperties;
  orientation?: AppendHandleOrientation;
  connectorTop?: number | string;
  containerOffsetY?: number;
}) {
  const previewStemLength = 44;
  const previewConnectorSize = 12;
  const previewContainerGap = 12;
  const previewNudgeLeft = -2;
  const connectorTopValue = typeof connectorTop === "number" ? `${connectorTop}px` : connectorTop;

  if (orientation === "vertical") {
    const previewContainerTop = previewStemLength + previewConnectorSize / 2 + previewContainerGap;

    return (
      <div className="sp-append-source-preview" style={style}>
        <div
          style={{
            position: "absolute",
            left: "50%",
            top: 0,
            width: 3,
            height: previewStemLength,
            transform: "translateX(-50%)",
            backgroundColor: CANVAS_CONNECTOR_COLOR,
            borderRadius: 999,
          }}
        />
        <div
          style={{
            ...APPEND_PREVIEW_DOT_BASE_STYLE,
            border: `3px solid ${CANVAS_CONNECTOR_COLOR}`,
            position: "absolute",
            left: "50%",
            top: previewStemLength + previewConnectorSize / 2,
            transform: "translate(-50%, -50%)",
            pointerEvents: "none",
          }}
        />
        {/* Absolute + translateX(-50%): parent is already centered on the
            stem; marginLeft:-halfWidth would double-offset and push the
            ghost left (line hits the ghost's right edge). */}
        <div
          style={{
            position: "absolute",
            left: "50%",
            top: previewContainerTop + containerOffsetY,
            transform: "translateX(-50%)",
            width: FACTORY_NODE_CARD_WIDTH,
            height: FACTORY_NODE_CARD_HEIGHT,
            borderRadius: 16,
            background: "var(--card, #ffffff)",
            outline: "1px solid rgb(15 23 42 / 0.08)",
            boxShadow: "0 1px 3px rgb(15 23 42 / 0.06), 0 2px 6px rgb(15 23 42 / 0.04)",
            opacity: 0.6,
          }}
        />
      </div>
    );
  }

  const previewContainerLeft = previewStemLength + previewConnectorSize / 2 + previewContainerGap;

  return (
    <div className="sp-append-source-preview" style={style}>
      <div
        style={{
          position: "absolute",
          left: 0,
          top: `calc(${connectorTopValue} + var(--sp-append-preview-connector-offset-y, 0px) + var(--sp-append-preview-offset-y, 0px))`,
          width: previewStemLength,
          height: 3,
          transform: "translateY(-50%)",
          backgroundColor: CANVAS_CONNECTOR_COLOR,
          borderRadius: 999,
        }}
      />
      <div
        style={{
          ...APPEND_PREVIEW_DOT_BASE_STYLE,
          border: `3px solid ${CANVAS_CONNECTOR_COLOR}`,
          position: "absolute",
          left: previewStemLength + previewConnectorSize / 2 + previewNudgeLeft,
          top: `calc(${connectorTopValue} + var(--sp-append-preview-connector-offset-y, 0px) + var(--sp-append-preview-offset-y, 0px))`,
          transform: "translate(-50%, -50%)",
          pointerEvents: "none",
        }}
      />
      <div
        style={{
          marginLeft: previewContainerLeft + previewNudgeLeft,
          marginTop: `calc(${containerOffsetY}px + var(--sp-append-preview-offset-y, 0px))`,
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
