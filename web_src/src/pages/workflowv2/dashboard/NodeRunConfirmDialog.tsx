import { useEffect, useMemo, useState, type ReactNode } from "react";

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
import { resolveStartTemplate } from "./dashboardTriggerParameters";

interface NodeRunConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Resolved node + display label, mirroring `resolveDashboardNode`. */
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

  // Re-seed parameter values whenever the dialog opens on a different
  // template — keeps stale inputs from leaking across templates when a
  // single panel hosts multiple Run buttons.
  useEffect(() => {
    if (open) setParameterValues(seedParameterValues(parameters));
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
    setSubmitting(true);
    try {
      await onConfirm(previewParameters ?? { template: template.name });
      onOpenChange(false);
    } catch {
      // Keep the dialog open so the user can retry with the same values.
    } finally {
      setSubmitting(false);
    }
  };

  const dialogTitle = template ? `Run ${template.name}` : "Run";

  return (
    <Dialog open={open} onOpenChange={(next) => (submitting ? null : onOpenChange(next))}>
      <DialogContent className="pb-6">
        <DialogHeader>
          <DialogTitle>{dialogTitle}</DialogTitle>
          <DialogDescription>
            {resolved ? (
              <>
                Manually run <span className="font-medium text-slate-700">{resolved.label}</span>
                {template ? (
                  <>
                    {" "}
                    using template{" "}
                    <code className="rounded bg-slate-100 px-1 py-0.5 font-mono text-[11px] text-slate-700">
                      {template.name}
                    </code>
                  </>
                ) : null}
                .
              </>
            ) : (
              "Resolve the node to display its run options."
            )}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3 text-xs" data-testid={`${testId}-body`}>
          {!template ? (
            <p className="text-amber-700">This node does not declare any runnable Start template.</p>
          ) : (
            <>
              {hasParameters ? (
                <div className="space-y-1.5" data-testid={`${testId}-fields`}>
                  <p className="text-[10px] font-semibold uppercase tracking-wide text-slate-500">Parameters</p>
                  <StartRunParameterFields
                    parameters={parameters}
                    parameterValues={parameterValues}
                    onParameterValuesChange={setParameterValues}
                  />
                </div>
              ) : null}
              <ConfirmFact label="Will submit">
                <pre
                  className="mt-1 max-h-40 overflow-auto rounded-md border border-slate-200 bg-slate-50 p-2 font-mono text-[11px] leading-snug text-slate-700"
                  data-testid={`${testId}-parameters`}
                >
                  {formatParameters(previewParameters)}
                </pre>
              </ConfirmFact>
            </>
          )}
        </div>
        <DialogFooter>
          <Button type="button" variant="ghost" onClick={() => onOpenChange(false)} disabled={submitting}>
            Cancel
          </Button>
          <LoadingButton
            type="button"
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

function ConfirmFact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-0.5">
      <p className="text-[10px] font-semibold uppercase tracking-wide text-slate-500">{label}</p>
      <div className="text-slate-700">{children}</div>
    </div>
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

function formatParameters(parameters: Record<string, unknown> | undefined): string {
  if (!parameters || Object.keys(parameters).length === 0) return "(empty)";
  try {
    return JSON.stringify(parameters, null, 2);
  } catch {
    return String(parameters);
  }
}
