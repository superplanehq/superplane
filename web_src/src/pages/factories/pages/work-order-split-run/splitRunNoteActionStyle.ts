import type { SplitRunDecisionTone, SplitRunFooterAction } from "./splitRunFooter";

const TONE_STRIP: Record<SplitRunDecisionTone, string> = {
  draft: "border-[color:var(--status-draft-border)] bg-[color:var(--status-draft-bg)]",
  "draft-blocked": "border-[color:var(--status-failed-border)] bg-[color:var(--status-failed-bg)]",
  "draft-caution": "border-[color:var(--status-waiting-border)] bg-[color:var(--status-waiting-bg)]",
  "draft-ready": "border-[color:var(--status-completed-border)] bg-[color:var(--status-completed-bg)]",
  waiting: "border-[color:var(--status-waiting-border)] bg-[color:var(--status-waiting-bg)]",
  failed: "border-[color:var(--status-failed-border)] bg-[color:var(--status-failed-bg)]",
  done: "border-[color:var(--status-completed-border)] bg-[color:var(--status-completed-bg)]",
  rejected: "border-[color:var(--status-cancelled-border)] bg-[color:var(--status-cancelled-bg)]",
};

/**
 * Border and background classes for a decision tone. Attention notes tint
 * their strip with these, and the console summary panel tints its strip
 * section the same way so a stacked note can stay flat and still read in
 * the tone's color.
 */
export function attentionToneClassName(tone: SplitRunDecisionTone): string {
  return TONE_STRIP[tone];
}

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
