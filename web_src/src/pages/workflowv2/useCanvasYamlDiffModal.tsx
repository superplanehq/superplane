import { lazy, Suspense, useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import type { CanvasNode } from "@/ui/CanvasPage";

const CanvasYamlDiffModal = lazy(() =>
  import("./CanvasYamlDiffModal").then((module) => ({ default: module.CanvasYamlDiffModal })),
);

type YamlExportPayload = {
  yamlText: string;
  filename: string;
};

type UseCanvasYamlDiffModalParams = {
  hasUnpublishedDraftChanges: boolean;
  liveCanvas?: CanvasesCanvas;
  liveCanvasVersion?: CanvasesCanvasVersion;
  draftCanvasVersion?: CanvasesCanvasVersion;
  draftCanvas?: CanvasesCanvas | null;
  draftNodes: CanvasNode[];
  activeCanvasVersionId: string;
  buildYamlExportPayload: (workflow?: CanvasesCanvas | null, overrideNodes?: CanvasNode[]) => YamlExportPayload | null;
};

export function useCanvasYamlDiffModal({
  hasUnpublishedDraftChanges,
  liveCanvas,
  liveCanvasVersion,
  draftCanvasVersion,
  draftCanvas,
  draftNodes,
  activeCanvasVersionId,
  buildYamlExportPayload,
}: UseCanvasYamlDiffModalParams): {
  onShowDiff: (() => void) | undefined;
  yamlDiffModal: ReactNode;
} {
  const [open, setOpen] = useState(false);
  const payload = useMemo(() => {
    if (!hasUnpublishedDraftChanges || !liveCanvas || !liveCanvasVersion?.spec || !draftCanvasVersion?.spec) {
      return null;
    }

    const livePayload = buildYamlExportPayload({
      ...liveCanvas,
      spec: liveCanvasVersion.spec,
    });
    const useCurrentDraftCanvas =
      !!draftCanvas && !!activeCanvasVersionId && draftCanvasVersion.metadata?.id === activeCanvasVersionId;
    const draftPayload = useCurrentDraftCanvas
      ? buildYamlExportPayload(draftCanvas, draftNodes)
      : buildYamlExportPayload({
          ...liveCanvas,
          spec: draftCanvasVersion.spec,
        });

    if (!livePayload || !draftPayload || livePayload.yamlText === draftPayload.yamlText) {
      return null;
    }

    return {
      liveYamlText: livePayload.yamlText,
      draftYamlText: draftPayload.yamlText,
      filename: draftPayload.filename || livePayload.filename,
    };
  }, [
    activeCanvasVersionId,
    buildYamlExportPayload,
    draftCanvas,
    draftCanvasVersion?.metadata?.id,
    draftCanvasVersion?.spec,
    draftNodes,
    hasUnpublishedDraftChanges,
    liveCanvas,
    liveCanvasVersion?.spec,
  ]);

  const onShowDiff = useCallback(() => setOpen(true), []);

  useEffect(() => {
    if (!payload && open) {
      setOpen(false);
    }
  }, [open, payload]);

  return {
    onShowDiff: payload ? onShowDiff : undefined,
    yamlDiffModal: payload ? (
      <Suspense fallback={null}>
        <CanvasYamlDiffModal
          open={open}
          onOpenChange={setOpen}
          liveYamlText={payload.liveYamlText}
          draftYamlText={payload.draftYamlText}
          filename={payload.filename}
        />
      </Suspense>
    ) : null,
  };
}
