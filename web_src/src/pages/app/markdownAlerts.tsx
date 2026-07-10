import { Children, cloneElement, isValidElement } from "react";
import type { ReactElement, ReactNode } from "react";

import {
  isMarkdownAlertType,
  MARKDOWN_ALERT_LABELS,
  type MarkdownAlertType,
  markdownAlertLabelClassName,
  markdownAlertShellClassName,
  MARKDOWN_ALERT_BODY_CLASSES,
} from "./markdownAlertStyles";

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
  const kids = skipLeadingWhitespaceNodes(Children.toArray(children));
  const first = kids[0];
  if (!isValidElement<{ children?: ReactNode }>(first)) {
    return null;
  }

  const inner = Children.toArray(first.props.children);
  const markerIndex = inner.findIndex((child) => typeof child === "string" && child.trim().length > 0);
  if (markerIndex < 0) {
    return null;
  }

  const markerText = String(inner[markerIndex]).trim();
  const match = markerText.match(ALERT_MARKER_RE);
  if (!match) {
    return null;
  }

  const typeName = match[1].toUpperCase();
  if (!isMarkdownAlertType(typeName)) {
    return null;
  }

  let restInner = inner.slice(markerIndex + 1);
  restInner = skipLeadingWhitespaceNodes(restInner);
  if (restInner.length > 0 && isBreakElement(restInner[0])) {
    restInner = restInner.slice(1);
    restInner = skipLeadingWhitespaceNodes(restInner);
  }
  if (restInner.length > 0 && typeof restInner[0] === "string") {
    restInner = [restInner[0].replace(/^\n/, ""), ...restInner.slice(1)];
  }

  const restKids = kids.slice(1);
  const body = buildAlertBody(first, restInner, restKids);

  return { type: typeName, body };
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

function buildAlertBody(
  firstParagraph: ReactElement<{ children?: ReactNode }>,
  restInner: ReactNode[],
  restKids: ReactNode[],
): ReactNode {
  if (restInner.length === 0) {
    return restKids;
  }

  const rewrittenFirst = cloneElement(firstParagraph, undefined, restInner);
  if (restKids.length === 0) {
    return rewrittenFirst;
  }

  return [rewrittenFirst, ...restKids];
}

function skipLeadingWhitespaceNodes(nodes: ReactNode[]): ReactNode[] {
  let index = 0;
  while (index < nodes.length && typeof nodes[index] === "string" && !String(nodes[index]).trim()) {
    index += 1;
  }
  return nodes.slice(index);
}

function isBreakElement(node: ReactNode): boolean {
  return isValidElement(node) && node.type === "br";
}
