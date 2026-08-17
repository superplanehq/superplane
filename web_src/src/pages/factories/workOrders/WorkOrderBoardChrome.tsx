import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

/**
 * Lane chrome for every board in the workspace. The Work Orders board and
 * the Lines phase board both render through it.
 */

/** Lane tint. Only in-flight and closed work get colour, as on the Work Orders board. */
export type BoardLaneTone = "neutral" | "running" | "done";

const LANE_TONE_CLASSNAME: Record<BoardLaneTone, string> = {
  neutral: "bg-card/60",
  running: "bg-[color:var(--status-running-lane-bg)]",
  done: "bg-[color:var(--status-done-lane-bg)]",
};

/** Shared card list inside a lane: fill leftover height, scroll cards. */
export const workOrderKanbanLaneScrollClassName =
  "flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto overscroll-contain [scrollbar-width:thin]";

interface WorkOrderKanbanBoardProps {
  children: ReactNode;
  testId: string;
}

/** One horizontal row of lanes. Extra columns grow right and scroll on x. */
export function WorkOrderKanbanBoard({ children, testId }: WorkOrderKanbanBoardProps) {
  return (
    <div
      className="flex h-full min-h-0 min-w-0 flex-1 gap-3 overflow-x-auto overflow-y-hidden [scrollbar-width:thin]"
      data-testid={testId}
    >
      {children}
    </div>
  );
}

interface WorkOrderBoardLaneProps {
  title: string;
  count: number;
  /** Replaces the body while the lane holds nothing. */
  emptyDescription: string;
  tone?: BoardLaneTone;
  /** Sits before the title, for example a phase status glyph. */
  leading?: ReactNode;
  /** Sits at the end of the header, for example a menu button. */
  actions?: ReactNode;
  /** Accessible name, when it must read differently from the title. */
  label?: string;
  testId?: string;
  children?: ReactNode;
}

export function WorkOrderBoardLane({
  title,
  count,
  emptyDescription,
  tone = "neutral",
  leading,
  actions,
  label,
  testId,
  children,
}: WorkOrderBoardLaneProps) {
  return (
    <section
      aria-label={label ?? title}
      className={cn(
        "flex h-full min-h-0 min-w-72 flex-1 flex-col rounded-lg border border-border/70 p-2",
        LANE_TONE_CLASSNAME[tone],
      )}
      data-testid={testId}
    >
      <header className="flex shrink-0 items-center justify-between gap-2 px-2 pb-2">
        <div className="inline-flex min-w-0 items-center gap-2">
          {leading}
          <h2 className="truncate text-[12px] font-semibold uppercase tracking-[0.06em] text-foreground/80">{title}</h2>
          <span className="shrink-0 rounded-full bg-background/70 px-1.5 py-0.5 text-[11px] font-medium text-muted-foreground">
            {count}
          </span>
        </div>
        {actions}
      </header>

      {count === 0 ? (
        <p className="mt-2 flex-1 rounded-md border border-dashed border-border/60 px-3 py-6 text-center text-[12px] text-muted-foreground">
          {emptyDescription}
        </p>
      ) : (
        children
      )}
    </section>
  );
}
