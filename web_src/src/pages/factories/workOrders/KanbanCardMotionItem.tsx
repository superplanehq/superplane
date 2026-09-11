import type { LiHTMLAttributes, ReactNode } from "react";

import { cn } from "@/lib/utils";

import {
  KANBAN_CARD_MOTION_ITEM_CLASS,
  KANBAN_CARD_TRANSITION_CLASS,
  kanbanViewTransitionName,
} from "./kanbanCardMotion";

type KanbanCardMotionItemProps = LiHTMLAttributes<HTMLLIElement> & {
  /** Work-order id. When unset, the row is not part of a view transition. */
  id?: string;
  children: ReactNode;
};

/** Card list item that shares a view-transition name across columns. */
export function KanbanCardMotionItem({ id, children, className, style, ...liProps }: KanbanCardMotionItemProps) {
  if (!id) {
    return (
      <li {...liProps} className={className}>
        {children}
      </li>
    );
  }

  return (
    <li
      {...liProps}
      className={cn(KANBAN_CARD_MOTION_ITEM_CLASS, className)}
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
