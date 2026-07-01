import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";

export function canvasSpecSignature(
  spec: CanvasesCanvas["spec"] | CanvasesCanvasVersion["spec"] | null | undefined,
): string | null {
  if (spec == null) {
    return null;
  }

  return JSON.stringify(spec);
}

export function canvasSpecsEqual(
  left: CanvasesCanvas["spec"] | CanvasesCanvasVersion["spec"] | null | undefined,
  right: CanvasesCanvas["spec"] | CanvasesCanvasVersion["spec"] | null | undefined,
): boolean {
  return canvasSpecSignature(left) === canvasSpecSignature(right);
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

  const incomingSignature = canvasSpecSignature(incomingSpec);
  const liveVersionSignature = canvasSpecSignature(liveVersionSpec);

  if (incomingSignature !== liveVersionSignature) {
    return false;
  }

  const currentDraftSignature = canvasSpecSignature(draftSpec);
  if (currentDraftSignature && currentDraftSignature !== incomingSignature) {
    return true;
  }

  const selectedDraftVersionSignature = canvasSpecSignature(selectedDraftVersionSpec);
  return selectedDraftVersionSignature != null && selectedDraftVersionSignature !== incomingSignature;
}
