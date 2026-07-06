import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { ArrowLeft } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { IdentityFields } from "./IdentityFields";
import type { SettingsViewProps } from "./types";

export function SettingsView({ initialValues, canUpdateCanvas, isSaving, onSave, onBackToCanvas }: SettingsViewProps) {
  const [name, setName] = useState(initialValues.name);
  const [description, setDescription] = useState(initialValues.description);
  const [saveMessage, setSaveMessage] = useState<string | null>(null);

  useEffect(() => {
    setName(initialValues.name);
    setDescription(initialValues.description);
  }, [initialValues]);

  const hasChanges = useMemo(() => {
    return name !== initialValues.name || description !== initialValues.description;
  }, [description, initialValues.description, initialValues.name, name]);

  const handleSave = useCallback(async () => {
    if (!canUpdateCanvas) {
      return;
    }

    setSaveMessage(null);

    try {
      await onSave({
        name,
        description,
      });
    } catch (error) {
      const responseMessage = (error as { response?: { data?: { message?: string } } })?.response?.data?.message;
      const errorMessage = responseMessage || (error as { message?: string })?.message || "Failed to update canvas";
      setSaveMessage(errorMessage);
      setTimeout(() => setSaveMessage(null), 3000);
    }
  }, [canUpdateCanvas, description, name, onSave]);

  return (
    <div className="px-4 py-6">
      <div className="mx-auto w-full max-w-3xl space-y-6">
        {onBackToCanvas ? (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="-ml-2 gap-1 px-2 text-slate-600 hover:bg-slate-950/5 hover:text-slate-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
            onClick={onBackToCanvas}
          >
            <ArrowLeft className="h-4 w-4 shrink-0" aria-hidden />
            Back to app
          </Button>
        ) : null}
        <IdentityFields
          name={name}
          description={description}
          onNameChange={setName}
          onDescriptionChange={setDescription}
          canUpdateCanvas={canUpdateCanvas}
        />
        <div className="flex items-center gap-4">
          <LoadingButton
            type="button"
            data-testid="canvas-settings-save-changes"
            onClick={handleSave}
            disabled={!canUpdateCanvas || !hasChanges}
            loading={isSaving}
            loadingText="Saving..."
          >
            Save Changes
          </LoadingButton>
          {saveMessage ? (
            <span
              className={`text-sm ${saveMessage.includes("successfully") ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"}`}
            >
              {saveMessage}
            </span>
          ) : null}
        </div>
      </div>
    </div>
  );
}
