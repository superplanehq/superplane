import { cn } from "@/lib/utils";
import { useEffect, useRef, type ReactNode } from "react";
import { ClickToRename } from "../layout/ClickToRename";
import { shouldRedirectWheelToHorizontalScroll } from "./kanbanBoardWheel";

/**
 * Lane chrome for every board in the workspace. The Tasks board and
 * the Lines phase board both render through it.
 */

/** Lane tint. Only in-flight and closed work get colour, as on the Tasks board. */
export type BoardLaneTone = "neutral" | "running" | "done";

const LANE_TONE_CLASSNAME: Record<BoardLaneTone, string> = {
  neutral: "bg-card/60",
  running: "bg-[color:var(--status-running-lane-bg)]",
  done: "bg-[color:var(--status-done-lane-bg)]",
};

/** Shared card list inside a lane: fill leftover height, scroll cards. */
export const workOrderKanbanLaneScrollClassName =
  "flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto [scrollbar-width:thin]";

/** Lane width: grow when there is room, never shrink below 18rem — extra lanes scroll on x. */
export const workOrderKanbanLaneSizeClassName = "min-w-72 basis-72 grow shrink-0";

interface WorkOrderKanbanBoardProps {
  children: ReactNode;
  testId: string;
}

/** One horizontal row of lanes. Extra columns grow right and scroll on x. */
export function WorkOrderKanbanBoard({ children, testId }: WorkOrderKanbanBoardProps) {
  const boardRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const board = boardRef.current;
    if (!board) {
      return;
    }
    const onWheel = (event: WheelEvent) => {
      if (!shouldRedirectWheelToHorizontalScroll(event, board)) {
        return;
      }
      board.scrollLeft += event.deltaY;
      event.preventDefault();
    };
    board.addEventListener("wheel", onWheel, { passive: false });
    return () => board.removeEventListener("wheel", onWheel);
  }, []);

  return (
    <div
      ref={boardRef}
      className="flex min-h-0 min-w-0 w-full flex-1 items-stretch gap-3 overflow-x-auto overflow-y-hidden [scrollbar-gutter:stable] [scrollbar-width:thin]"
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
  /** When set, replaces the dashed empty copy while the lane holds nothing. */
  emptyContent?: ReactNode;
  /** Keep the card list when count is 0, unless emptyContent is set. */
  keepChildrenWhenEmpty?: boolean;
  tone?: BoardLaneTone;
  /**
   * Optional pastel fill from the column color picker. When set, it replaces
   * the status tone background so the chosen color is visible.
   */
  surfaceClassName?: string;
  /** Sits at the end of the header, for example a menu button. */
  actions?: ReactNode;
  /** Pinned between the header and the card list, and kept while the lane is empty. */
  banner?: ReactNode;
  /** When set, a click on the title opens an inline rename field. */
  onRename?: (name: string) => void;
  canRename?: boolean;
  titleTestId?: string;
  /** Accessible name, when it must read differently from the title. */
  label?: string;
  className?: string;
  testId?: string;
  children?: ReactNode;
}

function LaneBody({
  count,
  emptyContent,
  emptyDescription,
  keepChildrenWhenEmpty,
  children,
}: {
  count: number;
  emptyContent?: ReactNode;
  emptyDescription: string;
  keepChildrenWhenEmpty: boolean;
  children?: ReactNode;
}) {
  if (count === 0 && emptyContent) {
    return <div className={cn(workOrderKanbanLaneScrollClassName, "justify-center")}>{emptyContent}</div>;
  }
  if (count === 0 && !keepChildrenWhenEmpty) {
    return (
      <p className="mt-2 flex-1 rounded-md border border-dashed border-border/60 px-3 py-6 text-center text-[12px] text-muted-foreground">
        {emptyDescription}
      </p>
    );
  }
  return children;
}

export function WorkOrderBoardLane({
  title,
  count,
  emptyDescription,
  emptyContent,
  keepChildrenWhenEmpty = false,
  tone = "neutral",
  surfaceClassName,
  actions,
  banner,
  onRename,
  canRename = false,
  titleTestId,
  label,
  className,
  testId,
  children,
}: WorkOrderBoardLaneProps) {
  return (
    <section
      aria-label={label ?? title}
      className={cn(
        "flex min-h-0 flex-col self-stretch rounded-lg border border-border/70 p-2",
        workOrderKanbanLaneSizeClassName,
        surfaceClassName ?? LANE_TONE_CLASSNAME[tone],
        className,
      )}
      data-testid={testId}
    >
      <header className="flex shrink-0 items-center justify-between gap-2 px-2 pb-2">
        <h2 className="workspace-section-title min-w-0 flex-1 overflow-visible">
          {onRename ? (
            <ClickToRename
              value={title}
              onSave={onRename}
              canEdit={canRename}
              testId={titleTestId ?? `${testId ?? "lane"}-title`}
              ariaLabel={`${title} column name`}
              inputClassName="text-[15px] font-semibold leading-[22.5px] tracking-[-0.01em]"
            />
          ) : (
            <span className="truncate">{title}</span>
          )}
        </h2>
        {actions}
      </header>

      {banner}

      <LaneBody
        count={count}
        emptyContent={emptyContent}
        emptyDescription={emptyDescription}
        keepChildrenWhenEmpty={keepChildrenWhenEmpty}
      >
        {children}
      </LaneBody>
    </section>
  );
}
