import { useEffect, useMemo, useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { showErrorToast } from "@/lib/toast";
import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client/types.gen";

import { StartRunParameterFields } from "../mappers/start/startRunParameterFields";
import {
  coerceParameterValue,
  initialParameterValue,
  parameterDisplayLabel,
  type StartTemplateParameter,
} from "../mappers/start/templatePayload";
import { ConfirmParametersPreview } from "./confirmDialogPreview";
import { formatParameters } from "./formatConfirmDialogParameters";
import { resolveStartTemplate } from "./consoleTriggerParameters";

interface NodeRunConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Resolved node + display label, mirroring `resolveConsoleNode`. */
  resolved: { node: ComponentsNode; label: string } | undefined;
  /** Optional template name override; defaults to the first declared template. */
  templateName?: string;
  /**
   * Final-step handler. Receives the merged parameters object
   * (`{ template, ...values }`). Must throw to keep the dialog open on
   * failure.
   */
  onConfirm: (parameters: Record<string, unknown>) => Promise<void>;
  /** Test id prefix; falls back to a stable default. */
  testId?: string;
}

/**
 * Confirmation dialog for Console Node / Key Nodes Run buttons. Mirrors the
 * table row-action confirm dialog (read-only JSON preview of the parameters
 * about to be submitted) and the canvas Start trigger run modal (optional
 * input fields when the resolved template declares `parameters`).
 *
 * The dialog always renders the preview so the user can verify what is
 * about to be sent before confirming, even for templates that take no
 * parameters. Submitting keeps the dialog open if `onConfirm` rejects so
 * users can retry without losing their inputs.
 */
export function NodeRunConfirmDialog({
  open,
  onOpenChange,
  resolved,
  templateName,
  onConfirm,
  testId = "node-run-confirm",
}: NodeRunConfirmDialogProps) {
  const template = useMemo(() => resolveStartTemplate(resolved?.node, templateName), [resolved?.node, templateName]);
  // `template?.parameters ?? []` returns a brand new array reference each
  // render when the template has no parameters, which would otherwise turn
  // any effect that depends on it into an infinite update loop. Memoizing
  // pins the reference per template so seeding only happens when the
  // template itself changes.
  const parameters = useMemo<StartTemplateParameter[]>(() => template?.parameters ?? [], [template]);
  const hasParameters = parameters.length > 0;

  const [parameterValues, setParameterValues] = useState<Record<string, string | number | boolean>>(() =>
    seedParameterValues(parameters),
  );
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | undefined>();
  const [payloadOpen, setPayloadOpen] = useState(false);

  // Re-seed parameter values whenever the dialog opens on a different
  // template — keeps stale inputs from leaking across templates when a
  // single panel hosts multiple Run buttons.
  useEffect(() => {
    if (open) {
      setParameterValues(seedParameterValues(parameters));
      setSubmitError(undefined);
      setPayloadOpen(false);
    }
  }, [open, parameters]);

  const previewParameters = useMemo(() => {
    if (!template) return undefined;
    return buildParameters(template.name, parameters, parameterValues);
  }, [template, parameters, parameterValues]);

  const handleConfirm = async () => {
    if (!template || !resolved?.node?.id) return;
    if (hasParameters) {
      for (const param of parameters) {
        const value = previewParameters?.[param.name];
        if (param.type === "number" && typeof value === "number" && Number.isNaN(value)) {
          showErrorToast(`"${parameterDisplayLabel(param)}" must be a valid number`);
          return;
        }
      }
    }
    setSubmitError(undefined);
    setSubmitting(true);
    try {
      await onConfirm(previewParameters ?? { template: template.name });
      onOpenChange(false);
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : "Failed to run");
    } finally {
      setSubmitting(false);
    }
  };

  const dialogTitle = template ? `Run ${template.name}` : "Run";

  return (
    <Dialog open={open} onOpenChange={(next) => (submitting ? null : onOpenChange(next))}>
      <DialogContent className="min-w-0 overflow-hidden pb-6">
        <DialogHeader className="min-w-0">
          <DialogTitle>{dialogTitle}</DialogTitle>
          {resolved ? (
            <DialogDescription className="sr-only">{dialogTitle}</DialogDescription>
          ) : (
            <DialogDescription className="min-w-0">Resolve the node to display its run options.</DialogDescription>
          )}
        </DialogHeader>
        <div className="min-w-0 space-y-3 text-[13px]" data-testid={`${testId}-body`}>
          {!template ? (
            <p className="text-amber-700">This node does not declare any runnable Start template.</p>
          ) : (
            <>
              {hasParameters ? (
                <div className="min-w-0 space-y-1.5" data-testid={`${testId}-fields`}>
                  <StartRunParameterFields
                    parameters={parameters}
                    parameterValues={parameterValues}
                    onParameterValuesChange={setParameterValues}
                  />
                </div>
              ) : null}
              <div className="min-w-0 space-y-0.5">
                <button
                  type="button"
                  onClick={() => setPayloadOpen((prev) => !prev)}
                  aria-expanded={payloadOpen}
                  className="flex items-center gap-1 rounded px-1 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-500 hover:bg-slate-100"
                  data-testid={`${testId}-payload-toggle`}
                >
                  {payloadOpen ? <ChevronDown className="size-3" /> : <ChevronRight className="size-3" />}
                  Payload
                </button>
                {payloadOpen ? (
                  <ConfirmParametersPreview testId={`${testId}-parameters`}>
                    {formatParameters(previewParameters)}
                  </ConfirmParametersPreview>
                ) : null}
              </div>
            </>
          )}
          <SubmitErrorMessage error={submitError} testId={testId} />
        </div>
        <DialogFooter className="min-w-0">
          <Button type="button" variant="ghost" size="xs" onClick={() => onOpenChange(false)} disabled={submitting}>
            Cancel
          </Button>
          <LoadingButton
            type="button"
            size="xs"
            loading={submitting}
            loadingText="Running…"
            onClick={handleConfirm}
            disabled={!template || !resolved?.node?.id}
            data-testid={`${testId}-submit`}
          >
            Run
          </LoadingButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function SubmitErrorMessage({ error, testId }: { error: string | undefined; testId: string }) {
  if (!error) return null;
  return (
    <p className="text-red-600" data-testid={`${testId}-error`}>
      {error}
    </p>
  );
}

function seedParameterValues(parameters: StartTemplateParameter[]): Record<string, string | number | boolean> {
  const seeded: Record<string, string | number | boolean> = {};
  for (const param of parameters) {
    if (!param.name || !param.type) continue;
    seeded[param.name] = initialParameterValue(param);
  }
  return seeded;
}

function buildParameters(
  templateName: string,
  parameters: StartTemplateParameter[],
  values: Record<string, string | number | boolean>,
): Record<string, unknown> {
  const out: Record<string, unknown> = { template: templateName };
  for (const param of parameters) {
    if (!param.name || !param.type) continue;
    out[param.name] = coerceParameterValue(param, values[param.name]);
  }
  return out;
}
