import { useCallback, useRef, useState } from "react";

import { useConsoleContext, type resolveConsoleNode } from "./ConsoleContext";
import { confirmConsoleTriggerNode } from "./confirmConsoleTriggerNode";
import { buildConsoleTriggerParameters, triggerHasParameters } from "./consoleTriggerParameters";
import { isManualRunNode } from "./manualRunTriggers";
import type { ConsoleTriggerLock } from "./useConsoleTriggerLock";

type ResolvedConsoleNode = ReturnType<typeof resolveConsoleNode>;

interface UseConsoleRunTriggerArgs {
  resolved: ResolvedConsoleNode;
  triggerName: string | undefined;
  promptConfirmation: boolean | undefined;
  /**
   * Shared submission lock, created once per widget (one per nodes panel)
   * so every Run button in that widget observes the same pending state.
   * Submissions are keyed by trigger node id — two entries pointing at the
   * same trigger lock together the moment either fires, instead of waiting
   * for the websocket-driven `STATE_STARTED` refresh to catch up.
   */
  lock: ConsoleTriggerLock;
}

export interface UseConsoleRunTriggerResult {
  canRun: boolean;
  running: boolean;
  /**
   * `true` when the button should be disabled — combines the runtime
   * authorization flag, node resolution, manual-run gate, in-flight run,
   * and local submission state. Callers can wire it straight to
   * `<Button disabled>` without recombining signals.
   */
  disabled: boolean;
  /**
   * Explains a `disabled` state so the button can surface a helpful
   * tooltip. `null` when the button is enabled.
   */
  disabledReason: null | "no-perm" | "no-resolved-node" | "not-manual-run" | "run-in-flight" | "submitting";
  dialogOpen: boolean;
  setDialogOpen: (next: boolean) => void;
  handleClick: () => void;
  runTrigger: (parameters: Record<string, unknown>) => void;
}

/**
 * Shared run-button orchestration for the merged console node panel and
 * any future widget that fires a trigger from a single-source button.
 *
 * Owns the `running` state, the confirm-dialog open state, and the routing
 * between the direct-run path (parameter-less trigger, no `promptConfirmation`)
 * and the confirm-dialog path. A synchronous `useRef` guard blocks re-entry
 * before React commits the `running` state, so a rapid double click on the
 * Run button (or double-confirm from the dialog) cannot fire the trigger
 * twice.
 *
 * The caller-provided {@link ConsoleTriggerLock} feeds in-flight run signals
 * so the button stays disabled until the run leaves `STATE_STARTED`, and
 * carries the submission lock keyed by trigger node id — because the lock
 * instance is shared across the widget, sibling buttons targeting the same
 * trigger disable together during the invoke flight and grace window.
 * `confirmConsoleTriggerNode` surfaces failures through a toast; the hook
 * swallows the rethrow so callers stay fire-and-forget.
 */
export function useConsoleRunTrigger({
  resolved,
  triggerName,
  promptConfirmation,
  lock,
}: UseConsoleRunTriggerArgs): UseConsoleRunTriggerResult {
  const ctx = useConsoleContext();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [running, setRunning] = useState(false);
  const runningRef = useRef(false);
  const triggerNodeId = resolved?.node?.id;

  const hasResolved = Boolean(resolved);
  const isManualRun = isManualRunNode(resolved?.node);
  // Panel entries lock per trigger — disable while the trigger has a run in
  // flight regardless of the originating source, so navigating to a canvas
  // where a different widget kicked off the same trigger also reflects the
  // "run in progress" state.
  const runInFlight = Boolean(triggerNodeId && lock.runInFlightIds.has(triggerNodeId));
  const submitting = Boolean(triggerNodeId && lock.pendingLockKeys.has(triggerNodeId));

  const disabledReason = resolveDisabledReason({
    canRunNodes: ctx?.canRunNodes ?? false,
    hasResolved,
    isManualRun,
    submitting: running || submitting,
    runInFlight,
  });

  const canRun = disabledReason === null || disabledReason === "run-in-flight" || disabledReason === "submitting";
  // `disabled` is true whenever there's any reason not to fire — the
  // "canRun" flag stays around for callers that only care about the
  // permission/resolution gate (e.g. whether to render the button at all).
  const disabled = disabledReason !== null;
  const shouldPrompt = triggerHasParameters(resolved?.node, triggerName) || Boolean(promptConfirmation);

  const runTrigger = useCallback(
    (parameters: Record<string, unknown>) => {
      const nodeId = resolved?.node?.id;
      if (!nodeId) return;
      if (runningRef.current) return;
      // A confirm dialog can outlive the state that opened it — e.g. another
      // widget fires the same trigger while the dialog is up. Re-check the
      // shared lock at fire time so a stale confirm can't enqueue a
      // duplicate run.
      if (lock.runInFlightIds.has(nodeId)) return;
      if (lock.pendingLockKeys.has(nodeId)) return;
      runningRef.current = true;
      setRunning(true);
      lock.beginSubmission(nodeId, nodeId);
      let succeeded = false;
      void (async () => {
        try {
          await confirmConsoleTriggerNode(ctx, nodeId, triggerName, parameters);
          succeeded = true;
        } catch {
          // Already reported to the user via toast.
        } finally {
          runningRef.current = false;
          setRunning(false);
          lock.endSubmission(nodeId, nodeId, succeeded);
        }
      })();
    },
    [ctx, resolved, triggerName, lock],
  );

  const handleClick = useCallback(() => {
    if (runningRef.current) return;
    if (shouldPrompt) {
      setDialogOpen(true);
      return;
    }
    if (!resolved?.node) return;
    runTrigger(buildConsoleTriggerParameters(resolved.node, "run", triggerName));
  }, [shouldPrompt, resolved, triggerName, runTrigger]);

  return { canRun, running, disabled, disabledReason, dialogOpen, setDialogOpen, handleClick, runTrigger };
}

function resolveDisabledReason({
  canRunNodes,
  hasResolved,
  isManualRun,
  submitting,
  runInFlight,
}: {
  canRunNodes: boolean;
  hasResolved: boolean;
  isManualRun: boolean;
  submitting: boolean;
  runInFlight: boolean;
}): UseConsoleRunTriggerResult["disabledReason"] {
  if (!canRunNodes) return "no-perm";
  if (!hasResolved) return "no-resolved-node";
  if (!isManualRun) return "not-manual-run";
  if (submitting) return "submitting";
  if (runInFlight) return "run-in-flight";
  return null;
}
