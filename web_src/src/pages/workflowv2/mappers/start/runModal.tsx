import React from "react";
import Editor from "@monaco-editor/react";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { showErrorToast } from "@/lib/toast";

export function StartRunModal({
  initialPayload,
  onRun,
  onClose,
}: {
  initialPayload: Record<string, unknown>;
  onRun: (payload: Record<string, unknown>) => Promise<void>;
  onClose: () => void;
}) {
  const [eventData, setEventData] = React.useState<string>(() => JSON.stringify(initialPayload, null, 2));
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  const handleSubmit = async () => {
    let parsedData: Record<string, unknown>;
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
