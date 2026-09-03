import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import {
  WORK_ORDER_ATTENTION_CHIP_CLASSNAME,
  WORK_ORDER_ATTENTION_ICON,
  WORK_ORDER_ATTENTION_LABEL,
  type WorkOrderAttentionReason,
} from "../lib/workOrderAttention";

const CHECKS_PASSED_TOOLTIP = "All checks on the pull request have passed.";

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

/**
 * Compact, icon-only mark for the "checksPassed" attention reason.
 *
 * getWorkOrderAttentionReasons only emits "checksPassed" alongside
 * "approval", so this mark sits next to the full Waiting for user review
 * chip. A second full-labeled chip would overflow the footer, so this
 * keeps the same color and icon but drops the visible label. The label
 * stays available as the accessible name (aria-label) so the meaning is
 * still announced to screen readers, and a Tooltip shows explanatory copy
 * on hover or keyboard focus.
 */
export function WorkOrderChecksPassedMark() {
  const Icon = WORK_ORDER_ATTENTION_ICON.checksPassed;
  const text = WORK_ORDER_ATTENTION_LABEL.checksPassed;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          className={cn(
            "inline-flex shrink-0 items-center rounded-md border px-1 py-0.5",
            WORK_ORDER_ATTENTION_CHIP_CLASSNAME.checksPassed,
          )}
          aria-label={text}
          tabIndex={0}
        >
          <Icon className="size-3 shrink-0" aria-hidden />
        </span>
      </TooltipTrigger>
      <TooltipContent side="top">{CHECKS_PASSED_TOOLTIP}</TooltipContent>
    </Tooltip>
  );
}
