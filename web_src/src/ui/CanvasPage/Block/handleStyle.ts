import type React from "react";
import { FACTORY_HANDLE_STYLE } from "@/lib/factoryCanvasChrome";

export const HANDLE_STYLE = {
  width: 12,
  height: 12,
  borderRadius: 100,
  border: "3px solid var(--sp-handle-border, #C9D5E1)",
  background: "transparent",
} satisfies React.CSSProperties;

/**
 * Half the factory handle height/width so square ports sit half in / half out
 * of the node edge (`bottom: -5` + height 10 → straddle border).
 */
export const FACTORY_HANDLE_OUTSET_PX = FACTORY_HANDLE_STYLE.height / 2;

export function resolveHandleStyle(isFactoryApp: boolean): React.CSSProperties {
  return isFactoryApp ? FACTORY_HANDLE_STYLE : HANDLE_STYLE;
}
