import { useEffect, useMemo, useState } from "react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { showErrorToast } from "@/lib/toast";
import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client/types.gen";

import { StartRunParameterFields } from "../mappers/start/startRunParameterFields";
import {
  coerceParameterValue,
  initialParameterValue,
  parameterDisplayLabel,
  type StartTemplateParameter,
} from "../mappers/start/templatePayload";
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
   * (`{ template, ...values }`). Fired as the dialog closes; the caller owns
   * the run (and its loading state on the widget's Run button), and surfaces
   * any failure via a toast — the dialog does not stay open on error.
   */
  onConfirm: (parameters: Record<string, unknown>) => void;
  /** Test id prefix; falls back to a stable default. */
  testId?: string;
}

/**
 * Confirmation dialog for Console Node / Key Nodes Run buttons. Renders the
 * canvas Start trigger run modal's input fields when the resolved template
 * declares `parameters`; templates with no parameters show a bare
 * confirmation (title + Cancel/Run). Callers decide whether to open this
 * dialog at all — see the widget run controls, which skip it for
 * parameter-less templates unless "Prompt confirmation" is enabled.
 *
 * Confirming validates the inputs, then hands the built parameters to
 * `onConfirm` and closes immediately. The run executes in the background with
 * the loading state shown on the widget's Run button, mirroring the
 * no-confirmation path.
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

  // Re-seed parameter values whenever the dialog opens on a different
  // template — keeps stale inputs from leaking across templates when a
  // single panel hosts multiple Run buttons.
  useEffect(() => {
    if (open) {
      setParameterValues(seedParameterValues(parameters));
    }
  }, [open, parameters]);

  const previewParameters = useMemo(() => {
    if (!template) return undefined;
    return buildParameters(template.name, parameters, parameterValues);
  }, [template, parameters, parameterValues]);

  const handleConfirm = () => {
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
    onConfirm(previewParameters ?? { template: template.name });
    onOpenChange(false);
  };

  const dialogTitle = template ? `Run ${template.name}` : "Run";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
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
          ) : hasParameters ? (
            <div className="min-w-0 space-y-1.5" data-testid={`${testId}-fields`}>
              <StartRunParameterFields
                parameters={parameters}
                parameterValues={parameterValues}
                onParameterValuesChange={setParameterValues}
              />
            </div>
          ) : (
            <p className="text-slate-600" data-testid={`${testId}-confirm-message`}>
              Run <span className="font-medium text-slate-800">{resolved?.label ?? template.name}</span>?
            </p>
          )}
        </div>
        <DialogFooter className="min-w-0">
          <Button type="button" variant="ghost" size="xs" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            size="xs"
            onClick={handleConfirm}
            disabled={!template || !resolved?.node?.id}
            data-testid={`${testId}-submit`}
          >
            Run
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
