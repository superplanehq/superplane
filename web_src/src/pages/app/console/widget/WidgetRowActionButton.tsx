import { useState } from "react";
import { ExternalLink, Play, RefreshCw, Square, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";

import { useConsoleContext, resolveConsoleNode } from "../ConsoleContext";
import { isManualRunNode } from "../manualRunTriggers";
import { mergeTriggerParameters } from "./mergeTriggerPayload";
import { RowActionConfirmDialog } from "./RowActionConfirmDialog";
import type { WidgetRowAction } from "./types";
import { useWidgetTableActionLock } from "./WidgetTableActionLockContext";

/**
 * Trigger-firing row button shared by every widget that renders one:
 * `WidgetTable` (per-row actions), `WidgetBoard` (per-card actions).
 *
 * The button owns the local `confirm` dialog state and submission
 * accounting via `useRowActionFire` / `useRowActionGate`, both of which
 * read the shared per-widget action lock through
 * {@link useWidgetTableActionLock}. Consumers must therefore wrap this
 * button in a {@link WidgetTableActionLockProvider} so the row-key /
 * trigger-node-id → lock accounting is coherent inside the widget.
 *
 * Behavior parity with the original inlined `RowActionButton`:
 * - `disabled` reasons combine perm / resolution / manual-run gates plus
 *   the shared per-row and per-trigger locks;
 * - templates without input fields fire immediately unless `action.confirm`
 *   is set;
 * - errors from the invoke call surface as an inline chip under the button;
 * - non-manual-run trigger nodes still render as disabled (defense in
 *   depth — table/board widgets already hide them upstream).
 */
export function WidgetRowActionButton({
  action,
  row,
  rowKey,
}: {
  action: WidgetRowAction;
  row: Record<string, unknown>;
  rowKey: string;
}) {
  const [confirmOpen, setConfirmOpen] = useState(false);

  const { resolved, isManualRun, disabled, reason, tooltip } = useRowActionGate(action, rowKey);
  const label = action.label ?? "Run";
  const hookName = action.hook ?? "run";
  const Icon = action.icon ? ACTION_ICONS[action.icon] : undefined;

  const { fire, error, pending } = useRowActionFire({
    action,
    row,
    rowKey,
    resolved,
    hookName,
    label,
    setConfirmOpen,
  });

  const handleClick = () => {
    if (disabled) return;
    if (action.confirm?.trim()) {
      setConfirmOpen(true);
      return;
    }
    void fire();
  };

  const testId = `widget-row-action-${action.node || "trigger"}`;

  return (
    <div className="inline-flex flex-col items-end gap-0.5">
      <Button
        type="button"
        size="xs"
        variant="outline"
        onClick={handleClick}
        disabled={disabled || pending}
        aria-disabled={disabled}
        title={tooltip}
        data-testid={testId}
        data-variant={action.variant ?? "default"}
        data-disabled-reason={reason ?? undefined}
      >
        {Icon ? <Icon className="mr-1 h-3 w-3" /> : null}
        {label}
      </Button>
      {error ? (
        <span
          className="max-w-48 text-right text-[10px] text-red-600 dark:text-red-400"
          data-testid={`${testId}-error`}
        >
          {error}
        </span>
      ) : null}
      {action.confirm ? (
        <RowActionConfirmDialog
          action={action}
          row={row}
          resolved={resolved}
          isManualRun={isManualRun}
          hookName={hookName}
          label={label}
          open={confirmOpen}
          onOpenChange={setConfirmOpen}
          confirmDisabled={pending || disabled}
          onConfirm={() => void fire()}
          testId={testId}
        />
      ) : null}
    </div>
  );
}

const ACTION_ICONS = {
  play: Play,
  stop: Square,
  trash: Trash2,
  refresh: RefreshCw,
  "external-link": ExternalLink,
} as const;

type ActionDisabledReason = "no-perm" | "no-node" | "not-manual-run" | "run-in-flight" | "submitting" | null;

type ResolvedNode = NonNullable<ReturnType<typeof resolveConsoleNode>>;

function useRowActionGate(action: WidgetRowAction, rowKey: string) {
  const ctx = useConsoleContext();
  const lock = useWidgetTableActionLock();
  const canRun = ctx?.canRunNodes ?? false;
  const resolved = resolveConsoleNode(ctx, action.node);
  const isManualRun = isManualRunNode(resolved?.node);
  const triggerNodeId = resolved?.node.id;
  const runInFlight = Boolean(
    triggerNodeId && lock.runInFlightIds.has(triggerNodeId) && lock.inFlightRowByTrigger.get(triggerNodeId) === rowKey,
  );
  const submitting = lock.pendingRowKeys.has(rowKey);
  const reason = disabledReasonFor(canRun, Boolean(resolved), isManualRun, runInFlight, submitting);
  return {
    resolved,
    isManualRun,
    disabled: reason !== null,
    reason,
    tooltip: disabledTooltip(reason, action.node),
  };
}

function useRowActionFire({
  action,
  row,
  rowKey,
  resolved,
  hookName,
  label,
  setConfirmOpen,
}: {
  action: WidgetRowAction;
  row: Record<string, unknown>;
  rowKey: string;
  resolved: ResolvedNode | undefined;
  hookName: string;
  label: string;
  setConfirmOpen: (open: boolean) => void;
}) {
  const ctx = useConsoleContext();
  const lock = useWidgetTableActionLock();
  const [error, setError] = useState<string | undefined>();
  const [pending, setPending] = useState(false);

  const fire = async () => {
    if (!ctx?.onTriggerNode || !resolved?.node.id) return;
    const triggerNodeId = resolved.node.id;
    setError(undefined);
    setPending(true);
    lock.beginSubmission(triggerNodeId, rowKey);
    let succeeded = false;
    try {
      const parameters = mergeTriggerParameters(resolved.node, hookName, action.template, row, action.payload);
      await ctx.onTriggerNode(triggerNodeId, {
        hookName,
        templateName: action.template,
        parameters,
        successLabel: label,
      });
      succeeded = true;
      setConfirmOpen(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to trigger");
    } finally {
      setPending(false);
      lock.endSubmission(triggerNodeId, rowKey, succeeded);
    }
  };

  return { fire, error, pending };
}

// `not-manual-run` is defense in depth: WidgetTable/WidgetBoard already
// hide non-manual-run actions upstream. This branch covers the transient
// case before the trigger catalog resolves — the action then renders
// disabled rather than as a button that would fail server-side.
function disabledReasonFor(
  canRun: boolean,
  hasResolvedNode: boolean,
  isManualRun: boolean,
  runInFlight: boolean,
  submitting: boolean,
): ActionDisabledReason {
  if (!canRun) return "no-perm";
  if (!hasResolvedNode) return "no-node";
  if (!isManualRun) return "not-manual-run";
  if (runInFlight) return "run-in-flight";
  if (submitting) return "submitting";
  return null;
}

function disabledTooltip(reason: ActionDisabledReason, node: string): string | undefined {
  switch (reason) {
    case "no-perm":
      return "You do not have permission to run actions in this canvas";
    case "no-node":
      return `Node "${node}" not found on this canvas`;
    case "not-manual-run":
      return "Only trigger nodes with a manual run can be fired from the console.";
    case "run-in-flight":
      return "A run for this trigger is already in progress.";
    case "submitting":
      return "Submitting trigger…";
    default:
      return undefined;
  }
}
