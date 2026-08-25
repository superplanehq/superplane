import type { FactoriesWorkOrderStatusNote } from "@/api-client";
import type { WorkOrderDisplayStatus } from "./workOrderProgress";

/**
 * A "what happens next" panel set by an automation while the order is
 * open — e.g. a PR watcher announcing that merging the tracked pull
 * request completes the order. Notes are latest-only per key; a
 * different key sits beside it. Cleared on any state transition.
 * A note with showOnlyWhenWaiting stays hidden while a line is running.
 */
export interface WorkOrderStatusNotePresentation {
  key: string;
  /** Action-first heading, e.g. "Review the pull request". */
  headline: string;
  /** Markdown body explaining what resolves the wait. */
  text: string;
  /** Primary call to action, usually the thing to review. */
  cta?: { label: string; href?: string };
  /** Automation that owns the wait, linked so users can inspect it. */
  source?: { name: string; appId?: string };
  updatedAt?: string;
}

function emptyToUndefined(value: string | undefined): string | undefined {
  return value || undefined;
}

export function presentWorkOrderStatusNote(
  note: FactoriesWorkOrderStatusNote | undefined,
): WorkOrderStatusNotePresentation | undefined {
  if (!note?.key || !note.headline) {
    return undefined;
  }

  const sourceName = emptyToUndefined(note.automation?.appName) ?? emptyToUndefined(note.automation?.nodeName);
  return {
    key: note.key,
    headline: note.headline,
    text: note.body ?? "",
    cta: note.ctaLabel && note.ctaUrl ? { label: note.ctaLabel, href: note.ctaUrl } : undefined,
    source: sourceName ? { name: sourceName, appId: emptyToUndefined(note.automation?.appId) } : undefined,
    updatedAt: note.updatedAt,
  };
}

export function presentWorkOrderStatusNotes(
  notes: FactoriesWorkOrderStatusNote[] | undefined,
  displayStatus?: WorkOrderDisplayStatus,
): WorkOrderStatusNotePresentation[] {
  return (notes ?? []).flatMap((note) => {
    if (note.showOnlyWhenWaiting && displayStatus !== undefined && displayStatus !== "waiting") {
      return [];
    }
    const presented = presentWorkOrderStatusNote(note);
    return presented ? [presented] : [];
  });
}
