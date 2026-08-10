import type { FactoryApp, FactoryLineStep } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useEffect, useMemo, useState } from "react";
import { FactoryLineStepsEditor } from "./FactoryLineStepsEditor";
import { draftStepsFromLine, draftStepsToProto, emptyStep, type DraftStep } from "./lib/factoryLineFormShared";

const stepFieldClassName = "w-full min-w-0";

interface FactoryLineFormProps {
  organizationId: string;
  apps: FactoryApp[];
  initialName?: string;
  initialSteps?: FactoryLineStep[];
  isSaving: boolean;
  submitLabel: string;
  errorMessage: string;
  onCancel?: () => void;
  onSave: (input: { name: string; steps: FactoryLineStep[] }) => Promise<void>;
}

export function FactoryLineForm({
  organizationId,
  apps,
  initialName = "",
  initialSteps,
  isSaving,
  submitLabel,
  errorMessage,
  onCancel,
  onSave,
}: FactoryLineFormProps) {
  const [name, setName] = useState("");
  const [steps, setSteps] = useState<DraftStep[]>([emptyStep()]);
  const [nameError, setNameError] = useState("");

  useEffect(() => {
    setName(initialName);
    setSteps(draftStepsFromLine(initialSteps));
    setNameError("");
  }, [initialName, initialSteps]);

  const appById = useMemo(() => {
    const map = new Map<string, FactoryApp>();
    for (const app of apps) {
      if (app.id) {
        map.set(app.id, app);
      }
    }
    return map;
  }, [apps]);

  const handleSave = async () => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      setNameError("Name is required");
      return;
    }

    if (apps.length === 0) {
      showErrorToast("Create at least one app before defining a line.");
      return;
    }

    try {
      await onSave({
        name: trimmedName,
        steps: draftStepsToProto(steps),
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, errorMessage));
    }
  };

  return (
    <div className="space-y-8">
      <div className="space-y-2">
        <Label htmlFor="factory-line-name-input">Line name</Label>
        <Input
          id="factory-line-name-input"
          data-testid="factory-line-name-input"
          className={stepFieldClassName}
          value={name}
          onChange={(event) => {
            setName(event.target.value);
            if (nameError) {
              setNameError("");
            }
          }}
        />
        {nameError ? <p className="text-xs text-red-600">{nameError}</p> : null}
      </div>

      <div className="space-y-4">
        <Label>Steps</Label>

        {apps.length === 0 ? (
          <p className="text-sm text-amber-700 dark:text-amber-300">
            Create an app first — lines run apps owned by this workspace.
          </p>
        ) : null}

        <FactoryLineStepsEditor
          organizationId={organizationId}
          steps={steps}
          apps={apps}
          appById={appById}
          onStepsChange={setSteps}
          onAddStep={() => setSteps((current) => [...current, emptyStep()])}
        />
      </div>

      <div className="flex flex-wrap gap-3 border-t border-slate-200 pt-6 dark:border-gray-700/70">
        <LoadingButton
          onClick={() => void handleSave()}
          disabled={!name.trim() || apps.length === 0}
          loading={isSaving}
          loadingText="Saving..."
          data-testid="factory-line-save-button"
        >
          {submitLabel}
        </LoadingButton>
        {onCancel ? (
          <Button type="button" variant="outline" onClick={onCancel} disabled={isSaving}>
            Cancel
          </Button>
        ) : null}
      </div>
    </div>
  );
}
