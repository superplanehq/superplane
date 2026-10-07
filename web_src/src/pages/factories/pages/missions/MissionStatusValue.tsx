import { cn } from "@/lib/utils";

import { MISSION_STATUS_META, type MissionDisplayStatus } from "./missionListModel";

type MissionStatusVariant = "card" | "row" | "plain";

export function MissionStatusValue({
  status,
  variant = "card",
}: {
  status: MissionDisplayStatus;
  variant?: MissionStatusVariant;
}) {
  const meta = MISSION_STATUS_META[status];

  if (variant === "plain") {
    return (
      <span className={cn("inline-flex items-center gap-1.5 text-[13px]", meta.textClass)}>
        <span className={cn("size-1.5 shrink-0 rounded-full", meta.dotClass)} aria-hidden />
        {meta.label}
      </span>
    );
  }

  return (
    <span
      className={cn(
        "inline-flex w-fit items-center gap-1 border px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-[0.04em]",
        variant === "row" ? "rounded-full" : "rounded-md",
        meta.className,
      )}
    >
      <span className={cn("size-1.5 rounded-full", meta.dotClass)} aria-hidden />
      {meta.label}
    </span>
  );
}
