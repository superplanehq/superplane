import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";
import {
  canvasesCancelExecution,
  canvasesCancelRun,
  canvasesDeleteNodeQueueItem,
  canvasesInvokeNodeExecutionHook,
  canvasesReemitTriggerEvent,
  type CanvasesCanvasRun,
} from "@/api-client";
import { buildNodeNamesById } from "@/lib/nodeNamesById";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import type { RunInspectorNodeSection } from "./types";

export type RunInspectorReplayTarget = {
  canvasId: string;
  nodeId: string;
  payload: unknown;
  sourceNodeId?: string;
  sourceExecutionId?: string;
};

//
// A replay reruns this one node against the input it already consumed, so it
// needs that past input: the trigger has none (it is replayed by Rerun), a
// queued step has not consumed one yet, and retention can null out an
// execution's input event long after the fact.
//
export function canReplaySection(section: RunInspectorNodeSection): boolean {
  return !section.isTrigger && !section.isQueued && Boolean(section.execution?.inputEvent);
}

function toReplayTarget(canvasId: string, section: RunInspectorNodeSection): RunInspectorReplayTarget | null {
  const inputEvent = section.execution?.inputEvent;
  if (!inputEvent) return null;

  return {
    canvasId,
    nodeId: section.nodeId,
    payload: inputEvent.data,
    sourceNodeId: inputEvent.nodeId,
    sourceExecutionId: section.execution?.id,
  };
}

