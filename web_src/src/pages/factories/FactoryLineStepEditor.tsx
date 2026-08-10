import type { FactoryApp } from "@/api-client";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useCanvas } from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
import { useMemo } from "react";
import { listTriggerNodes } from "./lib/factoryCanvasTriggers";
import type { DraftStep } from "./lib/factoryLineFormShared";
import { LineStepEditorShell } from "./FactoryLineStepFlow";

const stepFieldClassName = "w-full min-w-0";

function getTriggerMessage(state: {
  appId: string;
  appName: string;
  canvasLoading: boolean;
  canvasLoadFailed: boolean;
  triggerCount: number;
  selectedTriggerMissing: boolean;
}): string | null {
  const { appId, appName, canvasLoading, canvasLoadFailed, triggerCount, selectedTriggerMissing } = state;
  if (!appId) return null;
  if (canvasLoadFailed) return "Failed to load triggers for this app.";
  if (!canvasLoading && triggerCount === 0) return `${appName} has no triggers yet.`;
  if (selectedTriggerMissing) return "The selected trigger is no longer available. Choose another trigger.";
  return null;
}

interface FactoryLineStepEditorProps {
  organizationId: string;
  index: number;
  step: DraftStep;
  apps: FactoryApp[];
  appById: Map<string, FactoryApp>;
  onChange: (step: DraftStep) => void;
}

export function FactoryLineStepEditor({
  organizationId,
  index,
  step,
  apps,
  appById,
  onChange,
}: FactoryLineStepEditorProps) {
  const {
    data: canvas,
    isLoading: canvasLoading,
    error: canvasError,
  } = useCanvas(organizationId, step.appId, {
    enabled: Boolean(step.appId),
  });
  const triggers = useMemo(() => listTriggerNodes(canvas), [canvas]);

  const canvasLoadFailed = Boolean(canvasError && !canvas);
  const selectedTriggerMissing = Boolean(
    canvas && step.entrypoint && !triggers.some((trigger) => trigger.id === step.entrypoint),
  );
  const triggerMessage = getTriggerMessage({
    appId: step.appId,
    appName: appById.get(step.appId)?.name ?? "This app",
    canvasLoading,
    canvasLoadFailed,
    triggerCount: triggers.length,
    selectedTriggerMissing,
  });

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
            disabled={!step.appId || canvasLoading || canvasLoadFailed}
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
          {triggerMessage ? <p className="text-xs text-amber-700 dark:text-amber-300">{triggerMessage}</p> : null}
        </div>
      </div>
    </LineStepEditorShell>
  );
}
