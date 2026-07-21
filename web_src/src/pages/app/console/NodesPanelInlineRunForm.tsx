import { useEffect, useMemo, useState } from "react";
import { Play } from "lucide-react";

import { LoadingButton } from "@/components/ui/loading-button";
import { showErrorToast } from "@/lib/toast";

import type { ConsoleTriggerLock } from "./useConsoleTriggerLock";
import { StartRunParameterFields } from "../mappers/start/startRunParameterFields";
import {
  coerceParameterValue,
  initialParameterValue,
  parameterDisplayLabel,
  type StartTemplate,
  type StartTemplateParameter,
} from "../mappers/start/templatePayload";

interface NodesPanelInlineRunFormProps {
  template: StartTemplate;
  /** Called with `{ template, ...values }`. */
  onSubmit: (parameters: Record<string, unknown>) => void;
  running: boolean;
  disabled: boolean;
  disabledTitle?: string;
  submitLabel?: string;
  showFieldLabels?: boolean;
  testIdPrefix: string;
  /** Shared lock instance from the parent panel. Reused to disable the
   * submit button while an equivalent run is already in flight, matching
   * the modal path. Only read for `runInFlightIds` — submissions still go
   * through {@link useConsoleRunTrigger} which owns lock accounting. */
  lock: ConsoleTriggerLock;
  triggerNodeId: string | undefined;
}

/**
 * Inline (in-widget) counterpart to {@link NodeRunConfirmDialog}: renders
 * the resolved Start template's parameter form directly in the panel body
 * plus a submit button.
 *
 * Used by the `nodes` panel when an entry opts in via `formMode: "inline"`.
 * The submit path mirrors the modal one exactly — coerce inputs, validate
 * numbers, build `{ template, ...values }`, and hand off to the caller's
 * `runTrigger` — so both surfaces share the same payload contract and
 * lock semantics through the caller-provided {@link ConsoleTriggerLock}.
 */
export function NodesPanelInlineRunForm({
  template,
  onSubmit,
  running,
  disabled,
  disabledTitle,
  submitLabel,
  showFieldLabels = true,
  testIdPrefix,
  lock,
  triggerNodeId,
}: NodesPanelInlineRunFormProps) {
  const parameters = useMemo<StartTemplateParameter[]>(() => template.parameters ?? [], [template]);
  const [parameterValues, setParameterValues] = useState<Record<string, string | number | boolean>>(() =>
    seedParameterValues(parameters),
  );

  // Re-seed when the template identity changes (e.g. editor picks a
  // different one) so stale inputs don't leak across templates.
  useEffect(() => {
    setParameterValues(seedParameterValues(parameters));
  }, [parameters]);

  const runInFlight = Boolean(triggerNodeId && lock.runInFlightIds.has(triggerNodeId));

  const handleSubmit = () => {
    const out: Record<string, unknown> = { template: template.name };
    for (const param of parameters) {
      if (!param.name || !param.type) continue;
      const coerced = coerceParameterValue(param, parameterValues[param.name]);
      if (param.type === "number" && typeof coerced === "number" && Number.isNaN(coerced)) {
        showErrorToast(`"${parameterDisplayLabel(param)}" must be a valid number`);
        return;
      }
      out[param.name] = coerced;
    }
    onSubmit(out);
  };

  return (
    <div className="w-full space-y-3" data-testid={`${testIdPrefix}-inline-form`}>
      <StartRunParameterFields
        parameters={parameters}
        parameterValues={parameterValues}
        onParameterValuesChange={setParameterValues}
        showLabels={showFieldLabels}
      />
      <div className="flex justify-end">
        <LoadingButton
          type="button"
          size="xs"
          loading={running || runInFlight}
          loadingText="Running…"
          onClick={handleSubmit}
          disabled={disabled}
          title={disabled ? disabledTitle : undefined}
          data-testid={`${testIdPrefix}-inline-submit`}
        >
          <Play className="mr-1 h-3 w-3" />
          {submitLabel?.trim() || "Run"}
        </LoadingButton>
      </div>
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
