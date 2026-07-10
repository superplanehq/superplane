import { useMemo } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import { cn } from "@/lib/utils";

import { ConfirmFact, ConfirmParametersPreview } from "../confirmDialogPreview";
import { CONSOLE_CODE_BADGE_CLASSES } from "../consoleCodeStyles";
import { formatParameters } from "../formatConfirmDialogParameters";
import type { resolveConsoleNode } from "../ConsoleContext";
import { buildEnv, compileTemplate, evalTemplate } from "./celExpr";
import { mergeTriggerParameters } from "./mergeTriggerPayload";
import type { WidgetRowAction } from "./types";

type ResolvedNode = NonNullable<ReturnType<typeof resolveConsoleNode>>;

interface RowActionConfirmDialogProps {
  action: WidgetRowAction;
  row: Record<string, unknown>;
  resolved: ResolvedNode | undefined;
  /**
   * True when the resolved node is a trigger with a user-invokable `run`
   * hook (see `isManualRunNode`). The dialog uses this to warn when a row
   * is configured to target a node that the backend will not accept — a
   * defensive fallback since `WidgetTable` normally hides such rows.
   */
  isManualRun: boolean;
  hookName: string;
  label: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  confirmDisabled: boolean;
  onConfirm: () => void;
  testId: string;
}

/**
 * Confirmation dialog for a row action. Shows the resolved trigger node, the
 * hook/template that will be invoked, and a JSON preview of the merged
 * parameters so the user can verify exactly what the action will send before
 * confirming.
 */
export function RowActionConfirmDialog({
  action,
  row,
  resolved,
  isManualRun,
  hookName,
  label,
  open,
  onOpenChange,
  confirmDisabled,
  onConfirm,
  testId,
}: RowActionConfirmDialogProps) {
  const confirmBody = useMemo(() => {
    if (!action.confirm) return "";
    const env = buildEnv();
    return evalTemplate(compileTemplate(action.confirm), row, env, (v) => String(v ?? ""));
  }, [action.confirm, row]);

  const preview = useMemo(() => {
    if (!resolved?.node) return null;
    try {
      const parameters = mergeTriggerParameters(resolved.node, hookName, action.template, row, action.payload);
      return { parameters, error: undefined as string | undefined };
    } catch (err) {
      return { parameters: undefined, error: err instanceof Error ? err.message : String(err) };
    }
  }, [action.template, action.payload, resolved?.node, hookName, row]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="min-w-0 overflow-hidden pb-6 dark:border-gray-600 dark:bg-gray-900">
        <DialogHeader className="min-w-0">
          <DialogTitle>{label}</DialogTitle>
          <DialogDescription className="min-w-0">{confirmBody}</DialogDescription>
        </DialogHeader>
        <div className="min-w-0 space-y-3 text-xs" data-testid={`${testId}-preview`}>
          <ConfirmTriggerFact resolved={resolved} fallback={action.node} isManualRun={isManualRun} />
          <ConfirmHookFact hookName={hookName} templateName={extractTemplateName(preview?.parameters)} />
          <ConfirmParametersFact preview={preview} testId={testId} />
        </div>
        <DialogFooter className="min-w-0">
          <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            variant={action.variant === "danger" ? "destructive" : "default"}
            onClick={onConfirm}
            disabled={confirmDisabled}
            data-testid={`${testId}-confirm`}
          >
            {label}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ConfirmTriggerFact({
  resolved,
  fallback,
  isManualRun,
}: {
  resolved: ResolvedNode | undefined;
  fallback: string;
  isManualRun: boolean;
}) {
  return (
    <ConfirmFact label="Trigger">
      <span className="font-medium text-slate-800 dark:text-gray-100">{resolved?.label ?? fallback}</span>
      {resolved?.node.id ? (
        <span className="ml-1 font-mono text-[10px] text-slate-500 dark:text-gray-400">({resolved.node.id})</span>
      ) : null}
      {!resolved ? (
        <span className="ml-1 text-red-600 dark:text-red-400">— node not found on this canvas</span>
      ) : !isManualRun ? (
        <span className="ml-1 text-amber-600 dark:text-amber-400">— not a manual-run trigger</span>
      ) : null}
    </ConfirmFact>
  );
}

function ConfirmHookFact({ hookName, templateName }: { hookName: string; templateName: string | undefined }) {
  return (
    <ConfirmFact label="Hook">
      <code className={cn(CONSOLE_CODE_BADGE_CLASSES, "text-[11px]")}>{hookName}</code>
      {templateName ? (
        <>
          <span className="mx-1 text-slate-400 dark:text-gray-500">/</span>
          <code className={cn(CONSOLE_CODE_BADGE_CLASSES, "text-[11px]")}>{templateName}</code>
        </>
      ) : null}
    </ConfirmFact>
  );
}

function ConfirmParametersFact({
  preview,
  testId,
}: {
  preview: { parameters: Record<string, unknown> | undefined; error: string | undefined } | null;
  testId: string;
}) {
  return (
    <ConfirmFact label="Parameters">
      {preview?.error ? (
        <span className="text-red-600 dark:text-red-400">Failed to build parameters: {preview.error}</span>
      ) : (
        <ConfirmParametersPreview testId={`${testId}-parameters`}>
          {formatParameters(preview?.parameters)}
        </ConfirmParametersPreview>
      )}
    </ConfirmFact>
  );
}

function extractTemplateName(parameters: Record<string, unknown> | undefined): string | undefined {
  if (!parameters) return undefined;
  const name = parameters.template;
  return typeof name === "string" && name ? name : undefined;
}
