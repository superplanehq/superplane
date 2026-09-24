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

/** A Start fused to the model chevron loses its right radius. */
export function noteActionClassName({ grouped }: { grouped: boolean }) {
  return grouped ? "rounded-md rounded-r-none" : undefined;
}
