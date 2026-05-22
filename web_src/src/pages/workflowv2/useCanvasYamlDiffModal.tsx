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
  buildYamlExportPayload: (workflow?: CanvasesCanvas | null, overrideNodes?: CanvasNode[]) => YamlExportPayload | null;
};

export function useCanvasYamlDiffModal({
  hasUnpublishedDraftChanges,
  liveCanvas,
  liveCanvasVersion,
  draftCanvasVersion,
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
    const draftPayload = buildYamlExportPayload({
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
    buildYamlExportPayload,
    draftCanvasVersion?.spec,
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
