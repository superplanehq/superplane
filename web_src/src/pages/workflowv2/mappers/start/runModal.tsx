import React from "react";
import Editor from "@monaco-editor/react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { showErrorToast } from "@/lib/toast";
import { Checkbox } from "@/ui/checkbox";

import { coerceParameterValue, initialParameterValue, type StartTemplateParameter } from "./templatePayload";

export function StartRunModal({
  parameters,
  initialPayload,
  onRun,
  onClose,
}: {
  parameters?: StartTemplateParameter[];
  initialPayload: Record<string, unknown>;
  onRun: (payload: Record<string, unknown>) => Promise<void>;
  onClose: () => void;
}) {
  const useParameterForm = Boolean(parameters?.length);
  const [parameterValues, setParameterValues] = React.useState<Record<string, string | number | boolean>>(() => {
    const values: Record<string, string | number | boolean> = {};
    for (const param of parameters ?? []) {
      if (!param.name || !param.type) continue;
      values[param.name] = initialParameterValue(param, initialPayload);
    }
    return values;
  });
  const [eventData, setEventData] = React.useState<string>(() => JSON.stringify(initialPayload, null, 2));
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
          showErrorToast(`"${param.name}" must be a valid number`);
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
        <div className="space-y-3">
          {(parameters ?? []).map((param) => {
            if (!param.name || !param.type) return null;
            const id = `start-run-param-${param.name}`;
            return (
              <div key={param.name} className="space-y-1.5">
                <Label htmlFor={id}>{param.name}</Label>
                {param.type === "boolean" ? (
                  <Checkbox
                    id={id}
                    checked={Boolean(parameterValues[param.name])}
                    onCheckedChange={(checked) =>
                      setParameterValues((prev) => ({
                        ...prev,
                        [param.name]: checked === true,
                      }))
                    }
                  />
                ) : (
                  <Input
                    id={id}
                    type={param.type === "number" ? "number" : "text"}
                    value={String(parameterValues[param.name] ?? "")}
                    onChange={(e) =>
                      setParameterValues((prev) => ({
                        ...prev,
                        [param.name]: e.target.value,
                      }))
                    }
                  />
                )}
              </div>
            );
          })}
        </div>
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
