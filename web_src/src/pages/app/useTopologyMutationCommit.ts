import type { ActionsAction, CanvasesCanvas, ComponentsEdge, SuperplaneComponentsNode } from "@/api-client";
import { resolveCanvasFlowDirection } from "@/lib/canvasFlowDirection";
import { DefaultLayoutEngine } from "@/lib/layout";
import { useCallback, useRef, type MutableRefObject } from "react";

type CommitOptions = { addedNodeId?: string };
type TopologyMutationResult = {
  workflow: CanvasesCanvas;
  options?: CommitOptions;
  changed?: boolean;
};
type EdgeIdentity = Pick<ComponentsEdge, "sourceId" | "targetId" | "channel">;

function resolveMutationResult(result: CanvasesCanvas | TopologyMutationResult): TopologyMutationResult {
  if ("workflow" in result) return result;
  return { workflow: result };
}

export function appendWorkflowFragment(
  workflow: CanvasesCanvas,
  nodes: SuperplaneComponentsNode[],
  edges: ComponentsEdge[] = [],
): CanvasesCanvas {
  return {
    ...workflow,
    spec: {
      ...workflow.spec,
      nodes: [...(workflow.spec?.nodes || []), ...nodes],
      edges: [...(workflow.spec?.edges || []), ...edges],
    },
  };
}

export function removeWorkflowNodes(workflow: CanvasesCanvas, removedNodeIds: Set<string>): CanvasesCanvas {
  const nodes = (workflow.spec?.nodes || []).filter((node) => !node.id || !removedNodeIds.has(node.id));
  const survivingIds = new Set(nodes.map((node) => node.id).filter(Boolean));
  return {
    ...workflow,
    spec: {
      ...workflow.spec,
      nodes,
      edges: (workflow.spec?.edges || []).filter(
        (edge) =>
          (!edge.sourceId || survivingIds.has(edge.sourceId)) && (!edge.targetId || survivingIds.has(edge.targetId)),
      ),
    },
  };
}

export function removeWorkflowEdges(workflow: CanvasesCanvas, removedEdges: EdgeIdentity[]): CanvasesCanvas {
  return {
    ...workflow,
    spec: {
      ...workflow.spec,
      edges: (workflow.spec?.edges || []).filter(
        (edge) =>
          !removedEdges.some(
            (removed) =>
              edge.sourceId === removed.sourceId &&
              edge.targetId === removed.targetId &&
              edge.channel === removed.channel,
          ),
      ),
    },
  };
}

export function useTopologyMutationCommit({
  factoryAutoLayout,
  autoLayoutOnUpdate,
  components,
  getCurrentWorkflow,
  applyLocalWorkflow,
  saveWorkflow,
  saveSessionRef,
  readOnly,
}: {
  factoryAutoLayout: boolean;
  autoLayoutOnUpdate: boolean;
  components: ActionsAction[];
  getCurrentWorkflow: () => CanvasesCanvas | null | undefined;
  applyLocalWorkflow: (workflow: CanvasesCanvas) => void;
  saveWorkflow: (workflow: CanvasesCanvas, options: { showToast: false }) => Promise<unknown>;
  saveSessionRef: MutableRefObject<number>;
  readOnly: boolean;
}) {
  const queueRef = useRef<Promise<void>>(Promise.resolve());

  const applyRequiredLayout = useCallback(
    async (workflow: CanvasesCanvas, options?: CommitOptions) => {
      if (factoryAutoLayout) {
        return DefaultLayoutEngine.apply(workflow, {
          scope: "full-canvas",
          components,
          direction: "vertical",
        });
      }
      if (!autoLayoutOnUpdate || !options?.addedNodeId) {
        return workflow;
      }
      const node = workflow.spec?.nodes?.find((candidate) => candidate.id === options.addedNodeId);
      if (!node || node.type === "TYPE_WIDGET") {
        return workflow;
      }
      return DefaultLayoutEngine.apply(workflow, {
        scope: "connected-component",
        nodeIds: [options.addedNodeId],
        components,
        direction: resolveCanvasFlowDirection(workflow.metadata?.factoryId),
      });
    },
    [autoLayoutOnUpdate, components, factoryAutoLayout],
  );

  return useCallback(
    (mutate: (workflow: CanvasesCanvas) => CanvasesCanvas | TopologyMutationResult, options?: CommitOptions) => {
      const requestedSaveSession = saveSessionRef.current;
      const run = async () => {
        if (saveSessionRef.current !== requestedSaveSession) return null;
        const workflow = getCurrentWorkflow();
        if (!workflow) return null;
        const mutation = resolveMutationResult(mutate(workflow));
        if (mutation.changed === false) return mutation.workflow;
        const finalWorkflow = await applyRequiredLayout(mutation.workflow, mutation.options ?? options);
        if (saveSessionRef.current !== requestedSaveSession) return null;

        applyLocalWorkflow(finalWorkflow);
        if (!readOnly) await saveWorkflow(finalWorkflow, { showToast: false });
        return finalWorkflow;
      };

      if (!factoryAutoLayout) return run();
      const operation = queueRef.current.then(run, run);
      queueRef.current = operation.then(
        () => undefined,
        () => undefined,
      );
      return operation;
    },
    [
      applyLocalWorkflow,
      applyRequiredLayout,
      factoryAutoLayout,
      getCurrentWorkflow,
      readOnly,
      saveSessionRef,
      saveWorkflow,
    ],
  );
}
