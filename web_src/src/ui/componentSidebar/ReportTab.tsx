import type { ComponentsIntegrationRef } from "@/api-client";
import { useCallback, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";

interface ReportTabProps {
  configuration: Record<string, unknown>;
  onSave: (
    updatedConfiguration: Record<string, unknown>,
    updatedNodeName: string,
    integrationRef?: ComponentsIntegrationRef,
  ) => void | Promise<void>;
  nodeName: string;
  readOnly?: boolean;
  configurationSaveMode?: "manual" | "auto";
  autocompleteExampleObj?: Record<string, unknown> | null;
  integrationRef?: ComponentsIntegrationRef;
}

export function ReportTab({
  configuration,
  onSave,
  nodeName,
  readOnly,
  configurationSaveMode,
  autocompleteExampleObj,
  integrationRef,
}: ReportTabProps) {
  const serverValue = (configuration?.reportTemplate as string) ?? "";
  const [value, setValue] = useState(serverValue);
  const [saving, setSaving] = useState(false);
  const isAuto = configurationSaveMode === "auto";

  const valueRef = useRef(value);
  valueRef.current = value;
  const configRef = useRef(configuration);
  configRef.current = configuration;
  const nodeNameRef = useRef(nodeName);
  nodeNameRef.current = nodeName;
  const onSaveRef = useRef(onSave);
  onSaveRef.current = onSave;
  const integrationRefRef = useRef(integrationRef);
  integrationRefRef.current = integrationRef;
  const autosaveTimerRef = useRef<number | null>(null);
  const baselineRef = useRef(serverValue);
  const savingRef = useRef(false);

  useEffect(() => {
    if (!savingRef.current) {
      baselineRef.current = serverValue;
      setValue(serverValue);
    }
  }, [serverValue]);

  const doSave = useCallback(() => {
    const current = valueRef.current;
    if (current === baselineRef.current) return;
    if (savingRef.current) return;

    savingRef.current = true;
    setSaving(true);

    const mergedConfig = { ...configRef.current, reportTemplate: current || undefined };
    const result = onSaveRef.current(mergedConfig, nodeNameRef.current, integrationRefRef.current);

    const finish = () => {
      baselineRef.current = current;
      savingRef.current = false;
      setSaving(false);
    };

    if (result instanceof Promise) {
      result.then(finish, finish);
    } else {
      finish();
    }
  }, []);

  const handleSaveRef = useRef(doSave);
  handleSaveRef.current = doSave;

  const scheduleAutosave = useCallback(() => {
    if (autosaveTimerRef.current !== null) {
      window.clearTimeout(autosaveTimerRef.current);
    }
    autosaveTimerRef.current = window.setTimeout(() => {
      autosaveTimerRef.current = null;
      handleSaveRef.current();
    }, 600);
  }, []);

  useEffect(() => {
    return () => {
      if (autosaveTimerRef.current !== null) {
        window.clearTimeout(autosaveTimerRef.current);
      }
      if (isAuto) {
        handleSaveRef.current();
      }
    };
  }, [isAuto]);

  const handleChange = useCallback(
    (nextValue: string) => {
      setValue(nextValue);
      if (isAuto) {
        scheduleAutosave();
      }
    },
    [isAuto, scheduleAutosave],
  );

  const hasChanges = value !== baselineRef.current;

  return (
    <div className="flex flex-col gap-4 p-4">
      <div>
        <label className="mb-1.5 block text-xs font-medium text-gray-600">Report template</label>
        <p className="mb-2 text-xs text-gray-400">
          Markdown template resolved after each execution and appended to the run report. Use{" "}
          <code className="rounded bg-gray-100 px-1 py-0.5 text-[10px]">{"{{ root().field }}"}</code> for trigger data
          or the full expression syntax for component data.
        </p>
        <AutoCompleteInput
          exampleObj={autocompleteExampleObj ?? null}
          value={value}
          onChange={handleChange}
          placeholder="[View workflow]({{ root().data.workflow.url }})"
          startWord="{{"
          prefix="{{ "
          suffix=" }}"
          inputSize="md"
          showValuePreview
          quickTip="Tip: type `{{` to start an expression."
          disabled={readOnly}
        />
      </div>

      {!isAuto && !readOnly && (
        <div className="flex items-center justify-end gap-2">
          {hasChanges && (
            <Button variant="ghost" size="sm" onClick={() => setValue(baselineRef.current)}>
              Discard
            </Button>
          )}
          <LoadingButton size="sm" loading={saving} disabled={!hasChanges} onClick={doSave}>
            Save
          </LoadingButton>
        </div>
      )}
    </div>
  );
}
