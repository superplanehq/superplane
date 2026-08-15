import { useCallback, useMemo } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useTheme } from "@/contexts/useTheme";
import { useReplayInputsState } from "@/hooks/useReplayInputsState";
import { REPLAY_NODE_ERROR_FALLBACK, useReplayNode } from "@/hooks/useReplayNode";
import { getApiErrorMessage } from "@/lib/errors";
import { validateReplayLaunch, type ReplayInputDraft } from "@/lib/validateReplayLaunch";
import { ReplayInputsList, type ReplayInputListItem } from "./ReplayInputsList";

export interface ReplayNodeModalProps {
  onClose: () => void;
  canvasId: string | null;
  nodeId: string;
  originalPayload: unknown;
  sourceNodeId?: string;
  sourceExecutionId?: string;
  onReplayed?: (runId: string) => void;
  // The resolve-inputs response only carries node ids, not names. Callers that have
  // the canvas's nodes on hand can resolve labels here; callers that don't leave
  // editors labeled by raw id.
  nodeNamesById?: Record<string, string>;
}

function testIdPrefixFor(index: number, total: number): string {
  return total > 1 ? `replay-input-${index}` : "replay";
}

function labelFor(
  sourceNodeId: string | undefined,
  index: number,
  total: number,
  nodeNamesById: Record<string, string> | undefined,
): string {
  if (total <= 1) return "Payload";
  if (sourceNodeId && nodeNamesById?.[sourceNodeId]) return nodeNamesById[sourceNodeId];
  return sourceNodeId ?? `Input ${index + 1}`;
}

export function ReplayNodeModal({
  onClose,
  canvasId,
  nodeId,
  originalPayload,
  sourceNodeId,
  sourceExecutionId,
  onReplayed,
  nodeNamesById,
}: ReplayNodeModalProps) {
  const { resolvedTheme } = useTheme();
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";

  const { views, updateInput, resolveError, isResolving } = useReplayInputsState({
    canvasId,
    nodeId,
    sourceExecutionId,
    originalPayload,
    sourceNodeId,
  });

  const handleReplayed = useCallback(
    (runId?: string) => {
      onClose();
      if (runId) {
        onReplayed?.(runId);
      }
    },
    [onClose, onReplayed],
  );

  const replayMutation = useReplayNode({ canvasId, nodeId, onReplayed: handleReplayed });
  const { isError: replayFailed, reset: resetReplay, mutate: launchReplay } = replayMutation;

  const mutationError = replayMutation.error
    ? getApiErrorMessage(replayMutation.error, REPLAY_NODE_ERROR_FALLBACK)
    : null;

  const handleInputChange = useCallback(
    (index: number, value: string) => {
      updateInput(index, value);
      if (replayFailed) {
        resetReplay();
      }
    },
    [updateInput, replayFailed, resetReplay],
  );

  const drafts: ReplayInputDraft[] = useMemo(
    () =>
      views.map((view, index) => ({
        sourceNodeId: view.sourceNodeId,
        label: labelFor(view.sourceNodeId, index, views.length, nodeNamesById),
        editedText: view.editedText,
        status: view.status,
      })),
    [views, nodeNamesById],
  );

  const validation = useMemo(() => validateReplayLaunch(drafts), [drafts]);
  const replayError = mutationError ?? resolveError ?? validation.launchError ?? null;

  const listItems: ReplayInputListItem[] = drafts.map((draft, index) => {
    const field = validation.fields[index];
    return {
      testIdPrefix: testIdPrefixFor(index, drafts.length),
      label: draft.label,
      editedText: draft.editedText,
      originalText: views[index].originalText,
      status: draft.status,
      error: field && !field.valid ? field.error : null,
      errorKind: field && !field.valid ? field.kind : null,
    };
  });

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) {
        onClose();
      }
    },
    [onClose],
  );

  const handleLaunch = useCallback(() => {
    if (!validation.valid || !validation.request) return;
    launchReplay({ ...validation.request, sourceExecutionId });
  }, [validation, launchReplay, sourceExecutionId]);

  return (
    <Dialog open onOpenChange={handleOpenChange}>
      <DialogContent size="large" className="flex h-[80vh] w-[90vw] max-w-[1200px] flex-col gap-3">
        <DialogHeader>
          <DialogTitle>Replay node</DialogTitle>
          <DialogDescription>
            Edit the payload below, then launch the replay. The replay runs this node in isolation with the payload you
            provide.
          </DialogDescription>
        </DialogHeader>

        <ReplayInputsList
          inputs={listItems}
          monacoTheme={monacoTheme}
          readOnly={isResolving}
          onChange={handleInputChange}
        />

        {replayError && (
          <div className="text-xs text-red-600" data-testid="replay-api-error">
            {replayError}
          </div>
        )}

        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="outline">
              Cancel
            </Button>
          </DialogClose>
          <Button
            type="button"
            data-testid="launch-replay-button"
            disabled={!validation.valid || replayMutation.isPending || isResolving || Boolean(resolveError)}
            onClick={handleLaunch}
          >
            {replayMutation.isPending ? "Launching..." : "Launch replay"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
