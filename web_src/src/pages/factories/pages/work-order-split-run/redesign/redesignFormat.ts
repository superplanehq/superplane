/** Formatting helpers and class names shared by the redesign variants. */

export function formatClock(iso?: string): string {
  const time = Date.parse(iso ?? "");
  if (!Number.isFinite(time)) {
    return "";
  }
  return new Date(time).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", hour12: false });
}

export const MONO_LOG_CLASSNAME =
  "whitespace-pre-wrap break-words font-mono text-[12px] leading-5 text-foreground/90 [tab-size:2]";

export const META_TEXT_CLASSNAME = "text-[12px] text-muted-foreground tabular-nums";
