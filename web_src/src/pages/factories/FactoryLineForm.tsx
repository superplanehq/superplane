import type { FactoryApp, FactoryLineStep } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useCanvas } from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { Fragment, useEffect, useMemo, useState } from "react";
import { listTriggerNodes } from "./factoryCanvasTriggers";
import { LineStepAddButton, LineStepArrow, LineStepEditorShell, LineStepFlow } from "./FactoryLineStepFlow";
import { draftStepsFromLine, draftStepsToProto, emptyStep, type DraftStep } from "./factoryLineFormShared";

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
      showErrorToast("Create at least one factory app before defining a line.");
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

  const addStep = () => {
    setSteps((current) => [...current, emptyStep()]);
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
            Create a factory app first — lines run apps owned by this factory.
          </p>
        ) : null}

        <LineStepFlow variant="editor" className="pt-1">
          {steps.map((step, index) => (
            <Fragment key={index}>
              {index > 0 ? <LineStepArrow /> : null}
              <div className="w-full">
                {steps.length > 1 ? (
                  <div className="mb-2 flex justify-end">
                    <button
                      type="button"
                      onClick={() => setSteps((current) => current.filter((_, itemIndex) => itemIndex !== index))}
                      className="text-sm font-medium text-gray-500 hover:text-red-600 dark:text-gray-400 dark:hover:text-red-400"
                    >
                      Remove
                    </button>
                  </div>
                ) : null}
                <FactoryLineStepEditor
                  organizationId={organizationId}
                  index={index}
                  step={step}
                  apps={apps}
                  appById={appById}
                  onChange={(updated) =>
                    setSteps((current) => current.map((item, itemIndex) => (itemIndex === index ? updated : item)))
                  }
                />
              </div>
            </Fragment>
          ))}

          <LineStepAddButton onClick={addStep} disabled={apps.length === 0} />
        </LineStepFlow>
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

interface FactoryLineStepEditorProps {
  organizationId: string;
  index: number;
  step: DraftStep;
  apps: FactoryApp[];
  appById: Map<string, FactoryApp>;
  onChange: (step: DraftStep) => void;
}

function FactoryLineStepEditor({ organizationId, index, step, apps, appById, onChange }: FactoryLineStepEditorProps) {
  const { data: canvas, isLoading: canvasLoading } = useCanvas(organizationId, step.appId, {
    enabled: Boolean(step.appId),
  });
  const triggers = useMemo(() => listTriggerNodes(canvas), [canvas]);

  return (
    <LineStepEditorShell>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className={cn("space-y-2", stepFieldClassName)}>
          <Label htmlFor={`factory-line-step-name-${index}`}>Step name</Label>
          <Input
            id={`factory-line-step-name-${index}`}
            className={stepFieldClassName}
            value={step.name}
            onChange={(event) => onChange({ ...step, name: event.target.value })}
            placeholder="start-implementation"
          />
        </div>

        <div className={cn("space-y-2", stepFieldClassName)}>
          <Label htmlFor={`factory-line-step-app-${index}`}>App</Label>
          <Select
            value={step.appId || undefined}
            onValueChange={(appId) => onChange({ ...step, appId, entrypoint: "" })}
          >
            <SelectTrigger id={`factory-line-step-app-${index}`} className={stepFieldClassName}>
              <SelectValue placeholder="Select app" />
            </SelectTrigger>
            <SelectContent>
              {apps.map((app) => (
                <SelectItem key={app.id} value={app.id ?? ""}>
                  {app.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className={cn("space-y-2", stepFieldClassName)}>
          <Label htmlFor={`factory-line-step-entrypoint-${index}`}>Trigger</Label>
          <Select
            value={step.entrypoint || undefined}
            onValueChange={(entrypoint) => onChange({ ...step, entrypoint })}
            disabled={!step.appId || canvasLoading}
          >
            <SelectTrigger id={`factory-line-step-entrypoint-${index}`} className={stepFieldClassName}>
              <SelectValue placeholder={canvasLoading ? "Loading triggers…" : "Select trigger"} />
            </SelectTrigger>
            <SelectContent>
              {triggers.map((trigger) => (
                <SelectItem key={trigger.id} value={trigger.id ?? ""}>
                  {trigger.name || trigger.id}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {step.appId && !canvasLoading && triggers.length === 0 ? (
            <p className="text-xs text-amber-700 dark:text-amber-300">
              {appById.get(step.appId)?.name ?? "This app"} has no triggers yet.
            </p>
          ) : null}
        </div>
      </div>
    </LineStepEditorShell>
  );
}
