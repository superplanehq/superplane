import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";

type DraftSpec = CanvasesCanvas["spec"] | null;

type DraftSpecCache = Map<string, DraftSpec>;

export function activateDraftVersion(
  draftCanvasSpecs: DraftSpecCache,
  setActiveCanvasVersion: (version: CanvasesCanvasVersion | null) => void,
  setDraftCanvasSpec: (spec: DraftSpec) => void,
  version: CanvasesCanvasVersion,
) {
  const spec = version.spec ?? null;
  const versionID = version.metadata?.id;
  setActiveCanvasVersion(version);
  setDraftCanvasSpec(spec);
  if (versionID) {
    draftCanvasSpecs.set(versionID, spec);
  }
}

export function clearPublishedDraftVersion(
  draftCanvasSpecs: DraftSpecCache,
  setActiveCanvasVersion: (version: CanvasesCanvasVersion | null) => void,
  setDraftCanvasSpec: (spec: DraftSpec) => void,
  versionID: string,
) {
  setActiveCanvasVersion(null);
  setDraftCanvasSpec(null);
  draftCanvasSpecs.delete(versionID);
}
