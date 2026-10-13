import { CircleCheck, CircleMinus, CircleX, TriangleAlert } from "lucide-react";

import { cn } from "@/lib/utils";
import type { WorkOrderCheckLevel } from "../lib/workOrderChecks";

const CHECK_STATUS_ICON: Record<WorkOrderCheckLevel, { Icon: typeof CircleCheck; className: string }> = {
  positive: { Icon: CircleCheck, className: "fill-emerald-600 stroke-white" },
  neutral: { Icon: CircleMinus, className: "fill-slate-500 stroke-white" },
  caution: { Icon: TriangleAlert, className: "fill-amber-500 stroke-white" },
  critical: { Icon: CircleX, className: "fill-red-600 stroke-white" },
};

export function CheckStatusIcon({
  level,
  label,
  className,
}: {
  level: WorkOrderCheckLevel;
  label: string;
  className?: string;
}) {
  const { Icon, className: tone } = CHECK_STATUS_ICON[level];
  return (
    <Icon
      className={cn("size-4 shrink-0 dark:[&>:first-child]:stroke-transparent", tone, className)}
      role="img"
      aria-label={label}
    />
  );
}