export function useRunInspectorActions({
  canvasId,
  run,
  sections,
  executionsLoading,
  onRerunCreated,
  onReplayCreated,
}: {
  canvasId: string;
  run: CanvasesCanvasRun;
  sections: RunInspectorNodeSection[];
  executionsLoading: boolean;
  onRerunCreated?: (eventId: string) => void | Promise<void>;
  onReplayCreated?: (runId: string) => void;
}) {
  const queryClient = useQueryClient();
  const runningExecutionIds = useMemo(() => {
    const ids = new Set<string>();
    for (const executionRef of run.executions ?? []) {
      if (executionRef.id && executionRef.state === "STATE_STARTED") {
        ids.add(executionRef.id);
      }
    }

    for (const section of sections) {
      if (section.execution?.id && section.execution.state === "STATE_STARTED") {
        ids.add(section.execution.id);
      }

      if (section.executionRef?.id && section.executionRef.state === "STATE_STARTED") {
        ids.add(section.executionRef.id);
      }
    }

    return [...ids];
  }, [run.executions, sections]);
  const hasLoadedActionSection = useMemo(
    () => sections.some((section) => !section.isTrigger && section.execution),
    [sections],
  );
  const queuedItems = useMemo(
    () => run.queueItems?.filter(hasQueueItemIdentity).map(toQueuedItemReference) ?? [],
    [run.queueItems],
  );

  const refreshRunQueries = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: ["canvases"] });
  }, [queryClient]);

  const rerunMutation = useRerunMutation({ canvasId, run, refreshRunQueries, onRerunCreated });
  const stopMutation = useStopMutation({
    canvasId,
    runId: run.id ?? null,
    refreshRunQueries,
  });
  const stopNodeMutation = useStopNodeMutation({ canvasId, refreshRunQueries });
  const executionHookMutation = useExecutionHookMutation({ canvasId, refreshRunQueries });
  const cancelQueuedItemMutation = useCancelQueuedItemMutation({ canvasId, refreshRunQueries });
  const [replaySection, setReplaySection] = useState<RunInspectorNodeSection | null>(null);
  // The steps list only ever knows nodes that appear as sections in this run, so a
  // source outside it (e.g. never executed here) falls back to its raw id in the modal.
  const nodeNamesById = useMemo(
    () =>
      buildNodeNamesById(
        sections,
        (section) => section.nodeId,
        (section) => section.nodeName,
      ),
    [sections],
  );

  return {
    replayTarget: replaySection ? toReplayTarget(canvasId, replaySection) : null,
    nodeNamesById,
    openReplay: (section: RunInspectorNodeSection) => {
      if (!canReplaySection(section)) return;
      setReplaySection(section);
    },
    closeReplay: () => setReplaySection(null),
    onReplayCreated,
    rerun: () => rerunMutation.mutate(),
    rerunPending: rerunMutation.isPending,
    stop: () => stopMutation.mutate(),
    stopPending: stopMutation.isPending,
    stopDisabled:
      stopMutation.isPending ||
      run.state === "STATE_CANCELLING" ||
      (runningExecutionIds.length === 0 &&
        queuedItems.length === 0 &&
        (executionsLoading || !run.rootEvent?.id || !hasLoadedActionSection)),
    stopNode: (section: RunInspectorNodeSection) => {
      if (!section.execution?.id) return;
      stopNodeMutation.mutate(section.execution.id);
    },
    stopNodePending: stopNodeMutation.isPending,
    cancelQueuedItem: (section: RunInspectorNodeSection) => {
      if (!section.queueItem?.nodeId || !section.queueItem.id) return;
      cancelQueuedItemMutation.mutate({ nodeId: section.queueItem.nodeId, itemId: section.queueItem.id });
    },
    cancelQueuedItemPending: cancelQueuedItemMutation.isPending,
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
  onRerunCreated,
}: {
  canvasId: string;
  run: CanvasesCanvasRun;
  refreshRunQueries: () => Promise<void>;
  onRerunCreated?: (eventId: string) => void | Promise<void>;
}) {
  return useMutation({
    mutationFn: async () => {
      if (!run.rootEvent?.nodeId || !run.rootEvent?.id) {
        throw new Error("Run root event is missing");
      }

      const response = await canvasesReemitTriggerEvent(
        withOrganizationHeader({
          path: {
            canvasId,
            nodeId: run.rootEvent.nodeId,
            eventId: run.rootEvent.id,
          },
        }),
      );

      return response.data?.eventId;
    },
    onSuccess: async (eventId) => {
      await refreshRunQueries();
      if (eventId) {
        await onRerunCreated?.(eventId);
      }
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
  runId,
  refreshRunQueries,
}: {
  canvasId: string;
  runId: string | null;
  refreshRunQueries: () => Promise<void>;
}) {
  return useMutation({
    mutationFn: async () => {
      if (!runId) {
        return;
      }

      await canvasesCancelRun(
        withOrganizationHeader({
          path: {
            canvasId,
            runId,
          },
        }),
      );
    },
    onSuccess: async () => {
      await refreshRunQueries();
    },
    onError: (error) => {
      console.error("Failed to stop run", error);
      showErrorToast("Failed to stop run");
    },
  });
}

function useCancelQueuedItemMutation({
  canvasId,
  refreshRunQueries,
}: {
  canvasId: string;
  refreshRunQueries: () => Promise<void>;
}) {
  return useMutation({
    mutationFn: (queueItem: QueuedItemReference) => deleteQueuedItem(canvasId, queueItem.nodeId, queueItem.itemId),
    onSuccess: async () => {
      await refreshRunQueries();
      showSuccessToast("Queued step cancelled");
    },
    onError: (error) => {
      console.error("Failed to cancel queued step", error);
      showErrorToast("Failed to cancel queued step");
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

type QueueItemWithIdentity = {
  id: string;
  nodeId: string;
};

type QueuedItemReference = {
  nodeId: string;
  itemId: string;
};

function hasQueueItemIdentity(
  queueItem: NonNullable<CanvasesCanvasRun["queueItems"]>[number],
): queueItem is QueueItemWithIdentity {
  return Boolean(queueItem.id && queueItem.nodeId);
}

function toQueuedItemReference(queueItem: QueueItemWithIdentity): QueuedItemReference {
  return { nodeId: queueItem.nodeId, itemId: queueItem.id };
}
