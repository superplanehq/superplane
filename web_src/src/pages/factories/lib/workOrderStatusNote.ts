import type { FactoriesWorkOrderStatusNote } from "@/api-client";

/**
 * A "what happens next" panel set by an automation while the order is
 * Waiting — e.g. a PR watcher announcing that merging the tracked pull
 * request completes the order. One note per order, latest wins; cleared
 * on any state transition so it always describes the current wait.
 */
export interface WorkOrderStatusNotePresentation {
  /** Action-first heading, e.g. "Review the pull request". */
  headline: string;
  /** Markdown body explaining what resolves the wait. */
  text: string;
  /** Primary call to action, usually the thing to review. */
  cta?: { label: string; href: string };
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
  if (!note?.headline) {
    return undefined;
  }

  const sourceName = emptyToUndefined(note.automation?.appName) ?? emptyToUndefined(note.automation?.nodeName);
  return {
    headline: note.headline,
    text: note.body ?? "",
    cta: note.ctaLabel && note.ctaUrl ? { label: note.ctaLabel, href: note.ctaUrl } : undefined,
    source: sourceName ? { name: sourceName, appId: emptyToUndefined(note.automation?.appId) } : undefined,
    updatedAt: note.updatedAt,
  };
}
