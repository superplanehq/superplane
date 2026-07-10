import type { ReactNode } from "react";

import {
  isMarkdownAlertType,
  MARKDOWN_ALERT_LABELS,
  type MarkdownAlertType,
  markdownAlertLabelClassName,
  markdownAlertShellClassName,
  MARKDOWN_ALERT_BODY_CLASSES,
} from "./markdownAlertStyles";
import { splitBlockquoteMarkerLine } from "./markdownBlockquoteMarker";

const ALERT_MARKER_RE = /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]$/i;

type ParsedAlert = {
  type: MarkdownAlertType;
  body: ReactNode;
};

/**
 * If `children` are a GitHub alert blockquote (`[!NOTE]` …), return the type
 * and body with the marker line removed. Otherwise return null.
 *
 * With `remark-breaks`, GitHub’s two-line alert often arrives as a single `<p>`
 * whose children are `[!TYPE]`, `<br>`, then the body — not two paragraphs.
 */
export function parseGithubAlertChildren(children: ReactNode): ParsedAlert | null {
  const split = splitBlockquoteMarkerLine(children);
  if (!split) {
    return null;
  }

  const match = split.markerText.match(ALERT_MARKER_RE);
  if (!match) {
    return null;
  }

  const typeName = match[1].toUpperCase();
  if (!isMarkdownAlertType(typeName)) {
    return null;
  }

  return { type: typeName, body: split.body };
}

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
