import type { LiHTMLAttributes, ReactNode } from "react";

import { KANBAN_CARD_TRANSITION_CLASS, kanbanViewTransitionName } from "./kanbanCardMotion";

type KanbanCardMotionItemProps = LiHTMLAttributes<HTMLLIElement> & {
  /** Work-order id. When unset, the row is not part of a view transition. */
  id?: string;
  children: ReactNode;
};

/** Card list item that shares a view-transition name across columns. */
export function KanbanCardMotionItem({ id, children, style, ...liProps }: KanbanCardMotionItemProps) {
  if (!id) {
    return <li {...liProps}>{children}</li>;
  }

  return (
    <li
      {...liProps}
      style={{
        ...style,
        viewTransitionName: kanbanViewTransitionName(id),
        viewTransitionClass: KANBAN_CARD_TRANSITION_CLASS,
      }}
    >
      {children}
    </li>
  );
}
