import { MarkdownContent } from "@/pages/app/Markdown";
import { cn } from "@/lib/utils";

import {
  WORK_ORDER_ATTENTION_CHIP_CLASSNAME,
  WORK_ORDER_ATTENTION_ICON,
  WORK_ORDER_ATTENTION_LABEL,
  type WorkOrderAttentionReason,
} from "../lib/workOrderAttention";

const CHIP_MARKDOWN_LINK =
  "pointer-events-auto relative z-10 font-semibold text-current !underline !decoration-current underline-offset-2";

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
        "inline-flex max-w-full shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-[10px] font-medium",
        WORK_ORDER_ATTENTION_CHIP_CLASSNAME[reason],
        className,
      )}
      title={plainChipLabel(text)}
    >
      <Icon
        className={cn("size-3 shrink-0", (reason === "feedback" || reason === "checks") && "animate-spin")}
        aria-hidden
      />
      <MarkdownContent
        content={text}
        variant="workspace"
        openLinksInNewTab
        linkClassName={CHIP_MARKDOWN_LINK}
        className="min-w-0 truncate text-[10px] leading-none text-current [&_p]:m-0 [&_p]:inline"
      />
    </span>
  );
}

function plainChipLabel(value: string): string {
  return value
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/\*\*([^*]+)\*\*/g, "$1")
    .replace(/`([^`]+)`/g, "$1")
    .trim();
}
