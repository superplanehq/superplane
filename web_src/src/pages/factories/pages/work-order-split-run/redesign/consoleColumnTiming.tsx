import { Badge } from "@/components/reui/badge";
import { formatCompactDuration } from "@/lib/duration";
import { CalendarArrowDown, Timer } from "lucide-react";

import { formatWorkOrderDateTime } from "../../../lib/workOrderDateTime";
import type { ColumnTiming, ColumnTimingId } from "./columnTiming";

export function ColumnTimingMeta({ columnId, timing }: { columnId: ColumnTimingId; timing?: ColumnTiming }) {
  if (!timing) {
    return null;
  }
  const arrived = formatWorkOrderDateTime(new Date(timing.enteredAt));
  const spent = timing.durationMs != null ? formatCompactDuration(timing.durationMs) : "";
  return (
    <div
      className="pointer-events-none ml-auto flex shrink-0 items-center gap-1.5 opacity-0 transition-opacity duration-150 group-hover/column:opacity-100"
      data-testid={`redesign-console-column-timing-${columnId}`}
    >
      <Badge variant="secondary" size="sm" radius="full" className="gap-1 font-normal tabular-nums">
        <CalendarArrowDown className="size-3" aria-hidden />
        <span className="sr-only">Arrived </span>
        {arrived}
      </Badge>
      {spent ? (
        <Badge variant="secondary" size="sm" radius="full" className="gap-1 font-normal tabular-nums">
          <Timer className="size-3" aria-hidden />
          <span className="sr-only">Spent </span>
          {spent}
        </Badge>
      ) : null}
    </div>
  );
}
