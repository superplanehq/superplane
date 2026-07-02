/**
 * Pure helpers for deciding when the canvas should re-run its first-load
 * fit-to-view. Switching canvas or version must re-fit the whole graph instead
 * of restoring the previous content's persisted viewport.
 */

/** Builds the key identifying the displayed canvas/version, or undefined in run inspection. */
export function computeFitViewContentKey(params: {
  isRunInspectionMode: boolean;
  canvasId?: string;
  canvasViewKey: string;
}): string | undefined {
  if (params.isRunInspectionMode) {
    return undefined;
  }
  return `${params.canvasId ?? ""}:${params.canvasViewKey}`;
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
