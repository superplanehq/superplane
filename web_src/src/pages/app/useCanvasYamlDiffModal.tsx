import { lazy, Suspense, useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import type { CanvasNode } from "@/ui/CanvasPage";
import * as yaml from "js-yaml";

const CanvasYamlDiffModal = lazy(() =>
  import("./CanvasYamlDiffModal").then((module) => ({ default: module.CanvasYamlDiffModal })),
);

type YamlExportPayload = {
  yamlText: string;
  filename: string;
};

type NodeYamlDiffPayload = {
  liveYamlText: string;
  draftYamlText: string;
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

function findSpecNode(version: CanvasesCanvasVersion | undefined, nodeId: string) {
  const nodes = (version?.spec?.nodes || []) as Array<Record<string, unknown>>;
  return nodes.find((node) => String(node.id) === nodeId);
}

function applyRenderedNodeState(node: Record<string, unknown>, draftNodes: CanvasNode[]) {
  const renderedNode = draftNodes.find((draftNode) => draftNode.id === String(node.id));
  if (!renderedNode) {
    return node;
  }

  const componentType = (renderedNode.data?.type as string) || "";
  const renderedData = renderedNode.data[componentType] as { collapsed?: boolean } | undefined;

  return {
    ...node,
    position: {
      x: Math.round(renderedNode.position.x),
      y: Math.round(renderedNode.position.y),
    },
    isCollapsed: renderedData?.collapsed ?? node.isCollapsed ?? false,
  };
}

function dumpNodeYaml(node: Record<string, unknown>) {
  return yaml.dump(node, {
    forceQuotes: true,
    quotingType: '"',
    lineWidth: 0,
  });
}

function shouldUseCurrentDraftCanvas({
  activeCanvasVersionId,
  draftCanvas,
  draftCanvasVersion,
}: {
  activeCanvasVersionId: string;
  draftCanvas?: CanvasesCanvas | null;
  draftCanvasVersion?: CanvasesCanvasVersion;
}) {
  if (!draftCanvas || !activeCanvasVersionId) {
    return false;
  }

  return draftCanvasVersion?.metadata?.id === activeCanvasVersionId;
}

function getDraftSpecNodes({
  activeCanvasVersionId,
  draftCanvas,
  draftCanvasVersion,
}: {
  activeCanvasVersionId: string;
  draftCanvas?: CanvasesCanvas | null;
  draftCanvasVersion?: CanvasesCanvasVersion;
}) {
  if (shouldUseCurrentDraftCanvas({ activeCanvasVersionId, draftCanvas, draftCanvasVersion })) {
    return draftCanvas?.spec?.nodes || [];
  }

  return draftCanvasVersion?.spec?.nodes || [];
}

function buildNodeYamlDiffPayload({
  activeCanvasVersionId,
  draftCanvas,
  draftCanvasVersion,
  draftNodes,
  liveCanvasVersion,
  nodeId,
}: {
  activeCanvasVersionId: string;
  draftCanvas?: CanvasesCanvas | null;
  draftCanvasVersion?: CanvasesCanvasVersion;
  draftNodes: CanvasNode[];
  liveCanvasVersion?: CanvasesCanvasVersion;
  nodeId: string | null;
}): NodeYamlDiffPayload | null {
  if (!nodeId) {
    return null;
  }

  const liveNode = findSpecNode(liveCanvasVersion, nodeId);
  const draftSpecNodes = getDraftSpecNodes({ activeCanvasVersionId, draftCanvas, draftCanvasVersion });
  const draftNode = (draftSpecNodes as Array<Record<string, unknown>>).find((node) => String(node.id) === nodeId);

  if (!liveNode || !draftNode) {
    return null;
  }

  const liveYamlText = dumpNodeYaml(liveNode);
  const draftYamlText = dumpNodeYaml(applyRenderedNodeState(draftNode, draftNodes));
  if (liveYamlText === draftYamlText) {
    return null;
  }

  return {
    liveYamlText,
    draftYamlText,
    filename: `${String(draftNode.name || liveNode.name || nodeId)}.yaml`,
  };
}

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
  onShowNodeDiff: ((nodeId: string) => void) | undefined;
  yamlDiffModal: ReactNode;
} {
  const [open, setOpen] = useState(false);
  const [nodeDiffId, setNodeDiffId] = useState<string | null>(null);
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

  const nodePayload = useMemo(
    () =>
      buildNodeYamlDiffPayload({
        activeCanvasVersionId,
        draftCanvas,
        draftCanvasVersion,
        draftNodes,
        liveCanvasVersion,
        nodeId: nodeDiffId,
      }),
    [activeCanvasVersionId, draftCanvas, draftCanvasVersion, draftNodes, liveCanvasVersion, nodeDiffId],
  );

  const onShowDiff = useCallback(() => setOpen(true), []);
  const onShowNodeDiff = useCallback((nodeId: string) => setNodeDiffId(nodeId), []);

  useEffect(() => {
    if (!payload && open) {
      setOpen(false);
    }
  }, [open, payload]);

  useEffect(() => {
    if (!nodePayload && nodeDiffId) {
      setNodeDiffId(null);
    }
  }, [nodeDiffId, nodePayload]);

  return {
    onShowDiff: payload ? onShowDiff : undefined,
    onShowNodeDiff: hasUnpublishedDraftChanges ? onShowNodeDiff : undefined,
    yamlDiffModal: (
      <>
        {payload ? (
          <Suspense fallback={null}>
            <CanvasYamlDiffModal
              open={open}
              onOpenChange={setOpen}
              liveYamlText={payload.liveYamlText}
              draftYamlText={payload.draftYamlText}
              filename={payload.filename}
            />
          </Suspense>
        ) : null}
        {nodePayload ? (
          <Suspense fallback={null}>
            <CanvasYamlDiffModal
              open={!!nodeDiffId}
              onOpenChange={(nextOpen) => {
                if (!nextOpen) {
                  setNodeDiffId(null);
                }
              }}
              liveYamlText={nodePayload.liveYamlText}
              draftYamlText={nodePayload.draftYamlText}
              filename={nodePayload.filename}
            />
          </Suspense>
        ) : null}
      </>
    ),
  };
}
