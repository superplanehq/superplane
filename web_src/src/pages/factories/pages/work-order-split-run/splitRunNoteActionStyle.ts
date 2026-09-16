import { cn } from "@/lib/utils";

import type { SplitRunFooterAction } from "./splitRunFooter";

export function noteActionDisabled(
  kind: SplitRunFooterAction["kind"],
  flags: { actionBusy: boolean; startBusy: boolean; startDisabled: boolean; actionDisabled?: boolean },
) {
  if (kind !== "start") {
    return flags.actionBusy;
  }
  return flags.startDisabled || flags.startBusy || Boolean(flags.actionDisabled);
}

export function noteActionClassName({
  capsule,
  grouped,
  primary,
}: {
  capsule: boolean;
  grouped: boolean;
  primary: boolean;
}) {
  if (capsule) {
    return cn("!rounded-none h-7 border-0 shadow-none", !primary && "bg-background");
  }
  if (grouped) {
    return "rounded-md rounded-r-none";
  }
  return undefined;
}
