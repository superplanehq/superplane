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

import { ConfirmFact, ConfirmParametersPreview } from "../confirmDialogPreview";
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
  isTrigger: boolean;
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
  isTrigger,
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
      <DialogContent className="min-w-0 overflow-hidden pb-6">
        <DialogHeader className="min-w-0">
          <DialogTitle>{label}</DialogTitle>
          <DialogDescription className="min-w-0">{confirmBody}</DialogDescription>
        </DialogHeader>
        <div className="min-w-0 space-y-3 text-xs" data-testid={`${testId}-preview`}>
          <ConfirmTriggerFact resolved={resolved} fallback={action.node} isTrigger={isTrigger} />
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
  isTrigger,
}: {
  resolved: ResolvedNode | undefined;
  fallback: string;
  isTrigger: boolean;
}) {
  return (
    <ConfirmFact label="Trigger">
      <span className="font-medium text-slate-800">{resolved?.label ?? fallback}</span>
      {resolved?.node.id ? (
        <span className="ml-1 font-mono text-[10px] text-slate-500">({resolved.node.id})</span>
      ) : null}
      {!resolved ? (
        <span className="ml-1 text-red-600">— node not found on this canvas</span>
      ) : !isTrigger ? (
        <span className="ml-1 text-amber-600">— not a trigger node</span>
      ) : null}
    </ConfirmFact>
  );
}

function ConfirmHookFact({ hookName, templateName }: { hookName: string; templateName: string | undefined }) {
  return (
    <ConfirmFact label="Hook">
      <code className="rounded bg-slate-100 px-1 py-0.5 font-mono text-[11px] text-slate-700">{hookName}</code>
      {templateName ? (
        <>
          <span className="mx-1 text-slate-400">/</span>
          <code className="rounded bg-slate-100 px-1 py-0.5 font-mono text-[11px] text-slate-700">{templateName}</code>
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
        <span className="text-red-600">Failed to build parameters: {preview.error}</span>
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
