import React from "react";
import Editor from "@monaco-editor/react";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { showErrorToast } from "@/lib/toast";

import { StartRunParameterFields } from "./startRunParameterFields";
import {
  buildParameterFormPayload,
  initialParameterValue,
  parseJsonEventPayload,
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
    const result = useParameterForm
      ? buildParameterFormPayload(parameters, parameterValues)
      : parseJsonEventPayload(eventData);
    if ("error" in result) {
      showErrorToast(result.error);
      return;
    }

    setIsSubmitting(true);
    try {
      await onRun(result.payload);
      onClose();
    } catch {
      // Keep the modal open so users can retry with the same payload.
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="mt-1 space-y-4">
      {useParameterForm ? (
        <StartRunParameterFields
          parameters={parameters ?? []}
          parameterValues={parameterValues}
          onParameterValuesChange={setParameterValues}
        />
      ) : (
        <div className="border border-gray-200 dark:border-gray-600 rounded-md overflow-hidden">
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
