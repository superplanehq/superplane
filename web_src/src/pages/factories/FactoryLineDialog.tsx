import type { FactoryApp, FactoryLineStep } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useCanvas } from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { Plus, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { listTriggerNodes } from "./factoryCanvasTriggers";

const RUN_APP_TYPE = "runApp";

type DraftStep = {
  name: string;
  appId: string;
  entrypoint: string;
};

interface FactoryLineDialogProps {
  open: boolean;
  mode: "create" | "edit";
  organizationId: string;
  apps: FactoryApp[];
  initialName?: string;
  initialSteps?: FactoryLineStep[];
  isSaving: boolean;
  onClose: () => void;
  onSave: (input: { name: string; steps: FactoryLineStep[] }) => Promise<void>;
}

function emptyStep(): DraftStep {
  return { name: "", appId: "", entrypoint: "" };
}

function draftStepsFromLine(steps: FactoryLineStep[] | undefined): DraftStep[] {
  if (!steps?.length) {
    return [emptyStep()];
  }

  return steps.map((step) => ({
    name: step.name ?? "",
    appId: step.app?.app ?? "",
    entrypoint: step.app?.entrypoint ?? "",
  }));
}

function draftStepsToProto(steps: DraftStep[]): FactoryLineStep[] {
  return steps.map((step) => ({
    name: step.name.trim(),
    type: RUN_APP_TYPE,
    app: {
      app: step.appId,
      entrypoint: step.entrypoint.trim(),
    },
  }));
}

export function FactoryLineDialog({
  open,
  mode,
  organizationId,
  apps,
  initialName = "",
  initialSteps,
  isSaving,
  onClose,
  onSave,
}: FactoryLineDialogProps) {
  const [name, setName] = useState("");
  const [steps, setSteps] = useState<DraftStep[]>([emptyStep()]);
  const [nameError, setNameError] = useState("");

  useEffect(() => {
    if (open) {
      setName(initialName);
      setSteps(draftStepsFromLine(initialSteps));
      setNameError("");
    }
  }, [open, initialName, initialSteps]);

  const appById = useMemo(() => {
    const map = new Map<string, FactoryApp>();
    for (const app of apps) {
      if (app.id) {
        map.set(app.id, app);
      }
    }
    return map;
  }, [apps]);

  const handleClose = () => {
    if (isSaving) {
      return;
    }
    onClose();
  };

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
      showErrorToast(getApiErrorMessage(error, mode === "create" ? "Failed to create line" : "Failed to update line"));
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          handleClose();
        }
      }}
    >
      <DialogContent showCloseButton={false} className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{mode === "create" ? "Create line" : "Edit line"}</DialogTitle>
        </DialogHeader>

        <div className="space-y-6">
          <div className="space-y-2">
            <Label htmlFor="factory-line-name-input">Line name</Label>
            <Input
              id="factory-line-name-input"
              data-testid="factory-line-name-input"
              value={name}
              onChange={(event) => {
                setName(event.target.value);
                if (nameError) {
                  setNameError("");
                }
              }}
              autoFocus
            />
            {nameError ? <p className="text-xs text-red-600">{nameError}</p> : null}
          </div>

          <div className="space-y-3">
            <div className="flex items-center justify-between gap-3">
              <Label>Steps</Label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setSteps((current) => [...current, emptyStep()])}
              >
                <Plus className="h-4 w-4" aria-hidden />
                Add step
              </Button>
            </div>

            {apps.length === 0 ? (
              <p className="text-sm text-amber-700 dark:text-amber-300">
                Create a factory app first — lines run apps owned by this factory.
              </p>
            ) : null}

            <div className="space-y-4">
              {steps.map((step, index) => (
                <FactoryLineStepEditor
                  key={index}
                  organizationId={organizationId}
                  index={index}
                  step={step}
                  apps={apps}
                  appById={appById}
                  canRemove={steps.length > 1}
                  onChange={(updated) =>
                    setSteps((current) => current.map((item, itemIndex) => (itemIndex === index ? updated : item)))
                  }
                  onRemove={() => setSteps((current) => current.filter((_, itemIndex) => itemIndex !== index))}
                />
              ))}
            </div>
          </div>
        </div>

        <DialogFooter className="flex-row justify-start gap-3 sm:justify-start">
          <LoadingButton
            onClick={() => void handleSave()}
            disabled={!name.trim() || apps.length === 0}
            loading={isSaving}
            loadingText="Saving..."
            data-testid="factory-line-save-button"
          >
            {mode === "create" ? "Create" : "Save"}
          </LoadingButton>
          <Button type="button" variant="outline" onClick={handleClose} disabled={isSaving}>
            Cancel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface FactoryLineStepEditorProps {
  organizationId: string;
  index: number;
  step: DraftStep;
  apps: FactoryApp[];
  appById: Map<string, FactoryApp>;
  canRemove: boolean;
  onChange: (step: DraftStep) => void;
  onRemove: () => void;
}

function FactoryLineStepEditor({
  organizationId,
  index,
  step,
  apps,
  appById,
  canRemove,
  onChange,
  onRemove,
}: FactoryLineStepEditorProps) {
  const { data: canvas, isLoading: canvasLoading } = useCanvas(organizationId, step.appId, {
    enabled: Boolean(step.appId),
  });
  const triggers = useMemo(() => listTriggerNodes(canvas), [canvas]);

  return (
    <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-gray-700 dark:bg-gray-800/50">
      <div className="mb-3 flex items-center justify-between gap-3">
        <p className="text-sm font-medium text-slate-900 dark:text-gray-100">Step {index + 1}</p>
        {canRemove ? (
          <Button type="button" variant="ghost" size="icon-sm" onClick={onRemove} aria-label="Remove step">
            <Trash2 className="h-4 w-4" />
          </Button>
        ) : null}
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <div className="space-y-2">
          <Label htmlFor={`factory-line-step-name-${index}`}>Step name</Label>
          <Input
            id={`factory-line-step-name-${index}`}
            value={step.name}
            onChange={(event) => onChange({ ...step, name: event.target.value })}
            placeholder="start-implementation"
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor={`factory-line-step-app-${index}`}>App</Label>
          <Select
            value={step.appId || undefined}
            onValueChange={(appId) => onChange({ ...step, appId, entrypoint: "" })}
          >
            <SelectTrigger id={`factory-line-step-app-${index}`}>
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

        <div className="space-y-2">
          <Label htmlFor={`factory-line-step-entrypoint-${index}`}>Entrypoint trigger</Label>
          <Select
            value={step.entrypoint || undefined}
            onValueChange={(entrypoint) => onChange({ ...step, entrypoint })}
            disabled={!step.appId || canvasLoading}
          >
            <SelectTrigger id={`factory-line-step-entrypoint-${index}`}>
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
    </div>
  );
}
