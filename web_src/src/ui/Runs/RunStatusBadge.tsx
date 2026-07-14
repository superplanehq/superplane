import { cn } from "@/lib/utils";
import { RUN_STATUS_META, type RunStatusKey } from "./runPresentation";

export const RUN_STATUS_BADGE_BASE_CLASSES =
  "inline-flex shrink-0 items-center gap-1 rounded py-0.5 pl-1 pr-1.5 text-[12px] font-medium leading-4";

export function runStatusBadgeClassName(status: RunStatusKey): string {
  return cn(RUN_STATUS_BADGE_BASE_CLASSES, RUN_STATUS_META[status].badgeClassName);
}

export function RunStatusBadge({ status }: { status: RunStatusKey }) {
  const statusMeta = RUN_STATUS_META[status];
  const StatusIcon = statusMeta.icon;

  return (
    <span aria-label={statusMeta.label} title={statusMeta.label} className={runStatusBadgeClassName(status)}>
      <StatusIcon className="size-3.5" aria-hidden />
      <span>{statusMeta.label}</span>
    </span>
  );
}
