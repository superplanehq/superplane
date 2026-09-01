import { cn } from "@/lib/utils";

import {
  WORK_ORDER_ATTENTION_CHIP_CLASSNAME,
  WORK_ORDER_ATTENTION_ICON,
  WORK_ORDER_ATTENTION_LABEL,
  type WorkOrderAttentionReason,
} from "../lib/workOrderAttention";

export function WorkOrderAttentionChip({
  reason,
  label,
  className,
}: {
  reason: WorkOrderAttentionReason;
  label?: string;
  className?: string;
}) {
  const Icon = WORK_ORDER_ATTENTION_ICON[reason];
  const text = label?.trim() || WORK_ORDER_ATTENTION_LABEL[reason];
  return (
    <span
      className={cn(
        "inline-flex max-w-full shrink-0 items-center gap-1 rounded-md border px-1.5 py-0.5 text-[10px] font-medium",
        WORK_ORDER_ATTENTION_CHIP_CLASSNAME[reason],
        className,
      )}
      title={text}
    >
      <Icon
        className={cn("size-3 shrink-0", (reason === "feedback" || reason === "checks") && "animate-spin")}
        aria-hidden
      />
      <span className="truncate">{text}</span>
    </span>
  );
}
