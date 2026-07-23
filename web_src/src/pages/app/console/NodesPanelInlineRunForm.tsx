import { useEffect, useMemo, useRef, useState } from "react";
import { Play } from "lucide-react";

import { LoadingButton } from "@/components/ui/loading-button";
import { showErrorToast } from "@/lib/toast";

import type { ConsoleTriggerLock } from "./useConsoleTriggerLock";
import { StartRunParameterFields } from "../mappers/start/startRunParameterFields";
import {
  buildParameterFormPayload,
  initialParameterValue,
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
  disabledMessage?: string;
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
  disabledMessage,
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
  const parameterConfiguration = JSON.stringify({ template: template.name, parameters });
  const previousParameterConfiguration = useRef(parameterConfiguration);

  // Re-seed for actual template configuration changes, but preserve drafts
  // when query refetches only replace the template/parameters references.
  useEffect(() => {
    if (previousParameterConfiguration.current === parameterConfiguration) return;
    previousParameterConfiguration.current = parameterConfiguration;
    setParameterValues(seedParameterValues(parameters));
  }, [parameterConfiguration, parameters]);

  const runInFlight = Boolean(triggerNodeId && lock.runInFlightIds.has(triggerNodeId));

  const handleSubmit = () => {
    const result = buildParameterFormPayload(parameters, parameterValues);
    if ("error" in result) {
      showErrorToast(result.error);
      return;
    }
    onSubmit({ template: template.name, ...result.payload });
  };

  return (
    <div className="flex h-full min-h-0 w-full flex-col gap-3" data-testid={`${testIdPrefix}-inline-form`}>
      <div className="min-h-0 flex-1">
        <StartRunParameterFields
          parameters={parameters}
          parameterValues={parameterValues}
          onParameterValuesChange={setParameterValues}
          showLabels={showFieldLabels}
          fillAvailableHeight
        />
      </div>
      <div className="flex shrink-0 flex-col gap-1">
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
        {disabledMessage ? (
          <p
            className="text-right text-[11px] text-amber-600 dark:text-amber-400"
            data-testid={`${testIdPrefix}-inline-disabled-message`}
          >
            {disabledMessage}
          </p>
        ) : null}
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
