import { cn } from "@/lib/utils";
import { CircleDashed, Loader2 } from "lucide-react";
import type { WorkOrderListEntry } from "../lib/workOrderListModel";

const BASE_CLASSNAME = "inline-flex items-center gap-1 text-[11px] text-muted-foreground";

interface WorkOrderLineStepProps {
  entry: WorkOrderListEntry;
  className?: string;
  /**
   * Shown when the order has not run yet. Layouts with a fixed column pass
   * an em dash to keep the grid aligned; the rest render nothing.
   */
  fallback?: string;
}

/**
 * Line and step caption under the work order title. Board, List, and Table
 * share it so the three layouts always agree on the wording and on when the
 * running indicator spins.
 */
export function WorkOrderLineStep({ entry, className, fallback }: WorkOrderLineStepProps) {
  if (!entry.lineStepLabel) {
    return fallback ? <p className={cn(BASE_CLASSNAME, className)}>{fallback}</p> : null;
  }

  return (
    <p className={cn(BASE_CLASSNAME, className)}>
      {entry.displayStatus === "running" ? (
        <Loader2 className="size-3 shrink-0 animate-spin" aria-hidden />
      ) : (
        <CircleDashed className="size-3 shrink-0" aria-hidden />
      )}
      <span className="truncate">{entry.lineStepLabel}</span>
    </p>
  );
}
