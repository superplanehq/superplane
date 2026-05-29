import React from "react";
import Editor from "@monaco-editor/react";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { showErrorToast } from "@/lib/toast";

import { StartRunParameterFields } from "./startRunParameterFields";
import {
  coerceParameterValue,
  initialParameterValue,
  isValidSelectParameterValue,
  parameterDisplayLabel,
  type StartTemplateParameter,
} from "./templatePayload";

export function StartRunModal({
  parameters,
  initialPayload,
  onRun,
  onClose,
}: {
  parameters?: StartTemplateParameter[];
  initialPayload: Record<string, unknown> | string;
  onRun: (payload: Record<string, unknown>) => Promise<void>;
  onClose: () => void;
}) {
  const useParameterForm = Boolean(parameters?.length);
  const [parameterValues, setParameterValues] = React.useState<Record<string, string | number | boolean>>(() => {
    const values: Record<string, string | number | boolean> = {};
    for (const param of parameters ?? []) {
      if (!param.name || !param.type) continue;
      values[param.name] = initialParameterValue(param);
    }
    return values;
  });
  const [eventData, setEventData] = React.useState<string>(() =>
    typeof initialPayload === "string" ? initialPayload : JSON.stringify(initialPayload, null, 2),
  );
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  const handleSubmit = async () => {
    let parsedData: Record<string, unknown>;
    if (useParameterForm) {
      parsedData = {};
      for (const param of parameters ?? []) {
        if (!param.name || !param.type) continue;
        const raw = parameterValues[param.name];
        parsedData[param.name] = coerceParameterValue(param, raw);
        if (
          param.type === "number" &&
          typeof parsedData[param.name] === "number" &&
          Number.isNaN(parsedData[param.name])
        ) {
          showErrorToast(`"${parameterDisplayLabel(param)}" must be a valid number`);
          return;
        }
        if (
          param.type === "select" &&
          !isValidSelectParameterValue(param, String(parsedData[param.name] ?? ""))
        ) {
          showErrorToast(`"${parameterDisplayLabel(param)}" must be one of the configured options`);
          return;
        }
      }
    } else {
      try {
        const candidate = JSON.parse(eventData) as unknown;
        if (!candidate || typeof candidate !== "object" || Array.isArray(candidate)) {
          showErrorToast("Payload must be a JSON object");
          return;
        }
        parsedData = candidate as Record<string, unknown>;
      } catch {
        showErrorToast("Invalid JSON format");
        return;
      }
    }

    setIsSubmitting(true);
    try {
      await onRun(parsedData);
      onClose();
    } catch {
      // Keep the modal open so users can retry with the same payload.
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-4">
      {useParameterForm ? (
        <StartRunParameterFields
          parameters={parameters ?? []}
          parameterValues={parameterValues}
          onParameterValuesChange={setParameterValues}
        />
      ) : (
        <div className="border border-gray-200 dark:border-gray-700 rounded-md overflow-hidden">
          <Editor
            height="300px"
            defaultLanguage="json"
            value={eventData}
            onChange={(value) => setEventData(value || "{}")}
            options={{
              minimap: { enabled: false },
              fontSize: 13,
              lineNumbers: "on",
              scrollBeyondLastLine: false,
              automaticLayout: true,
            }}
          />
        </div>
      )}
      <div className="flex items-center justify-end gap-2">
        <Button variant="outline" onClick={onClose} disabled={isSubmitting}>
          Cancel
        </Button>
        <LoadingButton
          data-testid="emit-event-submit-button"
          loading={isSubmitting}
          loadingText="Running..."
          onClick={handleSubmit}
        >
          Run
        </LoadingButton>
      </div>
    </div>
  );
}
