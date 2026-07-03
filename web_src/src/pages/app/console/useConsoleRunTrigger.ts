import { useCallback, useRef, useState } from "react";

import { resolveConsoleNode, useConsoleContext } from "./ConsoleContext";
import { confirmConsoleTriggerNode } from "./confirmConsoleTriggerNode";
import { buildConsoleTriggerParameters, triggerHasParameters } from "./consoleTriggerParameters";

type ResolvedConsoleNode = ReturnType<typeof resolveConsoleNode>;

interface UseConsoleRunTriggerArgs {
  resolved: ResolvedConsoleNode;
  triggerName: string | undefined;
  promptConfirmation: boolean | undefined;
}

export interface UseConsoleRunTriggerResult {
  canRun: boolean;
  running: boolean;
  dialogOpen: boolean;
  setDialogOpen: (next: boolean) => void;
  handleClick: () => void;
  runTrigger: (parameters: Record<string, unknown>) => void;
}

/**
 * Shared run-button orchestration for the console `node` and `nodes` panels.
 *
 * Owns the `running` state, the confirm-dialog open state, and the routing
 * between the direct-run path (parameter-less trigger, no `promptConfirmation`)
 * and the confirm-dialog path. A synchronous `useRef` guard blocks re-entry
 * before React commits the `running` state, so a rapid double click on the
 * Run button (or double-confirm from the dialog) cannot fire the trigger
 * twice.
 *
 * `confirmConsoleTriggerNode` already surfaces failures through a toast; the
 * hook swallows the rethrow so the caller stays fire-and-forget.
 */
export function useConsoleRunTrigger({
  resolved,
  triggerName,
  promptConfirmation,
}: UseConsoleRunTriggerArgs): UseConsoleRunTriggerResult {
  const ctx = useConsoleContext();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [running, setRunning] = useState(false);
  const runningRef = useRef(false);

  const canRun = (ctx?.canRunNodes ?? false) && Boolean(resolved);
  const shouldPrompt = triggerHasParameters(resolved?.node, triggerName) || Boolean(promptConfirmation);

  const runTrigger = useCallback(
    (parameters: Record<string, unknown>) => {
      const nodeId = resolved?.node?.id;
      if (!nodeId) return;
      if (runningRef.current) return;
      runningRef.current = true;
      setRunning(true);
      void (async () => {
        try {
          await confirmConsoleTriggerNode(ctx, nodeId, triggerName, parameters);
        } catch {
          // Already reported to the user via toast.
        } finally {
          runningRef.current = false;
          setRunning(false);
        }
      })();
    },
    [ctx, resolved, triggerName],
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

  return { canRun, running, dialogOpen, setDialogOpen, handleClick, runTrigger };
}
