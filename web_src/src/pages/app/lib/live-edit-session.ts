import type { MutableRefObject } from "react";

import type { CanvasesCanvas } from "@/api-client";

import { clearComponentSidebarSearchParams } from "../viewState";

export function shouldReadStagedCanvasVersion({
  editSessionActive,
  activeCanvasVersionId,
  effectiveLiveCanvasVersionId,
  liveCanvasVersionId,
}: {
  editSessionActive: boolean;
  activeCanvasVersionId: string;
  effectiveLiveCanvasVersionId?: string;
  liveCanvasVersionId?: string;
}): boolean {
  return (
    editSessionActive &&
    !!activeCanvasVersionId &&
    ((!!effectiveLiveCanvasVersionId && activeCanvasVersionId === effectiveLiveCanvasVersionId) ||
      activeCanvasVersionId === liveCanvasVersionId)
  );
}

type LiveEditSessionDraftRefs = {
  draftCanvasSpecsRef: MutableRefObject<Map<string, CanvasesCanvas["spec"]>>;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  previewingCurrentVersionRef: MutableRefObject<boolean>;
};

export function clearLiveEditSessionDraftState({
  setEditSessionActive,
  setActiveCanvasVersion,
  setDraftCanvasSpec,
  draftCanvasSpecsRef,
  activeCanvasVersionIdRef,
  previewingCurrentVersionRef,
}: LiveEditSessionDraftRefs & {
  setEditSessionActive: (value: boolean) => void;
  setActiveCanvasVersion: (value: null) => void;
  setDraftCanvasSpec: (value: null) => void;
}): void {
  setEditSessionActive(false);
  setActiveCanvasVersion(null);
  setDraftCanvasSpec(null);
  draftCanvasSpecsRef.current.clear();
  activeCanvasVersionIdRef.current = "";
  previewingCurrentVersionRef.current = false;
}

export function clearLiveEditSessionSearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("version");
  next.delete("branch");
  return clearComponentSidebarSearchParams(next);
}
