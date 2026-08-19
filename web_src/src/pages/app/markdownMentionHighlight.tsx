import { Children, cloneElement, isValidElement, type ReactNode } from "react";

import type { WorkOrderMentionCandidate } from "@/lib/workOrderMentions";

import { WorkOrderMentionText } from "./markdownMentions";

const SKIP_MENTION_TAGS = new Set(["code", "pre", "button", "svg", "img", "iframe"]);

export function highlightMentionChildren(children: ReactNode, people: WorkOrderMentionCandidate[]): ReactNode {
  if (people.length === 0) {
    return children;
  }

  return Children.map(children, (child) => {
    if (typeof child === "string" || typeof child === "number") {
      return <WorkOrderMentionText text={String(child)} people={people} />;
    }
    if (!isValidElement<{ children?: ReactNode }>(child)) {
      return child;
    }
    if (typeof child.type === "string" && SKIP_MENTION_TAGS.has(child.type)) {
      return child;
    }
    if (child.type === WorkOrderMentionText) {
      return child;
    }
    if (child.props.children == null) {
      return child;
    }
    return cloneElement(child, undefined, highlightMentionChildren(child.props.children, people));
  });
}
