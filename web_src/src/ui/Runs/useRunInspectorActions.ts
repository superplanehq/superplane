import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import {
  canvasesCancelExecution,
  canvasesDeleteNodeQueueItem,
  canvasesInvokeNodeExecutionHook,
  canvasesListNodeQueueItems,
  canvasesReemitTriggerEvent,
  type CanvasesCanvasRun,
} from "@/api-client";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import type { RunInspectorNodeSection } from "./runNodeDetailModel";

export function useRunInspectorActions({
  canvasId,
  run,
  sections,
  executionsLoading,
}: {
  canvasId: string;
  run: CanvasesCanvasRun;
  sections: RunInspectorNodeSection[];
  executionsLoading: boolean;
}) {
  const queryClient = useQueryClient();
  const runningExecutionIds = useMemo(
    () =>
      sections
        .map((section) => section.execution)
        .filter((execution) => execution?.id && execution.state === "STATE_STARTED")
        .map((execution) => execution!.id!),
    [sections],
  );
  const hasActionSection = useMemo(() => sections.some((section) => !section.isTrigger), [sections]);
  const stoppableNodeIds = useMemo(() => [...new Set(sections.map((section) => section.nodeId))], [sections]);

  const refreshRunQueries = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: ["canvases"] });
  }, [queryClient]);

  const rerunMutation = useRerunMutation({ canvasId, run, refreshRunQueries });
  const stopMutation = useStopMutation({
    canvasId,
    runningExecutionIds,
    stoppableNodeIds,
    rootEventId: run.rootEvent?.id,
    refreshRunQueries,
  });
  const stopNodeMutation = useStopNodeMutation({ canvasId, refreshRunQueries });
  const executionHookMutation = useExecutionHookMutation({ canvasId, refreshRunQueries });

  return {
    rerun: () => rerunMutation.mutate(),
    rerunPending: rerunMutation.isPending,
    stop: () => stopMutation.mutate(),
    stopPending: stopMutation.isPending,
    stopDisabled:
      executionsLoading ||
      stopMutation.isPending ||
      (runningExecutionIds.length === 0 && (!run.rootEvent?.id || !hasActionSection)),
    stopNode: (section: RunInspectorNodeSection) => {
      if (!section.execution?.id) return;
      stopNodeMutation.mutate(section.execution.id);
    },
    stopNodePending: stopNodeMutation.isPending,
    invokeNodeHook: (
      section: RunInspectorNodeSection,
      hookName: string,
      parameters?: Record<string, unknown> | null,
    ) => {
      if (!section.execution?.id) return;
      executionHookMutation.mutate({ executionId: section.execution.id, hookName, parameters });
    },
    nodeHookPending: executionHookMutation.isPending,
  };
}

function useRerunMutation({
  canvasId,
  run,
  refreshRunQueries,
}: {
  canvasId: string;
  run: CanvasesCanvasRun;
  refreshRunQueries: () => Promise<void>;
}) {
  return useMutation({
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
}

function useStopMutation({
  canvasId,
  runningExecutionIds,
  stoppableNodeIds,
  rootEventId,
  refreshRunQueries,
}: {
  canvasId: string;
  runningExecutionIds: string[];
  stoppableNodeIds: string[];
  rootEventId?: string;
  refreshRunQueries: () => Promise<void>;
}) {
  return useMutation({
    mutationFn: async () => {
      const queuedItems = await listQueuedItemsForRun({
        canvasId,
        nodeIds: stoppableNodeIds,
        rootEventId,
      });

      if (runningExecutionIds.length === 0 && queuedItems.length === 0) {
        throw new Error("No running or queued steps to stop");
      }

      await Promise.all([
        ...runningExecutionIds.map((executionId) => cancelExecution(canvasId, executionId)),
        ...queuedItems.map((item) => deleteQueuedItem(canvasId, item.nodeId, item.itemId)),
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
}

function useStopNodeMutation({
  canvasId,
  refreshRunQueries,
}: {
  canvasId: string;
  refreshRunQueries: () => Promise<void>;
}) {
  return useMutation({
    mutationFn: (executionId: string) => cancelExecution(canvasId, executionId),
    onSuccess: async () => {
      await refreshRunQueries();
      showSuccessToast("Step stopped");
    },
    onError: (error) => {
      console.error("Failed to stop step", error);
      showErrorToast("Failed to stop step");
    },
  });
}

function useExecutionHookMutation({
  canvasId,
  refreshRunQueries,
}: {
  canvasId: string;
  refreshRunQueries: () => Promise<void>;
}) {
  return useMutation({
    mutationFn: invokeExecutionHook(canvasId),
    onSuccess: async (_data, variables) => {
      await refreshRunQueries();
      showSuccessToast(successMessageForHook(variables.hookName));
    },
    onError: (error, variables) => {
      console.error(`Failed to invoke ${variables.hookName} hook`, error);
      showErrorToast(errorMessageForHook(variables.hookName));
    },
  });
}

function successMessageForHook(hookName: string): string {
  if (hookName === "approve") return "Approval submitted";
  if (hookName === "reject") return "Rejection submitted";
  if (hookName === "pushThrough") return "Step pushed through";
  return "Action submitted";
}

function errorMessageForHook(hookName: string): string {
  if (hookName === "approve") return "Failed to approve";
  if (hookName === "reject") return "Failed to reject";
  if (hookName === "pushThrough") return "Failed to push through";
  return "Failed to run action";
}

async function cancelExecution(canvasId: string, executionId: string) {
  await canvasesCancelExecution(
    withOrganizationHeader({
      path: {
        canvasId,
        executionId,
      },
    }),
  );
}

async function deleteQueuedItem(canvasId: string, nodeId: string, itemId: string) {
  await canvasesDeleteNodeQueueItem(
    withOrganizationHeader({
      path: {
        canvasId,
        nodeId,
        itemId,
      },
    }),
  );
}

function invokeExecutionHook(canvasId: string) {
  return async ({
    executionId,
    hookName,
    parameters,
  }: {
    executionId: string;
    hookName: string;
    parameters?: Record<string, unknown> | null;
  }) => {
    await canvasesInvokeNodeExecutionHook(
      withOrganizationHeader({
        path: {
          canvasId,
          executionId,
          hookName,
        },
        body: {
          parameters: parameters ?? undefined,
        },
      }),
    );
  };
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
