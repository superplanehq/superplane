import { useCallback, useMemo, useRef, useState } from "react";

import { isManualRunNode, useConsoleContext, type resolveConsoleNode } from "./ConsoleContext";
import { confirmConsoleTriggerNode } from "./confirmConsoleTriggerNode";
import { buildConsoleTriggerParameters, triggerHasParameters } from "./consoleTriggerParameters";
import { useConsoleTriggerLock } from "./useConsoleTriggerLock";

type ResolvedConsoleNode = ReturnType<typeof resolveConsoleNode>;

interface UseConsoleRunTriggerArgs {
  resolved: ResolvedConsoleNode;
  triggerName: string | undefined;
  promptConfirmation: boolean | undefined;
  /**
   * Stable identifier for the source firing this trigger — the merged
   * node panel passes an entry-scoped key (e.g. `"nodes-panel-entry:0"`)
   * so its Run button stays locked while its own submission is in flight,
   * consistent with the table widget's per-row locking. Falls back to the
   * resolved node id when omitted.
   */
  lockKey?: string;
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
 * The shared {@link useConsoleTriggerLock} feeds in-flight run signals so
 * the button stays disabled until the run leaves `STATE_STARTED`, matching
 * the table row-action gate. `confirmConsoleTriggerNode` surfaces failures
 * through a toast; the hook swallows the rethrow so callers stay
 * fire-and-forget.
 */
export function useConsoleRunTrigger({
  resolved,
  triggerName,
  promptConfirmation,
  lockKey,
}: UseConsoleRunTriggerArgs): UseConsoleRunTriggerResult {
  const ctx = useConsoleContext();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [running, setRunning] = useState(false);
  const runningRef = useRef(false);
  const triggerNodeId = resolved?.node?.id;
  const effectiveLockKey = lockKey ?? triggerNodeId ?? "";

  const triggerNodeIds = useMemo(() => (triggerNodeId ? [triggerNodeId] : []), [triggerNodeId]);
  const lock = useConsoleTriggerLock({ triggerNodeIds });

  const hasResolved = Boolean(resolved);
  const isManualRun = isManualRunNode(ctx, resolved?.node);
  // Panel entries have one button per trigger — disable while the trigger
  // has a run in flight regardless of the originating source, so navigating
  // to a canvas where a different widget kicked off the same trigger also
  // reflects the "run in progress" state.
  const runInFlight = Boolean(triggerNodeId && lock.runInFlightIds.has(triggerNodeId));
  const submitting = Boolean(effectiveLockKey && lock.pendingLockKeys.has(effectiveLockKey));

  const disabledReason: UseConsoleRunTriggerResult["disabledReason"] = !(ctx?.canRunNodes ?? false)
    ? "no-perm"
    : !hasResolved
      ? "no-resolved-node"
      : !isManualRun
        ? "not-manual-run"
        : running || submitting
          ? "submitting"
          : runInFlight
            ? "run-in-flight"
            : null;

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
      runningRef.current = true;
      setRunning(true);
      lock.beginSubmission(nodeId, effectiveLockKey);
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
          lock.endSubmission(nodeId, effectiveLockKey, succeeded);
        }
      })();
    },
    [ctx, resolved, triggerName, lock, effectiveLockKey],
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
