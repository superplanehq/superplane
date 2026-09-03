import { useCallback, useMemo, useState } from "react";
import type { CanvasesResolvedReplayInput, CanvasesResolvedReplayInputStatus } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { stringifyReplayPayload } from "@/lib/replayPayloadText";
import { useResolveReplayInputs } from "./useResolveReplayInputs";

export const RESOLVE_INPUTS_ERROR_FALLBACK = "Failed to load this step's replay inputs";

export type ReplayInputView = {
  sourceNodeId?: string;
  status?: CanvasesResolvedReplayInputStatus;
  originalText: string;
  editedText: string;
};

function buildActiveInputs(
  resolved: CanvasesResolvedReplayInput[],
  fallback: { sourceNodeId?: string; payload: unknown },
): CanvasesResolvedReplayInput[] {
  // Nothing resolved yet (loading, error, or an endpoint with nothing to say) — fall
  // back to the payload the caller already had, with no status to report.
  if (resolved.length === 0) {
    return [{ sourceNodeId: fallback.sourceNodeId, payload: fallback.payload as Record<string, unknown> | undefined }];
  }
  return resolved;
}

function buildEditedText(inputs: CanvasesResolvedReplayInput[]): string[] {
  return inputs.map((input) => stringifyReplayPayload(input.payload));
}

export function useReplayInputsState({
  canvasId,
  nodeId,
  sourceExecutionId,
  originalPayload,
  sourceNodeId,
}: {
  canvasId: string | null;
  nodeId: string;
  sourceExecutionId?: string;
  originalPayload: unknown;
  sourceNodeId?: string;
}) {
  const resolveQuery = useResolveReplayInputs({ canvasId, nodeId, sourceExecutionId });

  const activeInputs = useMemo(
    () => buildActiveInputs(resolveQuery.data ?? [], { sourceNodeId, payload: originalPayload }),
    [resolveQuery.data, sourceNodeId, originalPayload],
  );

  // Prefixed with whether this is resolved or fallback data: a single-source node's
  // resolved entry shares its sourceNodeId with the fallback, so the source ids
  // alone can't tell the two states apart once the query settles.
  const activeInputsKey = useMemo(
    () =>
      `${resolveQuery.data ? "resolved" : "fallback"}:${activeInputs.map((input) => input.sourceNodeId ?? "").join("|")}`,
    [activeInputs, resolveQuery.data],
  );

  const [state, setState] = useState(() => ({
    key: activeInputsKey,
    editedByIndex: buildEditedText(activeInputs),
  }));

  // Derived-state-during-render, not an effect: avoids a stale-then-corrected render
  // each time the resolve query's data identity changes.
  if (state.key !== activeInputsKey) {
    setState({ key: activeInputsKey, editedByIndex: buildEditedText(activeInputs) });
  }

  const views: ReplayInputView[] = activeInputs.map((input, index) => {
    const originalText = stringifyReplayPayload(input.payload);
    return {
      sourceNodeId: input.sourceNodeId,
      status: input.status,
      originalText,
      editedText: state.editedByIndex[index] ?? originalText,
    };
  });

  const updateInput = useCallback((index: number, value: string) => {
    setState((previous) => ({
      ...previous,
      editedByIndex: previous.editedByIndex.map((text, position) => (position === index ? value : text)),
    }));
  }, []);

  // Surfaced rather than swallowed: without the resolved set an aggregating node
  // renders as a one-input step, which looks like a fact rather than a failure.
  const resolveError = resolveQuery.error
    ? getApiErrorMessage(resolveQuery.error, RESOLVE_INPUTS_ERROR_FALLBACK)
    : null;

  // isLoading, not isFetching: a background refetch must not disable the launch,
  // and a disabled query (no source execution) never counts as loading at all.
  return { views, updateInput, resolveError, isResolving: resolveQuery.isLoading };
}
