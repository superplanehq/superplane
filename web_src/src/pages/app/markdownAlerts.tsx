import type { ReactNode } from "react";

import {
  MARKDOWN_ALERT_LABELS,
  type MarkdownAlertType,
  markdownAlertLabelClassName,
  markdownAlertShellClassName,
  MARKDOWN_ALERT_BODY_CLASSES,
} from "./markdownAlertStyles";

export function MarkdownAlert({ type, children }: { type: MarkdownAlertType; children: ReactNode }) {
  return (
    <aside
      className={markdownAlertShellClassName(type)}
      data-testid={`markdown-alert-${type.toLowerCase()}`}
      data-alert={type.toLowerCase()}
    >
      <div className={markdownAlertLabelClassName(type)}>{MARKDOWN_ALERT_LABELS[type]}</div>
      <div className={MARKDOWN_ALERT_BODY_CLASSES}>{children}</div>
    </aside>
  );
}
