import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  canvasesCancelExecution,
  canvasesDeleteNodeQueueItem,
  canvasesListNodeQueueItems,
  canvasesReemitTriggerEvent,
  type CanvasesCanvasRun,
  type ComponentsEdge,
  type SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { useEventExecutions } from "@/hooks/useCanvasData";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { RunInspectorChrome } from "./RunInspectorChrome";
import { RunInspectorHeader } from "./RunInspectorHeader";
import { ResizeHandle, useResizableInspectorWidth } from "./RunInspectorResize";
import { RunInspectorStepsList } from "./RunInspectorStepsList";
import { buildNodeMap, buildRunPresentation } from "./runPresentation";
import { buildRunInspectorNodeSections, findRunInspectorErrorSummaries } from "./runNodeDetailModel";

export interface RunInspectorPanelProps {
  canvasId: string;
  run: CanvasesCanvasRun;
  workflowNodes: ComponentsNode[];
  workflowEdges?: ComponentsEdge[];
  componentIconMap?: Record<string, string>;
  selectedNodeId?: string | null;
  onSelectNode: (nodeId: string) => void;
  onClearSelectedNode?: () => void;
  onClose: () => void;
}

export function RunInspectorPanel({
  canvasId,
  run,
  workflowNodes,
  workflowEdges,
  componentIconMap = {},
  selectedNodeId = null,
  onSelectNode,
  onClearSelectedNode,
  onClose,
}: RunInspectorPanelProps) {
  const rootEventId = run.rootEvent?.id || null;
  const executionsQuery = useEventExecutions(canvasId, rootEventId);
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);
  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);
  const presentation = useMemo(() => buildRunPresentation(run, nodeMap), [nodeMap, run]);
  const sections = useMemo(
    () => buildRunInspectorNodeSections({ run, executions, workflowNodes, workflowEdges }),
    [executions, run, workflowEdges, workflowNodes],
  );
  const errorSummaries = useMemo(() => findRunInspectorErrorSummaries(sections), [sections]);
  const inspectorWidth = useResizableInspectorWidth();
  const queryClient = useQueryClient();
  const selectedValue = selectedNodeId ?? "";
  const [pendingErrorScrollNodeId, setPendingErrorScrollNodeId] = useState<string | null>(null);
  const runningExecutionIds = useMemo(
    () =>
      sections
        .map((section) => section.execution)
        .filter((execution) => execution?.id && execution.state === "STATE_STARTED")
        .map((execution) => execution!.id!),
    [sections],
  );
  const stoppableNodeIds = useMemo(() => [...new Set(sections.map((section) => section.nodeId))], [sections]);

  const refreshRunQueries = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: ["canvases"] });
  }, [queryClient]);

  const rerunMutation = useMutation({
    mutationFn: async () => {
      if (!run.rootEvent?.nodeId || !run.rootEvent?.id) {
        throw new Error("Run root event is missing");
      }

      await canvasesReemitTriggerEvent(
        withOrganizationHeader({
          path: {
            canvasId,
            nodeId: run.rootEvent.nodeId,
            eventId: run.rootEvent.id,
          },
        }),
      );
    },
    onSuccess: async () => {
      await refreshRunQueries();
      showSuccessToast("Run restarted");
    },
    onError: (error) => {
      console.error("Failed to restart run", error);
      showErrorToast("Failed to restart run");
    },
  });

  const stopMutation = useMutation({
    mutationFn: async () => {
      const queuedItems = await listQueuedItemsForRun({
        canvasId,
        nodeIds: stoppableNodeIds,
        rootEventId: run.rootEvent?.id,
      });

      if (runningExecutionIds.length === 0 && queuedItems.length === 0) {
        throw new Error("No running or queued steps to stop");
      }

      await Promise.all([
        ...runningExecutionIds.map((executionId) =>
          canvasesCancelExecution(
            withOrganizationHeader({
              path: {
                canvasId,
                executionId,
              },
            }),
          ),
        ),
        ...queuedItems.map((item) =>
          canvasesDeleteNodeQueueItem(
            withOrganizationHeader({
              path: {
                canvasId,
                nodeId: item.nodeId,
                itemId: item.itemId,
              },
            }),
          ),
        ),
      ]);
    },
    onSuccess: async () => {
      await refreshRunQueries();
      showSuccessToast("Run stopped");
    },
    onError: (error) => {
      console.error("Failed to stop run", error);
      showErrorToast("Failed to stop run");
    },
  });
  const stopActionDisabled =
    executionsQuery.isLoading || stopMutation.isPending || (runningExecutionIds.length === 0 && !run.rootEvent?.id);

  const handleValueChange = (value: string) => {
    if (value) {
      onSelectNode(value);
      return;
    }

    onClearSelectedNode?.();
  };

  const jumpToErrorOutput = (nodeId: string) => {
    setPendingErrorScrollNodeId(nodeId);
    onSelectNode(nodeId);
  };

  useEffect(() => {
    if (!pendingErrorScrollNodeId || selectedNodeId !== pendingErrorScrollNodeId) return;

    const frame = window.requestAnimationFrame(() => {
      const errorOutput = document.querySelector(`[data-run-error-output-node-id="${pendingErrorScrollNodeId}"]`);
      errorOutput?.scrollIntoView({ block: "center", behavior: "smooth" });
      setPendingErrorScrollNodeId(null);
    });

    return () => window.cancelAnimationFrame(frame);
  }, [pendingErrorScrollNodeId, selectedNodeId]);

  return (
    <aside
      className={cn(
        "relative z-20 flex h-full shrink-0 flex-col border-l bg-white shadow-sm dark:bg-gray-950",
        appDarkModeClasses.sidebarEdge,
      )}
      style={{ width: inspectorWidth.width }}
      data-testid="run-inspector-panel"
      aria-label="Run inspector"
    >
      <ResizeHandle onPointerDown={inspectorWidth.startResize} isResizing={inspectorWidth.isResizing} />
      <RunInspectorChrome onClose={onClose} />
      <RunInspectorHeader
        run={run}
        title={presentation.title}
        stepCount={sections.length || run.executions?.length || 0}
        onAction={() => (presentation.status === "running" ? stopMutation.mutate() : rerunMutation.mutate())}
        actionPending={presentation.status === "running" ? stopMutation.isPending : rerunMutation.isPending}
        actionDisabled={presentation.status === "running" ? stopActionDisabled : !run.rootEvent?.id}
      />

      <RunInspectorStepsList
        errorSummaries={errorSummaries}
        status={presentation.status}
        sections={sections}
        isLoading={executionsQuery.isLoading}
        selectedValue={selectedValue}
        componentIconMap={componentIconMap}
        onValueChange={handleValueChange}
        onJumpToError={jumpToErrorOutput}
        onRerun={() => rerunMutation.mutate()}
        rerunPending={rerunMutation.isPending}
      />
    </aside>
  );
}

async function listQueuedItemsForRun({
  canvasId,
  nodeIds,
  rootEventId,
}: {
  canvasId: string;
  nodeIds: string[];
  rootEventId?: string;
}) {
  if (!rootEventId || nodeIds.length === 0) {
    return [];
  }

  const responses = await Promise.all(
    nodeIds.map(async (nodeId) => {
      const response = await canvasesListNodeQueueItems(
        withOrganizationHeader({
          path: { canvasId, nodeId },
          query: { limit: 100 },
        }),
      );

      return (
        response.data?.items
          ?.filter((item) => item.id && item.rootEvent?.id === rootEventId)
          .map((item) => ({ nodeId, itemId: item.id! })) ?? []
      );
    }),
  );

  return responses.flat();
}
