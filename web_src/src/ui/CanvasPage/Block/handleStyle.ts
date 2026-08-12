import type React from "react";
import { FACTORY_HANDLE_STYLE } from "@/lib/factoryCanvasChrome";

export const HANDLE_STYLE = {
  width: 12,
  height: 12,
  borderRadius: 100,
  border: "3px solid var(--sp-handle-border, #C9D5E1)",
  background: "transparent",
} satisfies React.CSSProperties;

/** Pull factory square ports outside the card (flush `0` sat on the border and disappeared). */
export const FACTORY_HANDLE_OUTSET_PX = 12;

export function resolveHandleStyle(isFactoryApp: boolean): React.CSSProperties {
  return isFactoryApp ? FACTORY_HANDLE_STYLE : HANDLE_STYLE;
}
