import { Children, cloneElement, isValidElement } from "react";
import type { ReactElement, ReactNode } from "react";

/**
 * Shared split of a markdown blockquote’s first text line from the rest of the
 * body. With `remark-breaks`, GitHub-style two-line quotes often arrive as a
 * single `<p>` whose children are `marker`, `<br>`, then body — not two paragraphs.
 */
export type BlockquoteMarkerSplit = {
  firstParagraph: ReactElement<{ children?: ReactNode }>;
  markerText: string;
  body: ReactNode;
};

export function splitBlockquoteMarkerLine(children: ReactNode): BlockquoteMarkerSplit | null {
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
  let restInner = inner.slice(markerIndex + 1);
  restInner = skipLeadingWhitespaceNodes(restInner);
  if (restInner.length > 0 && isBreakElement(restInner[0])) {
    restInner = restInner.slice(1);
    restInner = skipLeadingWhitespaceNodes(restInner);
  }
  if (restInner.length > 0 && typeof restInner[0] === "string") {
    restInner = [restInner[0].replace(/^\n/, ""), ...restInner.slice(1)];
  }

  return {
    firstParagraph: first,
    markerText,
    body: buildBlockquoteBody(first, restInner, kids.slice(1)),
  };
}

function buildBlockquoteBody(
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
