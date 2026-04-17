import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";

function getWorkflowSpecSignature(
  spec: CanvasesCanvas["spec"] | CanvasesCanvasVersion["spec"] | null | undefined,
): string {
  return JSON.stringify(spec ?? null);
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
  return !!selectedDraftVersionSignature && selectedDraftVersionSignature !== incomingSignature;
}
