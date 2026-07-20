/**
 * Pure helpers for deciding when the canvas should re-run its first-load
 * fit-to-view. Switching canvas or version must re-fit the whole graph instead
 * of restoring the previous content's persisted viewport.
 */

/**
 * Resolves the version id of the graph actually rendered right now.
 *
 * It reflects the *rendered* content, not just the selected version: while a
 * previewed version's spec is still loading, the graph on screen is still the
 * previous (live) content, so this stays on the live id. That way the fit waits
 * for the real nodes instead of fitting (and stamping) the stale graph, which would
 * otherwise block the re-fit once the version's spec arrives without a remount.
 */
export function resolveFitViewVersionId(params: {
  liveCanvasVersionId?: string;
  activeCanvasVersionId?: string;
  isViewingDraftVersion: boolean;
  draftSpec?: unknown;
  selectedVersion?: { spec?: unknown } | null;
}): string {
  const showingSelectedVersion = params.isViewingDraftVersion ? !!params.draftSpec : !!params.selectedVersion?.spec;
  if (params.activeCanvasVersionId && showingSelectedVersion) {
    return params.activeCanvasVersionId;
  }
  return params.liveCanvasVersionId || "live";
}

/** True on first init or whenever the displayed content changed since the last fit. */
export function shouldRefitOnInit(params: {
  hasFittedBefore: boolean;
  fitViewContentKey?: string;
  lastFittedContentKey: string | null;
}): boolean {
  if (!params.hasFittedBefore) {
    return true;
  }
  if (params.fitViewContentKey === undefined) {
    return false;
  }
  return params.lastFittedContentKey !== params.fitViewContentKey;
}

/** Records the content key that was just fitted (only once real nodes were present). */
export function stampFittedContentKey(
  ref: { current: string | null } | undefined,
  fitViewContentKey: string | undefined,
): void {
  if (!ref || fitViewContentKey === undefined) {
    return;
  }
  ref.current = fitViewContentKey;
}
