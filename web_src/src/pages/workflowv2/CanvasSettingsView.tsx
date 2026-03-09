import { useEffect, useMemo, useState } from "react";
import { Field, Fieldset, Label } from "@/components/Fieldset/fieldset";
import { Input } from "@/components/Input/input";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/ui/tooltip";

type CanvasSettingsValues = {
  name: string;
  description: string;
  canvasVersioningEnabled: boolean;
};

interface CanvasSettingsViewProps {
  initialValues: CanvasSettingsValues;
  canUpdateCanvas: boolean;
  orgVersioningEnabled?: boolean;
  isSaving: boolean;
  onSave: (values: { name: string; description: string; canvasVersioningEnabled?: boolean }) => Promise<void>;
}

export function CanvasSettingsView({
  initialValues,
  canUpdateCanvas,
  orgVersioningEnabled,
  isSaving,
  onSave,
}: CanvasSettingsViewProps) {
  const [name, setName] = useState(initialValues.name);
  const [description, setDescription] = useState(initialValues.description);
  const [canvasVersioningEnabled, setCanvasVersioningEnabled] = useState(initialValues.canvasVersioningEnabled);
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const isVersioningEnforcedByOrganization = orgVersioningEnabled === true;
  const isVersioningToggleDisabled = !canUpdateCanvas || isVersioningEnforcedByOrganization;
  const versioningEnforcedTooltip = "Versioning is enabled by your organization settings for all canvases.";

  useEffect(() => {
    setName(initialValues.name);
    setDescription(initialValues.description);
    setCanvasVersioningEnabled(isVersioningEnforcedByOrganization ? true : initialValues.canvasVersioningEnabled);
  }, [initialValues, isVersioningEnforcedByOrganization]);

  const hasChanges = useMemo(() => {
    const effectiveCanvasVersioningEnabled = isVersioningEnforcedByOrganization ? true : canvasVersioningEnabled;
    return (
      name !== initialValues.name ||
      description !== initialValues.description ||
      effectiveCanvasVersioningEnabled !== initialValues.canvasVersioningEnabled
    );
  }, [canvasVersioningEnabled, description, initialValues, isVersioningEnforcedByOrganization, name]);

  const handleSave = async () => {
    if (!canUpdateCanvas) {
      return;
    }

    setSaveMessage(null);
    try {
      await onSave({
        name,
        description,
        canvasVersioningEnabled: isVersioningEnforcedByOrganization ? undefined : canvasVersioningEnabled,
      });
      setSaveMessage("Canvas updated successfully");
      setTimeout(() => setSaveMessage(null), 3000);
    } catch {
      setSaveMessage("Failed to update canvas");
      setTimeout(() => setSaveMessage(null), 3000);
    }
  };

  const versioningSection = (
    <Fieldset
      className={`rounded-lg border border-gray-300 bg-white p-6 ${isVersioningEnforcedByOrganization ? "opacity-60" : ""}`}
    >
      <div className="flex items-start justify-between gap-6">
        <div>
          <Label className="mb-1 block text-sm font-medium text-gray-700">Canvas Versioning</Label>
          <p className="text-sm text-gray-500">
            Manage canvas edits with drafts and publish flow. When disabled, users edit the live canvas directly.
            {isVersioningEnforcedByOrganization
              ? " Versioning is enabled by your organization settings for all canvases."
              : " This toggle controls versioning for this canvas."}
          </p>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-xs text-gray-500">
            {isVersioningEnforcedByOrganization ? "Enabled" : canvasVersioningEnabled ? "Enabled" : "Disabled"}
          </span>
          <Switch
            checked={isVersioningEnforcedByOrganization ? true : canvasVersioningEnabled}
            onCheckedChange={setCanvasVersioningEnabled}
            disabled={isVersioningToggleDisabled}
            aria-label="Toggle canvas versioning"
          />
        </div>
      </div>
    </Fieldset>
  );

  return (
    <div className="mx-auto max-w-3xl space-y-6 px-6 py-6">
      <Fieldset className="space-y-6 rounded-lg border border-gray-300 bg-white p-6">
        <Field className="space-y-3">
          <Label className="block text-sm font-medium text-gray-700">Canvas Name</Label>
          <Input
            type="text"
            value={name}
            onChange={(event) => setName(event.target.value)}
            disabled={!canUpdateCanvas}
          />
        </Field>

        <Field className="space-y-3">
          <Label className="block text-sm font-medium text-gray-700">Description</Label>
          <Textarea
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            disabled={!canUpdateCanvas}
            rows={4}
          />
        </Field>
      </Fieldset>

      {isVersioningEnforcedByOrganization ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <div aria-disabled="true" className="cursor-not-allowed">
              {versioningSection}
            </div>
          </TooltipTrigger>
          <TooltipContent side="top">{versioningEnforcedTooltip}</TooltipContent>
        </Tooltip>
      ) : (
        versioningSection
      )}

      <div className="flex items-center gap-4">
        <Button type="button" onClick={handleSave} disabled={isSaving || !canUpdateCanvas || !hasChanges}>
          {isSaving ? "Saving..." : "Save Changes"}
        </Button>
        {saveMessage ? (
          <span className={`text-sm ${saveMessage.includes("successfully") ? "text-green-600" : "text-red-600"}`}>
            {saveMessage}
          </span>
        ) : null}
      </div>
    </div>
  );
}
