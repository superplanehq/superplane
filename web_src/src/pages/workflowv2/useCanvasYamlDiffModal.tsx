import { lazy, Suspense, useCallback, useMemo, useState, type ReactNode } from "react";
import type { CanvasesCanvas } from "@/api-client";
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
  liveCanvasVersionSpec?: CanvasesCanvas["spec"];
  draftCanvas?: CanvasesCanvas | null;
  nodes: CanvasNode[];
  buildYamlExportPayload: (workflow?: CanvasesCanvas | null, overrideNodes?: CanvasNode[]) => YamlExportPayload | null;
};

export function useCanvasYamlDiffModal({
  hasUnpublishedDraftChanges,
  liveCanvas,
  liveCanvasVersionSpec,
  draftCanvas,
  nodes,
  buildYamlExportPayload,
}: UseCanvasYamlDiffModalParams): {
  onShowDiff: (() => void) | undefined;
  yamlDiffModal: ReactNode;
} {
  const [open, setOpen] = useState(false);
  const payload = useMemo(() => {
    if (!hasUnpublishedDraftChanges || !liveCanvas) {
      return null;
    }

    const livePayload = buildYamlExportPayload({
      ...liveCanvas,
      spec: liveCanvasVersionSpec || liveCanvas.spec,
    });
    const draftPayload = buildYamlExportPayload(draftCanvas, nodes);

    if (!livePayload || !draftPayload || livePayload.yamlText === draftPayload.yamlText) {
      return null;
    }

    return {
      liveYamlText: livePayload.yamlText,
      draftYamlText: draftPayload.yamlText,
      filename: draftPayload.filename || livePayload.filename,
    };
  }, [buildYamlExportPayload, draftCanvas, hasUnpublishedDraftChanges, liveCanvas, liveCanvasVersionSpec, nodes]);

  const onShowDiff = useCallback(() => setOpen(true), []);

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
