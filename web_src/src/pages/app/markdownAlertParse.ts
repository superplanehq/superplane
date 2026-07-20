import type { ReactNode } from "react";

import { isMarkdownAlertType, type MarkdownAlertType } from "./markdownAlertStyles";
import { splitBlockquoteMarkerLine } from "./markdownBlockquoteMarker";

const ALERT_MARKER_RE = /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]$/i;

export type ParsedMarkdownAlert = {
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
export function parseGithubAlertChildren(children: ReactNode): ParsedMarkdownAlert | null {
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
