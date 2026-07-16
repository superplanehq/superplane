import type { CSSProperties } from "react";

const JSON_VIEW_FONT_FAMILY =
  'Monaco, Menlo, "Cascadia Code", "Segoe UI Mono", "Roboto Mono", Consolas, "Courier New", monospace';

type CssCustomProperties = {
  [key: `--${string}`]: string | number | undefined;
};

type JsonViewStyle = CSSProperties & CssCustomProperties;

export const lightJsonViewStyle: JsonViewStyle = {
  fontSize: "12px",
  fontFamily: JSON_VIEW_FONT_FAMILY,
  backgroundColor: "#ffffff",
  color: "#24292e",
};

export const darkJsonViewStyle: JsonViewStyle = {
  ...lightJsonViewStyle,
  backgroundColor: "#27272a",
  color: "#e5e7eb",
  "--w-rjv-background-color": "#27272a",
  "--w-rjv-color": "#e5e7eb",
  "--w-rjv-line-color": "#404348",
  "--w-rjv-arrow-color": "#9ca3af",
  "--w-rjv-curlybraces-color": "#d1d5db",
  "--w-rjv-colon-color": "#9ca3af",
  "--w-rjv-brackets-color": "#d1d5db",
  "--w-rjv-key-string": "#a5b4fc",
  "--w-rjv-key-number": "#a5b4fc",
  "--w-rjv-type-string-color": "#fcd34d",
  "--w-rjv-type-int-color": "#86efac",
  "--w-rjv-type-float-color": "#86efac",
  "--w-rjv-type-boolean-color": "#93c5fd",
  "--w-rjv-type-null-color": "#93c5fd",
  "--w-rjv-info-color": "#6b7280",
};

export const jsonViewClassName = "json-viewer-hide-types json-viewer-wrap-values";

export function getJsonViewStyle(theme: "light" | "dark"): CSSProperties {
  return theme === "dark" ? darkJsonViewStyle : lightJsonViewStyle;
}
