import { AutoCompleteInput } from "@/components/AutoCompleteInput/AutoCompleteInput";
import { Label } from "@/components/ui/label";
import { useCallback, useEffect, useRef, useState } from "react";

const REPORT_FIELD = "reportTemplate" as const;

type ReportTabProps = {
  configuration: Record<string, unknown>;
  onSave: (nextConfiguration: Record<string, unknown>) => void | Promise<void>;
  configurationSaveMode?: "manual" | "auto";
  autocompleteExampleObj?: Record<string, unknown> | null;
  readOnly?: boolean;
};

const AUTOSAVE_MS = 600;

export function ReportTab({
  configuration,
  onSave,
  configurationSaveMode = "manual",
  autocompleteExampleObj,
  readOnly = false,
}: ReportTabProps) {
  const initial = typeof configuration[REPORT_FIELD] === "string" ? (configuration[REPORT_FIELD] as string) : "";
  const [value, setValue] = useState(initial);
  const [dirty, setDirty] = useState(false);
  const lastPersisted = useRef(initial);
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isReadOnly = readOnly ?? false;

  useEffect(() => {
    const fromParent = typeof configuration[REPORT_FIELD] === "string" ? (configuration[REPORT_FIELD] as string) : "";
    if (fromParent !== lastPersisted.current) {
      setValue(fromParent);
      setDirty(false);
      lastPersisted.current = fromParent;
    }
  }, [configuration]);

  const saveNow = useCallback(
    async (next: string) => {
      if (isReadOnly) return;
      await onSave({ ...configuration, [REPORT_FIELD]: next });
      lastPersisted.current = next;
      setDirty(false);
    },
    [configuration, isReadOnly, onSave],
  );

  const scheduleAutoSave = useCallback(
    (next: string) => {
      if (configurationSaveMode !== "auto" || isReadOnly) return;
      if (saveTimer.current) {
        clearTimeout(saveTimer.current);
      }
      saveTimer.current = setTimeout(() => {
        void saveNow(next);
      }, AUTOSAVE_MS);
    },
    [configurationSaveMode, isReadOnly, saveNow],
  );

  const handleChange = (next: string) => {
    setValue(next);
    setDirty(next !== lastPersisted.current);
    if (configurationSaveMode === "auto" && !isReadOnly) {
      scheduleAutoSave(next);
    }
  };

  useEffect(() => {
    return () => {
      if (saveTimer.current) {
        clearTimeout(saveTimer.current);
      }
    };
  }, []);

  return (
    <div className="space-y-3 p-4">
      <div>
        <Label className="text-sm font-medium text-gray-800">Report template (Markdown)</Label>
        <p className="mt-1 text-xs text-gray-500">
          Shown in Run View after this step. Use <code className="rounded bg-gray-100 px-0.5">{"{{ }}"}</code> for
          expressions. Use <code className="rounded bg-gray-100 px-0.5">$current()</code> in components for step output
          (see other configuration fields for examples).
        </p>
      </div>
      <AutoCompleteInput
        exampleObj={autocompleteExampleObj ?? null}
        value={value}
        onChange={handleChange}
        disabled={isReadOnly}
        minRows={6}
        showValuePreview
        className="min-h-[8rem] font-mono text-sm"
        inputSize="sm"
        quickTip="Expressions: try {{ $ }} to explore trigger payload or prior outputs in context of this field."
        placeholder="## What happened&#10;{{ $ }}"
      />
      {configurationSaveMode === "manual" && !isReadOnly && (
        <div className="flex items-center gap-2">
          <button
            type="button"
            className="inline-flex items-center justify-center rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-gray-800 shadow-sm transition hover:bg-slate-50"
            onClick={() => {
              setValue(lastPersisted.current);
              setDirty(false);
            }}
            disabled={!dirty}
          >
            Discard
          </button>
          <button
            type="button"
            className="inline-flex items-center justify-center rounded-md bg-slate-900 px-2.5 py-1.5 text-xs font-medium text-white transition hover:bg-slate-800 disabled:opacity-40"
            onClick={() => void saveNow(value)}
            disabled={!dirty}
          >
            Save
          </button>
        </div>
      )}
    </div>
  );
}
