import { cn } from "@/lib/utils";
import { RUN_STATUS_META, type RunStatusKey } from "./runPresentation";

export const RUN_STATUS_BADGE_BASE_CLASSES =
  "inline-flex shrink-0 items-center gap-1 rounded py-0.5 pl-1 pr-1.5 text-[12px] font-medium leading-4";

export function runStatusBadgeClassName(status: RunStatusKey): string {
  return cn(RUN_STATUS_BADGE_BASE_CLASSES, RUN_STATUS_META[status].badgeClassName);
}
