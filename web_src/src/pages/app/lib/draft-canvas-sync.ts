import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";

export function getWorkflowSpecSignature(
  spec: CanvasesCanvas["spec"] | CanvasesCanvasVersion["spec"] | null | undefined,
): string | null {
  if (spec == null) {
    return null;
  }

  return JSON.stringify(spec);
}

export function shouldPreserveDraftSpec({
  incomingSpec,
  draftSpec,
  selectedDraftVersionSpec,
  liveVersionSpec,
}: {
  incomingSpec: CanvasesCanvas["spec"] | null | undefined;
  draftSpec: CanvasesCanvas["spec"] | null | undefined;
  selectedDraftVersionSpec: CanvasesCanvasVersion["spec"] | null | undefined;
  liveVersionSpec: CanvasesCanvasVersion["spec"] | null | undefined;
}): boolean {
  if (!incomingSpec || !liveVersionSpec) {
    return false;
  }

  const incomingSignature = getWorkflowSpecSignature(incomingSpec);
  const liveVersionSignature = getWorkflowSpecSignature(liveVersionSpec);

  if (incomingSignature !== liveVersionSignature) {
    return false;
  }

  const currentDraftSignature = getWorkflowSpecSignature(draftSpec);
  if (currentDraftSignature && currentDraftSignature !== incomingSignature) {
    return true;
  }

  const selectedDraftVersionSignature = getWorkflowSpecSignature(selectedDraftVersionSpec);
  return selectedDraftVersionSignature != null && selectedDraftVersionSignature !== incomingSignature;
}

// The edit-session effect keeps per-version draft specs in a ref so branch
// switches preserve unsaved work. When entering edit on a version with remote
// staging, the ref can be seeded from committed content before the staged fetch
// completes; once staged content arrives, prefer it over the stale ref entry.
export function shouldApplyPreservedDraftSpec(
  preservedDraftSpec: CanvasesCanvas["spec"] | null | undefined,
  selectedDraftSpec: CanvasesCanvasVersion["spec"] | null | undefined,
): preservedDraftSpec is CanvasesCanvas["spec"] {
  if (!preservedDraftSpec) {
    return false;
  }

  if (!selectedDraftSpec) {
    return true;
  }

  return getWorkflowSpecSignature(preservedDraftSpec) === getWorkflowSpecSignature(selectedDraftSpec);
}

// When local draft state is ahead of a stale loaded version query, keep the
// local draft instead of overwriting it with older server/cache data.
export function shouldSkipDraftSpecSyncFromLoadedVersion(
  currentDraftSpec: CanvasesCanvas["spec"] | null | undefined,
  nextDraftSpec: CanvasesCanvasVersion["spec"] | null | undefined,
): boolean {
  if (!currentDraftSpec || !nextDraftSpec) {
    return false;
  }

  return getWorkflowSpecSignature(currentDraftSpec) !== getWorkflowSpecSignature(nextDraftSpec);
}
