import { cn } from "@/lib/utils";

import {
  WORK_ORDER_ATTENTION_CHIP_CLASSNAME,
  WORK_ORDER_ATTENTION_ICON,
  WORK_ORDER_ATTENTION_LABEL,
  type WorkOrderAttentionReason,
} from "../lib/workOrderAttention";

export function WorkOrderAttentionChip({
  reason,
  className,
}: {
  reason: WorkOrderAttentionReason;
  className?: string;
}) {
  const Icon = WORK_ORDER_ATTENTION_ICON[reason];
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center gap-1 rounded-md border px-1.5 py-0.5 text-[10px] font-medium",
        WORK_ORDER_ATTENTION_CHIP_CLASSNAME[reason],
        className,
      )}
    >
      <Icon className={cn("size-3", reason === "feedback" && "animate-spin")} aria-hidden />
      {WORK_ORDER_ATTENTION_LABEL[reason]}
    </span>
  );
}
