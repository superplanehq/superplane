import { cn } from "@/lib/utils";

export const MARKDOWN_ALERT_TYPES = ["NOTE", "TIP", "IMPORTANT", "WARNING", "CAUTION"] as const;

export type MarkdownAlertType = (typeof MARKDOWN_ALERT_TYPES)[number];

export const MARKDOWN_ALERT_LABELS: Record<MarkdownAlertType, string> = {
  NOTE: "Note",
  TIP: "Tip",
  IMPORTANT: "Important",
  WARNING: "Warning",
  CAUTION: "Caution",
};

/** Shared shell for SuperPlane-chrome GitHub alerts (white surface, thin accent). */
export const MARKDOWN_ALERT_SHELL_CLASSES = "my-3 border-l-2 bg-white px-3 py-2 dark:bg-gray-900/40";

export const MARKDOWN_ALERT_LABEL_CLASSES = "mb-1 text-[11px] font-semibold tracking-wide";

export const MARKDOWN_ALERT_BODY_CLASSES = "[&_p]:mb-2 [&_p:last-child]:mb-0";

const MARKDOWN_ALERT_ACCENT: Record<MarkdownAlertType, string> = {
  NOTE: "border-sky-600 dark:border-sky-400",
  TIP: "border-emerald-600 dark:border-emerald-400",
  IMPORTANT: "border-violet-600 dark:border-violet-400",
  WARNING: "border-amber-600 dark:border-amber-400",
  CAUTION: "border-red-600 dark:border-red-400",
};

const MARKDOWN_ALERT_LABEL_COLOR: Record<MarkdownAlertType, string> = {
  NOTE: "text-sky-700 dark:text-sky-300",
  TIP: "text-emerald-700 dark:text-emerald-300",
  IMPORTANT: "text-violet-700 dark:text-violet-300",
  WARNING: "text-amber-700 dark:text-amber-300",
  CAUTION: "text-red-700 dark:text-red-300",
};

export function markdownAlertShellClassName(type: MarkdownAlertType): string {
  return cn(MARKDOWN_ALERT_SHELL_CLASSES, MARKDOWN_ALERT_ACCENT[type]);
}

export function markdownAlertLabelClassName(type: MarkdownAlertType): string {
  return cn(MARKDOWN_ALERT_LABEL_CLASSES, MARKDOWN_ALERT_LABEL_COLOR[type]);
}

export function isMarkdownAlertType(value: string): value is MarkdownAlertType {
  return (MARKDOWN_ALERT_TYPES as readonly string[]).includes(value);
}
