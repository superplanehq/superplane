import JsonView from "@uiw/react-json-view";
import type { CSSProperties } from "react";

import { useTheme } from "@/contexts/useTheme";
import { escapeJsonStringValue, getJsonViewStyle, jsonViewClassName } from "@/lib/jsonViewTheme";
import { cn } from "@/lib/utils";

/**
 * The one JSON viewer used across the app (run inspector, node detail tabs,
 * component payload previews, event payloads). It keeps every string value
 * whole — the underlying viewer truncates strings after 30 chars by default,
 * which hides the tail of error messages users need — and wraps long lines so
 * they stay inside the panel.
 *
 * The style comes from the active theme; pass `jsonViewStyle` to layer extras
 * (e.g. padding) on top of it.
 */
export function JsonPayload({
  value,
  collapsed = 2,
  jsonViewStyle,
}: {
  value: unknown;
  collapsed?: boolean | number;
  jsonViewStyle?: CSSProperties;
}) {
  const { resolvedTheme } = useTheme();
  const style = { ...getJsonViewStyle(resolvedTheme), ...jsonViewStyle };

  return (
    <JsonView
      value={(value ?? {}) as object}
      collapsed={collapsed}
      shortenTextAfterLength={0}
      style={style}
      className={jsonViewClassName}
      displayObjectSize={false}
      enableClipboard={false}
    >
      <JsonView.String
        render={({ children, ...props }, { type, value: stringValue }) => {
          if (type !== "value") return undefined;

          const displayValue = typeof children === "string" ? children : String(stringValue ?? "");

          return (
            <>
              <span aria-hidden className={props.className}>
                &quot;
              </span>
              <span {...props} className={cn(props.className, "wrap-anywhere whitespace-pre-wrap")}>
                {escapeJsonStringValue(displayValue)}
              </span>
              <span aria-hidden className={props.className}>
                &quot;
              </span>
            </>
          );
        }}
      />
    </JsonView>
  );
}
