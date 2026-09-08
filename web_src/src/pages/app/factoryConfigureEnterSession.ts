import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";

export type FactoryConfigureEnterDeps = {
  activateCanvasVersionForEditing: (
    versionId: string,
    version: CanvasesCanvasVersion,
    options?: { preserveStagedLayer?: boolean },
  ) => void;
  draftCanvasSpecsRef: MutableRefObject<Map<string, CanvasesCanvas["spec"] | null>>;
  previewingCurrentVersionRef: MutableRefObject<boolean>;
  resyncStagedEditorState: (
    versionId: string,
    options?: { bumpResetNonce?: boolean; preferCachedStagedSpec?: boolean },
  ) => Promise<void>;
  setDraftCanvasSpec: Dispatch<SetStateAction<CanvasesCanvas["spec"] | null>>;
  setEditSessionActive: Dispatch<SetStateAction<boolean>>;
  setLastSavedWorkflowSnapshot: (workflow: CanvasesCanvas | null) => void;
};

type StartFactoryConfigureEnterOptions = {
  visitId: number;
  configureVersionId: string;
  version: CanvasesCanvasVersion;
  liveCanvasRef: MutableRefObject<CanvasesCanvas | null | undefined>;
  configureVisitIdRef: MutableRefObject<number>;
  inFlightVisitIdRef: MutableRefObject<number | null>;
  editEnabledVisitIdRef: MutableRefObject<number | null>;
  deps: FactoryConfigureEnterDeps;
};

/** Give the staged-spec fetch a short window, then open Configure on the seed. */
export const FACTORY_CONFIGURE_ENTER_RESYNC_TIMEOUT_MS = 2000;

async function awaitStagedResync(
  resync: FactoryConfigureEnterDeps["resyncStagedEditorState"],
  versionId: string,
): Promise<void> {
  let timeoutId: ReturnType<typeof setTimeout> | undefined;
  try {
    await Promise.race([
      resync(versionId, { bumpResetNonce: false }),
      new Promise<void>((resolve) => {
        timeoutId = setTimeout(resolve, FACTORY_CONFIGURE_ENTER_RESYNC_TIMEOUT_MS);
      }),
    ]);
  } finally {
    if (timeoutId !== undefined) {
      clearTimeout(timeoutId);
    }
  }
}

/** Seed draft + await staged resync for one Configure visit. Returns cancel cleanup. */
export function startFactoryConfigureEnter({
  visitId,
  configureVersionId,
  version,
  liveCanvasRef,
  configureVisitIdRef,
  inFlightVisitIdRef,
  editEnabledVisitIdRef,
  deps,
}: StartFactoryConfigureEnterOptions) {
  // Seed draft from the live version directly — do not wait on the versions
  // list (handleUseVersion), which often left Configure with no activeVersionId.
  const immediateSpec = liveCanvasRef.current?.spec ?? version.spec ?? { nodes: [], edges: [] };
  const versionForEdit: CanvasesCanvasVersion = {
    ...version,
    metadata: {
      ...version.metadata,
      id: configureVersionId,
    },
    spec: immediateSpec,
  };

  let cancelled = false;
  const {
    activateCanvasVersionForEditing: activate,
    draftCanvasSpecsRef: specsRef,
    previewingCurrentVersionRef: previewingRef,
    resyncStagedEditorState: resync,
    setDraftCanvasSpec: setDraft,
    setEditSessionActive: setEditActive,
    setLastSavedWorkflowSnapshot: setSnapshot,
  } = deps;

  previewingRef.current = true;
  activate(configureVersionId, versionForEdit, { preserveStagedLayer: true });
  specsRef.current.set(configureVersionId, immediateSpec);
  setDraft(immediateSpec);

  // Await staged resync before enabling edit so a late applyStagedSpec cannot
  // wipe edits typed against the immediate seed. A hung repository-file fetch
  // must not leave "Loading canvas..." forever: time-box it and keep the seed.
  void (async () => {
    try {
      await awaitStagedResync(resync, configureVersionId);
    } catch {
      // Keep the immediate live/committed spec so Configure stays usable.
    }
    if (cancelled || configureVisitIdRef.current !== visitId) {
      return;
    }
    editEnabledVisitIdRef.current = visitId;
    inFlightVisitIdRef.current = null;
    setEditActive(true);
    const canvas = liveCanvasRef.current;
    if (canvas) {
      const spec = specsRef.current.get(configureVersionId) ?? immediateSpec;
      setSnapshot({ ...canvas, spec });
    }
  })();

  return () => {
    cancelled = true;
    // Strict Mode remount: allow retry only when edit never enabled for visit.
    if (editEnabledVisitIdRef.current !== visitId && inFlightVisitIdRef.current === visitId) {
      inFlightVisitIdRef.current = null;
    }
  };
}
